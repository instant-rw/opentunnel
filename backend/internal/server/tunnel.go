package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/opentunnel/opentunnel/backend/internal/auth"
	tunnelv1 "github.com/opentunnel/opentunnel/shared/gen/tunnel/v1"
	"github.com/opentunnel/opentunnel/backend/internal/storage"
	"google.golang.org/protobuf/proto"
)

const tunnelProtocolVersion = 1

type tunnelStore interface {
	AuthorizeTunnel(context.Context, []byte, uuid.UUID) (storage.TunnelAuthorization, error)
	ConnectTunnelSession(context.Context, uuid.UUID, uuid.UUID, time.Time) (uuid.UUID, error)
	HeartbeatTunnelSession(context.Context, uuid.UUID) error
	DisconnectTunnelSession(context.Context, uuid.UUID) error
}

type sessionRegistry struct {
	mu     sync.RWMutex
	bySlug map[string]*tunnelSession
}

func newSessionRegistry() *sessionRegistry {
	return &sessionRegistry{bySlug: make(map[string]*tunnelSession)}
}

func (r *sessionRegistry) add(slug string, session *tunnelSession) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.bySlug[slug]; exists {
		return false
	}
	r.bySlug[slug] = session
	return true
}

func (r *sessionRegistry) remove(slug string, session *tunnelSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bySlug[slug] == session {
		delete(r.bySlug, slug)
	}
}

func (r *sessionRegistry) get(slug string) *tunnelSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.bySlug[slug]
}

func (r *sessionRegistry) getByDomainID(domainID uuid.UUID) *tunnelSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, session := range r.bySlug {
		if session.domain.ID == domainID {
			return session
		}
	}
	return nil
}

func (r *sessionRegistry) closeAll() {
	r.mu.RLock()
	sessions := make([]*tunnelSession, 0, len(r.bySlug))
	for _, session := range r.bySlug {
		sessions = append(sessions, session)
	}
	r.mu.RUnlock()
	for _, session := range sessions {
		session.close()
	}
}

type responseEvent struct {
	message proto.Message
}

type requestState struct {
	events        chan responseEvent
	requestWindow *creditWindow
}

type creditWindow struct {
	available uint64
	updates   chan uint64
}

func newCreditWindow(depth int) *creditWindow {
	return &creditWindow{updates: make(chan uint64, depth)}
}

func (w *creditWindow) add(credit uint64) bool {
	if credit == 0 {
		return false
	}
	select {
	case w.updates <- credit:
		return true
	default:
		return false
	}
}

func (w *creditWindow) take(ctx context.Context, bytes uint64) error {
	for w.available < bytes {
		select {
		case credit := <-w.updates:
			w.available += credit
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	w.available -= bytes
	return nil
}

type tunnelSession struct {
	id       uuid.UUID
	domain   storage.Domain
	server   *Server
	conn     *websocket.Conn
	send     chan *tunnelv1.Envelope
	done     chan struct{}
	closeOne sync.Once
	sequence uint64
	lastPong atomic.Int64

	requestsMu sync.Mutex
	requests   map[string]*requestState
}

var tunnelUpgrader = websocket.Upgrader{
	ReadBufferSize:  64 << 10,
	WriteBufferSize: 64 << 10,
	CheckOrigin: func(*http.Request) bool {
		return true
	},
}

func (s *Server) handleTunnel(w http.ResponseWriter, r *http.Request) {
	if s.tunnels == nil {
		writeProblem(w, http.StatusServiceUnavailable, "tunnel_unavailable", "Tunnel service is unavailable")
		return
	}
	token, err := auth.ParseBearer(r.Header.Get("Authorization"))
	if err != nil {
		writeProblem(w, http.StatusUnauthorized, "invalid_token", "Invalid token")
		return
	}
	domainID, err := uuid.Parse(r.URL.Query().Get("domainId"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_domain", "A valid domainId is required")
		return
	}
	authorization, err := s.tunnels.AuthorizeTunnel(r.Context(), auth.Digest(token), domainID)
	if err != nil {
		writeProblem(w, http.StatusForbidden, "domain_forbidden", "Token does not own this domain")
		return
	}

	connection, err := tunnelUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	connection.SetReadLimit(int64(s.config.MaxChunkBytes + 1<<20))
	_ = connection.SetReadDeadline(time.Now().Add(s.config.HeartbeatGrace))
	hello, err := readTunnelEnvelope(connection)
	if err != nil {
		_ = connection.Close()
		return
	}
	_ = connection.SetReadDeadline(time.Time{})
	clientHello := hello.GetClientHello()
	if hello.ProtocolVersion != tunnelProtocolVersion || clientHello == nil || clientHello.DomainId != domainID.String() {
		writeProtocolError(connection, tunnelv1.ProtocolErrorCode_PROTOCOL_ERROR_CODE_UNSUPPORTED_VERSION, "invalid tunnel hello")
		_ = connection.Close()
		return
	}

	sessionID, err := s.tunnels.ConnectTunnelSession(
		r.Context(),
		domainID,
		authorization.Token.ID,
		time.Now().Add(-s.config.HeartbeatGrace),
	)
	if err != nil {
		code := tunnelv1.ProtocolErrorCode_PROTOCOL_ERROR_CODE_INTERNAL
		detail := "could not register tunnel"
		if errors.Is(err, storage.ErrConflict) {
			code = tunnelv1.ProtocolErrorCode_PROTOCOL_ERROR_CODE_DUPLICATE_SESSION
			detail = "domain already has an active tunnel"
		}
		writeProtocolError(connection, code, detail)
		_ = connection.Close()
		return
	}

	session := &tunnelSession{
		id:       sessionID,
		domain:   authorization.Domain,
		server:   s,
		conn:     connection,
		send:     make(chan *tunnelv1.Envelope, s.config.QueueDepth),
		done:     make(chan struct{}),
		requests: make(map[string]*requestState),
	}
	session.lastPong.Store(time.Now().UnixNano())
	if !s.registry.add(authorization.Domain.Slug, session) {
		_ = s.tunnels.DisconnectTunnelSession(context.Background(), sessionID)
		writeProtocolError(connection, tunnelv1.ProtocolErrorCode_PROTOCOL_ERROR_CODE_DUPLICATE_SESSION, "domain already has an active tunnel")
		_ = connection.Close()
		return
	}
	s.events.publish(authorization.Domain.ID, domainEvent{name: "domain.status", data: s.apiDomain(storage.Domain{
		ID: authorization.Domain.ID, UserID: authorization.Domain.UserID, Slug: authorization.Domain.Slug,
		CreatedAt: authorization.Domain.CreatedAt, Online: true,
	})})

	go session.writeLoop()
	session.enqueue(context.Background(), &tunnelv1.Envelope{
		ProtocolVersion: tunnelProtocolVersion,
		Message: &tunnelv1.Envelope_ServerHello{ServerHello: &tunnelv1.ServerHello{
			SessionId:                sessionID.String(),
			HeartbeatIntervalSeconds: uint32(max(1, int(s.config.Heartbeat.Seconds()))),
			MaxInFlightRequests:      uint32(s.config.MaxInFlight),
			MaxChunkBytes:            uint32(s.config.MaxChunkBytes),
		}},
	})
	session.readLoop()
	session.close()
}

func (s *Server) publicSlug(rawHost string) (string, bool) {
	host := normalizedHost(rawHost)
	suffix := "." + strings.ToLower(strings.TrimSuffix(s.config.PublicHost, "."))
	if !strings.HasSuffix(host, suffix) {
		return "", false
	}
	slug := strings.TrimSuffix(host, suffix)
	return slug, domainPattern.MatchString(slug) && !strings.Contains(slug, ".")
}

func (s *Server) isPublicHostCandidate(rawHost string) bool {
	host := normalizedHost(rawHost)
	suffix := "." + strings.ToLower(strings.TrimSuffix(s.config.PublicHost, "."))
	return strings.HasSuffix(host, suffix)
}

func normalizedHost(rawHost string) string {
	host := strings.ToLower(strings.TrimSuffix(rawHost, "."))
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		return parsed
	}
	return host
}

func (s *Server) handlePublicRequest(w http.ResponseWriter, r *http.Request) {
	slug, ok := s.publicSlug(r.Host)
	if !ok {
		http.NotFound(w, r)
		return
	}
	session := s.registry.get(slug)
	if session == nil {
		writeTunnelOffline(w)
		return
	}
	tracked := &trackingResponseWriter{ResponseWriter: w}
	if err := session.proxy(tracked, r); err != nil && !tracked.wroteHeader && !errors.Is(err, context.Canceled) {
		writeTunnelOffline(w)
	}
}

type trackingResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *trackingResponseWriter) WriteHeader(status int) {
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *trackingResponseWriter) Write(body []byte) (int, error) {
	w.wroteHeader = true
	return w.ResponseWriter.Write(body)
}

func (w *trackingResponseWriter) Flush() {
	w.wroteHeader = true
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (s *tunnelSession) proxy(w http.ResponseWriter, r *http.Request) error {
	_, err := s.proxyRequest(w, r, false, true)
	return err
}

type proxyResult struct {
	status     *int
	durationMS *int64
}

func (s *tunnelSession) proxyRequest(
	w http.ResponseWriter,
	r *http.Request,
	replay bool,
	persist bool,
) (proxyResult, error) {
	requestID := uuid.NewString()
	state := &requestState{
		events:        make(chan responseEvent, s.server.config.QueueDepth),
		requestWindow: newCreditWindow(s.server.config.QueueDepth),
	}
	s.requestsMu.Lock()
	if len(s.requests) >= s.server.config.MaxInFlight {
		s.requestsMu.Unlock()
		w.Header().Set("Retry-After", "1")
		http.Error(w, "Tunnel Busy", http.StatusServiceUnavailable)
		return proxyResult{}, nil
	}
	s.requests[requestID] = state
	s.requestsMu.Unlock()
	defer s.removeRequest(requestID)

	ctx, cancel := context.WithTimeout(r.Context(), s.server.config.RequestTimeout)
	defer cancel()
	headers := protobufHeaders(r.Header)
	startedAt := time.Now()
	requestCapture := &captureBuffer{limit: s.server.config.CaptureBodyBytes}
	responseCapture := &captureBuffer{limit: s.server.config.CaptureBodyBytes}
	storedID, parseErr := uuid.Parse(requestID)
	if parseErr != nil {
		return proxyResult{}, parseErr
	}
	if persist && s.server.inspection != nil {
		stored, err := s.server.inspection.CreateCapturedRequest(ctx, storage.CapturedRequest{
			ID: storedID, DomainID: s.domain.ID, Method: r.Method, Path: r.URL.EscapedPath(),
			Query: r.URL.RawQuery, RequestHeaders: redactHeaders(r.Header), ReceivedAt: startedAt,
		}, s.server.config.MaxStoredRequests)
		if err != nil {
			return proxyResult{}, err
		}
		s.server.events.publish(s.domain.ID, domainEvent{name: "request.created", data: apiCapturedRequest(stored)})
	}
	start := &tunnelv1.RequestStart{
		RequestId:         requestID,
		Method:            r.Method,
		Path:              r.URL.EscapedPath(),
		RawQuery:          r.URL.RawQuery,
		Headers:           headers,
		ContentLength:     r.ContentLength,
		Replay:            replay,
		DeadlineUnixMilli: time.Now().Add(s.server.config.RequestTimeout).UnixMilli(),
	}
	if err := s.sendMessage(ctx, &tunnelv1.Envelope_RequestStart{RequestStart: start}); err != nil {
		return proxyResult{}, err
	}

	uploadDone := make(chan error, 1)
	go func() {
		err := s.streamRequestBody(ctx, requestID, &capturingReader{
			reader: r.Body, capture: requestCapture,
		}, state.requestWindow)
		if persist && s.server.inspection != nil {
			_ = s.server.inspection.UpdateCapturedRequestBody(
				context.Background(), storedID, requestCapture.bytes(),
				requestCapture.size, requestCapture.truncated,
			)
			if captured, fetchErr := s.server.inspection.CapturedRequest(
				context.Background(), s.domain.UserID, storedID,
			); fetchErr == nil {
				s.server.events.publish(s.domain.ID, domainEvent{
					name: "request.updated", data: apiCapturedRequest(captured),
				})
			}
		}
		uploadDone <- err
	}()

	started := false
	var responseStatus int
	var responseHeaders []storage.Header
	for {
		select {
		case err := <-uploadDone:
			if err != nil {
				return proxyResult{}, err
			}
			uploadDone = nil
		case <-ctx.Done():
			reason := tunnelv1.CancelReason_CANCEL_REASON_CALLER_DISCONNECTED
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				reason = tunnelv1.CancelReason_CANCEL_REASON_DEADLINE_EXCEEDED
			}
			s.trySend(&tunnelv1.Envelope_Cancel{Cancel: &tunnelv1.Cancel{
				RequestId: requestID,
				Reason:    reason,
				Detail:    ctx.Err().Error(),
			}})
			return proxyResult{}, ctx.Err()
		case <-s.done:
			return proxyResult{}, errors.New("tunnel disconnected")
		case event := <-state.events:
			switch message := event.message.(type) {
			case *tunnelv1.ResponseStart:
				if started {
					return proxyResult{}, errors.New("duplicate response start")
				}
				copyHTTPHeaders(w.Header(), message.Headers)
				w.WriteHeader(int(message.StatusCode))
				started = true
				responseStatus = int(message.StatusCode)
				responseHeaders = redactHeaders(httpHeaders(message.Headers))
				if err := s.grantCredit(ctx, requestID, s.server.config.MaxChunkBytes); err != nil {
					return proxyResult{}, err
				}
			case *tunnelv1.ResponseBody:
				if !started {
					return proxyResult{}, errors.New("response body before response start")
				}
				responseCapture.add(message.Data)
				if _, err := w.Write(message.Data); err != nil {
					return proxyResult{}, err
				}
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
				if err := s.grantCredit(ctx, requestID, len(message.Data)); err != nil {
					return proxyResult{}, err
				}
			case *tunnelv1.ResponseEnd:
				duration := time.Since(startedAt).Milliseconds()
				if persist && s.server.inspection != nil {
					_ = s.server.inspection.CompleteCapturedRequest(
						context.Background(), storedID, responseStatus, responseHeaders,
						responseCapture.bytes(), responseCapture.size, responseCapture.truncated, duration,
					)
					if captured, err := s.server.inspection.CapturedRequest(context.Background(), s.domain.UserID, storedID); err == nil {
						s.server.events.publish(s.domain.ID, domainEvent{name: "request.updated", data: apiCapturedRequest(captured)})
					}
				}
				return proxyResult{status: &responseStatus, durationMS: &duration}, nil
			case *tunnelv1.Cancel:
				if !started {
					http.Error(w, "Local Service Unavailable", http.StatusBadGateway)
				}
				if replay {
					return proxyResult{}, fmt.Errorf("local replay failed: %s", message.Detail)
				}
				return proxyResult{}, nil
			default:
				return proxyResult{}, fmt.Errorf("unexpected response event %T", event.message)
			}
		}
	}
}

type capturingReader struct {
	reader  io.Reader
	capture *captureBuffer
}

func (r *capturingReader) Read(buffer []byte) (int, error) {
	count, err := r.reader.Read(buffer)
	if count > 0 {
		r.capture.add(buffer[:count])
	}
	return count, err
}

type replayResponseWriter struct {
	header http.Header
}

func (w *replayResponseWriter) Header() http.Header {
	return w.header
}

func (*replayResponseWriter) WriteHeader(int) {}

func (*replayResponseWriter) Write(body []byte) (int, error) {
	return len(body), nil
}

func (s *tunnelSession) replay(ctx context.Context, captured storage.CapturedRequest) (proxyResult, error) {
	target := captured.Path
	if captured.Query != "" {
		target += "?" + captured.Query
	}
	request, err := http.NewRequestWithContext(ctx, captured.Method, target, bytes.NewReader(captured.RequestBody))
	if err != nil {
		return proxyResult{}, err
	}
	request.ContentLength = int64(len(captured.RequestBody))
	for _, header := range captured.RequestHeaders {
		if len(header.Values) == 1 && header.Values[0] == redactedHeaderValue {
			continue
		}
		request.Header[header.Name] = append([]string(nil), header.Values...)
	}
	return s.proxyRequest(&replayResponseWriter{header: make(http.Header)}, request, true, false)
}

func (s *tunnelSession) streamRequestBody(
	ctx context.Context,
	requestID string,
	body io.Reader,
	window *creditWindow,
) error {
	buffer := make([]byte, s.server.config.MaxChunkBytes)
	for {
		count, readErr := body.Read(buffer)
		if count > 0 {
			if err := window.take(ctx, uint64(count)); err != nil {
				return err
			}
			chunk := append([]byte(nil), buffer[:count]...)
			if err := s.sendMessage(ctx, &tunnelv1.Envelope_RequestBody{RequestBody: &tunnelv1.RequestBody{
				RequestId: requestID,
				Data:      chunk,
			}}); err != nil {
				return err
			}
		}
		if errors.Is(readErr, io.EOF) {
			return s.sendMessage(ctx, &tunnelv1.Envelope_RequestEnd{RequestEnd: &tunnelv1.RequestEnd{
				RequestId: requestID,
			}})
		}
		if readErr != nil {
			return readErr
		}
	}
}

func (s *tunnelSession) readLoop() {
	var lastSequence uint64
	for {
		envelope, err := readTunnelEnvelope(s.conn)
		if err != nil {
			return
		}
		if envelope.ProtocolVersion != tunnelProtocolVersion || envelope.Sequence <= lastSequence {
			return
		}
		lastSequence = envelope.Sequence
		switch message := envelope.Message.(type) {
		case *tunnelv1.Envelope_ResponseStart:
			s.deliver(message.ResponseStart.RequestId, message.ResponseStart)
		case *tunnelv1.Envelope_ResponseBody:
			if len(message.ResponseBody.Data) > s.server.config.MaxChunkBytes {
				return
			}
			s.deliver(message.ResponseBody.RequestId, message.ResponseBody)
		case *tunnelv1.Envelope_ResponseEnd:
			s.deliver(message.ResponseEnd.RequestId, message.ResponseEnd)
		case *tunnelv1.Envelope_Cancel:
			s.deliver(message.Cancel.RequestId, message.Cancel)
		case *tunnelv1.Envelope_Pong:
			s.lastPong.Store(time.Now().UnixNano())
			_ = s.server.tunnels.HeartbeatTunnelSession(context.Background(), s.id)
		case *tunnelv1.Envelope_WindowUpdate:
			s.requestsMu.Lock()
			request := s.requests[message.WindowUpdate.RequestId]
			s.requestsMu.Unlock()
			if request == nil || !request.requestWindow.add(message.WindowUpdate.CreditBytes) {
				return
			}
		default:
			return
		}
	}
}

func (s *tunnelSession) writeLoop() {
	ticker := time.NewTicker(s.server.config.Heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case envelope := <-s.send:
			s.sequence++
			envelope.Sequence = s.sequence
			if err := writeTunnelEnvelope(s.conn, envelope); err != nil {
				s.close()
				return
			}
		case now := <-ticker.C:
			lastPong := time.Unix(0, s.lastPong.Load())
			if now.Sub(lastPong) > s.server.config.HeartbeatGrace {
				s.close()
				return
			}
			s.trySend(&tunnelv1.Envelope_Ping{Ping: &tunnelv1.Ping{SentAtUnixMilli: now.UnixMilli()}})
		}
	}
}

func (s *tunnelSession) sendMessage(ctx context.Context, message any) error {
	envelope, err := s.envelope(message)
	if err != nil {
		return err
	}
	return s.enqueue(ctx, envelope)
}

func (s *tunnelSession) grantCredit(ctx context.Context, requestID string, bytes int) error {
	if bytes <= 0 {
		return nil
	}
	return s.sendMessage(ctx, &tunnelv1.Envelope_WindowUpdate{WindowUpdate: &tunnelv1.WindowUpdate{
		RequestId:   requestID,
		CreditBytes: uint64(bytes),
	}})
}

func (s *tunnelSession) trySend(message any) {
	envelope, err := s.envelope(message)
	if err != nil {
		return
	}
	select {
	case s.send <- envelope:
	case <-s.done:
	default:
	}
}

func (s *tunnelSession) envelope(message any) (*tunnelv1.Envelope, error) {
	envelope := &tunnelv1.Envelope{
		ProtocolVersion: tunnelProtocolVersion,
	}
	switch typed := message.(type) {
	case *tunnelv1.Envelope_RequestStart:
		envelope.Message = typed
	case *tunnelv1.Envelope_RequestBody:
		envelope.Message = typed
	case *tunnelv1.Envelope_RequestEnd:
		envelope.Message = typed
	case *tunnelv1.Envelope_Cancel:
		envelope.Message = typed
	case *tunnelv1.Envelope_Ping:
		envelope.Message = typed
	case *tunnelv1.Envelope_WindowUpdate:
		envelope.Message = typed
	default:
		return nil, fmt.Errorf("unsupported server tunnel message %T", message)
	}
	return envelope, nil
}

func (s *tunnelSession) enqueue(ctx context.Context, envelope *tunnelv1.Envelope) error {
	select {
	case s.send <- envelope:
		return nil
	case <-s.done:
		return errors.New("tunnel disconnected")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *tunnelSession) deliver(requestID string, message proto.Message) {
	s.requestsMu.Lock()
	request := s.requests[requestID]
	s.requestsMu.Unlock()
	if request == nil {
		return
	}
	select {
	case request.events <- responseEvent{message: message}:
	case <-s.done:
	}
}

func (s *tunnelSession) removeRequest(requestID string) {
	s.requestsMu.Lock()
	delete(s.requests, requestID)
	s.requestsMu.Unlock()
}

func (s *tunnelSession) close() {
	s.closeOne.Do(func() {
		close(s.done)
		s.server.registry.remove(s.domain.Slug, s)
		_ = s.conn.Close()
		_ = s.server.tunnels.DisconnectTunnelSession(context.Background(), s.id)
		s.server.events.publish(s.domain.ID, domainEvent{name: "domain.status", data: s.server.apiDomain(storage.Domain{
			ID: s.domain.ID, UserID: s.domain.UserID, Slug: s.domain.Slug,
			CreatedAt: s.domain.CreatedAt, Online: false,
		})})
	})
}

func readTunnelEnvelope(connection *websocket.Conn) (*tunnelv1.Envelope, error) {
	messageType, payload, err := connection.ReadMessage()
	if err != nil {
		return nil, err
	}
	if messageType != websocket.BinaryMessage {
		return nil, errors.New("tunnel frames must be binary")
	}
	var envelope tunnelv1.Envelope
	if err := proto.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}

func writeTunnelEnvelope(connection *websocket.Conn, envelope *tunnelv1.Envelope) error {
	payload, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}
	return connection.WriteMessage(websocket.BinaryMessage, payload)
}

func writeProtocolError(connection *websocket.Conn, code tunnelv1.ProtocolErrorCode, detail string) {
	_ = writeTunnelEnvelope(connection, &tunnelv1.Envelope{
		ProtocolVersion: tunnelProtocolVersion,
		Sequence:        1,
		Message: &tunnelv1.Envelope_Error{Error: &tunnelv1.ProtocolError{
			Code:   code,
			Detail: detail,
			Fatal:  true,
		}},
	})
}

func writeTunnelOffline(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Retry-After", "1")
	http.Error(w, "Tunnel Offline", http.StatusServiceUnavailable)
}

func protobufHeaders(headers http.Header) []*tunnelv1.Header {
	filtered := cloneWithoutHopByHop(headers)
	result := make([]*tunnelv1.Header, 0, len(filtered))
	for name, values := range filtered {
		result = append(result, &tunnelv1.Header{Name: name, Values: values})
	}
	return result
}

func copyHTTPHeaders(destination http.Header, headers []*tunnelv1.Header) {
	source := make(http.Header)
	for _, header := range headers {
		if header != nil {
			source[header.Name] = append([]string(nil), header.Values...)
		}
	}
	for name, values := range cloneWithoutHopByHop(source) {
		for _, value := range values {
			destination.Add(name, value)
		}
	}
}

func httpHeaders(headers []*tunnelv1.Header) http.Header {
	result := make(http.Header)
	for _, header := range headers {
		if header != nil {
			result[header.Name] = append([]string(nil), header.Values...)
		}
	}
	return result
}

func cloneWithoutHopByHop(headers http.Header) http.Header {
	result := headers.Clone()
	for _, value := range headers.Values("Connection") {
		for _, name := range strings.Split(value, ",") {
			result.Del(strings.TrimSpace(name))
		}
	}
	for _, name := range []string{
		"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate",
		"Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade",
	} {
		result.Del(name)
	}
	return result
}
