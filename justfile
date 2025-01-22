# just is a handy way to save and run project-specific commands # https://just.systems/

# List all recipes
default:
    @just --list

# Format all Go files
fmt:
    go fmt ./...

# Run tests
test:
    go test -v ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Build the project
build:
    go build -v ./...

# Build the CLI
build-cli:
    go build -v -o bin/evo ./cmd/evo

# Install the CLI to $GOPATH/bin
install: build-cli
    go install ./cmd/evo

# Run the main application
run:
    go run ./cmd/evo

# Install dependencies
deps:
    go mod download
    go mod tidy

# Verify dependencies
verify:
    go mod verify

# Run linter (requires golangci-lint)
lint:
    golangci-lint run

# Clean build artifacts
clean:
    go clean
    rm -f coverage.out coverage.html

# Update dependencies to latest versions
update-deps:
    go get -u ./...
    go mod tidy

# Run security check (requires gosec)
security-check:
    gosec ./...

# Generate documentation
docs:
    godoc -http=:6060

# Create a new release tag
release VERSION:
    git tag -a {{VERSION}} -m "Release {{VERSION}}"
    git push origin {{VERSION}}

# Install development tools
install-tools:
    go install golang.org/x/tools/cmd/godoc@latest
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    go install github.com/securego/gosec/v2/cmd/gosec@latest
