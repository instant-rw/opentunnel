package tunnel

import (
	"net/http"
	"testing"

	tunnelv1 "github.com/opentunnel/opentunnel/internal/gen/tunnel/v1"
)

func TestEndpoint(t *testing.T) {
	got, err := Endpoint("https://example.test/api/v1", "domain id")
	if err != nil {
		t.Fatalf("Endpoint() error = %v", err)
	}
	want := "wss://example.test/tunnel?domainId=domain+id"
	if got != want {
		t.Fatalf("Endpoint() = %q, want %q", got, want)
	}
}

func TestCopyRequestHeadersRemovesHopByHopHeaders(t *testing.T) {
	headers := http.Header{}
	copyRequestHeaders(headers, []*tunnelv1.Header{
		{Name: "X-Test", Values: []string{"one", "two"}},
		{Name: "Connection", Values: []string{"keep-alive"}},
	})
	if got := headers.Values("X-Test"); len(got) != 2 {
		t.Fatalf("X-Test values = %v, want two values", got)
	}
	if got := headers.Get("Connection"); got != "" {
		t.Fatalf("Connection = %q, want empty", got)
	}
}
