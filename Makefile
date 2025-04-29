# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=tx-aggregator
BINARY_UNIX=$(BINARY_NAME)_unix
MAIN_PACKAGE=./cmd/tx-aggregator

# Air for hot reload
AIR=air

# Default environment
APP_ENV ?= dev

.PHONY: all build clean run start dev build-linux deps install-air

all: build

build:
	@echo "Building binary..."
	$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PACKAGE)

clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

run:
	@echo "Running with APP_ENV=$(APP_ENV)"
	APP_ENV=$(APP_ENV) $(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PACKAGE)
	APP_ENV=$(APP_ENV) ./$(BINARY_NAME)

start:
	@echo "Starting with APP_ENV=$(APP_ENV)"
	APP_ENV=$(APP_ENV) $(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PACKAGE)
	APP_ENV=$(APP_ENV) ./$(BINARY_NAME)

dev:
	@echo "AIR variable => $(AIR)"
	@echo "Running dev mode with APP_ENV=$(APP_ENV)"
	APP_ENV=$(APP_ENV) $(AIR)

build-linux:
	@echo "Cross compiling for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v $(MAIN_PACKAGE)

deps:
	@echo "Installing Go dependencies..."
	$(GOGET) github.com/gofiber/fiber/v2
	$(GOGET) github.com/redis/go-redis/v9
	$(GOGET) github.com/spf13/viper
	$(GOGET) github.com/air-verse/air

install-air:
	@echo "Installing Air (hot reload)..."
	curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b $(GOPATH)/bin
