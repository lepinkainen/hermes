# Configuration Utilities

This document describes the configuration utility functions in Hermes, which provide centralized configuration management.

## Overview

The `config` package provides utilities for:

1. Managing global configuration settings
2. Setting default values
3. Accessing configuration values throughout the application
4. Controlling file overwrite behavior

The package uses [Viper](https://github.com/spf13/viper) for configuration management, which supports configuration via files, environment variables, command-line flags, and more.

## Configuration Loading

### InitConfig

The `InitConfig` function initializes the global configuration:

```go
func InitConfig()
```

This function:

1. Sets default values for key configuration options
2. Loads values from the configuration source (file, environment variables, etc.)
3. Populates global variables with the loaded values

The function is typically called during application startup in the `main.go` file or in the `init` function of the root command.

Example usage:

```go
func init() {
    // Initialize configuration
    config.InitConfig()

    // Now global configuration variables are available
    if config.OverwriteFiles {
        log.Info("Overwrite mode enabled")
    }
}
```

## Default Values

The `InitConfig` function sets the following default values:

| Configuration Key   | Default Value | Description                              |
| ------------------- | ------------- | ---------------------------------------- |
| `MarkdownOutputDir` | `./markdown/` | Base directory for Markdown output files |
| `JSONOutputDir`     | `./json/`     | Base directory for JSON output files     |
| `OverwriteFiles`    | `false`       | Whether to overwrite existing files      |

These defaults can be overridden in the configuration file or via command-line flags.

## Global Variables

The `config` package provides the following global variables:

| Variable         | Type   | Description                                                    |
| ---------------- | ------ | -------------------------------------------------------------- |
| `OverwriteFiles` | `bool` | Controls whether existing markdown files should be overwritten |

These variables can be accessed directly from any package that imports the `config` package.

Example usage:

```go
import "github.com/lepinkainen/hermes/internal/config"

func WriteFile(path string, data []byte) error {
    // Check if file exists
    if fileExists(path) && !config.OverwriteFiles {
        // Skip writing if file exists and overwrite is disabled
        return nil
    }

    // Write the file
    return os.WriteFile(path, data, 0644)
}
```

## Utility Functions

### SetOverwriteFiles

The `SetOverwriteFiles` function sets the `OverwriteFiles` flag:

```go
func SetOverwriteFiles(overwrite bool)
```

This function allows changing the overwrite behavior at runtime, which is useful when processing command-line flags.

Example usage:

```go
func init() {
    rootCmd.PersistentFlags().BoolVar(&overwriteFlag, "overwrite", false, "Overwrite existing files")
}

func Execute() {
    // Parse flags and execute command
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    // Update the global configuration based on the flag
    config.SetOverwriteFiles(overwriteFlag)
}
```

## Environment Variables

Viper automatically maps configuration keys to environment variables. By default, environment variables are prefixed with `HERMES_` and use underscores instead of dots or dashes.

For example:

- `MarkdownOutputDir` can be set with the environment variable `HERMES_MARKDOWNOUTPUTDIR`
- `OverwriteFiles` can be set with the environment variable `HERMES_OVERWRITEFILES`

Example usage:

```bash
# Set the Markdown output directory
export HERMES_MARKDOWNOUTPUTDIR="./output/markdown/"

# Enable file overwriting
export HERMES_OVERWRITEFILES=true

# Run the application
./hermes import goodreads
```

## Usage Examples

### Basic Configuration Setup

```go
// In cmd/root.go
func init() {
    cobra.OnInitialize(initConfig)

    // Add flags
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.hermes.yaml)")
    rootCmd.PersistentFlags().BoolVar(&overwriteFlag, "overwrite", false, "Overwrite existing files")
    rootCmd.PersistentFlags().StringVar(&markdownDir, "markdown-dir", "./markdown/", "Directory for markdown output")
    rootCmd.PersistentFlags().StringVar(&jsonDir, "json-dir", "./json/", "Directory for JSON output")

    // Bind flags to viper
    viper.BindPFlag("OverwriteFiles", rootCmd.PersistentFlags().Lookup("overwrite"))
    viper.BindPFlag("MarkdownOutputDir", rootCmd.PersistentFlags().Lookup("markdown-dir"))
    viper.BindPFlag("JSONOutputDir", rootCmd.PersistentFlags().Lookup("json-dir"))
}

func initConfig() {
    if cfgFile != "" {
        // Use config file from the flag
        viper.SetConfigFile(cfgFile)
    } else {
        // Find home directory
        home, err := os.UserHomeDir()
        if err != nil {
            fmt.Println(err)
            os.Exit(1)
        }

        // Search config in home directory with name ".hermes" (without extension)
        viper.AddConfigPath(home)
        viper.SetConfigName(".hermes")
    }

    // Read in environment variables that match
    viper.AutomaticEnv()

    // If a config file is found, read it in
    if err := viper.ReadInConfig(); err == nil {
        fmt.Println("Using config file:", viper.ConfigFileUsed())
    }

    // Initialize the global configuration
    config.InitConfig()

    // Update the overwrite flag based on the command-line flag
    config.SetOverwriteFiles(overwriteFlag)
}
```

### Accessing Configuration in Commands

```go
func importGoodreads(cmd *cobra.Command, args []string) error {
    // Get the output directory from the configuration
    outputDir := viper.GetString("MarkdownOutputDir")
    outputDir = filepath.Join(outputDir, "goodreads")

    // Create the output directory if it doesn't exist
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return fmt.Errorf("failed to create output directory: %w", err)
    }

    // Process files and write output
    for _, book := range books {
        filePath := filepath.Join(outputDir, book.Title+".md")

        // Check if file exists and respect the overwrite flag
        if fileutil.FileExists(filePath) && !config.OverwriteFiles {
            log.Infof("Skipping existing file: %s", filePath)
            continue
        }

        // Write the file
        if err := os.WriteFile(filePath, []byte(book.Content), 0644); err != nil {
            return fmt.Errorf("failed to write file: %w", err)
        }

        log.Infof("Wrote file: %s", filePath)
    }

    return nil
}
```

## API Reference

### Functions

| Function                            | Description                          |
| ----------------------------------- | ------------------------------------ |
| `InitConfig()`                      | Initializes the global configuration |
| `SetOverwriteFiles(overwrite bool)` | Sets the OverwriteFiles flag         |

### Global Variables

| Variable         | Type   | Description                                                    |
| ---------------- | ------ | -------------------------------------------------------------- |
| `OverwriteFiles` | `bool` | Controls whether existing markdown files should be overwritten |

### Configuration Keys

| Key                 | Type     | Default       | Description                              |
| ------------------- | -------- | ------------- | ---------------------------------------- |
| `MarkdownOutputDir` | `string` | `./markdown/` | Base directory for Markdown output files |
| `JSONOutputDir`     | `string` | `./json/`     | Base directory for JSON output files     |
| `OverwriteFiles`    | `bool`   | `false`       | Whether to overwrite existing files      |
