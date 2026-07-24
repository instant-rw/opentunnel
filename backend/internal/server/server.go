package server

import (
	"context"
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/opentunnel/opentunnel/backend/internal/auth"
	"github.com/opentunnel/opentunnel/backend/internal/storage"
	"github.com/opentunnel/opentunnel/shared/gen/api"
)

const (
	sessionCookie = "opentunnel_session"
	csrfCookie    = "opentunnel_csrf"
	csrfHeader    = "X-CSRF-Token"
)

type Config struct {
	BaseURL           string
	FrontendURL       string
	PublicHost        string
	CookieDomain      string
	CORSOrigins       []string
	SecureCookies     bool
	SessionLifetime   time.Duration
	Heartbeat         time.Duration
	HeartbeatGrace    time.Duration
	RequestTimeout    time.Duration
	MaxInFlight       int
	MaxChunkBytes     int
	QueueDepth        int
	CaptureBodyBytes  int
	MaxStoredRequests int
	ReadinessCheck    func(context.Context) error
}

type Store interface {
	CreateUser(context.Context, string, string) (storage.User, error)
	UserByEmail(context.Context, string) (storage.User, error)
	CreateSession(context.Context, uuid.UUID, []byte, []byte, time.Time) error
	SessionByToken(context.Context, []byte) (storage.Session, error)
	RevokeSession(context.Context, []byte) error
	UserByCLIToken(context.Context, []byte) (storage.User, error)
	CreateDeviceAuthorization(context.Context, []byte, string, int, time.Time) (storage.DeviceAuthorization, error)
	ApproveDeviceAuthorization(context.Context, uuid.UUID, string) error
	ExchangeDeviceCode(context.Context, []byte, []byte) (string, error)
	ListTokens(context.Context, uuid.UUID) ([]storage.Token, error)
	RevokeToken(context.Context, uuid.UUID, uuid.UUID) error
	CreateDomain(context.Context, uuid.UUID, string) (storage.Domain, error)
	ListDomains(context.Context, uuid.UUID) ([]storage.Domain, error)
	Domain(context.Context, uuid.UUID, uuid.UUID) (storage.Domain, error)
	DeleteDomain(context.Context, uuid.UUID, uuid.UUID) error
}

type contextKey int

const (
	userContextKey contextKey = iota
	sessionTokenContextKey
)

type Server struct {
	api.Unimplemented
	config     Config
	store      Store
	tunnels    tunnelStore
	inspection inspectionStore
	registry   *sessionRegistry
	events     *domainEventHub
	handler    http.Handler
}

func New(config Config, store Store) *Server {
	if config.SessionLifetime == 0 {
		config.SessionLifetime = 30 * 24 * time.Hour
	}
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:8080"
	}
	if config.FrontendURL == "" {
		config.FrontendURL = config.BaseURL
	}
	if config.PublicHost == "" {
		config.PublicHost = "localhost"
	}
	if config.Heartbeat == 0 {
		config.Heartbeat = 10 * time.Second
	}
	if config.HeartbeatGrace == 0 {
		config.HeartbeatGrace = 3 * config.Heartbeat
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 2 * time.Minute
	}
	if config.MaxInFlight == 0 {
		config.MaxInFlight = 64
	}
	if config.MaxChunkBytes == 0 {
		config.MaxChunkBytes = 64 << 10
	}
	if config.QueueDepth == 0 {
		config.QueueDepth = 128
	}
	if config.CaptureBodyBytes == 0 {
		config.CaptureBodyBytes = 256 << 10
	}
	if config.MaxStoredRequests == 0 {
		config.MaxStoredRequests = 1000
	}

	implementation := &Server{
		config:   config,
		store:    store,
		registry: newSessionRegistry(),
		events:   newDomainEventHub(),
	}
	implementation.tunnels, _ = store.(tunnelStore)
	implementation.inspection, _ = store.(inspectionStore)
	mux := http.NewServeMux()
	apiHandler := api.HandlerWithOptions(implementation, api.ChiServerOptions{BaseURL: "/api/v1"})
	mux.Handle("/api/v1/", implementation.cors(implementation.rateLimit(implementation.authenticate(apiHandler))))
	mux.HandleFunc("/healthz", implementation.health)
	mux.HandleFunc("/readyz", implementation.ready)
	mux.HandleFunc("/", implementation.handleApex)
	implementation.handler = implementation.accessLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := implementation.publicSlug(r.Host); ok {
			implementation.handlePublicRequest(w, r)
			return
		}
		if implementation.isPublicHostCandidate(r.Host) {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/tunnel" {
			implementation.handleTunnel(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	return implementation
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) Close() {
	s.registry.closeAll()
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublic(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		var user storage.User
		var sessionToken string
		authorization := r.Header.Get("Authorization")
		if authorization != "" {
			token, err := auth.ParseBearer(authorization)
			if err != nil {
				writeProblem(w, http.StatusUnauthorized, "invalid_token", "Invalid token")
				return
			}
			user, err = s.store.UserByCLIToken(r.Context(), auth.Digest(token))
			if err != nil {
				writeProblem(w, http.StatusUnauthorized, "invalid_token", "Invalid token")
				return
			}
		} else {
			cookie, err := r.Cookie(sessionCookie)
			if err != nil {
				writeProblem(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
				return
			}
			sessionToken = cookie.Value
			session, err := s.store.SessionByToken(r.Context(), auth.Digest(cookie.Value))
			if err != nil {
				s.clearAuthCookies(w)
				writeProblem(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
				return
			}
			user = session.User
			if requiresCSRF(r.Method) {
				csrf := r.Header.Get(csrfHeader)
				if csrf == "" || subtle.ConstantTimeCompare(auth.Digest(csrf), session.CSRFHash) != 1 {
					writeProblem(w, http.StatusForbidden, "csrf_failed", "CSRF validation failed")
					return
				}
			}
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		if sessionToken != "" {
			ctx = context.WithValue(ctx, sessionTokenContextKey, sessionToken)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublic(path string) bool {
	return path == "/api/v1/healthz" ||
		path == "/api/v1/auth/register" ||
		path == "/api/v1/auth/login" ||
		path == "/api/v1/device/authorizations" ||
		path == "/api/v1/device/token"
}

func requiresCSRF(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if s.allowsCORSOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Expose-Headers", "Retry-After")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) allowsCORSOrigin(origin string) bool {
	if origin == "" || len(s.config.CORSOrigins) == 0 {
		return false
	}
	for _, allowed := range s.config.CORSOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

type visitor struct {
	window time.Time
	count  int
}

func (s *Server) rateLimit(next http.Handler) http.Handler {
	var mu sync.Mutex
	visitors := make(map[string]visitor)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		key := host + "\x00" + r.URL.Path
		now := time.Now()
		limit := 120
		if strings.Contains(r.URL.Path, "/auth/") || r.URL.Path == "/api/v1/device/token" {
			limit = 20
		}
		mu.Lock()
		current := visitors[key]
		if now.Sub(current.window) >= time.Minute {
			current = visitor{window: now}
		}
		current.count++
		visitors[key] = current
		mu.Unlock()
		if current.count > limit {
			w.Header().Set("Retry-After", "60")
			writeProblem(w, http.StatusTooManyRequests, "rate_limited", "Too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func userFromContext(ctx context.Context) storage.User {
	return ctx.Value(userContextKey).(storage.User)
}
