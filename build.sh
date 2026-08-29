#!/usr/bin/env bash
# Cross-compiles the privatebox CLI for Windows, macOS, and Linux.
# Run from the project root: ./build.sh
set -e

VERSION="${1:-0.1.0}"
OUT="dist"
rm -rf "$OUT"
mkdir -p "$OUT"

build() {
  local goos=$1 goarch=$2 ext=$3
  local name="privatebox-${goos}-${goarch}${ext}"
  echo "Building $name..."
  GOOS=$goos GOARCH=$goarch go build -ldflags "-s -w" -o "$OUT/$name" .
}

build linux   amd64 ""
build linux   arm64 ""
build darwin  amd64 ""
build darwin  arm64 ""
build windows amd64 ".exe"
build windows arm64 ".exe"

echo ""
echo "Done. Binaries are in $OUT/"
ls -lh "$OUT"
