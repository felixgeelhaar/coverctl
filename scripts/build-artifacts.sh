#!/usr/bin/env bash
set -euo pipefail

rm -rf dist
mkdir -p dist

build() {
  local goos=$1
  local goarch=$2
  local name=$3
  local output=dist/$name
  env CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -o "$output" ./cmd/coverctl
  case "$goos" in
    linux)
      tar -C dist -czf "$output.tar.gz" "$name"
      rm "$output"
      ;;
    darwin)
      tar -C dist -czf "$output.tar.gz" "$name"
      rm "$output"
      ;;
    windows)
      zip -j dist/"$name.zip" "$output"
      rm "$output"
      ;;
  esac
}

build linux amd64 coverctl-linux-amd64
build darwin amd64 coverctl-darwin-amd64
build windows amd64 coverctl-windows-amd64.exe
