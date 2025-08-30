# Configurator

A robust, standalone configuration management system with comprehensive input validation and CLI interface. This tool can be used both as a Go library and as a command-line binary for TOML configuration handling.

## Build Status

---

## Architecture

This project provides both:

- **Library API** (`config.go`): Generic TOML loading and project discovery for Go applications.
- **CLI Binary** (`cmd/configurator/main.go`): Standalone executable for shell scripts and external tools.

---

## Features

- **Generic TOML Loading**: Load any TOML structure into Go structs.
- **Project Discovery**: Automatically find `project.toml` files by walking up the directory tree.
- **Security**: Path validation and cleaning to prevent directory traversal.
- **Robust Error Handling**: Clear error messages with context.
- **CLI Interface**: Validation, querying, and discovery commands.
- **Wrapper Compatibility**: Designed to work with existing internal config APIs.
- **Comprehensive Testing**: Edge cases, malformed input, and security scenarios are covered.

---

## Installation

### As Binary

```bash
# Build to ~/bin (default target)
make build

# Or build manually
cd cmd/configurator && go build -o ~/bin/configurator .
```

### As Go Module

```bash
go get configurator
```

---

## Usage

### Command Line Interface

#### Configuration Validation

```bash
# Auto-discover and validate project.toml
~/bin/configurator -validate

# Validate specific config file
~/bin/configurator -validate -config /path/to/project.toml
```

#### Configuration Querying

```bash
# Get specific configuration values using dot notation
~/bin/configurator -get project.name
~/bin/configurator -get settings.debug
```

#### Configuration Discovery

```bash
# List all available configuration keys
~/bin/configurator -list
```

#### Project Root Discovery

```bash
# Find project root from current directory
~/bin/configurator -find-root
```

### Library API

#### Basic Configuration Loading

```go
package main

import (
    "fmt"
    "configurator"
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
    var cfg Config
    if err := configurator.LoadInto("config.toml", &cfg); err != nil {
        panic(err)
    }
    fmt.Printf("Loaded: %s v%s\n", cfg.Project.Name, cfg.Project.Version)
}
```

---

## API

### Functions

#### `LoadInto(path string, target any) error`

Loads a TOML file and unmarshals it into the provided struct pointer.

#### `FindProjectRoot(startDir string) (projectRoot string, configPath string, err error)`

Walks up from `startDir` until it finds `project.toml` or reaches the filesystem root.

#### `LoadFromProject(startDir string, target any) (string, error)`

Combines project discovery and config loading in one call.

---

## Design Philosophy & Code Style

This configurator follows key design principles:

- **Unix Philosophy**: Does one thing well and integrates cleanly with other tools.
- **Security First**: Comprehensive input validation and path sanitization.
- **Auto-Discovery**: Intelligent `project.toml` discovery from any directory.
- **Code Style**: This project adheres to the standard Go practice of using **unnamed return values** for function signatures, reserving named returns only for exceptional cases where they significantly improve clarity. This convention is enforced by the `nonamedreturns` linter.

---

## Project Structure

```text
~/Dev/configurator/
├── cmd/configurator/
│   └── main.go           # CLI binary implementation
├── config.go              # Core configuration library
├── config_test.go         # Comprehensive test suite
├── go.mod                 # Go module definition
├── Makefile               # Build automation (targets ~/bin)
├── project.toml           # Project configuration
└── README.md              # This documentation
```
