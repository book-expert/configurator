# Configurator

## Project Summary

Configurator is a Go library that fetches a TOML configuration file from a URL and unmarshals it into a Go struct.

## Detailed Description

This library provides a simple and robust way to manage configuration in a distributed environment. It retrieves a TOML configuration file from a URL specified by the `PROJECT_TOML` environment variable. This allows for centralized configuration management, where services can fetch their configuration from a single source of truth.

The library includes features such as:
-   Fetching configuration from a URL.
-   Unmarshaling TOML data into a type-safe Go struct.
-   Configurable timeout for HTTP requests.
-   Comprehensive error handling.

## Technology Stack

-   **Programming Language:** Go 1.25
-   **Libraries:**
    -   `github.com/book-expert/logger`
    -   `github.com/pelletier/go-toml/v2`
    -   `github.com/stretchr/testify`

## Getting Started

### Prerequisites

-   Go 1.25 or later.

### Installation

To use this library in your project, you can use `go get`:

```bash
go get github.com/book-expert/configurator
```

## Usage

To use the configurator library, you need to set the `PROJECT_TOML` environment variable to the URL of your TOML configuration file.

```go
package main

import (
    "fmt"
    "os"

    "github.com/book-expert/configurator"
    "github.com/book-expert/logger"
)

type Config struct {
    Project struct {
        Name    string `toml:"name"`
        Version string `toml:"version"`
    } `toml:"project"`
    Settings struct {
        Debug bool `toml:"debug"`
        Port  int  `toml:"port"`
    } `toml:"settings"`
}

func main() {
    // Set the PROJECT_TOML environment variable to the URL of your configuration file.
    os.Setenv("PROJECT_TOML", "http://example.com/config.toml")

    var cfg Config
    log, err := logger.New("/tmp", "test.log")
    if err != nil {
        panic(err)
    }

    if err := configurator.Load(&cfg, log); err != nil {
        panic(err)
    }

    fmt.Printf("Loaded: %s v%s
", cfg.Project.Name, cfg.Project.Version)
}
```

## Testing

To run the tests for this library, you can use the `make test` command:

```bash
make test
```

This will run the tests and display the coverage.

## License

Distributed under the MIT License. See the `LICENSE` file for more information.