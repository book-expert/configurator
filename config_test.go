package configurator_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/book-expert/logger"
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

	IntegrationTestConfig = `[project]
name = "integration-test"
version = "2.0.0"`

	InvalidTOMLContent = `[project
name = "missing-bracket"`

	MalformedPortConfig = `[settings]
port = "not-a-number"`

	// Test project configurations with Unicode.
	UnicodeTestConfig = `[project]
name = "test-project_123!@#"
version = "1.0.0-αβγ"`

	TraversalTestConfig = `[project]
name="traversal"`

	DeepTestConfig = `[project]
name="deep"`

	SimpleTestConfig = `[project]
name = "test"`

	// Test file paths and names.
	TestConfigFileName      = "test.toml"
	EmptyConfigFileName     = "empty.toml"
	InvalidConfigFileName   = "invalid.toml"
	MalformedConfigFileName = "malformed.toml"
	UnicodeConfigFileName   = "unicode.toml"
	NonexistentConfigPath   = "/nonexistent/path/config.toml"
	TraversalTestFilePath   = "/tmp/traversal-test.toml"
	TraversalRelativePath   = "../../../tmp/traversal-test.toml"

	// Test directory names.
	SrcDirName      = "src"
	DeepDirName     = "deep"
	NestedDirName   = "nested"
	DeepPathPattern = "a/b/c/d/e/f/g/h"

	// Test version numbers.
	TestVersion100      = "1.0.0"
	TestVersion200      = "2.0.0"
	TestProjectName     = "test-project"
	IntegrationTestName = "integration-test"
	TraversalTestName   = "traversal"
	UnicodeProjectName  = "test-project_123!@#"

	// Test port numbers.
	TestPort = 8080

	// Error message templates for file operations.
	WriteFileErrorTemplate = "write file: %w"

	// Error message expectations.
	WriteTestConfigMsg       = "write test config: %v"
	LoadIntoFailedMsg        = "LoadInto failed: %v"
	FindProjectRootFailedMsg = "FindProjectRoot failed: %v"
	LoadFromProjectFailedMsg = "LoadFromProject failed: %v"
	ParseTOMLErrorMsg        = "expected 'parse toml' error, got: %v"
	ReadConfigErrorMsg       = "expected 'read config' error, got: %v"
	LoadIntoUnicodeFailedMsg = "LoadInto failed for unicode: %v"
	CreateNestedDirMsg       = "create nested dir: %v"
	CreateProjectTOMLMsg     = "create project.toml: %v"
	CreateOutsideFileMsg     = "create outside file: %v"
	CreateDeepPathMsg        = "create deep path: %v"
	WriteInvalidConfigMsg    = "write invalid config: %v"
	WriteEmptyConfigMsg      = "write empty config: %v"
	WriteMalformedConfigMsg  = "write malformed config: %v"
	WriteUnicodeConfigMsg    = "write unicode config: %v"

	// Validation error messages.
	ExpectedNameErrorMsg            = "expected name %q, got %q"
	ExpectedVersionErrorMsg         = "expected version %q, got %q"
	ExpectedDebugTrueMsg            = "expected debug to be true"
	ExpectedPortErrorMsg            = "expected port %d, got %d"
	ExpectedRootErrorMsg            = "expected root %q, got %q"
	ExpectedPathErrorMsg            = "expected path %q, got %q"
	ExpectedProjectNotFoundErrorMsg = "expected error when project.toml not found"
	ExpectedInvalidTOMLErrorMsg     = "expected error for invalid TOML syntax"
	ExpectedNonexistentFileErrorMsg = "expected error for nonexistent file"
	ExpectedZeroValuesErrorMsg      = "expected zero values for empty config"
	ExpectedTypeMismatchErrorMsg    = "expected error for type mismatch"
	ExpectedDeepPathErrorMsg        = "expected error for deep path with no project.toml"
	UnexpectedEmptyTOMLErrorMsg     = "unexpected error for empty TOML: %v"
	UnicodeNameNotPreservedMsg      = "unicode name not preserved: got %q"

	// Log messages.
	PathTraversalHandledMsg  = "Path traversal handled: %v"
	PathTraversalResolvedMsg = "Path traversal resolved correctly"

	// Error check strings.
	ParseTOMLString  = "parse toml"
	ReadConfigString = "read config"

	// File permissions.
	TestFilePermissions = 0o600
	TestDirPermissions  = 0o750
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

func TestLoadInto(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, TestConfigFileName)

	err := writeTestConfig(configPath, TestProjectConfig)
	if err != nil {
		t.Fatalf(WriteTestConfigMsg, err)
	}

	cfg := loadAndValidateConfig(t, configPath)
	validateProjectConfig(t, cfg)
}

func writeTestConfig(path, content string) error {
	err := os.WriteFile(path, []byte(content), TestFilePermissions)
	if err != nil {
		return fmt.Errorf(WriteFileErrorTemplate, err)
	}

	return nil
}

func loadAndValidateConfig(t *testing.T, configPath string) testConfig {
	t.Helper()

	var cfg testConfig

	err := configurator.LoadInto(configPath, &cfg)
	if err != nil {
		t.Fatalf(LoadIntoFailedMsg, err)
	}

	return cfg
}

func validateProjectConfig(t *testing.T, cfg testConfig) {
	t.Helper()

	validateProjectName(t, cfg.Project.Name, TestProjectName)
	validateProjectVersion(t, cfg.Project.Version, TestVersion100)
	validateDebugSetting(t, cfg.Settings.Debug)
	validatePortSetting(t, cfg.Settings.Port, TestPort)
}

func validateProjectName(t *testing.T, actual, expected string) {
	t.Helper()

	if actual != expected {
		t.Errorf(ExpectedNameErrorMsg, expected, actual)
	}
}

func validateProjectVersion(t *testing.T, actual, expected string) {
	t.Helper()

	if actual != expected {
		t.Errorf(ExpectedVersionErrorMsg, expected, actual)
	}
}

func validateDebugSetting(t *testing.T, debug bool) {
	t.Helper()

	if !debug {
		t.Error(ExpectedDebugTrueMsg)
	}
}

func validatePortSetting(t *testing.T, actual, expected int) {
	t.Helper()

	if actual != expected {
		t.Errorf(ExpectedPortErrorMsg, expected, actual)
	}
}

func TestFindProjectRoot(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	nestedDir := createNestedDirectory(t, tempDir)
	projectTomlPath := createProjectTOML(t, tempDir, SimpleTestConfig)

	foundRoot, foundPath := findAndValidateProjectRoot(t, nestedDir)
	validateFoundRoot(t, foundRoot, tempDir)
	validateFoundPath(t, foundPath, projectTomlPath)
}

func createNestedDirectory(t *testing.T, tempDir string) string {
	t.Helper()

	nestedDir := filepath.Join(tempDir, SrcDirName, DeepDirName, NestedDirName)

	err := os.MkdirAll(nestedDir, TestDirPermissions)
	if err != nil {
		t.Fatalf(CreateNestedDirMsg, err)
	}

	return nestedDir
}

func createProjectTOML(t *testing.T, dir, content string) string {
	t.Helper()

	projectTomlPath := filepath.Join(dir, configurator.ProjectConfigFile)

	err := writeTestConfig(projectTomlPath, content)
	if err != nil {
		t.Fatalf(CreateProjectTOMLMsg, err)
	}

	return projectTomlPath
}

func findAndValidateProjectRoot(t *testing.T, dir string) (foundRoot, foundPath string) {
	t.Helper()

	var err error

	foundRoot, foundPath, err = configurator.FindProjectRoot(dir)
	if err != nil {
		t.Fatalf(FindProjectRootFailedMsg, err)
	}

	return foundRoot, foundPath
}

func validateFoundRoot(t *testing.T, found, expected string) {
	t.Helper()

	if found != expected {
		t.Errorf(ExpectedRootErrorMsg, expected, found)
	}
}

func validateFoundPath(t *testing.T, found, expected string) {
	t.Helper()

	if found != expected {
		t.Errorf(ExpectedPathErrorMsg, expected, found)
	}
}

func TestFindProjectRootNotFound(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	_, _, err := configurator.FindProjectRoot(tempDir)
	if err == nil {
		t.Error(ExpectedProjectNotFoundErrorMsg)
	}
}

func TestLoadFromProject(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	nestedDir := createSrcDirectory(t, tempDir)
	createProjectTOML(t, tempDir, IntegrationTestConfig)

	foundRoot, cfg := loadFromProjectAndValidate(t, nestedDir)
	validateIntegrationTestConfig(t, foundRoot, tempDir, cfg)
}

func createSrcDirectory(t *testing.T, tempDir string) string {
	t.Helper()

	nestedDir := filepath.Join(tempDir, SrcDirName)

	err := os.MkdirAll(nestedDir, TestDirPermissions)
	if err != nil {
		t.Fatalf(CreateNestedDirMsg, err)
	}

	return nestedDir
}

func loadFromProjectAndValidate(t *testing.T, dir string) (string, testConfig) {
	t.Helper()

	var cfg testConfig

	foundRoot, err := configurator.LoadFromProject(dir, &cfg)
	if err != nil {
		t.Fatalf(LoadFromProjectFailedMsg, err)
	}

	return foundRoot, cfg
}

func validateIntegrationTestConfig(
	t *testing.T,
	foundRoot, expectedRoot string,
	cfg testConfig,
) {
	t.Helper()

	validateFoundRoot(t, foundRoot, expectedRoot)
	validateProjectName(t, cfg.Project.Name, IntegrationTestName)
}

func TestLoadInto_InvalidTOML(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, InvalidConfigFileName)

	err := writeTestConfig(configPath, InvalidTOMLContent)
	if err != nil {
		t.Fatalf(WriteInvalidConfigMsg, err)
	}

	validateInvalidTOMLError(t, configPath)
}

func validateInvalidTOMLError(t *testing.T, configPath string) {
	t.Helper()

	var cfg testConfig

	err := configurator.LoadInto(configPath, &cfg)
	if err == nil {
		t.Error(ExpectedInvalidTOMLErrorMsg)

		return
	}

	if !strings.Contains(err.Error(), ParseTOMLString) {
		t.Errorf(ParseTOMLErrorMsg, err)
	}
}

func TestLoadInto_NonexistentFile(t *testing.T) {
	t.Parallel()

	var cfg testConfig

	err := configurator.LoadInto(NonexistentConfigPath, &cfg)
	if err == nil {
		t.Error(ExpectedNonexistentFileErrorMsg)

		return
	}

	if !strings.Contains(err.Error(), ReadConfigString) {
		t.Errorf(ReadConfigErrorMsg, err)
	}
}

func TestLoadInto_EmptyFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, EmptyConfigFileName)

	err := writeTestConfig(configPath, "")
	if err != nil {
		t.Fatalf(WriteEmptyConfigMsg, err)
	}

	validateEmptyConfig(t, configPath)
}

func validateEmptyConfig(t *testing.T, configPath string) {
	t.Helper()

	var cfg testConfig

	err := configurator.LoadInto(configPath, &cfg)
	if err != nil {
		t.Errorf(UnexpectedEmptyTOMLErrorMsg, err)

		return
	}

	validateConfigIsEmpty(t, cfg)
}

func validateConfigIsEmpty(t *testing.T, cfg testConfig) {
	t.Helper()

	if cfg.Project.Name != "" || cfg.Settings.Port != 0 {
		t.Error(ExpectedZeroValuesErrorMsg)
	}
}

func TestLoadInto_MalformedValues(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, MalformedConfigFileName)

	err := writeTestConfig(configPath, MalformedPortConfig)
	if err != nil {
		t.Fatalf(WriteMalformedConfigMsg, err)
	}

	validateMalformedConfig(t, configPath)
}

func validateMalformedConfig(t *testing.T, configPath string) {
	t.Helper()

	var cfg testConfig

	err := configurator.LoadInto(configPath, &cfg)
	if err == nil {
		t.Error(ExpectedTypeMismatchErrorMsg)
	}
}

func TestLoadInto_UnicodeAndSpecialChars(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, UnicodeConfigFileName)

	err := writeTestConfig(configPath, UnicodeTestConfig)
	if err != nil {
		t.Fatalf(WriteUnicodeConfigMsg, err)
	}

	validateUnicodeConfig(t, configPath)
}

func validateUnicodeConfig(t *testing.T, configPath string) {
	t.Helper()

	var cfg testConfig

	err := configurator.LoadInto(configPath, &cfg)
	if err != nil {
		t.Fatalf(LoadIntoUnicodeFailedMsg, err)
	}

	if cfg.Project.Name != UnicodeProjectName {
		t.Errorf(UnicodeNameNotPreservedMsg, cfg.Project.Name)
	}
}

func TestLoadInto_PathTraversal(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	outsideFile := createOutsideFile(t)

	defer cleanupOutsideFile(outsideFile)

	validatePathTraversal(t, tempDir)
}

func createOutsideFile(t *testing.T) string {
	t.Helper()

	outsideFile := TraversalTestFilePath

	err := writeTestConfig(outsideFile, TraversalTestConfig)
	if err != nil {
		t.Fatalf(CreateOutsideFileMsg, err)
	}

	return outsideFile
}

func cleanupOutsideFile(_ string) {
	// Ignore cleanup errors in test
	//nolint:gosec // G104: intentionally ignoring error in test cleanup
	_ = os.Remove(TraversalTestFilePath)
}

func validatePathTraversal(t *testing.T, tempDir string) {
	t.Helper()

	var cfg testConfig

	traversalPath := filepath.Join(tempDir, TraversalRelativePath)

	err := configurator.LoadInto(traversalPath, &cfg)
	if err != nil {
		t.Logf(PathTraversalHandledMsg, err)
	} else if cfg.Project.Name == TraversalTestName {
		t.Log(PathTraversalResolvedMsg)
	}
}

func TestFindProjectRoot_DeepNesting(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	deepPath := createDeepPath(t, tempDir)

	validateDeepPathError(t, deepPath)

	projectTomlPath := createProjectTOML(t, tempDir, DeepTestConfig)
	validateDeepPathSuccess(t, deepPath, tempDir, projectTomlPath)
}

func createDeepPath(t *testing.T, tempDir string) string {
	t.Helper()

	deepPath := filepath.Join(tempDir, DeepPathPattern)

	err := os.MkdirAll(deepPath, TestDirPermissions)
	if err != nil {
		t.Fatalf(CreateDeepPathMsg, err)
	}

	return deepPath
}

func validateDeepPathError(t *testing.T, deepPath string) {
	t.Helper()

	_, _, err := configurator.FindProjectRoot(deepPath)
	if err == nil {
		t.Error(ExpectedDeepPathErrorMsg)
	}
}

func validateDeepPathSuccess(t *testing.T, deepPath, expectedRoot, expectedPath string) {
	t.Helper()

	foundRoot, foundPath := findAndValidateProjectRoot(t, deepPath)
	validateFoundRoot(t, foundRoot, expectedRoot)
	validateFoundPath(t, foundPath, expectedPath)
}

func TestLoadFromURL(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)

			_, err := io.WriteString(w, TestProjectConfig)
			if err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		}),
	)
	defer server.Close()

	var cfg testConfig

	log, err := logger.New(".", "test.log")
	require.NoError(t, err)

	defer func() {
		closeErr := log.Close()
		if closeErr != nil {
			t.Logf("failed to close logger: %v", closeErr)
		}
	}()

	err = configurator.LoadFromURL(server.URL, &cfg, log)
	require.NoError(t, err)

	validateProjectConfig(t, cfg)
}
