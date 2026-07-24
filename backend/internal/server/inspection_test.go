package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/opentunnel/opentunnel/backend/internal/storage"
	"github.com/opentunnel/opentunnel/shared/gen/api"
	tunnelv1 "github.com/opentunnel/opentunnel/shared/gen/tunnel/v1"
)

type inspectionFakeStore struct {
	*fakeStore
	mu       sync.Mutex
	owner    uuid.UUID
	domain   storage.Domain
	requests map[uuid.UUID]storage.CapturedRequest
	replays  map[uuid.UUID]storage.ReplayAttempt
}

func newInspectionFakeStore() *inspectionFakeStore {
	owner := uuid.New()
	domain := storage.Domain{
		ID: uuid.New(), UserID: owner, Slug: "inspect", CreatedAt: time.Now(),
	}
	return &inspectionFakeStore{
		fakeStore: &fakeStore{}, owner: owner, domain: domain,
		requests: make(map[uuid.UUID]storage.CapturedRequest),
		replays:  make(map[uuid.UUID]storage.ReplayAttempt),
	}
}

func (f *inspectionFakeStore) Domain(_ context.Context, userID, domainID uuid.UUID) (storage.Domain, error) {
	if userID != f.owner || domainID != f.domain.ID {
		return storage.Domain{}, storage.ErrNotFound
	}
	return f.domain, nil
}

func (f *inspectionFakeStore) CreateCapturedRequest(
	_ context.Context,
	request storage.CapturedRequest,
	_ int,
) (storage.CapturedRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests[request.ID] = request
	return request, nil
}

func (f *inspectionFakeStore) UpdateCapturedRequestBody(
	_ context.Context,
	requestID uuid.UUID,
	body []byte,
	size int64,
	truncated bool,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	request := f.requests[requestID]
	request.RequestBody = append([]byte(nil), body...)
	request.RequestBodySize = size
	request.RequestBodyTruncated = truncated
	f.requests[requestID] = request
	return nil
}

func (f *inspectionFakeStore) CompleteCapturedRequest(
	_ context.Context,
	requestID uuid.UUID,
	status int,
	headers []storage.Header,
	body []byte,
	size int64,
	truncated bool,
	duration int64,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	request := f.requests[requestID]
	request.ResponseStatus = &status
	request.ResponseHeaders = headers
	request.ResponseBody = append([]byte(nil), body...)
	request.ResponseBodySize = &size
	request.ResponseBodyTruncated = truncated
	request.DurationMS = &duration
	f.requests[requestID] = request
	return nil
}

func (f *inspectionFakeStore) ListCapturedRequests(
	_ context.Context,
	userID, domainID uuid.UUID,
	filter storage.CapturedRequestFilter,
) ([]storage.CapturedRequest, error) {
	if userID != f.owner || domainID != f.domain.ID {
		return nil, storage.ErrNotFound
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]storage.CapturedRequest, 0, len(f.requests))
	for _, request := range f.requests {
		if filter.Before != nil && !request.ReceivedAt.Before(*filter.Before) {
			continue
		}
		if filter.Method != "" && !strings.EqualFold(request.Method, filter.Method) {
			continue
		}
		if filter.Path != "" {
			matches := strings.Contains(strings.ToLower(request.Path), strings.ToLower(filter.Path))
			if filter.PathExclude == matches {
				continue
			}
		}
		if filter.StatusMin != nil && (request.ResponseStatus == nil || *request.ResponseStatus < *filter.StatusMin) {
			continue
		}
		if filter.StatusMax != nil && (request.ResponseStatus == nil || *request.ResponseStatus > *filter.StatusMax) {
			continue
		}
		result = append(result, request)
	}
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}
	return result, nil
}

func (f *inspectionFakeStore) CapturedRequest(
	_ context.Context,
	userID, requestID uuid.UUID,
) (storage.CapturedRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	request, ok := f.requests[requestID]
	if !ok || userID != f.owner {
		return storage.CapturedRequest{}, storage.ErrNotFound
	}
	return request, nil
}

func (f *inspectionFakeStore) CreateReplayAttempt(
	_ context.Context,
	userID, requestID uuid.UUID,
) (storage.ReplayAttempt, error) {
	if userID != f.owner {
		return storage.ReplayAttempt{}, storage.ErrNotFound
	}
	replay := storage.ReplayAttempt{
		ID: uuid.New(), RequestID: requestID, Status: "queued", CreatedAt: time.Now(),
	}
	f.mu.Lock()
	f.replays[replay.ID] = replay
	f.mu.Unlock()
	return replay, nil
}

func (f *inspectionFakeStore) UpdateReplayAttempt(
	_ context.Context,
	replayID uuid.UUID,
	status string,
	errorDetail *string,
	responseStatus *int,
	duration *int64,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	replay := f.replays[replayID]
	replay.Status = status
	replay.Error = errorDetail
	replay.ResponseStatus = responseStatus
	replay.DurationMS = duration
	f.replays[replayID] = replay
	return nil
}

func TestCaptureRedactsSensitiveHeadersAndMarksTruncation(t *testing.T) {
	t.Parallel()
	headers := http.Header{
		"Authorization": {"Bearer secret"},
		"Cookie":        {"session=secret"},
		"X-Trace":       {"safe"},
	}
	redacted := redactHeaders(headers)
	for _, header := range redacted {
		switch header.Name {
		case "Authorization", "Cookie":
			if len(header.Values) != 1 || header.Values[0] != redactedHeaderValue {
				t.Fatalf("%s was not redacted: %v", header.Name, header.Values)
			}
		case "X-Trace":
			if header.Values[0] != "safe" {
				t.Fatalf("safe header changed: %v", header.Values)
			}
		}
	}
	capture := &captureBuffer{limit: 4}
	capture.add([]byte("abcdef"))
	if string(capture.bytes()) != "abcd" || capture.size != 6 || !capture.truncated {
		t.Fatalf("unexpected capture: %q size=%d truncated=%v", capture.bytes(), capture.size, capture.truncated)
	}
}

func TestGetRequestIsOwnerScoped(t *testing.T) {
	t.Parallel()
	store := newInspectionFakeStore()
	requestID := uuid.New()
	store.requests[requestID] = storage.CapturedRequest{
		ID: requestID, DomainID: store.domain.ID, Method: "GET", Path: "/",
		ReceivedAt: time.Now(),
	}
	server := New(Config{}, store)
	for _, test := range []struct {
		name   string
		userID uuid.UUID
		status int
	}{
		{name: "owner", userID: store.owner, status: http.StatusOK},
		{name: "other user", userID: uuid.New(), status: http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request = request.WithContext(context.WithValue(
				request.Context(), userContextKey, storage.User{ID: test.userID},
			))
			response := httptest.NewRecorder()
			server.GetRequest(response, request, requestID)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
		})
	}
}

func TestListRequestsAppliesFilters(t *testing.T) {
	t.Parallel()
	store := newInspectionFakeStore()
	now := time.Now()
	statusOK := 200
	statusErr := 500
	getReq := uuid.New()
	postReq := uuid.New()
	store.requests[getReq] = storage.CapturedRequest{
		ID: getReq, DomainID: store.domain.ID, Method: "GET", Path: "/health",
		ResponseStatus: &statusOK, ReceivedAt: now.Add(-time.Minute),
	}
	store.requests[postReq] = storage.CapturedRequest{
		ID: postReq, DomainID: store.domain.ID, Method: "POST", Path: "/api/items",
		ResponseStatus: &statusErr, ReceivedAt: now,
	}
	server := New(Config{}, store)
	method := "POST"
	path := "/api"
	statusMin := 500
	statusMax := 599
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(context.WithValue(
		request.Context(), userContextKey, storage.User{ID: store.owner},
	))
	response := httptest.NewRecorder()
	server.ListRequests(response, request, store.domain.ID, api.ListRequestsParams{
		Method: &method, Path: &path, StatusMin: &statusMin, StatusMax: &statusMax,
	})
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var page api.RequestPage
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Id != postReq {
		t.Fatalf("items = %+v, want only POST /api/items", page.Items)
	}

	exclude := api.Exclude
	excludeResponse := httptest.NewRecorder()
	server.ListRequests(excludeResponse, request, store.domain.ID, api.ListRequestsParams{
		Path: &path, PathMode: &exclude,
	})
	if excludeResponse.Code != http.StatusOK {
		t.Fatalf("exclude status = %d, want %d", excludeResponse.Code, http.StatusOK)
	}
	var excludePage api.RequestPage
	if err := json.NewDecoder(excludeResponse.Body).Decode(&excludePage); err != nil {
		t.Fatalf("decode exclude page: %v", err)
	}
	if len(excludePage.Items) != 1 || excludePage.Items[0].Id != getReq {
		t.Fatalf("exclude items = %+v, want only GET /health", excludePage.Items)
	}
}

func TestDomainEventsRejectOtherOwnersAndSendInitialStatus(t *testing.T) {
	t.Parallel()
	store := newInspectionFakeStore()
	server := New(Config{PublicHost: "opts.ink"}, store)

	denied := httptest.NewRequest(http.MethodGet, "/", nil)
	denied = denied.WithContext(context.WithValue(
		denied.Context(), userContextKey, storage.User{ID: uuid.New()},
	))
	deniedResponse := httptest.NewRecorder()
	server.StreamDomainEvents(deniedResponse, denied, store.domain.ID)
	if deniedResponse.Code != http.StatusNotFound {
		t.Fatalf("other owner stream status = %d", deniedResponse.Code)
	}

	ctx, cancel := context.WithCancel(context.Background())
	allowed := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	allowed = allowed.WithContext(context.WithValue(
		allowed.Context(), userContextKey, storage.User{ID: store.owner},
	))
	allowedResponse := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		server.StreamDomainEvents(allowedResponse, allowed, store.domain.ID)
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-done
	if body := allowedResponse.Body.String(); !containsAll(body, "event: domain.status", `"status":"offline"`) {
		t.Fatalf("initial SSE event = %q", body)
	}
}

func TestReplayRejectsOfflineAndRunsDirectlyOnActiveTunnel(t *testing.T) {
	store := newInspectionFakeStore()
	requestID := uuid.New()
	store.requests[requestID] = storage.CapturedRequest{
		ID: requestID, DomainID: store.domain.ID, Method: "POST", Path: "/replay",
		RequestBody: []byte("payload"), RequestBodySize: 7, ReceivedAt: time.Now(),
	}
	server := New(Config{RequestTimeout: time.Second}, store)
	userContext := context.WithValue(context.Background(), userContextKey, storage.User{ID: store.owner})

	offlineRequest := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(userContext)
	offlineResponse := httptest.NewRecorder()
	server.ReplayRequest(offlineResponse, offlineRequest, requestID)
	if offlineResponse.Code != http.StatusConflict || len(store.replays) != 0 {
		t.Fatalf("offline replay status=%d attempts=%d", offlineResponse.Code, len(store.replays))
	}

	session := &tunnelSession{
		domain: store.domain, server: server, send: make(chan *tunnelv1.Envelope, 8),
		done: make(chan struct{}), requests: make(map[string]*requestState),
	}
	if !server.registry.add(store.domain.Slug, session) {
		t.Fatal("could not register test tunnel")
	}
	go answerReplay(session)
	onlineRequest := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(userContext)
	onlineResponse := httptest.NewRecorder()
	server.ReplayRequest(onlineResponse, onlineRequest, requestID)
	if onlineResponse.Code != http.StatusAccepted {
		t.Fatalf("online replay status=%d body=%s", onlineResponse.Code, onlineResponse.Body.String())
	}
	var response api.Replay
	if err := json.Unmarshal(onlineResponse.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		store.mu.Lock()
		replay := store.replays[response.Id]
		store.mu.Unlock()
		if replay.Status == "succeeded" {
			if replay.ResponseStatus == nil || *replay.ResponseStatus != http.StatusNoContent {
				t.Fatalf("replay result = %#v", replay)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("replay result was not persisted")
}

func TestProxyPersistsCappedRedactedRequestAndResponse(t *testing.T) {
	store := newInspectionFakeStore()
	server := New(Config{CaptureBodyBytes: 4, RequestTimeout: time.Second}, store)
	session := &tunnelSession{
		domain: store.domain, server: server, send: make(chan *tunnelv1.Envelope, 16),
		done: make(chan struct{}), requests: make(map[string]*requestState),
	}
	go func() {
		var requestID string
		for envelope := range session.send {
			if start := envelope.GetRequestStart(); start != nil {
				requestID = start.RequestId
				session.requestsMu.Lock()
				request := session.requests[requestID]
				session.requestsMu.Unlock()
				request.requestWindow.add(1024)
			}
			if envelope.GetRequestEnd() != nil {
				session.deliver(requestID, &tunnelv1.ResponseStart{
					RequestId: requestID, StatusCode: http.StatusCreated,
					Headers: []*tunnelv1.Header{
						{Name: "Set-Cookie", Values: []string{"secret=true"}},
						{Name: "X-Safe", Values: []string{"visible"}},
					},
				})
				session.deliver(requestID, &tunnelv1.ResponseBody{
					RequestId: requestID, Data: []byte("abcdef"),
				})
				session.deliver(requestID, &tunnelv1.ResponseEnd{RequestId: requestID})
				return
			}
		}
	}()
	request := httptest.NewRequest(http.MethodPost, "/capture?q=1", strings.NewReader("123456"))
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	if err := session.proxy(response, request); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		store.mu.Lock()
		for _, captured := range store.requests {
			store.mu.Unlock()
			if string(captured.RequestBody) != "1234" || !captured.RequestBodyTruncated || captured.RequestBodySize != 6 {
				t.Fatalf("request capture = %#v", captured)
			}
			if string(captured.ResponseBody) != "abcd" || !captured.ResponseBodyTruncated ||
				captured.ResponseBodySize == nil || *captured.ResponseBodySize != 6 {
				t.Fatalf("response capture = %#v", captured)
			}
			if !hasHeaderValue(captured.RequestHeaders, "Authorization", redactedHeaderValue) ||
				!hasHeaderValue(captured.ResponseHeaders, "Set-Cookie", redactedHeaderValue) {
				t.Fatalf("sensitive headers were persisted: %#v %#v", captured.RequestHeaders, captured.ResponseHeaders)
			}
			return
		}
		store.mu.Unlock()
		time.Sleep(time.Millisecond)
	}
	t.Fatal("captured request was not persisted")
}

func answerReplay(session *tunnelSession) {
	var requestID string
	for envelope := range session.send {
		if start := envelope.GetRequestStart(); start != nil {
			requestID = start.RequestId
			if !start.Replay || start.Path != "/replay" {
				return
			}
			session.requestsMu.Lock()
			request := session.requests[requestID]
			session.requestsMu.Unlock()
			request.requestWindow.add(1024)
		}
		if end := envelope.GetRequestEnd(); end != nil {
			session.deliver(requestID, &tunnelv1.ResponseStart{
				RequestId: requestID, StatusCode: http.StatusNoContent,
			})
			session.deliver(requestID, &tunnelv1.ResponseEnd{RequestId: requestID})
			return
		}
	}
}

func containsAll(value string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}

func hasHeaderValue(headers []storage.Header, name, value string) bool {
	for _, header := range headers {
		if header.Name == name && len(header.Values) == 1 && header.Values[0] == value {
			return true
		}
	}
	return false
}
