package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/creativeprojects/go-selfupdate"
)

const defaultRepositorySlug = "instant-rw/opentunnel"

// Result describes a completed or skipped update check.
type Result struct {
	Current   string
	Latest    string
	Updated   bool
	UpToDate  bool
	AssetName string
}

// CheckLatest returns the newest release without modifying the binary.
func CheckLatest(ctx context.Context) (*selfupdate.Release, bool, error) {
	updater, err := newUpdater()
	if err != nil {
		return nil, false, err
	}
	return updater.DetectLatest(ctx, selfupdate.ParseSlug(repositorySlug()))
}

// Apply downloads, verifies checksums.txt, and replaces the running binary.
func Apply(ctx context.Context, current string) (Result, error) {
	current = normalizeVersion(current)
	if current == "" || current == "dev" {
		return Result{Current: current}, errors.New("self-update requires a released binary version (not \"dev\")")
	}

	updater, err := newUpdater()
	if err != nil {
		return Result{Current: current}, err
	}
	latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(repositorySlug()))
	if err != nil {
		return Result{Current: current}, fmt.Errorf("detect latest release: %w", err)
	}
	if !found {
		return Result{Current: current}, errors.New("no GitHub release found for this platform")
	}

	result := Result{
		Current:   current,
		Latest:    latest.Version(),
		AssetName: latest.AssetName,
	}
	if latest.LessOrEqual(current) {
		result.UpToDate = true
		return result, nil
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return result, fmt.Errorf("locate executable: %w", err)
	}
	if err := updater.UpdateTo(ctx, latest, exe); err != nil {
		return result, fmt.Errorf("apply update: %w", err)
	}
	result.Updated = true
	return result, nil
}

func newUpdater() (*selfupdate.Updater, error) {
	return selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
}

func repositorySlug() string {
	if value := strings.TrimSpace(os.Getenv("OPENTUNNEL_REPOSITORY")); value != "" {
		return value
	}
	return defaultRepositorySlug
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}
