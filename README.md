# Configurator

## Project Summary

A Go library that loads TOML configuration from a URL defined by the `PROJECT_TOML` environment variable and unmarshals it into typed structs.

## Detailed Description

The configurator library centralizes configuration loading for the Book Expert services. It performs an HTTP GET against the URL provided in `PROJECT_TOML`, enforces a request timeout, and unmarshals the retrieved TOML payload into the caller's struct. This enables services to share a single `project.toml` file while keeping configuration data type-safe. The library uses the shared `logger` package to report issues such as missing environment variables, HTTP failures, or TOML parsing errors.

Key features include:

- URL-driven configuration sourcing via `PROJECT_TOML`.
- Context-based HTTP timeouts to prevent blocked startups.
- Strict error propagation with contextual wrapping for easier diagnosis.
- Integration with the shared `logger` package for structured error reporting.

## Technology Stack

- **Language:** Go 1.25
- **Parsing:** `github.com/pelletier/go-toml/v2`
- **Logging:** `github.com/book-expert/logger`
- **Testing:** `testing`, `net/http/httptest`, `github.com/stretchr/testify`

## Getting Started

### Prerequisites

- Go 1.25 or newer installed locally.

### Installation

```bash
go get github.com/book-expert/configurator
```

## Usage

```go
package main

import (
    "fmt"

    "github.com/book-expert/configurator"
    "github.com/book-expert/logger"
)

type ServiceConfig struct {
    Service struct {
        Name string `toml:"name"`
    } `toml:"service"`
}

func main() {
    // Initialize a shared logger.
    logInstance, createLoggerErr := logger.New("/tmp/logs", "configurator.log")
    if createLoggerErr != nil {
        panic(createLoggerErr)
    }
    defer logInstance.Close()

    // PROJECT_TOML must be set to a reachable URL that returns TOML content.
    var cfg ServiceConfig
    loadConfigErr := configurator.Load(&cfg, logInstance)
    if loadConfigErr != nil {
        panic(loadConfigErr)
    }

    fmt.Printf("Loaded configuration for %s\n", cfg.Service.Name)
}
```

## Testing

```bash
cd configurator
go test ./...
```

The test suite spins up an in-memory HTTP server that serves the shared `project.toml` file to verify that the loader correctly parses critical fields.

## License

Distributed under the MIT License. See the repository root `LICENSE` file for details.
