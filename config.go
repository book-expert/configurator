// Package configurator provides generic TOML configuration loading and project discovery.
// It supports rooted config discovery and type-safe unmarshaling for any project
// structure.
package configurator

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	// ProjectConfigFile is the name of the project configuration file.
	ProjectConfigFile = "project.toml"

	// ErrMsgProjectNotFound is returned when project.toml is not found.
	ErrMsgProjectNotFound = "project.toml not found"
	// ErrMsgGetCleanPath is returned when path cleaning fails.
	ErrMsgGetCleanPath = "get clean path: %w"
	// ErrMsgReadConfig is returned when config file reading fails.
	ErrMsgReadConfig = "read config: %w"
	// ErrMsgParseTOML is returned when TOML parsing fails.
	ErrMsgParseTOML = "parse toml: %w"
	// ErrMsgGetWorkingDirectory is returned when getting working directory fails.
	ErrMsgGetWorkingDirectory = "get working directory: %w"
	// ErrMsgFindProjectRoot is returned when finding project root fails.
	ErrMsgFindProjectRoot = "find project root: %w"
	// ErrMsgLoadConfig is returned when loading config fails.
	ErrMsgLoadConfig = "load config: %w"
	// ErrMsgReadFile is returned when file reading fails.
	ErrMsgReadFile = "read file: %w"
)

// ErrProjectNotFound indicates project.toml was not found in directory tree.
var ErrProjectNotFound = errors.New(ErrMsgProjectNotFound)

// LoadInto reads a TOML config file and unmarshals it into the provided struct.
// The target must be a pointer to the struct where config will be stored.
func LoadInto(path string, target any) error {
	cleanPath, err := getCleanPath(path)
	if err != nil {
		return fmt.Errorf(ErrMsgGetCleanPath, err)
	}

	data, err := readConfigFile(cleanPath)
	if err != nil {
		return fmt.Errorf(ErrMsgReadConfig, err)
	}

	return unmarshalTOML(data, target)
}

// getCleanPath cleans and validates the file path.
func getCleanPath(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) {
		return cleanPath, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf(ErrMsgGetWorkingDirectory, err)
	}

	return filepath.Join(wd, cleanPath), nil
}

// readConfigFile reads the configuration file content.
func readConfigFile(path string) ([]byte, error) {
	//nolint:gosec // G304: file path is cleaned and validated
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(ErrMsgReadFile, err)
	}

	return data, nil
}

// unmarshalTOML unmarshals TOML data into target.
func unmarshalTOML(data []byte, target any) error {
	err := toml.Unmarshal(data, target)
	if err != nil {
		return fmt.Errorf(ErrMsgParseTOML, err)
	}

	return nil
}

// FindProjectRoot walks up from startDir until it finds project.toml or reaches root.
// Returns the project root directory and the full path to project.toml.
func FindProjectRoot(startDir string) (projectRoot, configPath string, err error) {
	current := startDir

	for !isRootDirectory(current) {
		if projectRoot, configPath := checkForProjectFile(current); projectRoot != "" {
			return projectRoot, configPath, nil
		}

		current = filepath.Dir(current)
	}

	return "", "", ErrProjectNotFound
}

// checkForProjectFile checks if project.toml exists in the given directory.
func checkForProjectFile(dir string) (projectRoot, configPath string) {
	candidate := filepath.Join(dir, ProjectConfigFile)
	if fileExists(candidate) {
		return dir, candidate
	}

	return "", ""
}

// isRootDirectory checks if we've reached the filesystem root.
func isRootDirectory(current string) bool {
	parent := filepath.Dir(current)

	return parent == current
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

// LoadFromProject finds project.toml starting from startDir and loads it into target.
func LoadFromProject(startDir string, target any) (string, error) {
	projectRoot, configPath, err := FindProjectRoot(startDir)
	if err != nil {
		return "", fmt.Errorf(ErrMsgFindProjectRoot, err)
	}

	err = LoadInto(configPath, target)
	if err != nil {
		return "", fmt.Errorf(ErrMsgLoadConfig, err)
	}

	return projectRoot, nil
}

// LoadFromURL fetches a TOML config from a URL and unmarshals it into the provided struct.
func LoadFromURL(url string, target any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 10-second timeout
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create HTTP request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch TOML from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	return unmarshalTOML(data, target)
}
