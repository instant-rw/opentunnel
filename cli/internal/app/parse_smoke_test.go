package app

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opentunnel/opentunnel/cli/internal/config"
	"github.com/opentunnel/opentunnel/cli/internal/credentials"
)

func TestUpDomainFlagAfterPort(t *testing.T) {
	dir := t.TempDir()
	creds := credentials.NewStoreAt(filepath.Join(dir, "credentials"), "unsupported")
	if err := creds.Set("test-token"); err != nil {
		t.Fatal(err)
	}
	var errOut bytes.Buffer
	a := &App{
		Out:         &bytes.Buffer{},
		ErrOut:      &errOut,
		Config:      config.NewStoreAt(filepath.Join(dir, "config.json")),
		Credentials: creds,
		OpenBrowser: func(string) error { return nil },
	}
	// With a broken global interspersed setting, this fails as "unknown flag: --domain"
	// before any network I/O. Any other error means flag parsing succeeded.
	err := a.Run(context.Background(), []string{"--api-url", "http://127.0.0.1:1", "up", "3000", "--domain", "sike"})
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("global flag parse ate --domain: %v\nstderr: %s", err, errOut.String())
	}
	if strings.Contains(errOut.String(), "unknown flag") {
		t.Fatalf("stderr reported unknown flag: %s", errOut.String())
	}
}
