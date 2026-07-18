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
