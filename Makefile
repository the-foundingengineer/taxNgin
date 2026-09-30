.PHONY: dev test build

# 64 hex-zero dev key — local development only.
# Override by setting HMAC_KEY in your shell before running make.
HMAC_KEY_DEV ?= 0000000000000000000000000000000000000000000000000000000000000000

# Run the API (:8080) and Vite dev server (:5173) together.
# Ctrl+C stops both.
dev:
	@bash dev.sh

test:
	go test ./...

build:
	go build -o bin/api ./cmd/api
