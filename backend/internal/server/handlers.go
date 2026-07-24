package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/opentunnel/opentunnel/backend/internal/auth"
	"github.com/opentunnel/opentunnel/shared/gen/api"
	"github.com/opentunnel/opentunnel/backend/internal/storage"
)

const (
	deviceLifetime = 10 * time.Minute
	deviceInterval = 5
)

var (
	domainPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	reservedSlugs = map[string]struct{}{
		"api": {}, "app": {}, "dashboard": {}, "docs": {}, "help": {}, "status": {}, "www": {},
	}
)

func (s *Server) GetHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, api.Health{Status: api.Ok})
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	var request api.RegisterRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(string(request.Email)))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		writeProblem(w, http.StatusBadRequest, "invalid_email", "Invalid email address")
		return
	}
	if err := auth.ValidatePassword(request.Password); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_password", err.Error())
		return
	}
	passwordHash, err := auth.HashPassword(request.Password)
	if err != nil {
		writeInternal(w)
		return
	}
	user, err := s.store.CreateUser(r.Context(), email, passwordHash)
	if errors.Is(err, storage.ErrConflict) {
		writeProblem(w, http.StatusConflict, "email_exists", "An account already exists for this email")
		return
	}
	if err != nil {
		writeInternal(w)
		return
	}
	if !s.establishSession(w, r, user.ID) {
		return
	}
	writeJSON(w, http.StatusCreated, apiUser(user))
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	var request api.LoginRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	user, err := s.store.UserByEmail(r.Context(), string(request.Email))
	if err != nil || !auth.VerifyPassword(user.PasswordHash, request.Password) {
		writeProblem(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}
	if !s.establishSession(w, r, user.ID) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	if token, ok := r.Context().Value(sessionTokenContextKey).(string); ok {
		if err := s.store.RevokeSession(r.Context(), auth.Digest(token)); err != nil {
			writeInternal(w)
			return
		}
	}
	s.clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, apiUser(userFromContext(r.Context())))
}

func (s *Server) CreateDeviceAuthorization(w http.ResponseWriter, r *http.Request) {
	deviceCode, deviceHash, err := auth.NewSecret(32)
	if err != nil {
		writeInternal(w)
		return
	}
	var authorization storage.DeviceAuthorization
	for attempts := 0; attempts < 3; attempts++ {
		userCode, codeErr := auth.NewUserCode()
		if codeErr != nil {
			writeInternal(w)
			return
		}
		authorization, err = s.store.CreateDeviceAuthorization(
			r.Context(),
			deviceHash,
			userCode,
			deviceInterval,
			time.Now().Add(deviceLifetime),
		)
		if !errors.Is(err, storage.ErrConflict) {
			break
		}
	}
	if err != nil {
		writeInternal(w)
		return
	}
	verificationURI := strings.TrimRight(s.config.FrontendURL, "/") + "/device"
	writeJSON(w, http.StatusCreated, api.DeviceAuthorization{
		DeviceCode:              deviceCode,
		UserCode:                authorization.UserCode,
		VerificationUri:         verificationURI,
		VerificationUriComplete: verificationURI + "?user_code=" + url.QueryEscape(authorization.UserCode),
		ExpiresIn:               int(deviceLifetime.Seconds()),
		Interval:                authorization.IntervalSeconds,
	})
}

func (s *Server) ApproveDeviceAuthorization(w http.ResponseWriter, r *http.Request, userCode api.UserCode) {
	if _, ok := r.Context().Value(sessionTokenContextKey).(string); !ok {
		writeProblem(w, http.StatusForbidden, "web_session_required", "A web session is required")
		return
	}
	user := userFromContext(r.Context())
	if err := s.store.ApproveDeviceAuthorization(r.Context(), user.ID, userCode); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeProblem(w, http.StatusNotFound, "invalid_user_code", "Device code is invalid or expired")
			return
		}
		writeInternal(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ExchangeDeviceCode(w http.ResponseWriter, r *http.Request) {
	var request api.DeviceTokenRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	accessToken, tokenHash, err := auth.NewSecret(32)
	if err != nil {
		writeInternal(w)
		return
	}
	_, err = s.store.ExchangeDeviceCode(r.Context(), auth.Digest(request.DeviceCode), tokenHash)
	switch {
	case errors.Is(err, storage.ErrPending):
		w.Header().Set("Retry-After", "5")
		writeJSON(w, http.StatusAccepted, map[string]string{"code": "authorization_pending"})
	case errors.Is(err, storage.ErrSlowDown):
		w.Header().Set("Retry-After", "10")
		writeJSON(w, http.StatusAccepted, map[string]string{"code": "slow_down"})
	case errors.Is(err, storage.ErrExpired), errors.Is(err, storage.ErrNotFound):
		writeProblem(w, http.StatusBadRequest, "expired_token", "Device code is invalid or expired")
	case err != nil:
		writeInternal(w)
	default:
		writeJSON(w, http.StatusOK, api.DeviceToken{AccessToken: accessToken, TokenType: api.Bearer})
	}
}

func (s *Server) ListTokens(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	tokens, err := s.store.ListTokens(r.Context(), user.ID)
	if err != nil {
		writeInternal(w)
		return
	}
	response := make([]api.TokenSummary, 0, len(tokens))
	for _, token := range tokens {
		response = append(response, api.TokenSummary{
			Id: token.ID, Name: token.Name, LastUsedAt: token.LastUsedAt, CreatedAt: token.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) RevokeToken(w http.ResponseWriter, r *http.Request, tokenID openapi_types.UUID) {
	user := userFromContext(r.Context())
	if err := s.store.RevokeToken(r.Context(), user.ID, tokenID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeProblem(w, http.StatusNotFound, "token_not_found", "Token not found")
			return
		}
		writeInternal(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) CreateDomain(w http.ResponseWriter, r *http.Request) {
	var request api.CreateDomainRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	slug := strings.ToLower(strings.TrimSpace(request.Slug))
	if !domainPattern.MatchString(slug) {
		writeProblem(w, http.StatusBadRequest, "invalid_slug", "Domain slug is invalid")
		return
	}
	if _, reserved := reservedSlugs[slug]; reserved {
		writeProblem(w, http.StatusConflict, "domain_unavailable", "Domain is unavailable")
		return
	}
	user := userFromContext(r.Context())
	domain, err := s.store.CreateDomain(r.Context(), user.ID, slug)
	if errors.Is(err, storage.ErrConflict) {
		writeProblem(w, http.StatusConflict, "domain_unavailable", "Domain is unavailable")
		return
	}
	if err != nil {
		writeInternal(w)
		return
	}
	writeJSON(w, http.StatusCreated, s.apiDomain(domain))
}

func (s *Server) ListDomains(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	domains, err := s.store.ListDomains(r.Context(), user.ID)
	if err != nil {
		writeInternal(w)
		return
	}
	response := make([]api.Domain, 0, len(domains))
	for _, domain := range domains {
		response = append(response, s.apiDomain(domain))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) GetDomain(w http.ResponseWriter, r *http.Request, domainID api.DomainId) {
	user := userFromContext(r.Context())
	domain, err := s.store.Domain(r.Context(), user.ID, domainID)
	if errors.Is(err, storage.ErrNotFound) {
		writeProblem(w, http.StatusNotFound, "domain_not_found", "Domain not found")
		return
	}
	if err != nil {
		writeInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, s.apiDomain(domain))
}

func (s *Server) DeleteDomain(w http.ResponseWriter, r *http.Request, domainID api.DomainId) {
	user := userFromContext(r.Context())
	if err := s.store.DeleteDomain(r.Context(), user.ID, domainID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeProblem(w, http.StatusNotFound, "domain_not_found", "Domain not found")
			return
		}
		writeInternal(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) establishSession(w http.ResponseWriter, r *http.Request, userID uuid.UUID) bool {
	sessionToken, sessionHash, err := auth.NewSecret(32)
	if err != nil {
		writeInternal(w)
		return false
	}
	csrfToken, csrfHash, err := auth.NewSecret(32)
	if err != nil {
		writeInternal(w)
		return false
	}
	expiresAt := time.Now().Add(s.config.SessionLifetime)
	if err := s.store.CreateSession(r.Context(), userID, sessionHash, csrfHash, expiresAt); err != nil {
		writeInternal(w)
		return false
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: sessionToken, Path: "/", HttpOnly: true,
		Domain: s.config.CookieDomain, Secure: s.config.SecureCookies,
		SameSite: http.SameSiteLaxMode, Expires: expiresAt,
	})
	http.SetCookie(w, &http.Cookie{
		Name: csrfCookie, Value: csrfToken, Path: "/", HttpOnly: false,
		Domain: s.config.CookieDomain, Secure: s.config.SecureCookies,
		SameSite: http.SameSiteLaxMode, Expires: expiresAt,
	})
	return true
}

func (s *Server) clearAuthCookies(w http.ResponseWriter) {
	for _, name := range []string{sessionCookie, csrfCookie} {
		http.SetCookie(w, &http.Cookie{
			Name: name, Path: "/", Domain: s.config.CookieDomain,
			HttpOnly: name == sessionCookie, Secure: s.config.SecureCookies,
			SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0),
		})
	}
}

func (s *Server) apiDomain(domain storage.Domain) api.Domain {
	status := api.Offline
	if domain.Online {
		status = api.Online
	}
	return api.Domain{
		Id: domain.ID, Slug: domain.Slug, Hostname: domain.Slug + "." + s.config.PublicHost,
		Status: status, CreatedAt: domain.CreatedAt,
	}
}

func apiUser(user storage.User) api.User {
	return api.User{Id: user.ID, Email: openapi_types.Email(user.Email), CreatedAt: user.CreatedAt}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.Problem{
		Type: "about:blank", Title: http.StatusText(status), Status: status, Code: &code, Detail: &detail,
	})
}

func writeInternal(w http.ResponseWriter) {
	writeProblem(w, http.StatusInternalServerError, "internal_error", "An internal error occurred")
}
