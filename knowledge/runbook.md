# Runbook

## Build

```bash
make build
```

## Run locally

```bash
make run
```

## Cross-compile examples

```bash
GOOS=windows GOARCH=amd64 go build -o dist/integrationsdk.exe ./cmd/integrationsdk-app
GOOS=darwin GOARCH=arm64 go build -o dist/integrationsdk-macos ./cmd/integrationsdk-app
GOOS=linux GOARCH=amd64 go build -o dist/integrationsdk-linux ./cmd/integrationsdk-app
```

## Distribute

Share only the generated executable and any companion knowledge docs you want end users to receive.
