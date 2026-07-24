package update

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/creativeprojects/go-selfupdate"
	"github.com/creativeprojects/go-selfupdate/update"
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

// Release is the newest GitHub release that has an asset for this platform.
type Release struct {
	Tag       string
	Version   *semver.Version
	AssetName string
	AssetURL  string
	Checksums string
}

// VersionString returns the release version (e.g. "0.5.0" for tag v0.5).
func (r *Release) VersionString() string {
	if r == nil || r.Version == nil {
		return ""
	}
	return r.Version.String()
}

// LessOrEqual reports whether this release is <= current.
func (r *Release) LessOrEqual(current string) bool {
	if r == nil || r.Version == nil {
		return true
	}
	parsed, err := semver.NewVersion(normalizeVersion(current))
	if err != nil {
		return false
	}
	return r.Version.Compare(parsed) <= 0
}

// CheckLatest returns the newest release with a matching platform asset.
func CheckLatest(ctx context.Context) (*Release, bool, error) {
	release, err := detectLatest(ctx)
	if err != nil {
		if errors.Is(err, errNoPlatformAsset) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return release, true, nil
}

// Apply downloads, verifies checksums.txt, and replaces the running binary.
func Apply(ctx context.Context, current string) (Result, error) {
	current = normalizeVersion(current)
	if current == "" || current == "dev" {
		return Result{Current: current}, errors.New("self-update requires a released binary version (not \"dev\")")
	}

	latest, err := detectLatest(ctx)
	if err != nil {
		return Result{Current: current}, err
	}

	result := Result{
		Current:   current,
		Latest:    latest.VersionString(),
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
	if err := applyRelease(ctx, latest, exe); err != nil {
		return result, fmt.Errorf("apply update: %w", err)
	}
	result.Updated = true
	return result, nil
}

var errNoPlatformAsset = errors.New("no GitHub release asset found for this platform")

type githubRelease struct {
	TagName    string        `json:"tag_name"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	Assets     []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func detectLatest(ctx context.Context) (*Release, error) {
	release, err := fetchLatestRelease(ctx, repositorySlug())
	if err != nil {
		return nil, err
	}
	if release.Draft || release.Prerelease {
		return nil, fmt.Errorf("latest GitHub release %s is not a stable release", release.TagName)
	}

	version, err := semver.NewVersion(normalizeVersion(release.TagName))
	if err != nil {
		return nil, fmt.Errorf("parse release tag %q: %w", release.TagName, err)
	}

	assetName, assetURL, ok := findPlatformAsset(release.Assets, runtime.GOOS, runtime.GOARCH)
	if !ok {
		return nil, fmt.Errorf("%w (%s/%s)", errNoPlatformAsset, runtime.GOOS, runtime.GOARCH)
	}

	checksumsURL := ""
	for _, asset := range release.Assets {
		if asset.Name == "checksums.txt" {
			checksumsURL = asset.BrowserDownloadURL
			break
		}
	}
	if checksumsURL == "" {
		return nil, errors.New("release is missing checksums.txt")
	}

	return &Release{
		Tag:       release.TagName,
		Version:   version,
		AssetName: assetName,
		AssetURL:  assetURL,
		Checksums: checksumsURL,
	}, nil
}

func fetchLatestRelease(ctx context.Context, slug string) (githubRelease, error) {
	url := "https://api.github.com/repos/" + slug + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "opentunnel-cli")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return githubRelease{}, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return githubRelease{}, fmt.Errorf("repository %s has no releases", slug)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return githubRelease{}, fmt.Errorf("fetch latest release: GitHub API %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("decode latest release: %w", err)
	}
	return release, nil
}

func findPlatformAsset(assets []githubAsset, goos, goarch string) (name, url string, ok bool) {
	suffixes := []string{
		fmt.Sprintf("_%s_%s.tar.gz", goos, goarch),
		fmt.Sprintf("_%s_%s.tgz", goos, goarch),
		fmt.Sprintf("_%s_%s.zip", goos, goarch),
		fmt.Sprintf("-%s-%s.tar.gz", goos, goarch),
		fmt.Sprintf("-%s-%s.zip", goos, goarch),
	}
	if goarch == "amd64" {
		suffixes = append(suffixes,
			fmt.Sprintf("_%s_x86_64.tar.gz", goos),
			fmt.Sprintf("_%s_x86_64.zip", goos),
		)
	}
	for _, asset := range assets {
		lower := strings.ToLower(asset.Name)
		for _, suffix := range suffixes {
			if strings.HasSuffix(lower, suffix) {
				return asset.Name, asset.BrowserDownloadURL, true
			}
		}
	}
	return "", "", false
}

func applyRelease(ctx context.Context, release *Release, exe string) error {
	archive, err := download(ctx, release.AssetURL)
	if err != nil {
		return fmt.Errorf("download %s: %w", release.AssetName, err)
	}
	checksums, err := download(ctx, release.Checksums)
	if err != nil {
		return fmt.Errorf("download checksums.txt: %w", err)
	}
	if err := verifyChecksum(release.AssetName, archive, checksums); err != nil {
		return err
	}

	reader, err := selfupdate.DecompressCommand(
		bytes.NewReader(archive),
		release.AssetName,
		"opentunnel",
		runtime.GOOS,
		runtime.GOARCH,
	)
	if err != nil {
		return fmt.Errorf("decompress %s: %w", release.AssetName, err)
	}
	return update.Apply(reader, update.Options{TargetPath: exe})
}

func download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "opentunnel-cli")
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func verifyChecksum(assetName string, archive, checksums []byte) error {
	expected := ""
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == assetName {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksums.txt has no entry for %s", assetName)
	}
	sum := sha256.Sum256(archive)
	actual := hex.EncodeToString(sum[:])
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch for %s", assetName)
	}
	return nil
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
