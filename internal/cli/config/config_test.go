package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	store := NewStoreAt(path)
	want := Config{APIURL: "https://example.test/api/v1", LastDomainID: "domain-id", LastPort: 4321}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != want {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("config mode = %o, want no group/other permissions", info.Mode().Perm())
	}
}

func TestStoreRejectsInvalidPort(t *testing.T) {
	store := NewStoreAt(filepath.Join(t.TempDir(), "config.json"))
	if err := store.Save(Config{LastPort: 70000}); err == nil {
		t.Fatal("Save() error = nil, want invalid port error")
	}
}
