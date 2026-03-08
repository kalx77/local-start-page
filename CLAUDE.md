# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

A local start page application. The `.gitignore` is configured for a Go project, suggesting this will be a Go-based web app.

## Getting Started

The project is in early development. Before writing code, initialize the Go module:

```bash
go mod init github.com/kuzmin/local-start-page
```

## Common Commands (once implemented)

```bash
go build ./...       # Build
go test ./...        # Run all tests
go test ./pkg/...    # Run tests in a specific package
go run main.go       # Run the app
go vet ./...         # Lint/vet
```
