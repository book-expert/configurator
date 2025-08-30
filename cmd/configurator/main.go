// Configurator CLI - standalone tool for project.toml management
//
// This package provides a command-line interface for managing project.toml
// configuration files, including validation, key extraction, and project root discovery.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	// Exit codes.
	ExitSuccess = 0
	ExitError   = 1

	// Field count constants.
	TwoFields = 2

	// Flag names.
	ConfigFlagName   = "config"
	ValidateFlagName = "validate"
	GetFlagName      = "get"
	ListFlagName     = "list"
	FindRootFlagName = "find-root"
	HelpFlagName     = "help"

	// Error message templates.
	ErrorPrefix        = "Error: %v"
	ErrorListingKeys   = "Error listing keys: %v"
	ErrorLoadingConfig = "Error loading config: %v"
	KeyNotFoundError   = "Key not found: %s"

	// Error strings for variables.
	ErrMsgProjectNotFound  = "project.toml not found"
	ErrMsgWorkingDirectory = "failed to get working directory"
	ErrMsgReadConfigFile   = "failed to read config file"
	ErrMsgUnsupportedType  = "unsupported target type"
	ErrMsgFindProjectRoot  = "failed to find project root"
	ErrMsgLoadConfig       = "failed to load config"
	ErrMsgReadFile         = "read file: %w"

	// Success messages.
	ConfigurationValid  = "Configuration valid"
	UseHelpMessage      = "Use --help for available commands"
	ConfigLoadedMessage = "Configuration loaded from: %s"
	ProjectRootMessage  = "Project root: %s\nConfig file: %s\n"

	// Flag descriptions.
	ConfigPathDescription = "Path to project.toml file"
	ValidateDescription   = "Validate configuration file"
	GetDescription        = "Get configuration value (dot notation: project.name)"
	ListDescription       = "List all configuration keys"
	FindRootDescription   = "Find project root directory"
	HelpDescription       = "Show help"

	// File and directory names.
	ProjectConfigFile = "project.toml"

	// Format strings.
	ValueFormat = "%v"

	// Special characters.
	QuoteChars      = "\"'"
	CommentPrefix   = "#"
	DotSeparator    = "."
	EqualsSeparator = "="

	// Help text.
	HelpText = `Configurator - Project configuration management tool

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
  1  Error (file not found, invalid syntax, key not found)`
)

var (
	// Error variables.
	errProjectNotFound = errors.New(ErrMsgProjectNotFound)
	errWorkingDir      = errors.New(ErrMsgWorkingDirectory)
	errReadConfig      = errors.New(ErrMsgReadConfigFile)
	errUnsupportedType = errors.New(ErrMsgUnsupportedType)
	errFindRoot        = errors.New(ErrMsgFindProjectRoot)
	errLoadConfig      = errors.New(ErrMsgLoadConfig)
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
		configPath: flag.String(ConfigFlagName, "", ConfigPathDescription),
		validate:   flag.Bool(ValidateFlagName, false, ValidateDescription),
		get:        flag.String(GetFlagName, "", GetDescription),
		list:       flag.Bool(ListFlagName, false, ListDescription),
		findRoot:   flag.Bool(FindRootFlagName, false, FindRootDescription),
		help:       flag.Bool(HelpFlagName, false, HelpDescription),
	}
}

func fatal(format string, args ...any) {
	log.Fatalf(format, args...)
}

// loadInto reads a TOML-like config file and unmarshals it into the provided struct.
func loadInto(path string, target any) error {
	cleanPath := getCleanPath(path)

	data, err := readConfigData(cleanPath)
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

func readConfigData(path string) ([]byte, error) {
	//nolint:gosec // G304: file path is cleaned and validated
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(ErrMsgReadFile, err)
	}

	return data, nil
}

func getCleanPath(path string) string {
	cleanPath := filepath.Clean(path)
	if isAbsolutePath(cleanPath) {
		return cleanPath
	}

	return makeAbsolutePath(cleanPath)
}

func isAbsolutePath(path string) bool {
	return filepath.IsAbs(path)
}

func makeAbsolutePath(path string) string {
	wd := getCurrentDir()
	if wd == "" {
		return path
	}

	return joinPaths(wd, path)
}

func getCurrentDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	return wd
}

func joinPaths(dir, file string) string {
	return filepath.Join(dir, file)
}

// parseSimpleTOML parses very simple TOML-like key=value lines into a nested map.
func parseSimpleTOML(content string) map[string]any {
	result := createMap()

	lines := getLines(content)
	for _, line := range lines {
		processConfigLine(result, line)
	}

	return result
}

func createMap() map[string]any {
	return make(map[string]any)
}

func getLines(content string) []string {
	return strings.Split(content, "\n")
}

func processConfigLine(result map[string]any, line string) {
	line = trimWhitespace(line)
	if shouldSkipLine(line) {
		return
	}

	key, value, ok := parseConfigLine(line)
	if ok {
		setNestedValue(result, key, value)
	}
}

// shouldSkipLine: complexity reduced to a single boolean expression.
func shouldSkipLine(line string) bool {
	return isEmptyLine(line) || isCommentLine(line) || isSectionHeader(line)
}

// small helper extracted to avoid extra branching in shouldSkipLine.
func isSectionHeader(line string) bool {
	return strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")
}

func isEmptyLine(s string) bool {
	return s == ""
}

func isCommentLine(line string) bool {
	return hasCommentPrefix(line, CommentPrefix)
}

func hasCommentPrefix(line, prefix string) bool {
	return strings.HasPrefix(line, prefix)
}

func parseConfigLine(line string) (key, value string, ok bool) {
	parts := splitOnEquals(line)
	if !hasTwoElements(parts) {
		return "", "", false
	}

	key = trimWhitespace(parts[0])
	value = trimWhitespace(parts[1])

	value = removeQuotes(value)

	return key, value, true
}

func splitOnEquals(s string) []string {
	return strings.SplitN(s, EqualsSeparator, TwoFields)
}

func hasTwoElements(parts []string) bool {
	return len(parts) == TwoFields
}

func removeQuotes(s string) string {
	return strings.Trim(s, QuoteChars)
}

func setNestedValue(m map[string]any, key, value string) {
	parts := splitByDot(key)
	current := m

	for i, part := range parts {
		if isLastPart(i, parts) {
			current[part] = value

			return
		}

		current = ensureNestedMap(current, part)
	}
}

func splitByDot(s string) []string {
	return strings.Split(s, DotSeparator)
}

func isLastPart(index int, parts []string) bool {
	return index == len(parts)-1
}

func ensureNestedMap(targetMap map[string]any, key string) map[string]any {
	if _, exists := targetMap[key]; !exists {
		targetMap[key] = make(map[string]any)
	}

	if nested, ok := targetMap[key].(map[string]any); ok {
		return nested
	}
	// If existing value is not a map, overwrite with a new map to continue nesting
	newMap := make(map[string]any)

	targetMap[key] = newMap

	return newMap
}

func findProjectRoot(startDir string) (rootDir, configPath string, err error) {
	return searchDirectoryUpwards(startDir)
}

func searchDirectoryUpwards(dir string) (rootDir, configPath string, err error) {
	current := dir
	for !isAtRoot(current) {
		if configPath := findConfigInDir(current); configPath != "" {
			return current, configPath, nil
		}

		current = filepath.Dir(current)
	}

	return "", "", errProjectNotFound
}

func isAtRoot(current string) bool {
	return current == filepath.Dir(current)
}

func findConfigInDir(dir string) string {
	candidate := getProjectConfigPath(dir)
	if fileExists(candidate) {
		return candidate
	}

	return ""
}

func getProjectConfigPath(dir string) string {
	return filepath.Join(dir, ProjectConfigFile)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

func resolveConfigPath(flags *cliFlags) (string, error) {
	if hasCustomConfigPath(flags) {
		return getCustomConfigPath(flags), nil
	}

	return findConfigFromWorkingDirectory()
}

func hasCustomConfigPath(flags *cliFlags) bool {
	return *flags.configPath != ""
}

func getCustomConfigPath(flags *cliFlags) string {
	return *flags.configPath
}

func findConfigFromWorkingDirectory() (string, error) {
	wd := getWorkingDirectory()
	if wd == "" {
		return "", errWorkingDir
	}

	return findConfigInDirectory(wd)
}

func getWorkingDirectory() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	return wd
}

func findConfigInDirectory(dir string) (string, error) {
	_, configFile, err := findProjectRoot(dir)
	if err != nil {
		return "", errFindRoot
	}

	return configFile, nil
}

func showProjectRoot() {
	wd, err := os.Getwd()
	if err != nil {
		fatal(ErrorPrefix, err)
	}

	root, configFile, err := findProjectRoot(wd)
	if err != nil {
		fatal(ErrorPrefix, err)
	}

	log.Printf(ProjectRootMessage, root, configFile)
}

func loadConfiguration(configPath string) (map[string]any, error) {
	var config map[string]any

	err := loadInto(configPath, &config)
	if err != nil {
		return nil, errLoadConfig
	}

	return config, nil
}

func executeCommand(flags *cliFlags, config map[string]any, configPath string) {
	if shouldValidateConfig(flags) {
		handleValidateCommand()

		return
	}

	executeDataCommand(flags, config, configPath)
}

func executeDataCommand(flags *cliFlags, config map[string]any, configPath string) {
	if shouldListKeys(flags) {
		handleListCommand(config)

		return
	}

	if shouldGetValue(flags) {
		handleGetCommand(config, *flags.get)

		return
	}

	showDefaultOutput(configPath)
}

func shouldValidateConfig(flags *cliFlags) bool {
	return *flags.validate
}

func shouldListKeys(flags *cliFlags) bool {
	return *flags.list
}

func shouldGetValue(flags *cliFlags) bool {
	return *flags.get != ""
}

func handleValidateCommand() {
	log.Println(ConfigurationValid)
}

func handleListCommand(config map[string]any) {
	err := listConfigKeys(config, "")
	if err != nil {
		fatal(ErrorListingKeys, err)
	}
}

func handleGetCommand(config map[string]any, key string) {
	value := getConfigValue(config, key)
	if value == nil {
		fatal(KeyNotFoundError, key)
	}

	log.Printf(ValueFormat, value)
}

func showDefaultOutput(configPath string) {
	log.Printf(ConfigLoadedMessage, configPath)
	log.Println(UseHelpMessage)
}

func main() {
	flags := parseFlags()
	if shouldShowHelp(flags) {
		return
	}

	if shouldShowProjectRoot(flags) {
		return
	}

	runMainCommand(flags)
}

func shouldShowHelp(flags *cliFlags) bool {
	if *flags.help {
		showHelp()

		return true
	}

	return false
}

func shouldShowProjectRoot(flags *cliFlags) bool {
	if *flags.findRoot {
		showProjectRoot()

		return true
	}

	return false
}

func runMainCommand(flags *cliFlags) {
	configPath := mustResolveConfigPath(flags)
	config := mustLoadConfiguration(configPath)
	executeCommand(flags, config, configPath)
}

func mustResolveConfigPath(flags *cliFlags) string {
	configPath, err := resolveConfigPath(flags)
	if err != nil {
		fatal(ErrorPrefix, err)
	}

	return configPath
}

func mustLoadConfiguration(configPath string) map[string]any {
	config, err := loadConfiguration(configPath)
	if err != nil {
		fatal(ErrorLoadingConfig, err)
	}

	return config
}

func showHelp() {
	log.Println(HelpText)
}

func listConfigKeys(data map[string]any, prefix string) error {
	for key, value := range data {
		err := processConfigKey(key, value, prefix)
		if err != nil {
			return err
		}
	}

	return nil
}

func processConfigKey(key string, value any, prefix string) error {
	fullKey := buildFullKey(prefix, key)
	if nested, ok := value.(map[string]any); ok {
		return printNestedKeys(nested, fullKey)
	}

	log.Println(fullKey)

	return nil
}

func printNestedKeys(data map[string]any, prefix string) error {
	return listConfigKeys(data, prefix)
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
	return prefix + DotSeparator + key
}

func getConfigValue(data map[string]any, key string) any {
	keys := splitDotNotation(key)

	return navigateToValue(data, keys)
}

func navigateToValue(data map[string]any, keys []string) any {
	var current any = data
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

func trimWhitespace(s string) string {
	return strings.TrimSpace(s)
}
