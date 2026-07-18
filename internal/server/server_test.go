package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/opentunnel/opentunnel/internal/auth"
	"github.com/opentunnel/opentunnel/internal/storage"
)

type fakeStore struct {
	Store
	session      storage.Session
	createdSlug  string
	createdOwner uuid.UUID
}

func (f *fakeStore) SessionByToken(context.Context, []byte) (storage.Session, error) {
	return f.session, nil
}

func (f *fakeStore) CreateDomain(_ context.Context, owner uuid.UUID, slug string) (storage.Domain, error) {
	f.createdOwner = owner
	f.createdSlug = slug
	return storage.Domain{ID: uuid.New(), UserID: owner, Slug: slug, CreatedAt: time.Now()}, nil
}

func TestCreateDomainNormalizesSlugAndUsesAuthenticatedOwner(t *testing.T) {
	t.Parallel()
	user := storage.User{ID: uuid.New()}
	store := &fakeStore{}
	server := &Server{config: Config{PublicHost: "opts.ink"}, store: store}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/domains", bytes.NewBufferString(`{"slug":"My-Tunnel"}`))
	request = request.WithContext(context.WithValue(request.Context(), userContextKey, user))
	response := httptest.NewRecorder()

	server.CreateDomain(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("got status %d: %s", response.Code, response.Body.String())
	}
	if store.createdSlug != "my-tunnel" || store.createdOwner != user.ID {
		t.Fatalf("created slug %q for owner %s", store.createdSlug, store.createdOwner)
	}
}

func TestCreateDomainRejectsReservedSlug(t *testing.T) {
	t.Parallel()
	user := storage.User{ID: uuid.New()}
	store := &fakeStore{}
	server := &Server{config: Config{PublicHost: "opts.ink"}, store: store}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/domains", bytes.NewBufferString(`{"slug":"api"}`))
	request = request.WithContext(context.WithValue(request.Context(), userContextKey, user))
	response := httptest.NewRecorder()

	server.CreateDomain(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("got status %d", response.Code)
	}
	if store.createdSlug != "" {
		t.Fatal("reserved domain reached storage")
	}
}

func TestCookieMutationRequiresCSRFToken(t *testing.T) {
	t.Parallel()
	csrf := "csrf-secret"
	store := &fakeStore{session: storage.Session{
		User:     storage.User{ID: uuid.New()},
		CSRFHash: auth.Digest(csrf),
	}}
	server := &Server{store: store}
	called := false
	handler := server.authenticate(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/domains", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-secret"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || called {
		t.Fatalf("request without CSRF got status %d, called=%v", response.Code, called)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/domains", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-secret"})
	request.Header.Set(csrfHeader, csrf)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !called {
		t.Fatalf("request with CSRF got status %d, called=%v", response.Code, called)
	}
}
