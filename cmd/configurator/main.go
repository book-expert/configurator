// Configurator CLI - standalone tool for project.toml management
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nnikolov3/logger"

	"github.com/nnikolov3/configurator"
)

const (
	// Flag names.
	configFlagName   = "config"
	urlFlagName      = "url"
	validateFlagName = "validate"
	getFlagName      = "get"
	listFlagName     = "list"
	findRootFlagName = "find-root"
	helpFlagName     = "help"

	// Help text and messages.
	helpText = `Configurator - Project configuration management tool

Usage: configurator [options]

Options:
  -config PATH     Path to project.toml file (auto-discovered if not specified)
  -url URL         to remote project.toml file
  -validate        Validate configuration file syntax and structure
  -get KEY         Get configuration value (dot notation: project.name)
  -list            List all configuration keys
  -find-root       Find and display project root directory
  -help            Show this help message`
	useHelpMessage   = "Use --help for available commands"
	KeyNotFoundError = "key not found: %s"

	// Dot separator for nested keys.
	dotSeparator = "."
)

// cliFlags holds the parsed command-line flags.
type cliFlags struct {
	configPath string
	url        string
	get        string
	validate   bool
	list       bool
	findRoot   bool
	help       bool
}

func main() {
	flags := parseFlags()

	log, err := logger.New(".", "configurator.log")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = run(flags, log)
	if err != nil {
		log.Fatal("Error: %v", err)

		return
	}

	func() {
		closeErr := log.Close()
		if closeErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", closeErr)
		}
	}()
}

func run(flags *cliFlags, log *logger.Logger) error {
	if flags.help {
		log.Info(helpText)

		return nil
	}

	if flags.findRoot {
		return showProjectRoot(log)
	}

	config, source, err := loadConfiguration(flags, log)
	if err != nil {
		return err
	}

	return executeCommand(flags, config, source, log)
}

func parseFlags() *cliFlags {
	flags := &cliFlags{
		configPath: "",
		url:        "",
		get:        "",
		validate:   false,
		list:       false,
		findRoot:   false,
		help:       false,
	}
	flag.StringVar(&flags.configPath, configFlagName, "", "Path to project.toml file")
	flag.StringVar(&flags.url, urlFlagName, "", "URL to remote project.toml file")
	flag.BoolVar(
		&flags.validate,
		validateFlagName,
		false,
		"Validate configuration file",
	)
	flag.StringVar(
		&flags.get,
		getFlagName,
		"",
		"Get configuration value (dot notation)",
	)
	flag.BoolVar(&flags.list, listFlagName, false, "List all configuration keys")
	flag.BoolVar(
		&flags.findRoot,
		findRootFlagName,
		false,
		"Find project root directory",
	)
	flag.BoolVar(&flags.help, helpFlagName, false, "Show this help message")
	flag.Parse()

	return flags
}

func loadConfiguration(
	flags *cliFlags, log *logger.Logger,
) (config map[string]any, source string, err error) {
	switch {
	case flags.url != "":
		return loadFromURL(flags.url, log)
	case flags.configPath != "":
		return loadFromPath(flags.configPath)
	default:
		return loadFromDiscovery()
	}
}

func loadFromURL(
	url string,
	log *logger.Logger,
) (config map[string]any, source string, err error) {
	var configData map[string]any

	err = configurator.LoadFromURL(url, &configData, log)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load from URL: %w", err)
	}

	return configData, url, nil
}

func loadFromPath(path string) (config map[string]any, source string, err error) {
	var configData map[string]any

	err = configurator.LoadInto(path, &configData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load from path: %w", err)
	}

	return configData, path, nil
}

func loadFromDiscovery() (config map[string]any, source string, err error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get working directory: %w", err)
	}

	_, configPath, err := configurator.FindProjectRoot(workingDir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to find project root: %w", err)
	}

	var configData map[string]any

	err = configurator.LoadInto(configPath, &configData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load from discovery: %w", err)
	}

	return configData, configPath, nil
}

func executeCommand(
	flags *cliFlags,
	config map[string]any,
	source string,
	log *logger.Logger,
) error {
	switch {
	case flags.validate:
		log.Info("Configuration valid")
	case flags.list:
		listConfigKeys(config, log)
	case flags.get != "":
		printConfigValue(config, flags.get, log)
	default:
		log.Info("Configuration loaded from: %s", source)
		log.Info(useHelpMessage)
	}

	return nil
}

func showProjectRoot(log *logger.Logger) error {
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	root, configFile, err := configurator.FindProjectRoot(workingDir)
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	log.Info("Project root: %s\nConfig file: %s\n", root, configFile)

	return nil
}

func listConfigKeys(data map[string]any, log *logger.Logger) {
	printKeysRecursive(data, "", log)
}

func printKeysRecursive(data map[string]any, prefix string, log *logger.Logger) {
	for key, value := range data {
		fullKey := buildFullKey(prefix, key)
		if nested, ok := value.(map[string]any); ok {
			printKeysRecursive(nested, fullKey, log)
		} else {
			log.Info(fullKey)
		}
	}
}

func buildFullKey(prefix, key string) string {
	if prefix == "" {
		return key
	}

	return prefix + dotSeparator + key
}

func printConfigValue(config map[string]any, key string, log *logger.Logger) {
	value := getConfigValue(config, key)
	if value == nil {
		fatal(KeyNotFoundError, key, log)
	}

	log.Info("%v", value)
}

func getConfigValue(data map[string]any, key string) any {
	parts := strings.Split(key, dotSeparator)

	var current any = data

	for _, part := range parts {
		if m, ok := current.(map[string]any); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	return current
}

func fatal(format string, args ...any) {
	log, err := logger.New(".", "configurator.log")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.Fatal(format, args...)
}
