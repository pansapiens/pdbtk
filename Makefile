.PHONY: build clean test install

# Build the pdbtk binary
build:
	go build -o pdbtk .

# Clean build artifacts
clean:
	rm -f pdbtk
	rm -f test*.pdb

# Run tests
test:
	go test $(shell go list ./... | grep -v /.repos/)

# Install to GOPATH/bin
install:
	go install .

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o pdbtk-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o pdbtk-darwin-amd64 .
	GOOS=windows GOARCH=amd64 go build -o pdbtk-windows-amd64.exe .

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Update dependencies
deps:
	go mod tidy
	go mod download
