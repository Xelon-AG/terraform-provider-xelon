# Project variables
PROJECT_NAME := terraform-provider-xelon

# Build variables
.DEFAULT_GOAL = test
BUILD_DIR := build
DEV_GOARCH := $(shell go env GOARCH)
DEV_GOOS := $(shell go env GOOS)


## tools: Install required tooling.
.PHONY: tools
tools:
ifeq (,$(wildcard ./.bin/golangci-lint*))
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b .bin/ v1.45.0
else
	@echo "==> Required tooling is already installed"
endif


## clean: Delete the build directory.
.PHONY: clean
clean:
	@echo "==> Removing '$(BUILD_DIR)' directory..."
	@rm -rf $(BUILD_DIR)


## lint: Lint code with golangci-lint.
.PHONY: lint
lint:
	@echo "==> Linting code with 'golangci-lint'..."
	@.bin/golangci-lint run ./...


## test: Run all unit tests.
.PHONY: test
test:
	@echo "==> Running unit tests..."
	@mkdir -p $(BUILD_DIR)
	@go test -count=1 -v -cover -coverprofile=$(BUILD_DIR)/coverage.out -parallel=4 ./...


## testacc: Run all acceptance tests.
.PHONY: testacc
testacc: lint
	@echo "==> Running acceptance tests..."
	@mkdir -p $(BUILD_DIR)
	TF_ACC=1 go test -count=1 -v -cover -coverprofile=$(BUILD_DIR)/coverage-with-acceptance.out -timeout 120m ./...


## build: Build binary for default local system's operating system and architecture.
.PHONY: build
build:
	@echo "==> Building binary..."
	@echo "    running go build for GOOS=$(DEV_GOOS) GOARCH=$(DEV_GOARCH)"
# workaround for missing .exe extension on Windows
ifeq ($(OS),Windows_NT)
	@go build -o $(BUILD_DIR)/$(PROJECT_NAME).exe main.go
else
	@go build -o $(BUILD_DIR)/$(PROJECT_NAME) main.go
endif


## website: Build website for the provider.
.PHONY: website
website:
	@echo "Use this site to preview markdown rendering: https://registry.terraform.io/tools/doc-preview"


help: Makefile
	@echo "Usage: make <command>"
	@echo ""
	@echo "Commands:"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
