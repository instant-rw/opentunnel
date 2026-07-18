package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	clitunnel "github.com/opentunnel/opentunnel/internal/cli/tunnel"
	tunnelv1 "github.com/opentunnel/opentunnel/internal/gen/tunnel/v1"
	"github.com/opentunnel/opentunnel/internal/storage"
	"google.golang.org/protobuf/proto"
)

type tunnelFakeStore struct {
	*fakeStore
	mu         sync.Mutex
	domain     storage.Domain
	token      storage.Token
	active     bool
	heartbeats int
}

func newTunnelFakeStore() *tunnelFakeStore {
	userID := uuid.New()
	return &tunnelFakeStore{
		fakeStore: &fakeStore{},
		domain: storage.Domain{
			ID: uuid.New(), UserID: userID, Slug: "demo", CreatedAt: time.Now(),
		},
		token: storage.Token{ID: uuid.New(), UserID: userID},
	}
}

func (f *tunnelFakeStore) AuthorizeTunnel(
	context.Context,
	[]byte,
	uuid.UUID,
) (storage.TunnelAuthorization, error) {
	return storage.TunnelAuthorization{Domain: f.domain, Token: f.token}, nil
}

func (f *tunnelFakeStore) ConnectTunnelSession(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	time.Time,
) (uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.active {
		return uuid.Nil, storage.ErrConflict
	}
	f.active = true
	return uuid.New(), nil
}

func (f *tunnelFakeStore) HeartbeatTunnelSession(context.Context, uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.heartbeats++
	return nil
}

func (f *tunnelFakeStore) DisconnectTunnelSession(context.Context, uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.active = false
	return nil
}

func TestPublicHostRoutingAndOfflineResponse(t *testing.T) {
	t.Parallel()
	store := newTunnelFakeStore()
	handler := New(Config{PublicHost: "opts.ink"}, store)

	request := httptest.NewRequest(http.MethodGet, "http://demo.opts.ink/path", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || response.Body.String() != "Tunnel Offline\n" {
		t.Fatalf("offline route got %d %q", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "http://nested.demo.opts.ink/path", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("nested wildcard host got %d", response.Code)
	}
}

func TestTunnelRegistrationRequiresBearerToken(t *testing.T) {
	t.Parallel()
	store := newTunnelFakeStore()
	handler := New(Config{PublicHost: "opts.ink"}, store)
	request := httptest.NewRequest(http.MethodGet, "/tunnel?domainId="+store.domain.ID.String(), nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated registration got %d", response.Code)
	}
}

func TestRegistryAllowsOneActiveSessionPerDomain(t *testing.T) {
	t.Parallel()
	registry := newSessionRegistry()
	first := &tunnelSession{}
	second := &tunnelSession{}
	if !registry.add("demo", first) {
		t.Fatal("first session was rejected")
	}
	if registry.add("demo", second) {
		t.Fatal("duplicate session was accepted")
	}
	registry.remove("demo", second)
	if registry.get("demo") != first {
		t.Fatal("non-owner removed active session")
	}
	registry.remove("demo", first)
	if registry.get("demo") != nil {
		t.Fatal("active session was not removed")
	}
}

func TestDuplicateTunnelRegistrationIsRejected(t *testing.T) {
	store := newTunnelFakeStore()
	public := httptest.NewServer(New(Config{
		PublicHost: "opts.ink", Heartbeat: time.Second, HeartbeatGrace: 3 * time.Second,
	}, store))
	defer public.Close()

	first := dialTestTunnel(t, public.URL, store.domain.ID)
	defer first.Close()
	writeClientHello(t, first, store.domain.ID)
	serverHello := readTestEnvelope(t, first)
	if serverHello.GetServerHello() == nil {
		t.Fatalf("first registration got %T", serverHello.Message)
	}

	second := dialTestTunnel(t, public.URL, store.domain.ID)
	defer second.Close()
	writeClientHello(t, second, store.domain.ID)
	rejection := readTestEnvelope(t, second).GetError()
	if rejection == nil ||
		rejection.Code != tunnelv1.ProtocolErrorCode_PROTOCOL_ERROR_CODE_DUPLICATE_SESSION ||
		!rejection.Fatal {
		t.Fatalf("duplicate registration got %v", rejection)
	}

	_ = first.Close()
	disconnected := false
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		store.mu.Lock()
		disconnected = !store.active
		store.mu.Unlock()
		if disconnected {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !disconnected {
		t.Fatal("closed tunnel remained active")
	}
	reconnected := dialTestTunnel(t, public.URL, store.domain.ID)
	defer reconnected.Close()
	writeClientHello(t, reconnected, store.domain.ID)
	if readTestEnvelope(t, reconnected).GetServerHello() == nil {
		t.Fatal("replacement tunnel was rejected")
	}
}

func TestTunnelStreamsThroughLocalhostAndFiltersHopHeaders(t *testing.T) {
	store := newTunnelFakeStore()
	firstRequestChunk := make(chan struct{})
	firstResponseChunk := make(chan struct{})
	cancelStarted := make(chan struct{})
	cancelObserved := make(chan struct{})
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/probe" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/cancel" {
			close(cancelStarted)
			<-r.Context().Done()
			close(cancelObserved)
			return
		}
		if r.Header.Get("X-Hop") != "" {
			t.Error("connection-specific request header reached localhost")
		}
		first := make([]byte, 5)
		if _, err := io.ReadFull(r.Body, first); err != nil {
			t.Errorf("read first local request chunk: %v", err)
			return
		}
		close(firstRequestChunk)
		rest, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read local request: %v", err)
			return
		}
		w.Header().Set("Connection", "X-Response-Hop")
		w.Header().Set("X-Response-Hop", "remove-me")
		w.Header().Set("X-End-To-End", "preserve-me")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(append(append([]byte("first:"), first...), rest...))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		close(firstResponseChunk)
		time.Sleep(25 * time.Millisecond)
		_, _ = w.Write([]byte(":last"))
	}))
	defer local.Close()
	localURL, err := url.Parse(local.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(localURL.Port())
	if err != nil {
		t.Fatal(err)
	}

	public := httptest.NewServer(New(Config{
		PublicHost: "opts.ink", Heartbeat: 20 * time.Millisecond, HeartbeatGrace: 200 * time.Millisecond,
	}, store))
	defer public.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := &clitunnel.Runner{
		APIURL: public.URL, Token: "test-token", DomainID: store.domain.ID.String(),
		LocalPort: port, ClientVersion: "test",
	}
	runnerDone := make(chan error, 1)
	go func() { runnerDone <- runner.Run(ctx) }()

	client := public.Client()
	online := false
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		request, requestErr := http.NewRequest(http.MethodGet, public.URL+"/probe", nil)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		request.Host = "demo.opts.ink"
		probe, probeErr := client.Do(request)
		if probeErr == nil && probe.StatusCode == http.StatusNoContent {
			_ = probe.Body.Close()
			online = true
			break
		}
		if probe != nil {
			_ = probe.Body.Close()
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !online {
		t.Fatal("tunnel did not come online")
	}

	requestBodyReader, requestBodyWriter := io.Pipe()
	request, err := http.NewRequest(http.MethodPost, public.URL+"/stream?q=1", requestBodyReader)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "demo.opts.ink"
	request.Header.Set("Connection", "X-Hop")
	request.Header.Set("X-Hop", "remove-me")
	responseReady := make(chan *http.Response, 1)
	requestFailed := make(chan error, 1)
	go func() {
		response, requestErr := client.Do(request)
		if requestErr != nil {
			requestFailed <- requestErr
			return
		}
		responseReady <- response
	}()
	if _, err := requestBodyWriter.Write([]byte("first")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstRequestChunk:
	case <-time.After(time.Second):
		t.Fatal("localhost did not receive streaming request chunk")
	}
	if _, err := requestBodyWriter.Write([]byte("second")); err != nil {
		t.Fatal(err)
	}
	_ = requestBodyWriter.Close()

	var response *http.Response
	select {
	case response = <-responseReady:
	case err = <-requestFailed:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("public request did not complete")
	}
	if response == nil || response.StatusCode != http.StatusCreated {
		t.Fatalf("tunnel response was %#v", response)
	}
	select {
	case <-firstResponseChunk:
	case <-time.After(time.Second):
		t.Fatal("local response did not start streaming")
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "first:firstsecond:last" {
		t.Fatalf("response body %q", body)
	}
	if response.Header.Get("X-End-To-End") != "preserve-me" || response.Header.Get("X-Response-Hop") != "" {
		t.Fatalf("unexpected forwarded headers: %#v", response.Header)
	}

	requestContext, cancelRequest := context.WithCancel(context.Background())
	cancelRequestHTTP, err := http.NewRequestWithContext(requestContext, http.MethodGet, public.URL+"/cancel", nil)
	if err != nil {
		t.Fatal(err)
	}
	cancelRequestHTTP.Host = "demo.opts.ink"
	cancelResult := make(chan error, 1)
	go func() {
		_, requestErr := client.Do(cancelRequestHTTP)
		cancelResult <- requestErr
	}()
	select {
	case <-cancelStarted:
	case <-time.After(time.Second):
		t.Fatal("cancel test did not reach localhost")
	}
	cancelRequest()
	select {
	case <-cancelResult:
	case <-time.After(time.Second):
		t.Fatal("canceled public request did not return")
	}
	select {
	case <-cancelObserved:
	case <-time.After(time.Second):
		t.Fatal("public cancellation did not reach localhost")
	}

	cancel()
	select {
	case err := <-runnerDone:
		if err != nil {
			t.Fatalf("runner stopped: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not stop")
	}
}

func TestHeartbeatDisconnectsSilentClient(t *testing.T) {
	store := newTunnelFakeStore()
	public := httptest.NewServer(New(Config{
		PublicHost: "opts.ink", Heartbeat: 15 * time.Millisecond, HeartbeatGrace: 45 * time.Millisecond,
	}, store))
	defer public.Close()
	target := "ws" + public.URL[len("http"):] + "/tunnel?domainId=" + store.domain.ID.String()
	headers := http.Header{"Authorization": []string{"Bearer test-token"}}
	connection, _, err := websocket.DefaultDialer.Dial(target, headers)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	writeTestEnvelope(t, connection, &tunnelv1.Envelope{
		ProtocolVersion: tunnelProtocolVersion,
		Sequence:        1,
		Message: &tunnelv1.Envelope_ClientHello{ClientHello: &tunnelv1.ClientHello{
			DomainId: store.domain.ID.String(),
		}},
	})
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	for {
		_, _, err = connection.ReadMessage()
		if err != nil {
			break
		}
	}
	if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
		t.Fatal("silent tunnel was not disconnected after heartbeat grace")
	}
}

func TestTunnelEnvelopeFramingPreservesVersionAndMessage(t *testing.T) {
	t.Parallel()
	original := &tunnelv1.Envelope{
		ProtocolVersion: tunnelProtocolVersion,
		Sequence:        42,
		Message: &tunnelv1.Envelope_RequestEnd{RequestEnd: &tunnelv1.RequestEnd{
			RequestId: "request-id",
		}},
	}
	payload, err := proto.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded tunnelv1.Envelope
	if err := proto.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ProtocolVersion != tunnelProtocolVersion ||
		decoded.Sequence != 42 ||
		decoded.GetRequestEnd().GetRequestId() != "request-id" {
		t.Fatalf("unexpected decoded envelope: %v", &decoded)
	}
}

func writeTestEnvelope(t *testing.T, connection *websocket.Conn, envelope *tunnelv1.Envelope) {
	t.Helper()
	payload, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.WriteMessage(websocket.BinaryMessage, payload); err != nil {
		t.Fatal(err)
	}
}

func dialTestTunnel(t *testing.T, serverURL string, domainID uuid.UUID) *websocket.Conn {
	t.Helper()
	target := "ws" + serverURL[len("http"):] + "/tunnel?domainId=" + domainID.String()
	headers := http.Header{"Authorization": []string{"Bearer test-token"}}
	connection, _, err := websocket.DefaultDialer.Dial(target, headers)
	if err != nil {
		t.Fatal(err)
	}
	return connection
}

func writeClientHello(t *testing.T, connection *websocket.Conn, domainID uuid.UUID) {
	t.Helper()
	writeTestEnvelope(t, connection, &tunnelv1.Envelope{
		ProtocolVersion: tunnelProtocolVersion,
		Sequence:        1,
		Message: &tunnelv1.Envelope_ClientHello{ClientHello: &tunnelv1.ClientHello{
			DomainId: domainID.String(),
		}},
	})
}

func readTestEnvelope(t *testing.T, connection *websocket.Conn) *tunnelv1.Envelope {
	t.Helper()
	messageType, payload, err := connection.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("message type = %d", messageType)
	}
	var envelope tunnelv1.Envelope
	if err := proto.Unmarshal(payload, &envelope); err != nil {
		t.Fatal(err)
	}
	return &envelope
}

func TestTunnelFakeStoreConflict(t *testing.T) {
	t.Parallel()
	store := newTunnelFakeStore()
	if _, err := store.ConnectTunnelSession(context.Background(), uuid.New(), uuid.New(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConnectTunnelSession(context.Background(), uuid.New(), uuid.New(), time.Now()); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("duplicate connect error = %v", err)
	}
}
