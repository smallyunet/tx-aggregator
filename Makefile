# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=tx-aggregator
BINARY_UNIX=$(BINARY_NAME)_unix

# Air for hot reload
AIR=air

.PHONY: all build clean run start dev

all: build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

run:
	$(GOBUILD) -o $(BINARY_NAME) -v
	./$(BINARY_NAME)

start:
	$(GOBUILD) -o $(BINARY_NAME) -v
	./$(BINARY_NAME)

dev:
	$(AIR)

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v

# Install dependencies
deps:
	$(GOGET) github.com/gofiber/fiber/v2
	$(GOGET) github.com/redis/go-redis/v9
	$(GOGET) github.com/joho/godotenv
	$(GOGET) github.com/air-verse/air

# Install air for hot reload
install-air:
	curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b $(GOPATH)/bin 