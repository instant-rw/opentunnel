package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/opentunnel/opentunnel/internal/gen/api"
	"github.com/opentunnel/opentunnel/internal/storage"
)

const redactedHeaderValue = "[REDACTED]"

type inspectionStore interface {
	CreateCapturedRequest(context.Context, storage.CapturedRequest, int) (storage.CapturedRequest, error)
	UpdateCapturedRequestBody(context.Context, uuid.UUID, []byte, int64, bool) error
	CompleteCapturedRequest(context.Context, uuid.UUID, int, []storage.Header, []byte, int64, bool, int64) error
	ListCapturedRequests(context.Context, uuid.UUID, uuid.UUID, *time.Time, int) ([]storage.CapturedRequest, error)
	CapturedRequest(context.Context, uuid.UUID, uuid.UUID) (storage.CapturedRequest, error)
	CreateReplayAttempt(context.Context, uuid.UUID, uuid.UUID) (storage.ReplayAttempt, error)
	UpdateReplayAttempt(context.Context, uuid.UUID, string, *string, *int, *int64) error
}

type captureBuffer struct {
	limit     int
	data      bytes.Buffer
	size      int64
	truncated bool
}

func (c *captureBuffer) add(data []byte) {
	c.size += int64(len(data))
	remaining := c.limit - c.data.Len()
	if remaining > 0 {
		c.data.Write(data[:min(remaining, len(data))])
	}
	c.truncated = c.size > int64(c.limit)
}

func (c *captureBuffer) bytes() []byte {
	return append([]byte(nil), c.data.Bytes()...)
}

type domainEventHub struct {
	mu          sync.Mutex
	subscribers map[uuid.UUID]map[chan domainEvent]struct{}
}

type domainEvent struct {
	name string
	data any
}

func newDomainEventHub() *domainEventHub {
	return &domainEventHub{subscribers: make(map[uuid.UUID]map[chan domainEvent]struct{})}
}

func (h *domainEventHub) subscribe(domainID uuid.UUID) (<-chan domainEvent, func()) {
	channel := make(chan domainEvent, 16)
	h.mu.Lock()
	if h.subscribers[domainID] == nil {
		h.subscribers[domainID] = make(map[chan domainEvent]struct{})
	}
	h.subscribers[domainID][channel] = struct{}{}
	h.mu.Unlock()
	return channel, func() {
		h.mu.Lock()
		delete(h.subscribers[domainID], channel)
		if len(h.subscribers[domainID]) == 0 {
			delete(h.subscribers, domainID)
		}
		h.mu.Unlock()
	}
}

func (h *domainEventHub) publish(domainID uuid.UUID, event domainEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for channel := range h.subscribers[domainID] {
		select {
		case channel <- event:
		default:
		}
	}
}

func redactHeaders(headers http.Header) []storage.Header {
	result := make([]storage.Header, 0, len(headers))
	for name, values := range headers {
		copied := append([]string(nil), values...)
		if sensitiveHeader(name) {
			copied = []string{redactedHeaderValue}
		}
		result = append(result, storage.Header{Name: http.CanonicalHeaderKey(name), Values: copied})
	}
	return result
}

func sensitiveHeader(name string) bool {
	switch strings.ToLower(name) {
	case "authorization", "cookie", "proxy-authorization", "set-cookie", "x-api-key":
		return true
	default:
		return strings.Contains(strings.ToLower(name), "token") ||
			strings.Contains(strings.ToLower(name), "secret")
	}
}

func (s *Server) ListRequests(
	w http.ResponseWriter,
	r *http.Request,
	domainID api.DomainId,
	params api.ListRequestsParams,
) {
	if s.inspection == nil {
		writeProblem(w, http.StatusServiceUnavailable, "inspection_unavailable", "Request inspection is unavailable")
		return
	}
	limit := 50
	if params.Limit != nil {
		limit = *params.Limit
	}
	var before *time.Time
	if params.Cursor != nil {
		decoded, err := base64.RawURLEncoding.DecodeString(*params.Cursor)
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "invalid_cursor", "Request cursor is invalid")
			return
		}
		parsed, err := time.Parse(time.RFC3339Nano, string(decoded))
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "invalid_cursor", "Request cursor is invalid")
			return
		}
		before = &parsed
	}
	requests, err := s.inspection.ListCapturedRequests(r.Context(), userFromContext(r.Context()).ID, domainID, before, limit+1)
	if err != nil {
		writeInternal(w)
		return
	}
	page := api.RequestPage{Items: make([]api.CapturedRequest, 0, min(limit, len(requests)))}
	for _, request := range requests[:min(limit, len(requests))] {
		page.Items = append(page.Items, apiCapturedRequest(request))
	}
	if len(requests) > limit {
		cursor := base64.RawURLEncoding.EncodeToString([]byte(requests[limit-1].ReceivedAt.Format(time.RFC3339Nano)))
		page.NextCursor = &cursor
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) GetRequest(w http.ResponseWriter, r *http.Request, requestID api.RequestId) {
	request, err := s.inspection.CapturedRequest(r.Context(), userFromContext(r.Context()).ID, requestID)
	if errors.Is(err, storage.ErrNotFound) {
		writeProblem(w, http.StatusNotFound, "request_not_found", "Request not found")
		return
	}
	if err != nil {
		writeInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, apiCapturedRequest(request))
}

func (s *Server) StreamDomainEvents(w http.ResponseWriter, r *http.Request, domainID api.DomainId) {
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
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeProblem(w, http.StatusInternalServerError, "stream_unsupported", "Event streaming is unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	writeSSE(w, "domain.status", s.apiDomain(domain))
	flusher.Flush()

	events, unsubscribe := s.events.subscribe(domainID)
	defer unsubscribe()
	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-events:
			writeSSE(w, event.name, event.data)
			flusher.Flush()
		case <-keepAlive.C:
			_, _ = fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) ReplayRequest(w http.ResponseWriter, r *http.Request, requestID api.RequestId) {
	if s.inspection == nil {
		writeProblem(w, http.StatusServiceUnavailable, "inspection_unavailable", "Request inspection is unavailable")
		return
	}
	user := userFromContext(r.Context())
	captured, err := s.inspection.CapturedRequest(r.Context(), user.ID, requestID)
	if errors.Is(err, storage.ErrNotFound) {
		writeProblem(w, http.StatusNotFound, "request_not_found", "Request not found")
		return
	}
	if err != nil {
		writeInternal(w)
		return
	}
	if captured.RequestBodyTruncated {
		writeProblem(w, http.StatusConflict, "request_body_truncated", "A truncated request body cannot be replayed")
		return
	}
	session := s.registry.getByDomainID(captured.DomainID)
	if session == nil {
		writeProblem(w, http.StatusConflict, "tunnel_offline", "Tunnel must be online to replay")
		return
	}
	replay, err := s.inspection.CreateReplayAttempt(r.Context(), user.ID, requestID)
	if err != nil {
		writeInternal(w)
		return
	}
	go s.runReplay(session, captured, replay)
	writeJSON(w, http.StatusAccepted, api.Replay{
		Id: replay.ID, RequestId: replay.RequestID, Status: api.Queued, CreatedAt: replay.CreatedAt,
	})
}

func (s *Server) runReplay(session *tunnelSession, captured storage.CapturedRequest, replay storage.ReplayAttempt) {
	_ = s.inspection.UpdateReplayAttempt(context.Background(), replay.ID, "running", nil, nil, nil)
	s.events.publish(captured.DomainID, domainEvent{name: "replay.updated", data: map[string]any{
		"id": replay.ID, "requestId": replay.RequestID, "status": "running", "createdAt": replay.CreatedAt,
	}})
	ctx, cancel := context.WithTimeout(context.Background(), s.config.RequestTimeout)
	defer cancel()
	result, err := session.replay(ctx, captured)
	status := "succeeded"
	var detail *string
	if err != nil {
		status = "failed"
		message := err.Error()
		detail = &message
	}
	_ = s.inspection.UpdateReplayAttempt(context.Background(), replay.ID, status, detail, result.status, result.durationMS)
	s.events.publish(captured.DomainID, domainEvent{name: "replay.updated", data: map[string]any{
		"id": replay.ID, "requestId": replay.RequestID, "status": status, "createdAt": replay.CreatedAt,
	}})
}

func writeSSE(w http.ResponseWriter, name string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, data)
}

func apiCapturedRequest(request storage.CapturedRequest) api.CapturedRequest {
	result := api.CapturedRequest{
		Id: request.ID, DomainId: request.DomainID, Method: request.Method,
		Path: request.Path, Query: request.Query, Headers: apiHeaders(request.RequestHeaders),
		ReceivedAt: request.ReceivedAt,
	}
	if request.RequestBodySize > 0 || len(request.RequestBody) > 0 {
		result.Body = &api.BodyCapture{
			Base64: request.RequestBody,
			Size:   request.RequestBodySize, Truncated: request.RequestBodyTruncated,
		}
	}
	if request.ResponseStatus != nil {
		response := api.CapturedResponse{
			Status: *request.ResponseStatus, Headers: apiHeaders(request.ResponseHeaders),
		}
		if request.DurationMS != nil {
			response.DurationMs = *request.DurationMS
		}
		if request.ResponseBodySize != nil && (*request.ResponseBodySize > 0 || len(request.ResponseBody) > 0) {
			response.Body = &api.BodyCapture{
				Base64: request.ResponseBody,
				Size:   *request.ResponseBodySize, Truncated: request.ResponseBodyTruncated,
			}
		}
		result.Response = &response
	}
	return result
}

func apiHeaders(headers []storage.Header) []api.Header {
	result := make([]api.Header, 0, len(headers))
	for _, header := range headers {
		result = append(result, api.Header{Name: header.Name, Values: header.Values})
	}
	return result
}
