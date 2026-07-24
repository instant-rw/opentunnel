package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/opentunnel/opentunnel/backend/internal/server"
	"github.com/opentunnel/opentunnel/backend/internal/storage"
	"github.com/opentunnel/opentunnel/backend/migrations"
)

func main() {
	configureLogging()
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	store, err := storage.Open(ctx, databaseURL)
	cancel()
	if err != nil {
		return err
	}
	defer store.Close()

	migrationContext, migrationCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer migrationCancel()
	if err := migrations.Up(migrationContext, store.Pool()); err != nil {
		return err
	}

	address := os.Getenv("OPENTUNNEL_HTTP_ADDR")
	if address == "" {
		address = ":8080"
	}
	baseURL := os.Getenv("OPENTUNNEL_BASE_URL")
	frontendURL := os.Getenv("OPENTUNNEL_FRONTEND_URL")
	publicHost := os.Getenv("OPENTUNNEL_PUBLIC_HOST")
	cookieDomain := os.Getenv("OPENTUNNEL_COOKIE_DOMAIN")
	secureCookies := os.Getenv("OPENTUNNEL_SECURE_COOKIES") != "false"

	handler := server.New(server.Config{
		BaseURL:           baseURL,
		FrontendURL:       frontendURL,
		PublicHost:        publicHost,
		CookieDomain:      cookieDomain,
		CORSOrigins:       splitCSV(os.Getenv("OPENTUNNEL_CORS_ORIGINS")),
		SecureCookies:     secureCookies,
		CaptureBodyBytes:  envInt("OPENTUNNEL_CAPTURE_BODY_BYTES"),
		MaxStoredRequests: envInt("OPENTUNNEL_MAX_STORED_REQUESTS"),
		MaxInFlight:       envInt("OPENTUNNEL_MAX_IN_FLIGHT"),
		MaxChunkBytes:     envInt("OPENTUNNEL_MAX_CHUNK_BYTES"),
		QueueDepth:        envInt("OPENTUNNEL_QUEUE_DEPTH"),
		Heartbeat:         envDuration("OPENTUNNEL_HEARTBEAT"),
		HeartbeatGrace:    envDuration("OPENTUNNEL_HEARTBEAT_GRACE"),
		RequestTimeout:    envDuration("OPENTUNNEL_REQUEST_TIMEOUT"),
		ReadinessCheck:    store.Pool().Ping,
	}, store)
	httpServer := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	httpServer.RegisterOnShutdown(handler.Close)

	shutdownSignal, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	errs := make(chan error, 1)
	go func() {
		slog.Info("listening", "address", address)
		errs <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-shutdownSignal.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownContext)
	}
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			origins = append(origins, part)
		}
	}
	return origins
}

func envInt(name string) int {
	value, _ := strconv.Atoi(os.Getenv(name))
	return value
}

func envDuration(name string) time.Duration {
	value, _ := time.ParseDuration(os.Getenv(name))
	return value
}

func configureLogging() {
	level := slog.LevelInfo
	switch os.Getenv("OPENTUNNEL_LOG_LEVEL") {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
}
