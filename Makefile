.PHONY: build test clean run install lint lint-fix vuln tests

BINARY_NAME=codalf
BUILD_DIR=bin
GO=go
GOLANGCI_LINT=$(GOPATH)/bin/golangci-lint
GOVULNCHECK=$(GOPATH)/bin/govulncheck
STATICCHECK=$(GOPATH)/bin/staticcheck

build:
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/codalf

test:
	$(GO) test -v -race -cover ./...

tests: test

lint:
	$(GO) vet ./...
	$(GOPATH)/bin/golangci-lint run ./...

lint-fix:
	$(GOPATH)/bin/golangci-lint run ./... --fix

vuln:
	$(GOPATH)/bin/govulncheck ./...

sec: vuln lint

install:
	$(GO) install ./cmd/codalf

deps:
	$(GO) mod download
	$(GO) mod tidy

clean:
	rm -rf $(BUILD_DIR)

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

tools:
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	$(GO) install honnef.co/go/tools/cmd/staticcheck@latest
