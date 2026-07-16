MODULE := github.com/iome-sh/iomesh-tui
BIN    := iomesh
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: all build test vet fmt tidy run models clean

all: vet test build

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/$(BIN) ./cmd/iomesh

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

tidy:
	go mod tidy

run:
	go run ./cmd/iomesh

models:
	go run ./cmd/iomesh models

# Headless smoke (requires DEEPSEEK_API_KEY for live call)
smoke-prompt:
	go run ./cmd/iomesh -p "Reply with exactly: pong" -m deepseek-v4-flash

clean:
	rm -rf bin/
