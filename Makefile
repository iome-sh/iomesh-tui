MODULE  := github.com/iome-sh/iomesh-tui
BIN     := iomesh
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COVER   ?= coverage.out

.PHONY: all build test test-race cover vet fmt tidy vuln run models clean check ci

all: check build

build:
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/$(BIN) ./cmd/iomesh

test:
	go test ./... -count=1

test-race:
	go test ./... -race -count=1

cover:
	go test ./... -coverprofile=$(COVER) -covermode=atomic
	go tool cover -func=$(COVER) | tail -20

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "$$unformatted"; exit 1; fi

tidy:
	go mod tidy
	go mod verify

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

check: fmt-check vet test

ci: check test-race cover vuln build

run:
	go run ./cmd/iomesh

models:
	go run ./cmd/iomesh models

# Headless smoke (requires DEEPSEEK_API_KEY for live call)
smoke-prompt:
	go run ./cmd/iomesh -p "Reply with exactly: pong" -m deepseek-v4-flash

clean:
	rm -rf bin/ $(COVER) coverage.html
