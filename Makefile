# TinyMuscle Makefile
# Minimal, explicit, no magic — Go style

BINARY_NAME := tinymuscle
CMD_PATH := ./cmd/main.go
BUILD_DIR := build

# Default target
.PHONY: all
all: build

## -------------------------
## Local Development
## -------------------------

.PHONY: build
build:
	@echo "→ Building $(BINARY_NAME)"
	@go build -o $(BINARY_NAME) $(CMD_PATH)

.PHONY: run
run: build
	@echo "→ Running $(BINARY_NAME)"
	@export $$(cat .env | xargs) && ./$(BINARY_NAME)

.PHONY: clean
clean:
	@echo "→ Cleaning binaries"
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)

## -------------------------
## Cross Compilation
## -------------------------

PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

.PHONY: cross
cross:
	@echo "→ Cross compiling..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		output_name=$(BINARY_NAME)-$$GOOS-$$GOARCH; \
		if [ "$$GOOS" = "windows" ]; then output_name="$$output_name.exe"; fi; \
		echo "→ $$output_name"; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -o $(BUILD_DIR)/$$output_name $(CMD_PATH); \
	done

## -------------------------
## Install (local system)
## -------------------------

.PHONY: install
install:
	@echo "→ Installing $(BINARY_NAME) to GOPATH/bin"
	@go install $(CMD_PATH)

## -------------------------
## Quality
## -------------------------

.PHONY: fmt
fmt:
	@echo "→ Formatting"
	@gofmt -s -w .

.PHONY: vet
vet:
	@echo "→ Vetting"
	@go vet ./...

.PHONY: tidy
tidy:
	@echo "→ Tidying modules"
	@go mod tidy
