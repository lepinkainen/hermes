# CLI Configuration Architecture: Viper + Kong Hybrid

## Overview

This document explains the architectural decisions behind our hybrid CLI configuration approach using both Kong and Viper, and why we deliberately maintain this separation rather than migrating to a single solution.

## Current Architecture

### Kong - CLI Argument Parsing
- **Purpose**: Command-line interface definition and parsing
- **Responsibilities**:
  - CLI flag definitions and validation
  - Command structure and subcommands
  - Help text generation
  - Flag-to-parameter mapping

### Viper - Configuration Management
- **Purpose**: Runtime configuration management
- **Responsibilities**:
  - Configuration file loading (YAML)
  - Environment variable binding
  - Default value management
  - Runtime configuration updates
  - Configuration file auto-creation

## File Structure

```
cmd/root.go                 - Kong CLI definitions + Viper integration
internal/config/config.go    - Global configuration state via Viper
config.yml                  - User configuration file
```

## Implementation Pattern

### 1. CLI Structure (Kong)
```go
type CLI struct {
    Overwrite    bool   `help:"Overwrite existing files"`
    Datasette   bool   `help:"Enable Datasette" default:"true"`
    CacheDBFile string `help:"Cache DB path" default:"./cache.db"`
    // ... other flags
}
```

### 2. Configuration Integration
```go
func updateGlobalConfig(cli *CLI) {
    // Update Viper config from Kong-parsed CLI flags
    viper.Set("datasette.enabled", cli.Datasette)
    viper.Set("cache.dbfile", cli.CacheDBFile)
}
```

### 3. Runtime Access (Viper)
```go
// Throughout the codebase
dbPath := viper.GetString("cache.dbfile")
ttlStr := viper.GetString("cache.ttl")
```

## Why Not Full Kong Migration?

### Analysis Results

We evaluated moving from the hybrid approach to Kong-only configuration management. The findings:

**Kong Can Replace:**
- ✅ CLI flag parsing (already implemented)
- ✅ Environment variable binding via `env:` tags
- ✅ Default values via `default:` tags
- ✅ Configuration file loading via `kong.Configuration()`

**Kong Cannot Replace:**
- ❌ Runtime configuration updates (`viper.Set()`)
- ❌ Configuration file auto-creation (`viper.SafeWriteConfig()`)
- ❌ Complex file watching (`viper.WatchConfig()`)
- ❌ Non-CLI environment variable binding

### Migration Complexity

**Files Requiring Changes:** 50+ files
**Viper Usage Points:** 100+ locations
**Test Files to Update:** 15+ files
**Estimated Effort:** 40-60 hours (2-3 weeks)

### Key Blocking Issues

1. **Runtime Configuration**: Many parts of the codebase update configuration at runtime using `viper.Set()`

2. **Global State**: `internal/config/config.go` provides global configuration access that would require dependency injection throughout the codebase

3. **Configuration File Management**: Viper's auto-creation and migration capabilities would need custom implementation

4. **Testing Infrastructure**: Test utilities heavily rely on Viper's reset and mock capabilities

## Architectural Benefits of Hybrid Approach

### 1. **Clear Separation of Concerns**
- Kong handles user interaction (CLI)
- Viper handles system state (configuration)

### 2. **Minimal Cognitive Overhead**
- Each tool does what it does best
- No need to work around limitations

### 3. **Established Patterns**
- Well-understood in Go ecosystem
- Easy for new developers to grasp

### 4. **Future-Proof**
- Can replace either tool independently if needed
- No architectural lock-in

## Best Practices

### 1. Configuration Priority Order
1. CLI flags (highest priority)
2. Environment variables
3. Configuration file values
4. Default values (lowest priority)

### 2. Pattern for New Commands
```go
type NewCmd struct {
    Input  string `short:"f" help:"Input file"`
    Output string `short:"o" help:"Output directory" default:"newcmd"`
}

func (n *NewCmd) Run() error {
    // Fallback to config if CLI flag not provided
    input := n.Input
    if input == "" {
        input = viper.GetString("newcmd.inputfile")
    }
    
    // Execute with validated parameters
    return parseNewCmd(input, n.Output)
}
```

### 3. Configuration File Structure
```yaml
# Namespaced by command
newcmd:
  inputfile: "data.csv"
  output: "custom-output"

# Global settings
cache:
  dbfile: "./cache.db"
  ttl: "720h"

datasette:
  enabled: true
  dbfile: "./hermes.db"
```

## When to Reconsider This Decision

Reconsider the hybrid approach only if:

1. **Performance Issues**: If configuration loading becomes a bottleneck
2. **Tool Consolidation**: If one tool clearly supersedes the other in capabilities
3. **Maintenance Burden**: If maintaining both tools becomes significantly more work than the benefits
4. **Major Rewrite**: If undertaking a major architectural refactor anyway

## Conclusion

The hybrid Kong+Viper approach is the optimal solution for this project:

- **Maintainable**: Clear separation of concerns
- **Extensible**: Easy to add new commands and configuration options
- **Stable**: Based on mature, well-supported libraries
- **Efficient**: Minimal code duplication and complexity

**Decision**: Maintain the current hybrid approach indefinitely. The cost and complexity of migration outweigh any perceived benefits.

---

*Document created: 2025-01-19*
*Last reviewed: 2025-01-19*