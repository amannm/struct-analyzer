#!/usr/bin/env bash
set -eu

EXEC_NAME="structanalyzer"
BUILD_DIR="build"

build() {
  go fmt ./...
  go test ./...
  local os="darwin"
  local arch="arm64"
  rm -rf "${BUILD_DIR}"
  mkdir -p "${BUILD_DIR}"
  GOOS="${os}" GOARCH="${arch}" go build -o "${BUILD_DIR}/${EXEC_NAME}"
}

"$@"