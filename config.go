// Package configurator provides generic TOML configuration loading and project discovery.
// It supports rooted config discovery and type-safe unmarshaling for any project structure.
package configurator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// LoadInto reads a TOML config file and unmarshals it into the provided struct.
// The target must be a pointer to the struct where config will be stored.
func LoadInto(path string, target any) error {
	// Clean and validate the path to prevent directory traversal
	cleanPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanPath) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		cleanPath = filepath.Join(wd, cleanPath)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := toml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse toml: %w", err)
	}
	return nil
}

// FindProjectRoot walks up from startDir until it finds project.toml or reaches root.
// Returns the project root directory and the full path to project.toml.
func FindProjectRoot(startDir string) (string, string, error) {
	cur := startDir
	for {
		candidate := filepath.Join(cur, "project.toml")
		if _, err := os.Stat(candidate); err == nil {
			return cur, candidate, nil
		}
		next := filepath.Dir(cur)
		if next == cur {
			break
		}
		cur = next
	}
	return "", "", errors.New("project.toml not found")
}

// LoadFromProject finds project.toml starting from startDir and loads it into target.
func LoadFromProject(startDir string, target any) (string, error) {
	projectRoot, configPath, err := FindProjectRoot(startDir)
	if err != nil {
		return "", fmt.Errorf("find project root: %w", err)
	}
	if err := LoadInto(configPath, target); err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}
	return projectRoot, nil
}
