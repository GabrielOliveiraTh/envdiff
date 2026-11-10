# Build metadata derived from git, falling back to "dev" outside a repo.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

PKG     := github.com/GabrielOliveiraTh/envdiff/internal/cli
LDFLAGS := -s -w -X $(PKG).version=$(VERSION) -X $(PKG).commit=$(COMMIT) -X $(PKG).date=$(DATE)

.PHONY: build test cover lint fuzz snapshot

build:
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/envdiff ./cmd/envdiff

test:
	go test -race ./...

cover:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run

fuzz:
	go test -fuzz=FuzzParse -fuzztime=30s ./internal/envfile/

snapshot:
	goreleaser release --snapshot --clean
