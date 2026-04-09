.PHONY: test lint install-dirty

.DEFAULT_GOAL := test

BINARY_NAME := fraga
MAIN_PACKAGE := ./cmd/fraga

test:
	go test ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

install-dirty:
	@mkdir -p ~/bin
	go build -ldflags "-X main.version=dirty-$(shell date +%Y%m%d-%H%M%S)-$(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" -o ~/bin/$(BINARY_NAME)-dirty $(MAIN_PACKAGE)
	@echo "Installed dirty version to ~/bin/$(BINARY_NAME)-dirty"
	@echo "Version: $(shell ~/bin/$(BINARY_NAME)-dirty --version 2>/dev/null || echo 'built with dev flags')"
	@echo "Make sure ~/bin is in your PATH"

ci: lint test
