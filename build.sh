#!/usr/bin/env bash
set -eu

EXEC_NAME="structanalyzer"
BUILD_DIR="build"

go mod vendor
go fmt ./...
go test ./...
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"
go build -o "${BUILD_DIR}/${EXEC_NAME}"