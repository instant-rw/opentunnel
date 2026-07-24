package update

import (
	"testing"

	"github.com/Masterminds/semver/v3"
)

func TestFindPlatformAsset(t *testing.T) {
	assets := []githubAsset{
		{Name: "checksums.txt", BrowserDownloadURL: "https://example/checksums.txt"},
		{Name: "opentunnel_0.5_darwin_amd64.tar.gz", BrowserDownloadURL: "https://example/amd64"},
		{Name: "opentunnel_0.5_darwin_arm64.tar.gz", BrowserDownloadURL: "https://example/arm64"},
		{Name: "opentunnel_0.5_windows_arm64.zip", BrowserDownloadURL: "https://example/win"},
	}

	name, url, ok := findPlatformAsset(assets, "darwin", "arm64")
	if !ok {
		t.Fatal("expected darwin/arm64 asset")
	}
	if name != "opentunnel_0.5_darwin_arm64.tar.gz" || url != "https://example/arm64" {
		t.Fatalf("unexpected asset: %s %s", name, url)
	}

	name, url, ok = findPlatformAsset(assets, "windows", "arm64")
	if !ok || name != "opentunnel_0.5_windows_arm64.zip" {
		t.Fatalf("unexpected windows asset: %s %v", name, ok)
	}

	_, _, ok = findPlatformAsset(assets, "linux", "arm64")
	if ok {
		t.Fatal("expected no linux/arm64 asset")
	}
}

func TestTwoPartTagSemver(t *testing.T) {
	version, err := semver.NewVersion(normalizeVersion("v0.5"))
	if err != nil {
		t.Fatalf("parse v0.5: %v", err)
	}
	release := &Release{Version: version}
	if !release.LessOrEqual("0.5") {
		t.Fatal("0.5 should be equal to tag v0.5")
	}
	if release.LessOrEqual("0.4.9") {
		t.Fatal("v0.5 should be newer than 0.4.9")
	}
	if release.VersionString() == "" {
		t.Fatal("expected version string")
	}
}

func TestVerifyChecksum(t *testing.T) {
	archive := []byte("hello")
	// echo -n hello | shasum -a 256
	checksums := []byte("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824  opentunnel_0.5_darwin_arm64.tar.gz\n")
	if err := verifyChecksum("opentunnel_0.5_darwin_arm64.tar.gz", archive, checksums); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum("missing.tar.gz", archive, checksums); err == nil {
		t.Fatal("expected missing entry error")
	}
}
