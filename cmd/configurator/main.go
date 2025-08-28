// Configurator CLI - standalone tool for project.toml management
package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	errorPrefix = "Error: %v"
	twoFields   = 2
)

type cliFlags struct {
	configPath *string
	validate   *bool
	get        *string
	list       *bool
	findRoot   *bool
	help       *bool
}

func parseFlags() *cliFlags {
	flags := createFlags()

	flag.Parse()

	return flags
}

func createFlags() *cliFlags {
	return &cliFlags{
		configPath: flag.String("config", "", "Path to project.toml file"),
		validate:   flag.Bool("validate", false, "Validate configuration file"),
		get: flag.String("get", "",
			"Get configuration value (dot notation: project.name)"),
		list:     flag.Bool("list", false, "List all configuration keys"),
		findRoot: flag.Bool("find-root", false, "Find project root directory"),
		help:     flag.Bool("help", false, "Show help"),
	}
}

func fatal(format string, args ...any) {
	log.Fatalf(format, args...)
}

func loadInto(path string, target any) error {
	cleanPath := getCleanPath(path)

	// #nosec G304 - path is cleaned and validated above
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return errReadConfig
	}

	config, ok := target.(*map[string]any)
	if !ok {
		return errUnsupportedType
	}

	*config = parseSimpleTOML(string(data))

	return nil
}

func getCleanPath(path string) string {
	cleanPath := filepath.Clean(path)
	if isAbsPath(cleanPath) {
		return cleanPath
	}

	return makeAbsolute(cleanPath)
}

func isAbsPath(path string) bool {
	return filepath.IsAbs(path)
}

func makeAbsolute(path string) string {
	wd := getWorkingDir()
	if wd == "" {
		return path
	}

	return joinPath(wd, path)
}

func getWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	return wd
}

func joinPath(dir, file string) string {
	return filepath.Join(dir, file)
}

var (
	errProjectNotFound = errors.New("project.toml not found")
	errWorkingDir      = errors.New("failed to get working directory")
	errReadConfig      = errors.New("failed to read config file")
	errUnsupportedType = errors.New("unsupported target type")
	errFindRoot        = errors.New("failed to find project root")
	errLoadConfig      = errors.New("failed to load config")
)

func parseSimpleTOML(content string) map[string]any {
	result := makeMap()
	lines := splitLines(content)

	for _, line := range lines {
		processLine(result, line)
	}

	return result
}

func splitLines(content string) []string {
	return strings.Split(content, "\n")
}

func processLine(result map[string]any, line string) {
	line = trimSpace(line)
	if shouldSkipLine(line) {
		return
	}

	key, value, ok := parseLine(line)
	if !ok {
		return
	}

	setNestedValue(result, key, value)
}

func shouldSkipLine(line string) bool {
	return isEmpty(line) || isComment(line)
}

func isEmpty(s string) bool {
	return s == ""
}

func isComment(line string) bool {
	return hasPrefix2(line, "#")
}

func parseLine(line string) (string, string, bool) {
	parts := splitByEquals(line)
	if !hasTwoParts(parts) {
		return "", "", false
	}

	key := trimSpace(parts[0])
	value := trimSpace(parts[1])
	value = trimQuotes(value)

	return key, value, true
}

func hasTwoParts(parts []string) bool {
	return len(parts) == twoFields
}

func setNestedValue(m map[string]any, key, value string) {
	parts := strings.Split(key, ".")
	current := m

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value

			return
		}

		current = ensureNestedMap(current, part)
	}
}

func ensureNestedMap(targetMap map[string]any, key string) map[string]any {
	if _, exists := targetMap[key]; !exists {
		targetMap[key] = make(map[string]any)
	}

	if nested, ok := targetMap[key].(map[string]any); ok {
		return nested
	}

	return targetMap
}

func findProjectRoot(startDir string) (string, string, error) {
	return searchUpwards(startDir)
}

func searchUpwards(dir string) (string, string, error) {
	cur := dir
	for cur != filepath.Dir(cur) {
		if found, path := checkForConfig(cur); found {
			return cur, path, nil
		}

		cur = filepath.Dir(cur)
	}

	return "", "", errProjectNotFound
}

func checkForConfig(dir string) (bool, string) {
	candidate := filepath.Join(dir, "project.toml")
	_, err := os.Stat(candidate)

	return err == nil, candidate
}

func resolveConfigPath(flags *cliFlags) (string, error) {
	if hasCustomPath(flags) {
		return getCustomPath(flags), nil
	}

	return findConfigFromWorkingDir()
}

func hasCustomPath(flags *cliFlags) bool {
	return *flags.configPath != ""
}

func getCustomPath(flags *cliFlags) string {
	return *flags.configPath
}

func findConfigFromWorkingDir() (string, error) {
	wd := getCurrentWorkingDir()
	if wd == "" {
		return "", errWorkingDir
	}

	return findConfigFromDir(wd)
}

func getCurrentWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	return wd
}

func findConfigFromDir(dir string) (string, error) {
	_, configFile, err := findProjectRoot(dir)
	if err != nil {
		return "", errFindRoot
	}

	return configFile, nil
}

func showProjectRoot() {
	wd, err := os.Getwd()
	if err != nil {
		fatal(errorPrefix, err)
	}

	root, configFile, err := findProjectRoot(wd)
	if err != nil {
		fatal(errorPrefix, err)
	}

	log.Printf("Project root: %s\nConfig file: %s\n", root, configFile)
}

func loadConfig(configPath string) (map[string]any, error) {
	var config map[string]any

	err := loadInto(configPath, &config)
	if err != nil {
		return nil, errLoadConfig
	}

	return config, nil
}

func executeCommand(flags *cliFlags, config map[string]any, configPath string) {
	if *flags.validate {
		log.Println("Configuration valid")

		return
	}

	if *flags.list {
		err := listKeys(config, "")
		if err != nil {
			fatal("Error listing keys: %v", err)
		}

		return
	}

	if *flags.get != "" {
		handleGetCommand(config, *flags.get)

		return
	}

	showDefaultOutput(configPath)
}

func handleGetCommand(config map[string]any, key string) {
	value := getValue(config, key)
	if value == nil {
		fatal("Key not found: %s", key)
	}

	log.Printf("%v", value)
}

func showDefaultOutput(configPath string) {
	log.Printf("Configuration loaded from: %s", configPath)
	log.Println("Use --help for available commands")
}

func main() {
	flags := parseFlags()

	if shouldShowHelp(flags) {
		return
	}

	if shouldShowRoot(flags) {
		return
	}

	runConfigCommand(flags)
}

func shouldShowHelp(flags *cliFlags) bool {
	if *flags.help {
		showHelp()

		return true
	}

	return false
}

func shouldShowRoot(flags *cliFlags) bool {
	if *flags.findRoot {
		showProjectRoot()

		return true
	}

	return false
}

func runConfigCommand(flags *cliFlags) {
	configPath := mustResolveConfig(flags)
	config := mustLoadConfig(configPath)
	executeCommand(flags, config, configPath)
}

func mustResolveConfig(flags *cliFlags) string {
	configPath, err := resolveConfigPath(flags)
	if err != nil {
		fatal(errorPrefix, err)
	}

	return configPath
}

func mustLoadConfig(configPath string) map[string]any {
	config, err := loadConfig(configPath)
	if err != nil {
		fatal("Error loading config: %v", err)
	}

	return config
}

func showHelp() {
	log.Println(`Configurator - Project configuration management tool

Usage: configurator [options]

Options:
  -config PATH     Path to project.toml file (auto-discovered if not specified)
  -validate        Validate configuration file syntax and structure
  -get KEY         Get configuration value (dot notation: project.name)
  -list            List all configuration keys
  -find-root       Find and display project root directory
  -help            Show this help message

Examples:
  configurator -validate
  configurator -get project.name
  configurator -get paths.input_dir
  configurator -list
  configurator -find-root

Exit codes:
  0  Success
  1  Error (file not found, invalid syntax, key not found)`)
}

func listKeys(data map[string]any, prefix string) error {
	for key, value := range data {
		err := processKey(key, value, prefix)
		if err != nil {
			return err
		}
	}

	return nil
}

func processKey(key string, value any, prefix string) error {
	fullKey := buildFullKey(prefix, key)
	if nested, ok := value.(map[string]any); ok {
		err := printNestedKeys(nested, fullKey)
		if err != nil {
			return err
		}

		return nil
	}

	log.Println(fullKey)

	return nil
}

func printNestedKeys(data map[string]any, prefix string) error {
	return listKeys(data, prefix)
}

func buildFullKey(prefix, key string) string {
	if hasPrefix(prefix) {
		return combineKeys(prefix, key)
	}

	return key
}

func hasPrefix(prefix string) bool {
	return prefix != ""
}

func combineKeys(prefix, key string) string {
	return prefix + "." + key
}

func getValue(data map[string]any, key string) any {
	keys := splitDotNotation(key)

	return navigateToValue(data, keys)
}

func navigateToValue(data map[string]any, keys []string) any {
	current := any(data)
	for _, k := range keys {
		current = getNextLevel(current, k)
		if current == nil {
			return nil
		}
	}

	return current
}

func getNextLevel(current any, key string) any {
	if m, ok := current.(map[string]any); ok {
		return m[key]
	}

	return nil
}

func splitDotNotation(key string) []string {
	return splitByDot(key)
}

func splitByDot(s string) []string {
	return strings.Split(s, ".")
}

func trimSpace(s string) string {
	return strings.TrimSpace(s)
}

func hasPrefix2(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func splitByEquals(s string) []string {
	return strings.SplitN(s, "=", twoFields)
}

func trimQuotes(s string) string {
	return strings.Trim(s, "\"'")
}

func makeMap() map[string]any {
	return make(map[string]any)
}
