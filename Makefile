.PHONY: build run vet test race lint clean install ci release

PREFIX ?= $(HOME)/.local
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/RakhaYandra/kage/internal/cli.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/kage ./cmd/kage

release:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/kage-linux-amd64 ./cmd/kage
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/kage-linux-arm64 ./cmd/kage
	sha256sum dist/kage-linux-* > dist/SHA256SUMS

lint:
	test -z "$$(gofmt -l cmd internal tests)" || (gofmt -l cmd internal tests && exit 1)
	go vet ./...

install: build
	install -Dm755 bin/kage $(PREFIX)/bin/kage

run: build
	./bin/kage --help

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

ci: lint test
	go test -race ./internal/... ./tests/...

clean:
	rm -rf bin/ dist/
