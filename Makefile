.PHONY: dev build frontend frontend-build proto-lint proto-generate proto-check proto-verify backend-test backend-verify ci

FRONTEND_DIR := website
BUF_VERSION ?= 1.69.0
GRPC_GATEWAY_OPENAPIV3_VERSION ?= 6352855b82dc
BUF ?= .bin/buf
HOST_GOOS := $(shell go env GOOS 2>/dev/null || echo linux)
HOST_GOARCH := $(shell go env GOARCH 2>/dev/null || echo amd64)
HOST_GOVERSION := $(shell go env GOVERSION 2>/dev/null || echo go1.26.2)
GO_TOOLCHAIN_ROOT := $(shell go env GOMODCACHE 2>/dev/null)/golang.org/toolchain@v0.0.1-$(HOST_GOVERSION).$(HOST_GOOS)-$(HOST_GOARCH)
ifneq ($(wildcard $(GO_TOOLCHAIN_ROOT)/bin/go),)
GO ?= GOROOT=$(GO_TOOLCHAIN_ROOT) $(GO_TOOLCHAIN_ROOT)/bin/go
else
GO ?= go
endif

dev:
	@echo "Starting frontend dev server..."
	@cd $(FRONTEND_DIR) && pnpm dev &
	@sleep 2
	@echo "Starting Go server..."
	@CONFIG_FILE=config.yaml $(GO) run ./cmd/server/
	@kill %1 2>/dev/null

frontend:
	cd $(FRONTEND_DIR) && pnpm dev

frontend-build:
	cd $(FRONTEND_DIR) && pnpm build

.bin/buf:
	mkdir -p .bin
	curl -fsSL https://github.com/bufbuild/buf/releases/download/v$(BUF_VERSION)/buf-Linux-x86_64 -o .bin/buf
	chmod +x .bin/buf

.bin/protoc-gen-openapiv3:
	mkdir -p .bin
	GOBIN=$(CURDIR)/.bin $(GO) install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv3@$(GRPC_GATEWAY_OPENAPIV3_VERSION)

proto-lint: .bin/buf
	$(BUF) lint

proto-generate: .bin/buf .bin/protoc-gen-openapiv3
	$(BUF) generate

proto-check: proto-lint proto-generate

proto-verify: proto-check
	git diff --exit-code -- buf.lock gen

backend-test:
	$(GO) mod tidy
	$(GO) test ./...

backend-verify: backend-test
	git diff --exit-code -- go.mod go.sum

ci: proto-check
	cd $(FRONTEND_DIR) && pnpm lint && pnpm typecheck && pnpm build
	$(MAKE) backend-test

build: frontend-build
	$(GO) build -o slot-art ./cmd/server/

clean:
	rm -f slot-art
	rm -rf $(FRONTEND_DIR)/dist

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-logs:
	docker compose logs -f app

docker-down:
	docker compose down

docker-full: docker-build docker-up
