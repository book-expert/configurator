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
	"time"

	"github.com/book-expert/logger"
	"github.com/pelletier/go-toml/v2"
)

const (
	// DefaultURLTimeout is the default timeout for fetching a configuration from a
	// URL.
	DefaultURLTimeout = 10 * time.Second
)

// Static errors for the configurator package.
var (
	ErrUnexpectedHTTPStatus = errors.New("unexpected HTTP status")
	ErrProjectTomlNotSet    = errors.New("PROJECT_TOML environment variable not set")
)

// Load fetches the configuration from the URL specified in the PROJECT_TOML
// environment variable and unmarshals it into the provided struct.
func Load(target any, log *logger.Logger) error {
	url := os.Getenv("PROJECT_TOML")
	if url == "" {
		return ErrProjectTomlNotSet
	}

	data, err := fetchURL(url, log)
	if err != nil {
		return err
	}

	return unmarshalTOML(data, target)
}

// unmarshalTOML unmarshals TOML data into the target.
func unmarshalTOML(data []byte, target any) error {
	err := toml.Unmarshal(data, target)
	if err != nil {
		return fmt.Errorf("parse toml: %w", err)
	}

	return nil
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
