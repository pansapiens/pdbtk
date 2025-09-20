.PHONY: build clean test install

# Build the pdbtk binary
build:
	mkdir -p bin
	go build -o bin/pdbtk .

# Clean build artifacts
clean:
	rm -f bin/pdbtk
	rm -f test*.pdb

# Run tests - we exclude test discovery in the 'repos' folder
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

# Update dependencies
deps:
	go mod tidy
	go mod download
