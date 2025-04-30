# Contributing to Hermes

This document provides guidelines for contributing to the Hermes project. Whether you're fixing bugs, adding features, or improving documentation, your contributions are welcome!

## Getting Started

### Prerequisites

- **Go** (version 1.18 or later)
- **Git** for version control
- **Task** for running build tasks

### Setting Up the Development Environment

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/yourusername/hermes.git
   cd hermes
   ```
3. Add the original repository as an upstream remote:
   ```bash
   git remote add upstream https://github.com/originalowner/hermes.git
   ```
4. Install dependencies:
   ```bash
   go mod download
   ```
5. Install Task (if not already installed):
   ```bash
   go install github.com/go-task/task/v3/cmd/task@latest
   ```

## Development Workflow

### Creating a New Feature or Fix

1. Create a new branch for your feature or fix:

   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-fix-name
   ```

2. Make your changes, following the coding standards and guidelines

3. Run tests to ensure your changes don't break existing functionality:

   ```bash
   task test
   ```

4. Commit your changes with a descriptive commit message:

   ```bash
   git commit -m "Add feature: your feature description"
   ```

5. Push your branch to your fork:

   ```bash
   git push origin feature/your-feature-name
   ```

6. Create a pull request from your fork to the original repository

### Adding a New Importer

To add a new data source importer to Hermes, follow these steps:

1. Create a new directory under `cmd/` for your importer:

   ```bash
   mkdir -p cmd/yourimporter
   ```

2. Create the following files in your importer directory:

   - `cmd.go`: Command registration with Cobra
   - `parser.go`: Data parsing logic
   - `types.go`: Data models
   - `api.go` or similar: External API integration (if applicable)
   - `cache.go`: Caching implementation
   - `json.go`: JSON output formatter
   - `markdown.go`: Markdown output formatter
   - `testdata/`: Directory for test files

3. Register your importer command in `cmd/import.go`

4. Add documentation for your importer in `docs/importers/yourimporter.md`

5. Update the README.md to include your new importer in the list of supported sources

### Example: Minimal Importer Command

Here's a minimal example of a new importer command:

```go
// cmd/yourimporter/cmd.go
package yourimporter

import (
	"github.com/spf13/cobra"
	"github.com/yourusername/hermes/internal/cmdutil"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "yourimporter",
		Short: "Import data from YourSource",
		Long:  `Import and enrich data from YourSource exports.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Command implementation
			return nil
		},
	}

	// Add flags
	cmdutil.AddCommonFlags(cmd)
	cmd.Flags().String("csvfile", "", "Path to YourSource CSV export file")

	return cmd
}
```

## Coding Standards

### Go Style

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` or `goimports` to format your code
- Write idiomatic Go code
- Keep functions small and focused on a single responsibility
- Use meaningful variable and function names

### Error Handling

- Follow the error handling patterns described in [Logging & Error Handling](07_logging_error_handling.md)
- Return errors rather than panicking
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Log errors appropriately before returning them

### Logging

- Use the logrus library for logging
- Choose the appropriate log level for your messages
- Add context to log messages using fields
- See [Logging & Error Handling](07_logging_error_handling.md) for more details

### Testing

- Write unit tests for all new functionality
- Place tests in `_test.go` files within the same package
- Use table-driven tests where appropriate
- Create test fixtures in a `testdata/` directory
- Aim for high test coverage, especially for critical functionality

## Pull Request Process

1. Ensure your code passes all tests
2. Update documentation to reflect any changes
3. Add or update tests as necessary
4. Make sure your code follows the project's coding standards
5. Submit a pull request with a clear description of the changes
6. Address any feedback from code reviews

## Documentation

- Update documentation for any changes you make
- Follow the documentation structure described in [Documentation Rules](../docs/documentation-rules.md)
- Write clear, concise documentation with examples where appropriate
- Keep the README.md up to date with any significant changes

## Community Guidelines

- Be respectful and inclusive in all interactions
- Provide constructive feedback on pull requests
- Help others who are contributing to the project
- Report any issues or bugs you encounter

## License

By contributing to Hermes, you agree that your contributions will be licensed under the project's license.

## Questions?

If you have any questions about contributing, please open an issue or contact the project maintainers.

Thank you for contributing to Hermes!
