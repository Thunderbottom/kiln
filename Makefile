.PHONY: build test clean install dev

BINARY_NAME=kiln
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS=-ldflags "-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)"

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME) .

build-all: build-linux build-darwin build-windows

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 .

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 .

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe .

test:
	go test -v -race -bench=. -benchmem -coverprofile=coverage.out ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

install:
	CGO_ENABLED=0 go install $(LDFLAGS) .

dev:
	go run . $(ARGS)

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

mod-tidy:
	go mod tidy

deps:
	go mod download

release: clean build-all
	@echo "Built release binaries in bin/"
