.PHONY: build test lint clean install

# Build the binary
build:
	go build -o bin/trakt-mcp ./cmd/trakt-mcp

# Run tests
test:
	go test -v -race ./...

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf bin/

# Install to $GOPATH/bin
install:
	go install ./cmd/trakt-mcp

# Build for all platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o bin/trakt-mcp-darwin-amd64 ./cmd/trakt-mcp
	GOOS=darwin GOARCH=arm64 go build -o bin/trakt-mcp-darwin-arm64 ./cmd/trakt-mcp
	GOOS=linux GOARCH=amd64 go build -o bin/trakt-mcp-linux-amd64 ./cmd/trakt-mcp
	GOOS=windows GOARCH=amd64 go build -o bin/trakt-mcp-windows-amd64.exe ./cmd/trakt-mcp

# Run the server (for local testing)
run:
	go run ./cmd/trakt-mcp
