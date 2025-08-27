// Configurator CLI - standalone tool for project.toml management
package main

import (
	"flag"
	"fmt"
	"os"

	"configurator"
)

func main() {
	var (
		configPath = flag.String("config", "", "Path to project.toml file")
		validate   = flag.Bool("validate", false, "Validate configuration file")
		get        = flag.String("get", "", "Get configuration value (dot notation: project.name)")
		list       = flag.Bool("list", false, "List all configuration keys")
		findRoot   = flag.Bool("find-root", false, "Find project root directory")
		help       = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Find project root if no config specified
	if *configPath == "" {
		if *findRoot {
			wd, _ := os.Getwd()
			root, configFile, err := configurator.FindProjectRoot(wd)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Project root: %s\nConfig file: %s\n", root, configFile)
			return
		}

		wd, _ := os.Getwd()
		_, configFile, err := configurator.FindProjectRoot(wd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		*configPath = configFile
	}

	// Load config into generic map
	var config map[string]any
	if err := configurator.LoadInto(*configPath, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Execute commands
	switch {
	case *validate:
		fmt.Println("Configuration valid ✅")
	case *list:
		listKeys(config, "")
	case *get != "":
		value := getValue(config, *get)
		if value != nil {
			fmt.Printf("%v\n", value)
		} else {
			fmt.Fprintf(os.Stderr, "Key not found: %s\n", *get)
			os.Exit(1)
		}
	default:
		fmt.Printf("Configuration loaded from: %s\n", *configPath)
		fmt.Println("Use --help for available commands")
	}
}

func showHelp() {
	fmt.Println(`Configurator - Project configuration management tool

Usage: configurator [options]

Options:
  -config PATH     Path to project.toml file (auto-discovered if not specified)
  -validate        Validate configuration file syntax and structure
  -get KEY         Get configuration value using dot notation (e.g., project.name)
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

func listKeys(data map[string]any, prefix string) {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if nested, ok := value.(map[string]any); ok {
			listKeys(nested, fullKey)
		} else {
			fmt.Println(fullKey)
		}
	}
}

func getValue(data map[string]any, key string) any {
	keys := splitDotNotation(key)
	current := any(data)

	for _, k := range keys {
		if m, ok := current.(map[string]any); ok {
			current = m[k]
		} else {
			return nil
		}
	}
	return current
}

func splitDotNotation(key string) []string {
	var result []string
	var current string

	for _, char := range key {
		if char == '.' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
