// Package configurator provides generic TOML configuration loading and project discovery.
// It supports rooted config discovery and type-safe unmarshaling for any project
// structure.
package configurator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nnikolov3/logger"
	"github.com/pelletier/go-toml/v2"
)

const (
	// ProjectConfigFile is the name of the project configuration file.
	ProjectConfigFile = "project.toml"

	// DefaultURLTimeout is the default timeout for fetching a configuration from a
	// URL.
	DefaultURLTimeout = 10 * time.Second
)

// Static errors for the configurator package.
var (
	ErrProjectConfigNotFound = errors.New("project.toml not found")
	ErrUnexpectedHTTPStatus  = errors.New("unexpected HTTP status")
	ErrPathTraversalAttempt  = errors.New("path is outside the current directory")
)

// LoadInto reads a TOML config file and unmarshals it into the provided struct.
func LoadInto(path string, target any) error {
	cleanPath, err := getCleanPath(path)
	if err != nil {
		return fmt.Errorf("get clean path: %w", err)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	return unmarshalTOML(data, target)
}

// getCleanPath cleans and validates a file path to prevent directory traversal.
func getCleanPath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	absPath := filepath.Join(workingDir, path)
	cleanedPath := filepath.Clean(absPath)

	rel, err := filepath.Rel(workingDir, cleanedPath)
	if err != nil {
		return "", fmt.Errorf("could not compute relative path: %w", err)
	}

	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("%w: path %q", ErrPathTraversalAttempt, path)
	}

	return cleanedPath, nil
}

// unmarshalTOML unmarshals TOML data into the target.
func unmarshalTOML(data []byte, target any) error {
	err := toml.Unmarshal(data, target)
	if err != nil {
		return fmt.Errorf("parse toml: %w", err)
	}

	return nil
}

// FindProjectRoot walks up from startDir until it finds project.toml.
func FindProjectRoot(startDir string) (projectRoot, configPath string, err error) {
	current := startDir
	for {
		candidate := filepath.Join(current, ProjectConfigFile)
		if fileExists(candidate) {
			return current, candidate, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", "", ErrProjectConfigNotFound
		}

		current = parent
	}
}

// fileExists checks if a file exists at a given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

// LoadFromProject finds and loads project.toml from a starting directory.
func LoadFromProject(startDir string, target any) (string, error) {
	projectRoot, configPath, err := FindProjectRoot(startDir)
	if err != nil {
		return "", fmt.Errorf("find project root: %w", err)
	}

	err = LoadInto(configPath, target)
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}

	return projectRoot, nil
}

// LoadFromURL fetches a TOML config from a URL.
func LoadFromURL(url string, target any, log *logger.Logger) error {
	data, err := fetchURL(url, log)
	if err != nil {
		return err
	}

	return unmarshalTOML(data, target)
}

func fetchURL(url string, log *logger.Logger) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultURLTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch toml from url: %w", err)
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Warn("failed to close response body: %v", err)
		}
	}()

	return processResponse(resp)
}

func processResponse(resp *http.Response) ([]byte, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrUnexpectedHTTPStatus, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return data, nil
}
