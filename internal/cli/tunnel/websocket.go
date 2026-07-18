package tunnel

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type webSocket struct {
	connection net.Conn
	reader     *bufio.Reader
	writeMu    sync.Mutex
}

func dialWebSocket(ctx context.Context, rawURL, token string) (*webSocket, error) {
	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse tunnel URL: %w", err)
	}
	if target.Scheme != "ws" && target.Scheme != "wss" {
		return nil, fmt.Errorf("unsupported tunnel URL scheme %q", target.Scheme)
	}

	port := target.Port()
	if port == "" {
		if target.Scheme == "wss" {
			port = "443"
		} else {
			port = "80"
		}
	}
	host := target.Hostname()
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, err
	}
	if target.Scheme == "wss" {
		tlsConnection := tls.Client(connection, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host})
		if err := tlsConnection.HandshakeContext(ctx); err != nil {
			_ = connection.Close()
			return nil, err
		}
		connection = tlsConnection
	}

	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		_ = connection.Close()
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)
	request := &http.Request{
		Method: http.MethodGet,
		URL:    target,
		Host:   target.Host,
		Header: http.Header{
			"Authorization":         []string{"Bearer " + token},
			"Connection":            []string{"Upgrade"},
			"Sec-WebSocket-Key":     []string{key},
			"Sec-WebSocket-Version": []string{"13"},
			"Upgrade":               []string{"websocket"},
		},
	}
	if err := request.Write(connection); err != nil {
		_ = connection.Close()
		return nil, err
	}

	reader := bufio.NewReader(connection)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		_ = connection.Close()
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusSwitchingProtocols {
		_ = connection.Close()
		return nil, fmt.Errorf("tunnel endpoint returned %s", response.Status)
	}
	expected := sha1.Sum([]byte(key + websocketGUID))
	if !strings.EqualFold(response.Header.Get("Upgrade"), "websocket") ||
		response.Header.Get("Sec-WebSocket-Accept") != base64.StdEncoding.EncodeToString(expected[:]) {
		_ = connection.Close()
		return nil, errors.New("invalid WebSocket upgrade response")
	}
	return &webSocket{connection: connection, reader: reader}, nil
}

func (w *webSocket) Close() error {
	_ = w.writeFrame(0x8, nil)
	return w.connection.Close()
}

func (w *webSocket) WriteBinary(payload []byte) error {
	return w.writeFrame(0x2, payload)
}

func (w *webSocket) ReadBinary() ([]byte, error) {
	for {
		opcode, payload, err := w.readFrame()
		if err != nil {
			return nil, err
		}
		switch opcode {
		case 0x2:
			return payload, nil
		case 0x8:
			return nil, io.EOF
		case 0x9:
			if err := w.writeFrame(0xA, payload); err != nil {
				return nil, err
			}
		case 0xA:
			continue
		default:
			return nil, fmt.Errorf("unexpected WebSocket opcode %d", opcode)
		}
	}
}

func (w *webSocket) writeFrame(opcode byte, payload []byte) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	header := []byte{0x80 | opcode}
	length := len(payload)
	switch {
	case length < 126:
		header = append(header, 0x80|byte(length))
	case uint64(length) <= uint64(^uint16(0)):
		header = append(header, 0x80|126, 0, 0)
		binary.BigEndian.PutUint16(header[len(header)-2:], uint16(length))
	default:
		header = append(header, 0x80|127, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.BigEndian.PutUint64(header[len(header)-8:], uint64(length))
	}
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	header = append(header, mask...)
	masked := make([]byte, len(payload))
	for index := range payload {
		masked[index] = payload[index] ^ mask[index%4]
	}
	if _, err := w.connection.Write(header); err != nil {
		return err
	}
	_, err := w.connection.Write(masked)
	return err
}

func (w *webSocket) readFrame() (byte, []byte, error) {
	first, err := w.reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	second, err := w.reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	if first&0x80 == 0 {
		return 0, nil, errors.New("fragmented WebSocket frames are unsupported")
	}
	if second&0x80 != 0 {
		return 0, nil, errors.New("server sent a masked WebSocket frame")
	}
	length := uint64(second & 0x7F)
	switch length {
	case 126:
		var value uint16
		if err := binary.Read(w.reader, binary.BigEndian, &value); err != nil {
			return 0, nil, err
		}
		length = uint64(value)
	case 127:
		if err := binary.Read(w.reader, binary.BigEndian, &length); err != nil {
			return 0, nil, err
		}
	}
	if length > 64<<20 {
		return 0, nil, errors.New("WebSocket frame exceeds 64 MiB")
	}
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(w.reader, payload); err != nil {
		return 0, nil, err
	}
	return first & 0x0F, payload, nil
}
