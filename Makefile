# ---------------------------------------------------------------------------
# Go build / test parameters
# ---------------------------------------------------------------------------
GOCMD      := go
GOBUILD    := $(GOCMD) build
GOCLEAN    := $(GOCMD) clean
GOTEST     := $(GOCMD) test
GOGET      := $(GOCMD) get

BINARY_NAME := tx-aggregator
BINARY_UNIX := $(BINARY_NAME)_unix        # Linux cross‑build output
MAIN_PACKAGE := ./cmd/tx-aggregator

# ---------------------------------------------------------------------------
# Hot‑reload tool
# ---------------------------------------------------------------------------
AIR := air

# ---------------------------------------------------------------------------
# Application run‑time environment (dev / test / prod)
# Passed to the Go program through the APP_ENV variable
# ---------------------------------------------------------------------------
APP_ENV ?= local

# ---------------------------------------------------------------------------
# Integration‑test environment selector (local / test / prod / all)
# Used only by the integration‑test targets
# ---------------------------------------------------------------------------
TEST_ENV ?= local

# ---------------------------------------------------------------------------
# Phony targets
# ---------------------------------------------------------------------------
.PHONY: all build clean run start dev build-linux deps install-air \
        unit-test integration-test integration-test-all

# ---------------------------------------------------------------------------
# Default target
# ---------------------------------------------------------------------------
all: build

# ---------------------------------------------------------------------------
# Build application binary
# ---------------------------------------------------------------------------
build:
	@echo "Building binary…"
	$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PACKAGE)

# ---------------------------------------------------------------------------
# Remove build artefacts
# ---------------------------------------------------------------------------
clean:
	@echo "Cleaning build artefacts…"
	$(GOCLEAN)
	@rm -f $(BINARY_NAME) $(BINARY_UNIX)

# ---------------------------------------------------------------------------
# Compile & run once (respecting APP_ENV)
# ---------------------------------------------------------------------------
run:
	@echo "Running (APP_ENV=$(APP_ENV))"
	APP_ENV=$(APP_ENV) $(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PACKAGE)
	APP_ENV=$(APP_ENV) ./$(BINARY_NAME)

start: run   # alias

# ---------------------------------------------------------------------------
# Hot‑reload development mode
# ---------------------------------------------------------------------------
dev:
	@echo "Hot‑reload with Air (APP_ENV=$(APP_ENV))"
	APP_ENV=$(APP_ENV) $(AIR)

# ---------------------------------------------------------------------------
# Cross‑compile for Linux amd64 (static binary)
# ---------------------------------------------------------------------------
build-linux:
	@echo "Cross‑compiling for Linux (amd64)…"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v $(MAIN_PACKAGE)

# ---------------------------------------------------------------------------
# Fetch Go dependencies
# ---------------------------------------------------------------------------
deps:
	@echo "Installing Go dependencies…"
	$(GOGET) github.com/gofiber/fiber/v2
	$(GOGET) github.com/redis/go-redis/v9
	$(GOGET) github.com/spf13/viper
	$(GOGET) github.com/air-verse/air

# ---------------------------------------------------------------------------
# Install Air (hot‑reload) to GOPATH/bin
# ---------------------------------------------------------------------------
install-air:
	@echo "Installing Air (hot reload)…"
	curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b $(GOPATH)/bin

# ---------------------------------------------------------------------------
# Unit tests (all packages)
# ---------------------------------------------------------------------------
unit-test:
	@echo "Running unit tests…"
	$(GOTEST) ./...

# ---------------------------------------------------------------------------
# Integration tests
#   make integration-test            # default TEST_ENV=local
#   make integration-test TEST_ENV=test
#   make integration-test-all        # runs all environments
# ---------------------------------------------------------------------------
integration-test:
	@echo "Running integration tests (env=$(TEST_ENV))…"
	$(GOCMD) run ./cmd/tx-aggregator-integration -env=$(TEST_ENV)

integration-test-all:
	@echo "Running integration tests for all environments…"
	$(GOCMD) run ./cmd/tx-aggregator-integration -env=all
