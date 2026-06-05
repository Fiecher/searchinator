.PHONY: all build test cover vet fmt fmt-check run demo server bench clean tidy help

BIN_DIR := bin
GUI     := ./cmd/gui
SERVER  := ./cmd/server
GUI_BIN := $(BIN_DIR)/searchinator-gui
ADDR    := :8080

export CGO_ENABLED := 1

export GOTOOLCHAIN := local

all: fmt vet test build

LDFLAGS := -s -w

build: ## Build the windowed GUI demo into ./bin (requires cgo + a C compiler)
	go build -ldflags "$(LDFLAGS)" -o $(GUI_BIN) $(GUI)

run: build
	$(GUI_BIN)

demo: run

server: ## Run the HTTP JSON API. Pass ADDR=:9090 to change the port
	go run $(SERVER) -addr $(ADDR)

test: ## Run all tests
	go test ./...

cover: ## Run tests with a coverage summary
	go test -cover ./...
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

vet:
	go vet ./...

fmt: ## Format all Go source
	gofmt -w .

fmt-check: ## Fail if any file is not gofmt-clean
	@test -z "$$(gofmt -l .)" || (echo "unformatted files:"; gofmt -l .; exit 1)

bench:
	go test -bench=. -benchmem ./...

tidy: ## Tidy go.mod / go.sum
	go mod tidy

clean:
	rm -rf $(BIN_DIR) coverage.out

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
