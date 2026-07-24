#!/bin/sh
set -eu

REPOSITORY="${OPENTUNNEL_REPOSITORY:-instant-rw/opentunnel}"
INSTALL_DIR="${OPENTUNNEL_INSTALL_DIR:-/usr/local/bin}"
VERSION="${OPENTUNNEL_VERSION:-latest}"

fail() {
  printf 'opentunnel installer: %s\n' "$1" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) fail "unsupported operating system: $(uname -s)" ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) fail "unsupported architecture: $(uname -m)" ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPOSITORY}/releases/latest" |
    awk -F '"' '/"tag_name":/ { print $4; exit }')"
  [ -n "$VERSION" ] || fail "could not determine the latest version"
fi

number="${VERSION#v}"
archive="opentunnel_${number}_${os}_${arch}.tar.gz"
base_url="https://github.com/${REPOSITORY}/releases/download/${VERSION}"
temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT HUP INT TERM

curl -fsSL "${base_url}/${archive}" -o "${temp_dir}/${archive}"
curl -fsSL "${base_url}/checksums.txt" -o "${temp_dir}/checksums.txt"

(
  cd "$temp_dir"
  expected="$(awk -v file="$archive" '$2 == file { print $1 }' checksums.txt)"
  [ -n "$expected" ] || fail "archive is missing from checksums.txt"
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$archive" | awk '{ print $1 }')"
  else
    actual="$(shasum -a 256 "$archive" | awk '{ print $1 }')"
  fi
  [ "$actual" = "$expected" ] || fail "checksum verification failed"
  tar -xzf "$archive" opentunnel
)

if [ -w "$INSTALL_DIR" ]; then
  install -m 0755 "${temp_dir}/opentunnel" "${INSTALL_DIR}/opentunnel"
else
  command -v sudo >/dev/null 2>&1 || fail "${INSTALL_DIR} is not writable and sudo is unavailable"
  sudo install -m 0755 "${temp_dir}/opentunnel" "${INSTALL_DIR}/opentunnel"
fi

printf 'Installed OpenTunnel %s to %s/opentunnel\n' "$VERSION" "$INSTALL_DIR"
