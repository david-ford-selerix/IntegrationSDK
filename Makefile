APP_NAME := integrationsdk
GO ?= go

EXE_EXT :=
ifeq ($(OS),Windows_NT)
EXE_EXT := .exe
else ifeq ($(GOOS),windows)
EXE_EXT := .exe
endif

.PHONY: build run clean check-go

check-go:
	@command -v $(GO) >/dev/null 2>&1 || ( \
		echo "Error: could not find '$(GO)' in PATH."; \
		echo "Install Go from https://go.dev/dl/ and ensure its bin directory is in PATH."; \
		echo "On Windows + Git Bash, restart the shell after updating PATH."; \
		exit 1 \
	)

build: check-go
	@mkdir -p dist
	$(GO) build -o dist/$(APP_NAME)$(EXE_EXT) ./cmd/integrationsdk-app

run: check-go
	$(GO) run ./cmd/integrationsdk-app

clean:
	rm -rf dist
