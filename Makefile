APP_NAME := integrationsdk

EXE_EXT :=
ifeq ($(OS),Windows_NT)
EXE_EXT := .exe
else ifeq ($(GOOS),windows)
EXE_EXT := .exe
endif

.PHONY: build run clean

build:
	@mkdir -p dist
	go build -o dist/$(APP_NAME)$(EXE_EXT) ./cmd/integrationsdk-app

run:
	go run ./cmd/integrationsdk-app

clean:
	rm -rf dist
