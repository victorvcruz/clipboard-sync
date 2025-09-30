# Clipboard Sync - Makefile

.PHONY: build run clean install-deps test

# Variables
BINARY_NAME=clipboard-sync
BUILD_DIR=build
GO_FILES=$(shell find . -name "*.go" -type f)

# Default target
all: build

# Build the application
build: $(BUILD_DIR)/$(BINARY_NAME)

$(BUILD_DIR)/$(BINARY_NAME): $(GO_FILES)
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

# Install dependencies (xclip)
install-deps:
	@echo "Installing system dependencies..."
	@if command -v apt-get >/dev/null 2>&1; then \
		echo "Installing xclip via apt-get..."; \
		sudo apt-get update && sudo apt-get install -y xclip; \
	elif command -v yum >/dev/null 2>&1; then \
		echo "Installing xclip via yum..."; \
		sudo yum install -y xclip; \
	elif command -v pacman >/dev/null 2>&1; then \
		echo "Installing xclip via pacman..."; \
		sudo pacman -S --noconfirm xclip; \
	else \
		echo "Package manager not found. Please install xclip manually."; \
		exit 1; \
	fi
	@echo "Go dependencies..."
	go mod tidy

# Run the application (requires target IP)
run: build
	@if [ -z "$(TARGET)" ]; then \
		echo "Usage: make run TARGET=<IP_ADDRESS> [PORT=8080] [TARGET_PORT=8080]"; \
		echo "Example: make run TARGET=192.168.1.100"; \
		exit 1; \
	fi
	$(BUILD_DIR)/$(BINARY_NAME) -target $(TARGET) $(if $(PORT),-port $(PORT)) $(if $(TARGET_PORT),-target-port $(TARGET_PORT))

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)

# Test the application
test:
	@echo "Running tests..."
	go test -v ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Install the binary to system PATH
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installation complete. You can now run 'clipboard-sync' from anywhere."

# Uninstall the binary from system PATH
uninstall:
	@echo "Removing $(BINARY_NAME) from /usr/local/bin..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstall complete."

# Development mode - build and run with file watching (requires entr)
dev:
	@if ! command -v entr >/dev/null 2>&1; then \
		echo "entr not found. Install it with: sudo apt-get install entr"; \
		exit 1; \
	fi
	@if [ -z "$(TARGET)" ]; then \
		echo "Usage: make dev TARGET=<IP_ADDRESS> [PORT=8080] [TARGET_PORT=8080]"; \
		echo "Example: make dev TARGET=192.168.1.100"; \
		exit 1; \
	fi
	@echo "Starting development mode (auto-rebuild on file changes)..."
	find . -name "*.go" | entr -r make run TARGET=$(TARGET) $(if $(PORT),PORT=$(PORT)) $(if $(TARGET_PORT),TARGET_PORT=$(TARGET_PORT))

# Help
help:
	@echo "Clipboard Sync - Makefile Commands"
	@echo ""
	@echo "Building:"
	@echo "  build          Build the application"
	@echo "  clean          Clean build artifacts"
	@echo "  install-deps   Install system dependencies (xclip)"
	@echo ""
	@echo "Running:"
	@echo "  run TARGET=IP  Run the application with target IP"
	@echo "                 Example: make run TARGET=192.168.1.100"
	@echo "  dev TARGET=IP  Development mode with auto-rebuild"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  test           Run tests"
	@echo "  fmt            Format code"
	@echo "  lint           Lint code (requires golangci-lint)"
	@echo ""
	@echo "Installation:"
	@echo "  install        Install binary to /usr/local/bin"
	@echo "  uninstall      Remove binary from /usr/local/bin"
	@echo ""
	@echo "Optional parameters for run/dev:"
	@echo "  PORT=8080             Local port (default: 8080)"
	@echo "  TARGET_PORT=8080      Target port (default: 8080)"