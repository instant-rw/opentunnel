package credentials

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestProtectedFileFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials")
	store := NewStoreAt(path, "unsupported")

	if err := store.Set("secret-token"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	got, err := store.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "secret-token" {
		t.Fatalf("Get() = %q, want secret-token", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("credential mode = %o, want no group/other permissions", info.Mode().Perm())
	}
	if err := store.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Get(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}
