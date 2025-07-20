# Command Utilities

This document describes the command utility functions in Hermes, which provide common operations for setting up and managing commands.

**Note**: While Hermes has transitioned to Kong for its primary CLI, some legacy Cobra-based command utilities may still exist in individual command packages for backwards compatibility.

## Overview

The `cmdutil` package provides utilities for:

1. Managing common command configuration
2. Setting up output directories
3. Adding standardized flags to commands
4. Handling shared command logic

These utilities ensure consistent behavior across all Hermes importers and reduce code duplication. Note that these utilities primarily support legacy Cobra-based command implementations that exist alongside the primary Kong CLI.

## Command Setup

### BaseCommandConfig

The `BaseCommandConfig` struct holds common configuration for import commands:

```go
type BaseCommandConfig struct {
    OutputDir     string
    OutputDirFlag string
    ConfigKey     string
    JSONOutput    string
    WriteJSON     bool
    Overwrite     bool
}
```

Fields:

- `OutputDir`: The directory where output files will be written
- `OutputDirFlag`: The flag name for the output directory
- `ConfigKey`: The key in the configuration file for this command (e.g., "goodreads", "steam")
- `JSONOutput`: The path to the JSON output file
- `WriteJSON`: Whether to write JSON output
- `Overwrite`: Whether to overwrite existing files

### SetupOutputDir

The `SetupOutputDir` function handles the common output directory setup logic:

```go
func SetupOutputDir(cfg *BaseCommandConfig) error
```

This function:

1. Checks for output directory in command-line flags or config file
2. Combines the base markdown directory with the specific subdirectory
3. Sets up the JSON output path if JSON output is enabled
4. Creates necessary directories

Example usage:

```go
cmdConfig := &cmdutil.BaseCommandConfig{
    OutputDir:  outputDir,
    ConfigKey:  "steam",
    WriteJSON:  writeJSON,
    JSONOutput: jsonOutput,
    Overwrite:  overwrite,
}
if err := cmdutil.SetupOutputDir(cmdConfig); err != nil {
    return err
}
// Now cmdConfig.OutputDir and cmdConfig.JSONOutput are fully resolved
```

## Kong Integration

> **Note**: The cmdutil package has been updated to work with Kong CLI framework. Legacy Cobra flag management functions have been removed as Kong handles flags declaratively through struct tags.

With Kong, command flags are defined using struct tags in the command definitions in `cmd/root.go`. For example:

```go
type GoodreadsCmd struct {
    Input      string `short:"f" help:"Path to Goodreads library export CSV file"`
    Output     string `short:"o" help:"Subdirectory under markdown output directory" default:"goodreads"`
    JSON       bool   `help:"Write data to JSON format"`
    JSONOutput string `help:"Path to JSON output file (defaults to json/goodreads.json)"`
}
```

The `BaseCommandConfig` struct is still used internally for configuration management and can be populated from Kong command structs.

This adds two flags to the command:

- `--json`: A boolean flag to enable JSON output
- `--json-output`: A string flag to specify the JSON output file path

## Usage Examples

### Basic Command Setup

```go
var (
    steamID    string
    apiKey     string
    outputDir  string
    writeJSON  bool
    jsonOutput string
    overwrite  bool
    cmdConfig  *cmdutil.BaseCommandConfig
)

var importCmd = &cobra.Command{
    Use:   "steam",
    Short: "Import games from Steam library",
    Long:  `Import games from your Steam library and create markdown files with detailed information.`,
    PreRunE: func(cmd *cobra.Command, args []string) error {
        // If flags weren't provided, try to get values from config
        if steamID == "" {
            steamID = viper.GetString("steam.steamid")
        }
        if apiKey == "" {
            apiKey = viper.GetString("steam.apikey")
        }

        cmdConfig = &cmdutil.BaseCommandConfig{
            OutputDir:  outputDir,
            ConfigKey:  "steam",
            WriteJSON:  writeJSON,
            JSONOutput: jsonOutput,
            Overwrite:  overwrite,
        }
        if err := cmdutil.SetupOutputDir(cmdConfig); err != nil {
            return err
        }
        outputDir = cmdConfig.OutputDir
        jsonOutput = cmdConfig.JSONOutput

        // Still require the values to be present somewhere
        if steamID == "" {
            return cmd.MarkFlagRequired("steamid")
        }
        if apiKey == "" {
            return cmd.MarkFlagRequired("apikey")
        }
        return nil
    },
    RunE: func(cmd *cobra.Command, args []string) error {
        // Command implementation
        return nil
    },
}

func init() {
    importCmd.Flags().StringVarP(&steamID, "steamid", "s", "", "Steam ID of the user (required if not in config)")
    importCmd.Flags().StringVarP(&apiKey, "apikey", "k", "", "Steam API key (required if not in config)")
    
    // Legacy Cobra pattern (Kong handles flags declaratively through struct tags)
    importCmd.Flags().StringVarP(&outputDir, "output", "o", "steam", "Subdirectory under markdown output directory for Steam files")
    importCmd.Flags().BoolVar(&writeJSON, "json", false, "Write data to JSON format")
    importCmd.Flags().StringVar(&jsonOutput, "json-output", "", "Path to JSON output file (defaults to json/steam.json)")

    // Use the global overwrite flag by default
    overwrite = config.OverwriteFiles
}
```

### Handling Output Directories

```go
func ParseSteam() error {
    // cmdConfig was set up in PreRunE
    outputDir := cmdConfig.OutputDir
    jsonOutput := cmdConfig.JSONOutput

    // Ensure output directory exists
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return fmt.Errorf("error creating output directory: %w", err)
    }

    // Process data and write markdown files to outputDir
    // ...

    // Write to JSON if enabled
    if cmdConfig.WriteJSON {
        if err := writeGameToJson(processedGames, jsonOutput); err != nil {
            log.Errorf("Error writing games to JSON: %v\n", err)
        }
    }

    return nil
}
```

## API Reference

### Types

| Type                | Description                                             |
| ------------------- | ------------------------------------------------------- |
| `BaseCommandConfig` | Struct holding common configuration for import commands |

### Functions

| Function                                                                              | Description                                        |
| ------------------------------------------------------------------------------------- | -------------------------------------------------- |
| `SetupOutputDir(cfg *BaseCommandConfig) error`                                        | Handles the common output directory setup logic    |

### BaseCommandConfig Fields

| Field           | Type     | Description                                        |
| --------------- | -------- | -------------------------------------------------- |
| `OutputDir`     | `string` | The directory where output files will be written   |
| `OutputDirFlag` | `string` | The flag name for the output directory             |
| `ConfigKey`     | `string` | The key in the configuration file for this command |
| `JSONOutput`    | `string` | The path to the JSON output file                   |
| `WriteJSON`     | `bool`   | Whether to write JSON output                       |
| `Overwrite`     | `bool`   | Whether to overwrite existing files                |
