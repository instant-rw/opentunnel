package storage

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	ErrExpired  = errors.New("expired")
	ErrPending  = errors.New("authorization pending")
	ErrSlowDown = errors.New("slow down")
)

type Store struct {
	pool *pgxpool.Pool
}

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type Session struct {
	ID       uuid.UUID
	User     User
	CSRFHash []byte
}

type DeviceAuthorization struct {
	ID              uuid.UUID
	UserCode        string
	IntervalSeconds int
	ExpiresAt       time.Time
}

type Domain struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Slug      string
	CreatedAt time.Time
	Online    bool
}

type Token struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Name       string
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

type TunnelAuthorization struct {
	Domain Domain
	Token  Token
}

type Header struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type CapturedRequest struct {
	ID                    uuid.UUID
	DomainID              uuid.UUID
	Method                string
	Path                  string
	Query                 string
	RequestHeaders        []Header
	RequestBody           []byte
	RequestBodySize       int64
	RequestBodyTruncated  bool
	ResponseStatus        *int
	ResponseHeaders       []Header
	ResponseBody          []byte
	ResponseBodySize      *int64
	ResponseBodyTruncated bool
	DurationMS            *int64
	ReceivedAt            time.Time
}

type ReplayAttempt struct {
	ID             uuid.UUID
	RequestID      uuid.UUID
	Status         string
	Error          *string
	ResponseStatus *int
	DurationMS     *int64
	CreatedAt      time.Time
	CompletedAt    *time.Time
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var user User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users(id, email, password_hash) VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, created_at`,
		uuid.New(), strings.ToLower(strings.TrimSpace(email)), passwordHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	return user, translate(err)
}

func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at FROM users WHERE email = lower($1)`,
		strings.TrimSpace(email),
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	return user, translate(err)
}

func (s *Store) CreateSession(
	ctx context.Context,
	userID uuid.UUID,
	tokenHash, csrfHash []byte,
	expiresAt time.Time,
) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO web_sessions(id, user_id, token_hash, csrf_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5)`,
		uuid.New(), userID, tokenHash, csrfHash, expiresAt,
	)
	return translate(err)
}

func (s *Store) SessionByToken(ctx context.Context, tokenHash []byte) (Session, error) {
	var session Session
	err := s.pool.QueryRow(ctx, `
		UPDATE web_sessions s SET last_seen_at = now()
		FROM users u
		WHERE s.token_hash = $1 AND s.user_id = u.id AND s.expires_at > now()
		RETURNING s.id, s.csrf_hash, u.id, u.email, u.password_hash, u.created_at`,
		tokenHash,
	).Scan(
		&session.ID,
		&session.CSRFHash,
		&session.User.ID,
		&session.User.Email,
		&session.User.PasswordHash,
		&session.User.CreatedAt,
	)
	return session, translate(err)
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM web_sessions WHERE token_hash = $1", tokenHash)
	return err
}

func (s *Store) UserByCLIToken(ctx context.Context, tokenHash []byte) (User, error) {
	var user User
	err := s.pool.QueryRow(ctx, `
		UPDATE cli_tokens t SET last_used_at = now()
		FROM users u
		WHERE t.token_hash = $1 AND t.user_id = u.id AND t.revoked_at IS NULL
		  AND (t.expires_at IS NULL OR t.expires_at > now())
		RETURNING u.id, u.email, u.password_hash, u.created_at`,
		tokenHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	return user, translate(err)
}

func (s *Store) AuthorizeTunnel(
	ctx context.Context,
	tokenHash []byte,
	domainID uuid.UUID,
) (TunnelAuthorization, error) {
	var authorization TunnelAuthorization
	err := s.pool.QueryRow(ctx, `
		UPDATE cli_tokens t SET last_used_at = now()
		FROM domains d
		WHERE t.token_hash = $1
		  AND t.user_id = d.user_id
		  AND d.id = $2
		  AND t.revoked_at IS NULL
		  AND (t.expires_at IS NULL OR t.expires_at > now())
		RETURNING d.id, d.user_id, d.slug, d.created_at,
		          t.id, t.user_id, t.name, t.last_used_at, t.created_at`,
		tokenHash, domainID,
	).Scan(
		&authorization.Domain.ID,
		&authorization.Domain.UserID,
		&authorization.Domain.Slug,
		&authorization.Domain.CreatedAt,
		&authorization.Token.ID,
		&authorization.Token.UserID,
		&authorization.Token.Name,
		&authorization.Token.LastUsedAt,
		&authorization.Token.CreatedAt,
	)
	return authorization, translate(err)
}

func (s *Store) ConnectTunnelSession(
	ctx context.Context,
	domainID, tokenID uuid.UUID,
	staleBefore time.Time,
) (uuid.UUID, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE tunnel_sessions
		SET disconnected_at = now()
		WHERE domain_id = $1
		  AND disconnected_at IS NULL
		  AND last_heartbeat_at < $2`,
		domainID, staleBefore,
	); err != nil {
		return uuid.Nil, err
	}
	sessionID := uuid.New()
	if _, err := tx.Exec(ctx, `
		INSERT INTO tunnel_sessions(id, domain_id, cli_token_id)
		VALUES ($1, $2, $3)`,
		sessionID, domainID, tokenID,
	); err != nil {
		return uuid.Nil, translate(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return sessionID, nil
}

func (s *Store) HeartbeatTunnelSession(ctx context.Context, sessionID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE tunnel_sessions
		SET last_heartbeat_at = now()
		WHERE id = $1 AND disconnected_at IS NULL`,
		sessionID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DisconnectTunnelSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tunnel_sessions
		SET disconnected_at = COALESCE(disconnected_at, now())
		WHERE id = $1`,
		sessionID,
	)
	return err
}

func (s *Store) CreateDeviceAuthorization(
	ctx context.Context,
	deviceHash []byte,
	userCode string,
	interval int,
	expiresAt time.Time,
) (DeviceAuthorization, error) {
	var authorization DeviceAuthorization
	err := s.pool.QueryRow(ctx, `
		INSERT INTO device_authorizations(
			id, device_code_hash, user_code, interval_seconds, expires_at
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_code, interval_seconds, expires_at`,
		uuid.New(), deviceHash, userCode, interval, expiresAt,
	).Scan(
		&authorization.ID,
		&authorization.UserCode,
		&authorization.IntervalSeconds,
		&authorization.ExpiresAt,
	)
	return authorization, translate(err)
}

func (s *Store) ApproveDeviceAuthorization(ctx context.Context, userID uuid.UUID, userCode string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE device_authorizations
		SET status = 'approved', user_id = $1, approved_at = now()
		WHERE user_code = upper($2) AND status = 'pending' AND expires_at > now()`,
		userID, userCode,
	)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ExchangeDeviceCode(
	ctx context.Context,
	deviceHash, tokenHash []byte,
) (string, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id uuid.UUID
	var userID *uuid.UUID
	var status string
	var interval int
	var lastPolledAt *time.Time
	var expiresAt time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, user_id, status, interval_seconds, last_polled_at, expires_at
		FROM device_authorizations WHERE device_code_hash = $1 FOR UPDATE`,
		deviceHash,
	).Scan(&id, &userID, &status, &interval, &lastPolledAt, &expiresAt)
	if err != nil {
		return "", translate(err)
	}
	if time.Now().After(expiresAt) {
		return "", ErrExpired
	}
	switch status {
	case "pending":
		// Rate limiting applies only while pending. Approved exchanges must not be
		// delayed — a slow_down death spiral previously blocked token issuance.
		now := time.Now()
		if lastPolledAt != nil && now.Before(lastPolledAt.Add(time.Duration(interval)*time.Second)) {
			if _, err := tx.Exec(ctx, "UPDATE device_authorizations SET interval_seconds = interval_seconds + 5 WHERE id = $1", id); err != nil {
				return "", err
			}
			if err := tx.Commit(ctx); err != nil {
				return "", err
			}
			return "", ErrSlowDown
		}
		if _, err := tx.Exec(ctx, "UPDATE device_authorizations SET last_polled_at = now() WHERE id = $1", id); err != nil {
			return "", err
		}
		if err := tx.Commit(ctx); err != nil {
			return "", err
		}
		return "", ErrPending
	case "approved":
		tokenID := uuid.New()
		if _, err := tx.Exec(ctx, `
			INSERT INTO cli_tokens(id, user_id, token_hash) VALUES ($1, $2, $3)`,
			tokenID, *userID, tokenHash,
		); err != nil {
			return "", translate(err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE device_authorizations SET status = 'consumed', consumed_at = now() WHERE id = $1`,
			id,
		); err != nil {
			return "", err
		}
		if err := tx.Commit(ctx); err != nil {
			return "", err
		}
		return tokenID.String(), nil
	default:
		return "", ErrNotFound
	}
}

func (s *Store) ListTokens(ctx context.Context, userID uuid.UUID) ([]Token, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, name, last_used_at, created_at
		FROM cli_tokens WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Token, error) {
		var token Token
		err := row.Scan(&token.ID, &token.UserID, &token.Name, &token.LastUsedAt, &token.CreatedAt)
		return token, err
	})
}

func (s *Store) RevokeToken(ctx context.Context, userID, tokenID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE cli_tokens SET revoked_at = now()
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`,
		tokenID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateDomain(ctx context.Context, userID uuid.UUID, slug string) (Domain, error) {
	var domain Domain
	err := s.pool.QueryRow(ctx, `
		INSERT INTO domains(id, user_id, slug) VALUES ($1, $2, lower($3))
		RETURNING id, user_id, slug, created_at`,
		uuid.New(), userID, slug,
	).Scan(&domain.ID, &domain.UserID, &domain.Slug, &domain.CreatedAt)
	return domain, translate(err)
}

func (s *Store) ListDomains(ctx context.Context, userID uuid.UUID) ([]Domain, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT d.id, d.user_id, d.slug, d.created_at,
			EXISTS(
				SELECT 1 FROM tunnel_sessions t
				WHERE t.domain_id = d.id
				  AND t.disconnected_at IS NULL
				  AND t.last_heartbeat_at >= now() - interval '30 seconds'
			)
		FROM domains d WHERE d.user_id = $1 ORDER BY d.created_at`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, scanDomain)
}

func (s *Store) Domain(ctx context.Context, userID, domainID uuid.UUID) (Domain, error) {
	var domain Domain
	err := s.pool.QueryRow(ctx, `
		SELECT d.id, d.user_id, d.slug, d.created_at,
			EXISTS(
				SELECT 1 FROM tunnel_sessions t
				WHERE t.domain_id = d.id
				  AND t.disconnected_at IS NULL
				  AND t.last_heartbeat_at >= now() - interval '30 seconds'
			)
		FROM domains d WHERE d.id = $1 AND d.user_id = $2`,
		domainID, userID,
	).Scan(&domain.ID, &domain.UserID, &domain.Slug, &domain.CreatedAt, &domain.Online)
	return domain, translate(err)
}

func (s *Store) DeleteDomain(ctx context.Context, userID, domainID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM domains WHERE id = $1 AND user_id = $2", domainID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateCapturedRequest(
	ctx context.Context,
	request CapturedRequest,
	maxStored int,
) (CapturedRequest, error) {
	headers, err := json.Marshal(request.RequestHeaders)
	if err != nil {
		return CapturedRequest{}, err
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO requests(
			id, domain_id, method, path, query, request_headers, request_body,
			request_body_size, request_body_truncated, received_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING received_at`,
		request.ID, request.DomainID, request.Method, request.Path, request.Query,
		headers, request.RequestBody, request.RequestBodySize,
		request.RequestBodyTruncated, request.ReceivedAt,
	).Scan(&request.ReceivedAt)
	if err != nil {
		return CapturedRequest{}, translate(err)
	}
	if maxStored > 0 {
		_, err = s.pool.Exec(ctx, `
			DELETE FROM requests
			WHERE domain_id = $1 AND id IN (
				SELECT id FROM requests WHERE domain_id = $1
				ORDER BY received_at DESC, id DESC OFFSET $2
			)`,
			request.DomainID, maxStored,
		)
	}
	return request, err
}

func (s *Store) CompleteCapturedRequest(
	ctx context.Context,
	requestID uuid.UUID,
	status int,
	headers []Header,
	body []byte,
	bodySize int64,
	truncated bool,
	durationMS int64,
) error {
	encodedHeaders, err := json.Marshal(headers)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE requests SET
			response_status = $2, response_headers = $3, response_body = $4,
			response_body_size = $5, response_body_truncated = $6, duration_ms = $7
		WHERE id = $1`,
		requestID, status, encodedHeaders, body, bodySize, truncated, durationMS,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateCapturedRequestBody(
	ctx context.Context,
	requestID uuid.UUID,
	body []byte,
	bodySize int64,
	truncated bool,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE requests SET request_body = $2, request_body_size = $3,
			request_body_truncated = $4 WHERE id = $1`,
		requestID, body, bodySize, truncated,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListCapturedRequests(
	ctx context.Context,
	userID, domainID uuid.UUID,
	before *time.Time,
	limit int,
) ([]CapturedRequest, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.domain_id, r.method, r.path, r.query, r.request_headers,
			r.request_body, COALESCE(r.request_body_size, 0), r.request_body_truncated,
			r.response_status, COALESCE(r.response_headers, '[]'::jsonb),
			r.response_body, r.response_body_size, r.response_body_truncated,
			r.duration_ms, r.received_at
		FROM requests r
		JOIN domains d ON d.id = r.domain_id
		WHERE d.user_id = $1 AND d.id = $2 AND ($3::timestamptz IS NULL OR r.received_at < $3)
		ORDER BY r.received_at DESC, r.id DESC LIMIT $4`,
		userID, domainID, before, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, scanCapturedRequest)
}

func (s *Store) CapturedRequest(
	ctx context.Context,
	userID, requestID uuid.UUID,
) (CapturedRequest, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT r.id, r.domain_id, r.method, r.path, r.query, r.request_headers,
			r.request_body, COALESCE(r.request_body_size, 0), r.request_body_truncated,
			r.response_status, COALESCE(r.response_headers, '[]'::jsonb),
			r.response_body, r.response_body_size, r.response_body_truncated,
			r.duration_ms, r.received_at
		FROM requests r
		JOIN domains d ON d.id = r.domain_id
		WHERE r.id = $1 AND d.user_id = $2`,
		requestID, userID,
	)
	return scanCapturedRow(row)
}

func (s *Store) CreateReplayAttempt(
	ctx context.Context,
	userID, requestID uuid.UUID,
) (ReplayAttempt, error) {
	var replay ReplayAttempt
	err := s.pool.QueryRow(ctx, `
		INSERT INTO replay_attempts(id, request_id)
		SELECT $1, r.id FROM requests r
		JOIN domains d ON d.id = r.domain_id
		WHERE r.id = $2 AND d.user_id = $3
		RETURNING id, request_id, status::text, created_at`,
		uuid.New(), requestID, userID,
	).Scan(&replay.ID, &replay.RequestID, &replay.Status, &replay.CreatedAt)
	return replay, translate(err)
}

func (s *Store) UpdateReplayAttempt(
	ctx context.Context,
	replayID uuid.UUID,
	status string,
	errorDetail *string,
	responseStatus *int,
	durationMS *int64,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE replay_attempts SET status = $2, error = $3, response_status = $4,
			duration_ms = $5,
			completed_at = CASE WHEN $2 IN ('succeeded', 'failed') THEN now() ELSE NULL END
		WHERE id = $1`,
		replayID, status, errorDetail, responseStatus, durationMS,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanCapturedRequest(row pgx.CollectableRow) (CapturedRequest, error) {
	return scanCapturedRow(row)
}

type rowScanner interface {
	Scan(...any) error
}

func scanCapturedRow(row rowScanner) (CapturedRequest, error) {
	var request CapturedRequest
	var requestHeaders []byte
	var responseHeaders []byte
	err := row.Scan(
		&request.ID, &request.DomainID, &request.Method, &request.Path, &request.Query,
		&requestHeaders, &request.RequestBody, &request.RequestBodySize,
		&request.RequestBodyTruncated, &request.ResponseStatus, &responseHeaders,
		&request.ResponseBody, &request.ResponseBodySize, &request.ResponseBodyTruncated,
		&request.DurationMS, &request.ReceivedAt,
	)
	if err != nil {
		return CapturedRequest{}, translate(err)
	}
	if err := json.Unmarshal(requestHeaders, &request.RequestHeaders); err != nil {
		return CapturedRequest{}, err
	}
	if err := json.Unmarshal(responseHeaders, &request.ResponseHeaders); err != nil {
		return CapturedRequest{}, err
	}
	return request, nil
}

func scanDomain(row pgx.CollectableRow) (Domain, error) {
	var domain Domain
	err := row.Scan(&domain.ID, &domain.UserID, &domain.Slug, &domain.CreatedAt, &domain.Online)
	return domain, err
}

func translate(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return ErrConflict
	}
	return err
}
