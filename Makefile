APP_NAME := integrationsdk

.PHONY: build run clean

build:
	@mkdir -p dist
	go build -o dist/$(APP_NAME) ./cmd/integrationsdk-app

run:
	go run ./cmd/integrationsdk-app

clean:
	rm -rf dist
