package configurator_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/book-expert/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/book-expert/configurator"
)

const (
	// Test configuration content.
	TestProjectConfig = `[project]
name = "test-project"
version = "1.0.0"
[settings]
debug = true
port = 8080`

	TestProjectName = "test-project"
	TestVersion100  = "1.0.0"
	TestPort        = 8080
)

type testConfig struct {
	Project struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	} `toml:"project"`
	Settings struct {
		Debug bool `toml:"debug"`
		Port  int  `toml:"port"`
	} `toml:"settings"`
}

func TestLoad_Success(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := fmt.Fprint(w, TestProjectConfig)
			assert.NoError(t, err)
		}),
	)
	defer server.Close()

	t.Setenv("PROJECT_TOML", server.URL)

	var cfg testConfig

	log, err := logger.New(t.TempDir(), "test.log")
	require.NoError(t, err)

	err = configurator.Load(&cfg, log)
	require.NoError(t, err)

	require.Equal(t, TestProjectName, cfg.Project.Name)
	require.Equal(t, TestVersion100, cfg.Project.Version)
	require.True(t, cfg.Settings.Debug)
	require.Equal(t, TestPort, cfg.Settings.Port)
}

func TestLoad_NoEnvVar(t *testing.T) {
	t.Parallel()
	// Ensure the environment variable is not set
	err := os.Unsetenv("PROJECT_TOML")
	require.NoError(t, err)

	var cfg testConfig

	log, err := logger.New(t.TempDir(), "test.log")
	require.NoError(t, err)

	err = configurator.Load(&cfg, log)
	require.Error(t, err)
	require.ErrorIs(t, err, configurator.ErrProjectTomlNotSet)
}

func TestLoad_InvalidURL(t *testing.T) {
	t.Setenv("PROJECT_TOML", "http://invalid-url-that-does-not-exist.local")

	var cfg testConfig

	log, err := logger.New(t.TempDir(), "test.log")
	require.NoError(t, err)

	err = configurator.Load(&cfg, log)
	require.Error(t, err)
}

func TestLoad_BadResponse(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer server.Close()

	t.Setenv("PROJECT_TOML", server.URL)

	var cfg testConfig

	log, err := logger.New(t.TempDir(), "test.log")
	require.NoError(t, err)

	err = configurator.Load(&cfg, log)
	require.Error(t, err)
	require.ErrorIs(t, err, configurator.ErrUnexpectedHTTPStatus)
}

func TestLoad_InvalidToml(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := fmt.Fprint(w, `[project] name = "test-project"`)
			assert.NoError(t, err)
		}),
	)
	defer server.Close()

	t.Setenv("PROJECT_TOML", server.URL)

	var cfg testConfig

	log, err := logger.New(t.TempDir(), "test.log")
	require.NoError(t, err)

	err = configurator.Load(&cfg, log)
	require.Error(t, err)
}
