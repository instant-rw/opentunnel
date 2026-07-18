package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tunnelv1 "github.com/opentunnel/opentunnel/internal/gen/tunnel/v1"
	"google.golang.org/protobuf/proto"
)

const (
	protocolVersion = 1
	defaultChunk    = 64 << 10
)

type State struct {
	Connected         bool
	RequestCount      uint64
	ReconnectAttempts int
}

type Runner struct {
	APIURL        string
	Token         string
	DomainID      string
	LocalPort     int
	ClientVersion string
	OnState       func(State)

	sequence atomic.Uint64
	sendMu   sync.Mutex
}

type pendingRequest struct {
	start          *tunnelv1.RequestStart
	body           *io.PipeWriter
	chunks         chan []byte
	responseWindow *creditWindow
	cancel         context.CancelFunc
	closed         atomic.Bool
}

type creditWindow struct {
	available uint64
	updates   chan uint64
}

func newCreditWindow() *creditWindow {
	return &creditWindow{updates: make(chan uint64, 16)}
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

// Endpoint documents the data-plane endpoint expected from the server-owned
// implementation. It intentionally derives from the configured control URL.
func Endpoint(apiURL, domainID string) (string, error) {
	base, err := url.Parse(apiURL)
	if err != nil {
		return "", err
	}
	switch base.Scheme {
	case "http":
		base.Scheme = "ws"
	case "https":
		base.Scheme = "wss"
	default:
		return "", fmt.Errorf("unsupported API URL scheme %q", base.Scheme)
	}
	base.Path = "/tunnel"
	query := base.Query()
	query.Set("domainId", domainID)
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func (r *Runner) Run(ctx context.Context) error {
	if r.LocalPort < 1 || r.LocalPort > 65535 {
		return fmt.Errorf("invalid local port %d", r.LocalPort)
	}
	endpoint, err := Endpoint(r.APIURL, r.DomainID)
	if err != nil {
		return err
	}

	attempt := 0
	requestCount := uint64(0)
	for {
		if ctx.Err() != nil {
			return nil
		}
		r.report(State{ReconnectAttempts: attempt, RequestCount: requestCount})
		connection, err := dialWebSocket(ctx, endpoint, r.Token)
		if err == nil {
			r.report(State{Connected: true, ReconnectAttempts: attempt, RequestCount: requestCount})
			count, sessionErr := r.runSession(ctx, connection)
			requestCount += count
			_ = connection.Close()
			if ctx.Err() != nil {
				return nil
			}
			err = sessionErr
		}

		attempt++
		r.report(State{ReconnectAttempts: attempt, RequestCount: requestCount})
		delay := time.Second << min(attempt-1, 5)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
		if err != nil && attempt == 1 {
			continue
		}
	}
}

func (r *Runner) runSession(ctx context.Context, connection *webSocket) (uint64, error) {
	sessionContext, cancelSession := context.WithCancel(ctx)
	defer cancelSession()
	hello := &tunnelv1.Envelope{
		ProtocolVersion: protocolVersion,
		Sequence:        r.sequence.Add(1),
		Message: &tunnelv1.Envelope_ClientHello{ClientHello: &tunnelv1.ClientHello{
			DomainId:      r.DomainID,
			ClientVersion: r.ClientVersion,
		}},
	}
	if err := writeEnvelope(connection, hello); err != nil {
		return 0, err
	}

	pending := make(map[string]*pendingRequest)
	var pendingMu sync.Mutex
	defer func() {
		pendingMu.Lock()
		defer pendingMu.Unlock()
		for _, request := range pending {
			request.cancel()
			request.stop(errors.New("tunnel disconnected"))
		}
	}()
	var count atomic.Uint64
	chunkSize := defaultChunk
	var lastSequence uint64
	for {
		if ctx.Err() != nil {
			return count.Load(), nil
		}
		payload, err := connection.ReadBinary()
		if err != nil {
			return count.Load(), err
		}
		var envelope tunnelv1.Envelope
		if err := proto.Unmarshal(payload, &envelope); err != nil {
			return count.Load(), fmt.Errorf("decode tunnel frame: %w", err)
		}
		if envelope.ProtocolVersion != protocolVersion {
			return count.Load(), fmt.Errorf("server uses unsupported tunnel protocol %d", envelope.ProtocolVersion)
		}
		if envelope.Sequence <= lastSequence {
			return count.Load(), fmt.Errorf("server sent out-of-order tunnel sequence %d", envelope.Sequence)
		}
		lastSequence = envelope.Sequence

		switch message := envelope.Message.(type) {
		case *tunnelv1.Envelope_ServerHello:
			if message.ServerHello.MaxChunkBytes > 0 {
				chunkSize = int(message.ServerHello.MaxChunkBytes)
			}
		case *tunnelv1.Envelope_RequestStart:
			requestContext, cancel := requestContext(sessionContext, message.RequestStart)
			reader, writer := io.Pipe()
			requestID := message.RequestStart.RequestId
			request := &pendingRequest{
				start: message.RequestStart, body: writer,
				chunks: make(chan []byte, 16), responseWindow: newCreditWindow(), cancel: cancel,
			}
			pendingMu.Lock()
			if existing := pending[requestID]; existing != nil {
				existing.cancel()
				existing.stop(errors.New("duplicate request start"))
			}
			pending[requestID] = request
			pendingMu.Unlock()
			go request.pump(func(bytes int) {
				_ = r.send(connection, &tunnelv1.Envelope_WindowUpdate{WindowUpdate: &tunnelv1.WindowUpdate{
					RequestId:   requestID,
					CreditBytes: uint64(bytes),
				}})
			})
			if err := r.send(connection, &tunnelv1.Envelope_WindowUpdate{WindowUpdate: &tunnelv1.WindowUpdate{
				RequestId:   requestID,
				CreditBytes: uint64(16 * chunkSize),
			}}); err != nil {
				request.cancel()
				request.stop(err)
			}
			count.Add(1)
			go r.forward(requestContext, connection, message.RequestStart, reader, request.responseWindow, chunkSize, func() {
				cancel()
				pendingMu.Lock()
				if pending[requestID] == request {
					delete(pending, requestID)
				}
				pendingMu.Unlock()
			})
		case *tunnelv1.Envelope_RequestBody:
			pendingMu.Lock()
			request := pending[message.RequestBody.RequestId]
			pendingMu.Unlock()
			if request != nil {
				if len(message.RequestBody.Data) > chunkSize {
					request.cancel()
					request.stop(errors.New("request chunk exceeds negotiated limit"))
					continue
				}
				if request.closed.Load() {
					continue
				}
				data := append([]byte(nil), message.RequestBody.Data...)
				select {
				case request.chunks <- data:
				default:
					request.cancel()
					request.stop(errors.New("request body queue is full"))
					r.sendLocalFailure(connection, message.RequestBody.RequestId, errors.New("request body backpressure limit exceeded"))
				}
			}
		case *tunnelv1.Envelope_RequestEnd:
			pendingMu.Lock()
			request := pending[message.RequestEnd.RequestId]
			pendingMu.Unlock()
			if request != nil {
				request.stop(nil)
			}
		case *tunnelv1.Envelope_Ping:
			_ = r.send(connection, &tunnelv1.Envelope_Pong{Pong: &tunnelv1.Pong{SentAtUnixMilli: message.Ping.SentAtUnixMilli}})
		case *tunnelv1.Envelope_Cancel:
			pendingMu.Lock()
			request := pending[message.Cancel.RequestId]
			pendingMu.Unlock()
			if request != nil {
				request.cancel()
				request.stop(context.Canceled)
			}
		case *tunnelv1.Envelope_WindowUpdate:
			pendingMu.Lock()
			request := pending[message.WindowUpdate.RequestId]
			pendingMu.Unlock()
			if request != nil && !request.responseWindow.add(message.WindowUpdate.CreditBytes) {
				request.cancel()
				request.stop(errors.New("response credit queue is full"))
			}
		case *tunnelv1.Envelope_Error:
			if message.Error.Fatal {
				return count.Load(), fmt.Errorf("server tunnel error: %s", message.Error.Detail)
			}
		default:
			return count.Load(), errors.New("unexpected tunnel protocol message")
		}
	}
}

func (p *pendingRequest) pump(grantCredit func(int)) {
	for chunk := range p.chunks {
		if _, err := p.body.Write(chunk); err != nil {
			p.cancel()
			return
		}
		grantCredit(len(chunk))
	}
	_ = p.body.Close()
}

func (p *pendingRequest) stop(err error) {
	if !p.closed.CompareAndSwap(false, true) {
		return
	}
	close(p.chunks)
	if err != nil {
		_ = p.body.CloseWithError(err)
	}
}

func (r *Runner) forward(
	ctx context.Context,
	connection *webSocket,
	start *tunnelv1.RequestStart,
	body io.Reader,
	window *creditWindow,
	chunkSize int,
	done func(),
) {
	defer done()
	target := "http://127.0.0.1:" + strconv.Itoa(r.LocalPort) + start.Path
	if start.RawQuery != "" {
		target += "?" + start.RawQuery
	}
	request, err := http.NewRequestWithContext(ctx, start.Method, target, body)
	if err != nil {
		r.sendLocalFailure(connection, start.RequestId, err)
		return
	}
	request.ContentLength = start.ContentLength
	copyRequestHeaders(request.Header, start.Headers)
	client := &http.Client{
		Transport: &http.Transport{Proxy: nil},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		r.sendLocalFailure(connection, start.RequestId, err)
		return
	}
	defer response.Body.Close()

	headers := responseHeaders(response.Header)
	if err := r.send(connection, &tunnelv1.Envelope_ResponseStart{ResponseStart: &tunnelv1.ResponseStart{
		RequestId:  start.RequestId,
		StatusCode: uint32(response.StatusCode),
		Headers:    headers,
	}}); err != nil {
		return
	}

	buffer := make([]byte, max(1, chunkSize))
	for {
		read, readErr := response.Body.Read(buffer)
		if read > 0 {
			if err := window.take(ctx, uint64(read)); err != nil {
				return
			}
			data := append([]byte(nil), buffer[:read]...)
			if err := r.send(connection, &tunnelv1.Envelope_ResponseBody{ResponseBody: &tunnelv1.ResponseBody{
				RequestId: start.RequestId,
				Data:      data,
			}}); err != nil {
				return
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			r.sendLocalFailure(connection, start.RequestId, readErr)
			return
		}
	}
	_ = r.send(connection, &tunnelv1.Envelope_ResponseEnd{ResponseEnd: &tunnelv1.ResponseEnd{RequestId: start.RequestId}})
}

func (r *Runner) sendLocalFailure(connection *webSocket, requestID string, err error) {
	_ = r.send(connection, &tunnelv1.Envelope_Cancel{Cancel: &tunnelv1.Cancel{
		RequestId: requestID,
		Reason:    tunnelv1.CancelReason_CANCEL_REASON_LOCAL_FAILURE,
		Detail:    err.Error(),
	}})
}

func (r *Runner) send(connection *webSocket, message any) error {
	r.sendMu.Lock()
	defer r.sendMu.Unlock()
	envelope := &tunnelv1.Envelope{
		ProtocolVersion: protocolVersion,
		Sequence:        r.sequence.Add(1),
	}
	switch typed := message.(type) {
	case *tunnelv1.Envelope_ResponseStart:
		envelope.Message = typed
	case *tunnelv1.Envelope_ResponseBody:
		envelope.Message = typed
	case *tunnelv1.Envelope_ResponseEnd:
		envelope.Message = typed
	case *tunnelv1.Envelope_Cancel:
		envelope.Message = typed
	case *tunnelv1.Envelope_Pong:
		envelope.Message = typed
	case *tunnelv1.Envelope_WindowUpdate:
		envelope.Message = typed
	default:
		return fmt.Errorf("unsupported outbound tunnel message %T", message)
	}
	return writeEnvelope(connection, envelope)
}

func writeEnvelope(connection *webSocket, envelope *tunnelv1.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}
	return connection.WriteBinary(data)
}

func (r *Runner) report(state State) {
	if r.OnState != nil {
		r.OnState(state)
	}
}

func copyRequestHeaders(destination http.Header, headers []*tunnelv1.Header) {
	connectionHeaders := make(map[string]struct{})
	for _, header := range headers {
		if header != nil && strings.EqualFold(header.Name, "Connection") {
			for _, value := range header.Values {
				for _, name := range strings.Split(value, ",") {
					connectionHeaders[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
				}
			}
		}
	}
	for _, header := range headers {
		_, connectionSpecific := connectionHeaders[strings.ToLower(header.GetName())]
		if header != nil && !connectionSpecific && !isHopByHop(header.Name) {
			for _, value := range header.Values {
				destination.Add(header.Name, value)
			}
		}
	}
}

func requestContext(parent context.Context, start *tunnelv1.RequestStart) (context.Context, context.CancelFunc) {
	if start.DeadlineUnixMilli > 0 {
		return context.WithDeadline(parent, time.UnixMilli(start.DeadlineUnixMilli))
	}
	return context.WithCancel(parent)
}

func responseHeaders(source http.Header) []*tunnelv1.Header {
	headers := source.Clone()
	for _, value := range source.Values("Connection") {
		for _, name := range strings.Split(value, ",") {
			headers.Del(strings.TrimSpace(name))
		}
	}
	result := make([]*tunnelv1.Header, 0, len(headers))
	for name, values := range headers {
		if !isHopByHop(name) {
			result = append(result, &tunnelv1.Header{Name: name, Values: values})
		}
	}
	return result
}

func isHopByHop(name string) bool {
	switch strings.ToLower(name) {
	case "connection", "proxy-connection", "keep-alive", "proxy-authenticate",
		"proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}
