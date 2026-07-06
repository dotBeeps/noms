set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

# List available project recipes.
default:
    @just --list --unsorted

# Install git hooks and the commit template.
setup:
    lefthook install
    git config commit.template .gitmessage

# Build all packages.
check:
    go build ./...

# Run go vet and golangci-lint across the module.
lint:
    go vet ./...
    golangci-lint run --timeout=5m

# Fail if go.mod/go.sum need `go mod tidy`.
tidy-check:
    go mod tidy -diff

# Format tracked Go files with gofmt.
format:
    gofmt -w $(git ls-files '*.go')

# Fail if any tracked Go file is not gofmt-formatted.
format-check:
    #!/usr/bin/env bash
    set -euo pipefail
    unformatted="$(gofmt -l $(git ls-files '*.go'))"
    if [ -n "$unformatted" ]; then
      echo "gofmt: needs formatting (run: just format):"
      echo "$unformatted"
      exit 1
    fi

alias fmt := format

# Run check, lint, and format (mutating aggregate).
yummers: check lint format

# Run the test suite.
test:
    go test ./...

# Run the test suite with the race detector (the CI gate).
test-race:
    go test -race -count=1 ./...

# Build the noms binary.
build:
    go build -o noms ./cmd/noms

# Build and run the TUI.
run: build
    ./noms

# Non-mutating full gate: format-check, lint, tidy-check, check, race tests.
ci: format-check lint tidy-check check test-race
