# Configurator

A robust, standalone configuration management system with comprehensive input
validation and CLI interface. This tool can be used both as a Go library
and as a command-line binary for TOML configuration handling.

## Architecture

This project provides both:

- **Library API** (`config.go`): Generic TOML loading and project
  discovery for Go applications
- **CLI Binary** (`cmd/configurator/main.go`): Standalone executable for
  shell scripts and external tools

## Features

- **Generic TOML Loading**: Load any TOML structure into Go structs
- **Project Discovery**: Automatically find `project.toml` files by walking
  up directory tree
- **Security**: Path validation and cleaning to prevent directory traversal
- **Robust Error Handling**: Clear error messages with context
- **CLI Interface**: Validation, querying, and discovery commands
- **Wrapper Compatibility**: Designed to work with existing internal config APIs
- **Comprehensive Testing**: Edge cases, malformed input, and security
  scenarios covered

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

## Usage

### Command Line Interface

#### Configuration Validation

```bash
# Auto-discover and validate project.toml
~/bin/configurator -validate

# Validate specific config file
~/bin/configurator -validate -config /path/to/project.toml

# Example output:
# Configuration valid ✅
```

#### Configuration Querying

```bash
# Get specific configuration values using dot notation (auto-discovers config)
~/bin/configurator -get project.name
~/bin/configurator -get project.version
~/bin/configurator -get settings.debug
~/bin/configurator -get paths.input_dir
~/bin/configurator -get tesseract.language

# With specific config file
~/bin/configurator -get project.name -config project.toml

# Example outputs:
# book_expert
# 1.0.0
# true
# ./data/raw
# eng
```

#### Configuration Discovery

```bash
# List all available configuration keys
~/bin/configurator -list

# Sample output:
# project.name
# project.version
# paths.input_dir
# paths.output_dir
# settings.dpi
# settings.workers
# tesseract.language
# google_api.max_retries
```

#### Project Root Discovery

```bash
# Find project root from current directory
~/bin/configurator -find-root

# Example output:
# Project root: /home/user/Dev/book_expert
# Config file: /home/user/Dev/book_expert/project.toml

# Works from any subdirectory
cd src/deep/nested/dir && ~/bin/configurator -find-root
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

### Project Discovery

```go
func main() {
    var cfg Config
    projectRoot, err := configurator.LoadFromProject(".", &cfg)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Found project at: %s\n", projectRoot)
}
```

### Manual Project Root Discovery

```go
func main() {
    root, configPath, err := configurator.FindProjectRoot("./src/deep/nested")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Project root: %s\nConfig file: %s\n", root, configPath)
}
```

## API

### Functions

#### `LoadInto(path string, target any) error`

Loads a TOML file and unmarshals it into the provided struct pointer.
- `path`: File path to TOML config (cleaned and validated)
- `target`: Pointer to struct where config will be unmarshaled
- Returns error if file cannot be read or TOML is invalid

#### `FindProjectRoot(startDir string) (string, string, error)`

Walks up from startDir until it finds `project.toml` or reaches
filesystem root.
- `startDir`: Directory to start search from
- Returns: project root directory, path to project.toml, error

#### `LoadFromProject(startDir string, target any) (string, error)`

Combines project discovery and config loading in one call.
- `startDir`: Directory to start search from
- `target`: Pointer to struct for config
- Returns: project root directory, error

### Wrapper Integration

The configurator is designed to work with wrapper functions that maintain existing APIs:

```go
// Example wrapper that calls the binary
func Load(path string) (Config, error) {
    var cfg Config
    
    // Use configurator binary for validation first
    cmd := exec.Command(os.ExpandEnv("$HOME/bin/configurator"),
        "-validate", "-config", path)
    if err := cmd.Run(); err != nil {
        return cfg, fmt.Errorf("config validation failed: %w",
            err)
    }
    
    // Then load normally
    data, err := os.ReadFile(path)
    if err != nil {
        return cfg, fmt.Errorf("read config: %w", err)
    }
    if err := toml.Unmarshal(data, &cfg); err != nil {
        return cfg, fmt.Errorf("parse toml: %w", err)
    }
    return cfg, nil
}
```

## Project Structure

```text
~/Dev/configurator/
├── config.go              # Core configuration library
├── cmd/configurator/
│   └── main.go           # CLI binary implementation
├── config_test.go        # Comprehensive test suite
├── Makefile             # Build automation (targets ~/bin)
├── go.mod               # Go module definition
├── project.toml         # Project configuration
└── README.md            # This documentation
```

## Development Workflow

```bash
# Format, lint, test, and build in proper sequence
make all

# Individual steps
make format     # Format code with gofmt
make lint       # Run comprehensive linting (go vet, staticcheck, gosec)
make test       # Run test suite with coverage
make build      # Build binary to ~/bin/configurator
```

## Testing

Run all tests including edge cases:

```bash
go test -v
```

Tests cover:
- ✅ Basic TOML loading (78.6% coverage)
- ✅ Project discovery from nested directories
- ✅ Invalid TOML syntax handling
- ✅ Nonexistent file handling
- ✅ Empty file handling
- ✅ Type mismatch errors
- ✅ Unicode and special character support
- ✅ Path traversal security validation
- ✅ Deep directory nesting (10+ levels)
- ✅ CLI functionality (validation, querying, discovery)
- ✅ Dot notation parsing and key resolution

## Security

- **Path Validation**: All paths are cleaned using `filepath.Clean()`
- **Directory Traversal Protection**: Relative paths are resolved safely
- **No External Dependencies**: Only uses standard library and go-toml/v2

## Integration Examples

### book_expert Integration

The configurator is used in book_expert through wrapper functions that
maintain the original internal API while calling the standalone binary
for validation underneath.

### Shell Script Integration

```bash
#!/bin/bash
# Configuration management functions for shell scripts

# Function to validate config (auto-discovery)
validate_config() {
    if ~/bin/configurator -validate; then
        echo "✅ Configuration is valid"
        return 0
    else
        echo "❌ Configuration validation failed"
        return 1
    fi
}

# Function to get config values (auto-discovery)
get_config() {
    local key="$1"
    ~/bin/configurator -get "$key"
}

# Function to check if we're in a project directory
check_project() {
    if ~/bin/configurator -find-root >/dev/null 2>&1; then
        return 0
    else
        echo "❌ Not in a project directory (no project.toml found)" >&2
        return 1
    fi
}

# Usage example
check_project || exit 1
validate_config || exit 1

# Get project configuration
PROJECT_NAME=$(get_config "project.name")
PROJECT_VERSION=$(get_config "project.version")
INPUT_DIR=$(get_config "paths.input_dir")
OUTPUT_DIR=$(get_config "paths.output_dir")
WORKERS=$(get_config "settings.workers")
DPI=$(get_config "settings.dpi")

echo "Processing $PROJECT_NAME v$PROJECT_VERSION"
echo "Input: $INPUT_DIR -> Output: $OUTPUT_DIR"
echo "Workers: $WORKERS, DPI: $DPI"
```

### External Process Integration

The binary design allows any language or system to use the configurator:

```python
import subprocess
import json
import sys

def validate_config():
    """Validate TOML configuration file (auto-discovery)."""
    try:
        subprocess.run([
            f"{os.path.expanduser('~/bin/configurator')}",
            "-validate"
        ], check=True, capture_output=True)
        return True
    except subprocess.CalledProcessError:
        return False

def get_config_value(key):
    """Get specific configuration value (auto-discovery)."""
    try:
        result = subprocess.run([
            f"{os.path.expanduser('~/bin/configurator')}",
            "-get", key
        ], check=True, capture_output=True, text=True)
        return result.stdout.strip()
    except subprocess.CalledProcessError:
        return None

def get_all_config_keys():
    """Get list of all configuration keys."""
    try:
        result = subprocess.run([
            f"{os.path.expanduser('~/bin/configurator')}",
            "-list"
        ], check=True, capture_output=True, text=True)
        return result.stdout.strip().split('\n')
    except subprocess.CalledProcessError:
        return []

def find_project_root():
    """Find project root directory."""
    try:
        result = subprocess.run([
            f"{os.path.expanduser('~/bin/configurator')}",
            "-find-root"
        ], check=True, capture_output=True, text=True)
        lines = result.stdout.strip().split('\n')
        root = lines[0].replace('Project root: ', '')
        config_file = lines[1].replace('Config file: ', '')
        return root, config_file
    except subprocess.CalledProcessError:
        return None, None

# Usage examples
if validate_config():
    project_name = get_config_value("project.name")
    version = get_config_value("project.version")
    workers = get_config_value("settings.workers")
    
    print(f"Project: {project_name} v{version}")
    print(f"Workers: {workers}")
    
    # Get project structure
    root, config = find_project_root()
    print(f"Project root: {root}")
    print(f"Config file: {config}")
    
    # List all available keys
    keys = get_all_config_keys()
    print(f"Available config keys ({len(keys)}):")
    for key in keys[:10]:  # Show first 10
        print(f"  {key}")
else:
    print("Configuration validation failed", file=sys.stderr)
```

## Design Philosophy

This configurator follows key design principles:
- **No Mocks**: Real implementations only, no fake/mock objects
- **Security First**: Comprehensive input validation and path sanitization
- **Wrapper Compatibility**: Maintains existing APIs while leveraging standalone architecture
- **Unix Philosophy**: Does one thing well, integrates cleanly with
  other tools
- **Defensive Programming**: Graceful handling of edge cases and failures
- **Configuration Validation**: Always validate before processing
- **Auto-Discovery**: Intelligent project.toml discovery from any directory
- **Dot Notation**: Intuitive nested configuration access

## Requirements

- Go 1.24+ (tested with Go 1.24 and 1.25)
- github.com/pelletier/go-toml/v2 v2.2.4+
- Unix-like environment (for ~/bin path)

## Testing Environment

Tested on:
- **OS**: Fedora 42
- **Kernel**: Linux 6.15.9-201.fc42.x86_64+debug  
- **Go**: 1.24 and 1.25

## License

This project follows the same license as the parent projects it serves.
