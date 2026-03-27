#!/usr/bin/env bash
set -euo pipefail

mkdir -p dist

echo "Building Linux amd64"
GOOS=linux GOARCH=amd64 go build -o dist/integrationsdk-linux-amd64 ./cmd/integrationsdk-app

echo "Building Windows amd64"
GOOS=windows GOARCH=amd64 go build -o dist/integrationsdk-windows-amd64.exe ./cmd/integrationsdk-app

echo "Building macOS arm64"
GOOS=darwin GOARCH=arm64 go build -o dist/integrationsdk-macos-arm64 ./cmd/integrationsdk-app

echo "Done"
