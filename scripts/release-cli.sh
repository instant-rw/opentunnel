#!/bin/sh
set -eu

if [ $# -ne 1 ]; then
    echo "Usage: $0 <version>"
    exit 1
fi

VERSION="$1"

release() {
    git tag "$VERSION"
    git push origin "$VERSION"
}

release