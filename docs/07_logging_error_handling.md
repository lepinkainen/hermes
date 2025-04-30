# Logging & Error Handling

This document describes the logging and error handling approaches used in Hermes.

## Logging

Hermes uses the [logrus](https://github.com/sirupsen/logrus) library for structured logging. Logging is configured globally in `cmd/root.go` and used consistently throughout the application.

### Log Levels

Hermes uses the following log levels:

- **Debug**: Detailed information useful for debugging

  - API request/response details
  - Cache hits/misses
  - Detailed processing steps
  - Configuration values

- **Info**: General progress information

  - Import start/completion
  - Number of items processed
  - Output file locations
  - General progress updates

- **Warn**: Recoverable issues that don't stop processing

  - Skipping items due to missing data
  - Rate limiting delays
  - Non-critical API errors
  - Using fallback methods

- **Error**: Significant problems that may affect functionality
  - API authentication failures
  - File access errors
  - Parsing failures
  - Critical configuration issues

### Log Configuration

The log level can be configured in several ways:

1. **Configuration File**:

   ```yaml
   loglevel: "info" # debug, info, warn, error
   ```

2. **Command-Line Flag**:

   ```bash
   ./hermes --loglevel debug
   ```

3. **Environment Variable**:
   ```bash
   HERMES_LOGLEVEL=debug ./hermes
   ```

The `--verbose` flag is also available as a shorthand for `--loglevel debug`.

### Log Format

Logs are formatted as text by default, with each line containing:

- Timestamp
- Log level
- Message
- Additional fields (if any)

Example log output:

```
INFO[2023-04-30T16:30:45+03:00] Starting Goodreads import                   items=142
INFO[2023-04-30T16:30:46+03:00] Enriching data from OpenLibrary API          book="1984" author="George Orwell"
WARN[2023-04-30T16:30:47+03:00] Could not find cover image                   book="Obscure Title" isbn=""
INFO[2023-04-30T16:30:50+03:00] Import completed                             success=140 skipped=2
```

### Contextual Logging

Logrus supports adding context to log entries using fields:

```go
log.WithFields(log.Fields{
    "book":   book.Title,
    "author": book.Author,
    "isbn":   book.ISBN,
}).Info("Processing book")
```

This approach is used throughout Hermes to provide contextual information in log messages.

## Error Handling

Hermes follows standard Go error handling practices with some additional patterns for better error management.

### Error Types

Hermes uses several types of errors:

1. **Standard Errors**: Created using `errors.New()` or `fmt.Errorf()`
2. **Custom Error Types**: Defined in `internal/errors/` for specific error conditions
3. **Wrapped Errors**: Using `fmt.Errorf("context: %w", err)` to add context

### Custom Error Types

Custom error types are defined in the `internal/errors/` package:

- **RateLimitError**: Indicates an API rate limit has been reached

  ```go
  type RateLimitError struct {
      RetryAfter time.Duration
      Message    string
  }
  ```

- Other custom error types as needed for specific scenarios

### Error Handling Patterns

#### Error Propagation

Errors are typically returned up the call stack:

```go
func processItem(item Item) error {
    data, err := fetchData(item.ID)
    if err != nil {
        return fmt.Errorf("failed to fetch data for %s: %w", item.ID, err)
    }

    // Process data...

    return nil
}
```

#### Error Wrapping

Errors are wrapped with context to provide more information:

```go
if err != nil {
    return fmt.Errorf("failed to process item %s: %w", itemID, err)
}
```

This preserves the original error while adding context, making it easier to debug issues.

#### Error Logging

Significant errors are logged before being returned:

```go
if err != nil {
    log.WithFields(log.Fields{
        "item": item.ID,
        "error": err.Error(),
    }).Error("Failed to process item")
    return err
}
```

#### Error Recovery

For non-critical errors, Hermes attempts to recover and continue processing:

```go
for _, item := range items {
    err := processItem(item)
    if err != nil {
        log.WithFields(log.Fields{
            "item": item.ID,
            "error": err.Error(),
        }).Warn("Skipping item due to error")
        skipped++
        continue
    }
    processed++
}
```

### API Error Handling

API errors are handled based on their type:

- **Rate Limit Errors**: Trigger a delay and retry mechanism
- **Not Found Errors**: Log a warning and continue with default/fallback data
- **Authentication Errors**: Log an error and abort the operation
- **Network Errors**: Implement retries with exponential backoff

Example rate limit handling:

```go
resp, err := client.Get(url)
if err != nil {
    return nil, err
}

if resp.StatusCode == 429 {
    retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
    return nil, &errors.RateLimitError{
        RetryAfter: retryAfter,
        Message:    "API rate limit exceeded",
    }
}
```

### Command Error Handling

Cobra commands use the `RunE` function signature, which returns an error:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    // Command implementation
    if err := doSomething(); err != nil {
        return err
    }
    return nil
}
```

Cobra handles the error by:

1. Printing the error message to stderr
2. Setting a non-zero exit code
3. Optionally showing command usage information

### Error Reporting

For user-facing errors, Hermes aims to provide:

1. Clear error messages that explain what went wrong
2. Suggestions for how to fix the issue
3. References to documentation when appropriate

Example user-facing error message:

```
Error: Failed to authenticate with Steam API: invalid API key

Please check your API key in config.yaml or provide a valid key using the --apikey flag.
See docs/02_installation_setup.md for instructions on obtaining a Steam API key.
```

## Best Practices

When working with Hermes code, follow these best practices for logging and error handling:

1. **Use appropriate log levels** based on the information's importance
2. **Add context to logs** using logrus fields
3. **Wrap errors** with context using `fmt.Errorf("context: %w", err)`
4. **Handle recoverable errors** gracefully, allowing the program to continue
5. **Log errors** before returning them up the call stack
6. **Use custom error types** for specific error conditions that require special handling

## Next Steps

- See the importer-specific documentation for details on error handling in each importer
- See [Contributing](contributing.md) for guidelines on logging and error handling in contributions
