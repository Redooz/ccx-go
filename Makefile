BINARY = claude
BUILD_DIR = bin
MODULE = github.com/anton-abyzov/ccx-go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build test lint clean fmt vet release snapshot

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/claude

test:
	go test ./... -v -count=1

lint: vet
	@which golangci-lint > /dev/null 2>&1 || echo "golangci-lint not installed, skipping"
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || true

vet:
	go vet ./...

fmt:
	gofmt -s -w .

clean:
	rm -rf $(BUILD_DIR)
	go clean -testcache

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

release:
	goreleaser release --clean

snapshot:
	goreleaser build --snapshot --clean
