.PHONY: test lint lint-fix build

BINARY := plint
TOOLS_DIR := bin
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint

test:
	go test ./...

$(GOLANGCI_LINT):
	mkdir -p $(TOOLS_DIR)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(TOOLS_DIR)

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run

lint-fix: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --fix

build:
	go build -o $(BINARY) ./cmd/promptlinter
