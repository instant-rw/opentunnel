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

func TestApexReturnsServiceJSON(t *testing.T) {
	t.Parallel()
	handler := New(Config{}, &fakeStore{})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("got status %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"service":"opentunnel"`) {
		t.Fatalf("apex response = %q", response.Body.String())
	}
}

func TestApexRedirectsToFrontendURL(t *testing.T) {
	t.Parallel()
	handler := New(Config{
		BaseURL:     "https://api.opts.ink",
		FrontendURL: "https://app.opts.ink",
	}, &fakeStore{})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusFound {
		t.Fatalf("got status %d", response.Code)
	}
	if got := response.Header().Get("Location"); got != "https://app.opts.ink/" {
		t.Fatalf("location = %q", got)
	}
}

func TestCORSPreflightAllowsConfiguredOrigin(t *testing.T) {
	t.Parallel()
	handler := New(Config{
		CORSOrigins: []string{"http://localhost:3000"},
	}, &fakeStore{})
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/me", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", "GET")
	request.Header.Set("Access-Control-Request-Headers", "X-CSRF-Token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("got status %d", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("allow origin = %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
	if response.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("credentials not allowed")
	}
	if !strings.Contains(response.Header().Get("Access-Control-Allow-Headers"), "X-CSRF-Token") {
		t.Fatalf("allow headers = %q", response.Header().Get("Access-Control-Allow-Headers"))
	}
}
