.PHONY: all build build-server build-client build-frontend deps build-windows build-linux build-darwin clean help

# Build flags
LDFLAGS := -ldflags="-s -w"

# Default target
all: build

# Build both server and client for current platform
build: build-frontend deps build-server build-client

# Install/update Go dependencies
deps:
	@echo "Installing Go dependencies..."
	go mod tidy
	go mod download

# Build frontend
build-frontend:
	@echo "Building frontend..."
	cd frontend && yarn install && yarn build
	@echo "Copying frontend build to internal/server/public..."
	rm -rf internal/server/public/*
	cp -r frontend/dist/* internal/server/public/

# Build server
build-server:
	@echo "Building server..."
	@mkdir -p build/bin
	go build -o build/bin/digit-link-server ./cmd/server

# Build client (static binary, no CGO)
build-client:
	@echo "Building client (static)..."
	@mkdir -p build/bin
	CGO_ENABLED=0 go build $(LDFLAGS) -o build/bin/digit-link ./cmd/client

# Cross-compilation targets

# Windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p build/bin/windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o build/bin/windows/digit-link-server.exe ./cmd/server
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o build/bin/windows/digit-link.exe ./cmd/client

# Linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p build/bin/linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o build/bin/linux/digit-link-server ./cmd/server
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o build/bin/linux/digit-link ./cmd/client

# macOS Intel
build-darwin:
	@echo "Building for macOS (Intel)..."
	@mkdir -p build/bin/darwin
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o build/bin/darwin/digit-link-server ./cmd/server
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o build/bin/darwin/digit-link ./cmd/client

# macOS Apple Silicon
build-darwin-arm:
	@echo "Building for macOS (Apple Silicon)..."
	@mkdir -p build/bin/darwin-arm64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o build/bin/darwin-arm64/digit-link-server ./cmd/server
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o build/bin/darwin-arm64/digit-link ./cmd/client

# Build all platforms
build-all: build-windows build-linux build-darwin build-darwin-arm

# Docker
docker-build:
	docker build -t digit-link-server .

docker-run:
	docker run -p 8080:8080 -e DOMAIN=link.digit.zone digit-link-server

# Run locally
run-server:
	go run ./cmd/server

run-client:
	@echo "Usage: make run-client SUBDOMAIN=myapp PORT=3000"
	go run ./cmd/client --subdomain=$(SUBDOMAIN) --port=$(PORT)

# Setup admin account
setup-admin:
	go run ./cmd/server --setup-admin

# Clean build artifacts
clean:
	rm -rf build/

# Help
help:
	@echo "digit-link Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build          - Build frontend, server and client for current platform"
	@echo "  make build-frontend - Build Vue.js frontend"
	@echo "  make deps           - Install/update Go dependencies"
	@echo "  make build-server   - Build server only"
	@echo "  make build-client   - Build client only"
	@echo "  make build-windows  - Cross-compile for Windows"
	@echo "  make build-linux    - Cross-compile for Linux"
	@echo "  make build-darwin   - Cross-compile for macOS Intel"
	@echo "  make build-darwin-arm - Cross-compile for macOS Apple Silicon"
	@echo "  make build-all      - Build for all platforms"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run Docker container"
	@echo "  make run-server     - Run server locally"
	@echo "  make run-client     - Run client (SUBDOMAIN=x PORT=y required)"
	@echo "  make setup-admin    - Create initial admin account"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make help           - Show this help"
