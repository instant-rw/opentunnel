package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOperationalHealthEndpoints(t *testing.T) {
	t.Parallel()
	handler := New(Config{}, &fakeStore{})

	for _, path := range []string{"/healthz", "/readyz", "/api/v1/healthz"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s got status %d", path, response.Code)
		}
	}
}

func TestReadinessReportsDependencyFailure(t *testing.T) {
	t.Parallel()
	handler := New(Config{
		ReadinessCheck: func(context.Context) error {
			return errors.New("database unavailable")
		},
	}, &fakeStore{})

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("got status %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "not_ready") {
		t.Fatalf("got body %q", response.Body.String())
	}
}

func TestDashboardAssetsAreServedWithSecurityHeaders(t *testing.T) {
	t.Parallel()
	handler := New(Config{}, &fakeStore{})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("got status %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "<title>OpenTunnel</title>") {
		t.Fatalf("dashboard response did not contain title: %q", response.Body.String())
	}
	if response.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("dashboard response omitted clickjacking protection")
	}
}

func TestDashboardServesDeviceApprovalPath(t *testing.T) {
	t.Parallel()
	handler := New(Config{BaseURL: "https://opts.ink", PublicHost: "opts.ink"}, &fakeStore{})
	request := httptest.NewRequest(http.MethodGet, "/device?user_code=B182-3514", nil)
	request.Host = "opts.ink"
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("got status %d body %q", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "<title>OpenTunnel</title>") {
		t.Fatalf("device path did not serve dashboard: %q", response.Body.String())
	}
}

func TestDashboardServesUnderscorePrefixedNextAssets(t *testing.T) {
	t.Parallel()
	handler := New(Config{}, &fakeStore{})
	request := httptest.NewRequest(http.MethodGet, "/_next/static/chunks/embed-probe.js", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("got status %d body %q", response.Code, response.Body.String())
	}
	contentType := response.Header().Get("Content-Type")
	if !strings.Contains(contentType, "javascript") && !strings.Contains(contentType, "ecmascript") {
		t.Fatalf("unexpected content type %q", contentType)
	}
	if !strings.Contains(response.Body.String(), "__OPENTUNNEL_EMBED_PROBE__") {
		t.Fatalf("probe asset missing from embed: %q", response.Body.String())
	}
}
