package storage

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/opentunnel/opentunnel/backend/migrations"
)

func TestPostgresAuthAndDomainOwnership(t *testing.T) {
	databaseURL := os.Getenv("OPENTUNNEL_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENTUNNEL_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := migrations.Up(ctx, store.Pool()); err != nil {
		t.Fatal(err)
	}

	email := "integration-" + time.Now().Format("20060102150405.000000000") + "@example.com"
	owner, err := store.CreateUser(ctx, email, "hash")
	if err != nil {
		t.Fatal(err)
	}
	other, err := store.CreateUser(ctx, "other-"+email, "hash")
	if err != nil {
		t.Fatal(err)
	}
	domain, err := store.CreateDomain(ctx, owner.ID, "integration-"+time.Now().Format("150405"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Domain(ctx, other.ID, domain.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user read domain: %v", err)
	}
	if err := store.DeleteDomain(ctx, other.ID, domain.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user deleted domain: %v", err)
	}
	if _, err := store.Domain(ctx, owner.ID, domain.ID); err != nil {
		t.Fatalf("owner could not read domain: %v", err)
	}
}
