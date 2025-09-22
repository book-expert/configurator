// Package configurator sets the global configuration
package configurator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/book-expert/logger"
	"github.com/pelletier/go-toml/v2"
)

// DefaultURLTimeout defines the default timeout for fetching the configuration URL.
const DefaultURLTimeout = 10 * time.Second

// ErrUnexpectedHTTPStatus is returned when the HTTP request to fetch the TOML file does not return a 200 OK status.
var ErrUnexpectedHTTPStatus = errors.New("unexpected HTTP status")

// ErrProjectTomlNotSet is returned when the PROJECT_TOML environment variable is not set.
var ErrProjectTomlNotSet = errors.New("PROJECT_TOML environment variable not set")

// Load fetches application configuration from a remote URL, specified by the PROJECT_TOML
// environment variable, and unmarshals it into a type-safe Go struct.
// It acts as a centralized configuration client for other services within the Book Expert project.
func Load(target any, logger *logger.Logger) error {
	projectTOMLURL := os.Getenv("PROJECT_TOML")
	if projectTOMLURL == "" {
		return ErrProjectTomlNotSet
	}

	tomlContent, fetchErr := fetchURL(projectTOMLURL, logger)
	if fetchErr != nil {
		return fmt.Errorf("failed to fetch TOML from %s: %w", projectTOMLURL, fetchErr)
	}

	unmarshalErr := unmarshalTOML(tomlContent, target)
	if unmarshalErr != nil {
		return fmt.Errorf("failed to unmarshal TOML: %w", unmarshalErr)
	}

	return nil
}

// fetchURL handles the HTTP request to fetch the TOML file from the specified URL.
func fetchURL(url string, logger *logger.Logger) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultURLTimeout)
	defer cancel()

	req, newRequestErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if newRequestErr != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", newRequestErr)
	}

	resp, doRequestErr := http.DefaultClient.Do(req)
	if doRequestErr != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", doRequestErr)
	}

	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			logger.Error("failed to close response body: %v", closeErr)
		}
	}()

	body, processResponseErr := processResponse(resp)
	if processResponseErr != nil {
		return nil, fmt.Errorf("failed to process HTTP response: %w", processResponseErr)
	}

	return body, nil
}

// processResponse validates the HTTP response status and reads the response body.
func processResponse(resp *http.Response) ([]byte, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", ErrUnexpectedHTTPStatus, resp.StatusCode)
	}

	body, readAllErr := io.ReadAll(resp.Body)
	if readAllErr != nil {
		return nil, fmt.Errorf("failed to read response body: %w", readAllErr)
	}

	return body, nil
}

// unmarshalTOML parses the raw TOML data into the provided Go struct.
func unmarshalTOML(data []byte, target interface{}) error {
	unmarshalErr := toml.Unmarshal(data, target)
	if unmarshalErr != nil {
		return fmt.Errorf("failed to unmarshal TOML data: %w", unmarshalErr)
	}

	return nil
}
