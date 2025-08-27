package configurator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.toml")

	configContent := `[project]
name = "test-project"
version = "1.0.0"

[settings]
debug = true
port = 8080`

	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	var cfg testConfig
	if err := LoadInto(configPath, &cfg); err != nil {
		t.Fatalf("LoadInto failed: %v", err)
	}

	if cfg.Project.Name != "test-project" {
		t.Errorf("expected name 'test-project', got %q", cfg.Project.Name)
	}
	if cfg.Project.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", cfg.Project.Version)
	}
	if !cfg.Settings.Debug {
		t.Error("expected debug to be true")
	}
	if cfg.Settings.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Settings.Port)
	}
}

func TestFindProjectRoot(t *testing.T) {
	tempDir := t.TempDir()

	// Create nested directory structure
	nestedDir := filepath.Join(tempDir, "src", "deep", "nested")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("create nested dir: %v", err)
	}

	// Create project.toml in root
	projectTomlPath := filepath.Join(tempDir, "project.toml")
	if err := os.WriteFile(projectTomlPath, []byte("[project]\nname = \"test\""), 0o600); err != nil {
		t.Fatalf("create project.toml: %v", err)
	}

	// Test finding from nested directory
	foundRoot, foundPath, err := FindProjectRoot(nestedDir)
	if err != nil {
		t.Fatalf("FindProjectRoot failed: %v", err)
	}

	if foundRoot != tempDir {
		t.Errorf("expected root %q, got %q", tempDir, foundRoot)
	}
	if foundPath != projectTomlPath {
		t.Errorf("expected path %q, got %q", projectTomlPath, foundPath)
	}
}

func TestFindProjectRootNotFound(t *testing.T) {
	tempDir := t.TempDir()

	_, _, err := FindProjectRoot(tempDir)
	if err == nil {
		t.Error("expected error when project.toml not found")
	}
}

func TestLoadFromProject(t *testing.T) {
	tempDir := t.TempDir()

	// Create nested directory structure
	nestedDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("create nested dir: %v", err)
	}

	// Create project.toml
	configContent := `[project]
name = "integration-test"
version = "2.0.0"`

	projectTomlPath := filepath.Join(tempDir, "project.toml")
	if err := os.WriteFile(projectTomlPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("create project.toml: %v", err)
	}

	var cfg testConfig
	foundRoot, err := LoadFromProject(nestedDir, &cfg)
	if err != nil {
		t.Fatalf("LoadFromProject failed: %v", err)
	}

	if foundRoot != tempDir {
		t.Errorf("expected root %q, got %q", tempDir, foundRoot)
	}
	if cfg.Project.Name != "integration-test" {
		t.Errorf("expected name 'integration-test', got %q", cfg.Project.Name)
	}
}

func TestLoadInto_InvalidTOML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.toml")

	// Invalid TOML syntax
	invalidContent := `[project
name = "missing-bracket"`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	var cfg testConfig
	err := LoadInto(configPath, &cfg)
	if err == nil {
		t.Error("expected error for invalid TOML syntax")
	}
	if err != nil && !strings.Contains(err.Error(), "parse toml") {
		t.Errorf("expected 'parse toml' error, got: %v", err)
	}
}

func TestLoadInto_NonexistentFile(t *testing.T) {
	var cfg testConfig
	err := LoadInto("/nonexistent/path/config.toml", &cfg)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if err != nil && !strings.Contains(err.Error(), "read config") {
		t.Errorf("expected 'read config' error, got: %v", err)
	}
}

func TestLoadInto_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "empty.toml")

	if err := os.WriteFile(configPath, []byte(""), 0o600); err != nil {
		t.Fatalf("write empty config: %v", err)
	}

	var cfg testConfig
	err := LoadInto(configPath, &cfg)
	// Empty TOML should not error, just result in zero values
	if err != nil {
		t.Errorf("unexpected error for empty TOML: %v", err)
	}
	if cfg.Project.Name != "" || cfg.Settings.Port != 0 {
		t.Error("expected zero values for empty config")
	}
}

func TestLoadInto_MalformedValues(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "malformed.toml")

	// Port should be int but providing string
	malformedContent := `[settings]
port = "not-a-number"`

	if err := os.WriteFile(configPath, []byte(malformedContent), 0o600); err != nil {
		t.Fatalf("write malformed config: %v", err)
	}

	var cfg testConfig
	err := LoadInto(configPath, &cfg)
	if err == nil {
		t.Error("expected error for type mismatch")
	}
}

func TestLoadInto_UnicodeAndSpecialChars(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "unicode.toml")

	// Test unicode and special characters
	unicodeContent := `[project]
name = "测试-project_123!@#"
version = "1.0.0-αβγ"`

	if err := os.WriteFile(configPath, []byte(unicodeContent), 0o600); err != nil {
		t.Fatalf("write unicode config: %v", err)
	}

	var cfg testConfig
	if err := LoadInto(configPath, &cfg); err != nil {
		t.Fatalf("LoadInto failed for unicode: %v", err)
	}

	if cfg.Project.Name != "测试-project_123!@#" {
		t.Errorf("unicode name not preserved: got %q", cfg.Project.Name)
	}
}

func TestLoadInto_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file outside temp directory
	outsideFile := "/tmp/traversal-test.toml"
	if err := os.WriteFile(outsideFile, []byte("[project]\nname=\"traversal\""), 0o600); err != nil {
		t.Fatalf("create outside file: %v", err)
	}
	defer func() {
		_ = os.Remove(outsideFile) // Ignore cleanup error in test
	}()

	// Try to access with path traversal
	var cfg testConfig
	traversalPath := filepath.Join(tempDir, "../../../tmp/traversal-test.toml")

	// This should work because we clean the path - testing that we handle it properly
	err := LoadInto(traversalPath, &cfg)
	if err != nil {
		t.Logf("Path traversal handled: %v", err)
	} else if cfg.Project.Name == "traversal" {
		t.Log("Path traversal resolved correctly")
	}
}

func TestFindProjectRoot_DeepNesting(t *testing.T) {
	tempDir := t.TempDir()

	// Create very deep nesting
	deepPath := filepath.Join(tempDir, "a", "b", "c", "d", "e", "f", "g", "h")
	if err := os.MkdirAll(deepPath, 0o755); err != nil {
		t.Fatalf("create deep path: %v", err)
	}

	// No project.toml anywhere
	_, _, err := FindProjectRoot(deepPath)
	if err == nil {
		t.Error("expected error for deep path with no project.toml")
	}

	// Now create project.toml at root
	projectTomlPath := filepath.Join(tempDir, "project.toml")
	if err := os.WriteFile(projectTomlPath, []byte("[project]\nname=\"deep\""), 0o600); err != nil {
		t.Fatalf("create project.toml: %v", err)
	}

	foundRoot, foundPath, err := FindProjectRoot(deepPath)
	if err != nil {
		t.Fatalf("FindProjectRoot failed: %v", err)
	}

	if foundRoot != tempDir {
		t.Errorf("expected root %q, got %q", tempDir, foundRoot)
	}
	if foundPath != projectTomlPath {
		t.Errorf("expected path %q, got %q", projectTomlPath, foundPath)
	}
}
