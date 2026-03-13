BINARY_NAME=symgrep
BIN_DIR=bin
MAIN_PKG=./cmd/symgrep

build:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 go build -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PKG)

test:
	go test ./...

integration-test:
	bash ./scripts/integration_index_test.sh

clean:
	rm -rf $(BIN_DIR)
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux-amd64 $(BINARY_NAME)-darwin-amd64 $(BINARY_NAME)-darwin-arm64

# Cross-compilation targets (requires zig for easy CGO cross-compilation)
# You might need to install zig: brew install zig
build-linux-amd64:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC="zig cc -target x86_64-linux" go build -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PKG)

build-darwin-amd64:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 CC="zig cc -target x86_64-macos" go build -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PKG)

build-darwin-arm64:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 CC="zig cc -target aarch64-macos" go build -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PKG)
