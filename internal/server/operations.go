package server

import (
	"bufio"
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

func newDashboardHandler(assets fs.FS) http.Handler {
	root, err := fs.Sub(assets, "out")
	if err != nil {
		panic(err)
	}
	if _, err := fs.Stat(root, "_next"); err != nil {
		slog.Warn("dashboard _next assets missing from embed; rebuild the Docker image so Next.js chunks are included")
	}
	files := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline' https://static.cloudflareinsights.com")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		path := r.URL.Path
		if path == "/" || strings.HasSuffix(path, ".html") || !looksLikeStaticAsset(path) {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		if shouldServeDashboardIndex(root, path) {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	})
}

func looksLikeStaticAsset(path string) bool {
	base := path
	if index := strings.LastIndex(path, "/"); index >= 0 {
		base = path[index+1:]
	}
	return strings.Contains(base, ".")
}

func shouldServeDashboardIndex(root fs.FS, path string) bool {
	if path == "/" || looksLikeStaticAsset(path) {
		return false
	}
	cleaned := strings.Trim(path, "/")
	if cleaned == "" {
		return false
	}
	if _, err := fs.Stat(root, cleaned); err == nil {
		return false
	}
	if _, err := fs.Stat(root, cleaned+"/index.html"); err == nil {
		return false
	}
	return true
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeStatus(w, http.StatusOK, "ok")
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if s.config.ReadinessCheck != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.config.ReadinessCheck(ctx); err != nil {
			slog.Warn("readiness check failed", "error", err)
			writeStatus(w, http.StatusServiceUnavailable, "not_ready")
			return
		}
	}
	writeStatus(w, http.StatusOK, "ready")
}

func writeStatus(w http.ResponseWriter, status int, value string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": value})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

func (w *statusWriter) Flush() {
	http.NewResponseController(w.ResponseWriter).Flush()
}

func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(writer, r)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"host", r.Host,
			"status", writer.status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}
