# Error Utilities

This document describes the error utility functions in Hermes, which provide standardized error handling mechanisms for common scenarios.

## Overview

The `errors` package provides custom error types and utility functions for handling specific error conditions that may occur during the operation of Hermes. These utilities help standardize error handling across the application, making it easier to:

1. Identify specific types of errors
2. Handle errors appropriately based on their type
3. Provide consistent error messages to users
4. Implement retry logic for recoverable errors

Currently, the package focuses on API rate limiting errors, which are common when interacting with external services.

## Custom Error Types

### RateLimitError

The `RateLimitError` type represents a rate limit error from any API. This error occurs when an external API rejects requests due to exceeding the allowed request rate.

```go
type RateLimitError struct {
    Message string
}
```

The `RateLimitError` implements the standard Go `error` interface through its `Error()` method, which returns the error message.

```go
func (e *RateLimitError) Error() string {
    return e.Message
}
```

## Error Handling Patterns

### Type Assertion

To check if an error is a specific type, use type assertion:

```go
if err != nil {
    // Check if the error is a rate limit error
    if rateLimitErr, ok := err.(*errors.RateLimitError); ok {
        log.Warnf("Rate limit exceeded: %s", rateLimitErr.Error())
        // Implement retry logic with backoff
        time.Sleep(5 * time.Second)
        return doRequest() // Retry the request
    }

    // Handle other types of errors
    return err
}
```

### Error Wrapping

When propagating errors up the call stack, it's often useful to wrap them with additional context:

```go
import (
    "fmt"
    "github.com/lepinkainen/hermes/internal/errors"
)

func fetchGameDetails(gameID string) (*GameDetails, error) {
    resp, err := apiClient.Get(fmt.Sprintf("/games/%s", gameID))
    if err != nil {
        // Check for rate limit errors
        if resp != nil && resp.StatusCode == 429 {
            return nil, errors.NewRateLimitError(fmt.Sprintf("Rate limit exceeded for game ID %s", gameID))
        }
        return nil, fmt.Errorf("failed to fetch game details for %s: %w", gameID, err)
    }

    // Process response...
}
```

### Retry Logic

When encountering rate limit errors, it's common to implement retry logic with exponential backoff:

```go
func fetchWithRetry(url string, maxRetries int) (*http.Response, error) {
    var resp *http.Response
    var err error

    for i := 0; i < maxRetries; i++ {
        resp, err = http.Get(url)
        if err == nil {
            return resp, nil
        }

        // Check if it's a rate limit error
        if _, ok := err.(*errors.RateLimitError); ok {
            // Exponential backoff
            sleepTime := time.Duration(math.Pow(2, float64(i))) * time.Second
            log.Warnf("Rate limited, retrying in %v seconds", sleepTime.Seconds())
            time.Sleep(sleepTime)
            continue
        }

        // For other errors, don't retry
        return nil, err
    }

    return nil, fmt.Errorf("max retries exceeded: %w", err)
}
```

## Usage Examples

### Handling Rate Limit Errors in API Clients

```go
// In cmd/steam/steam.go
func (c *SteamClient) GetGameDetails(appID string) (*GameDetails, error) {
    url := fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=%s", appID)

    resp, err := c.httpClient.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to get game details: %w", err)
    }
    defer resp.Body.Close()

    // Check for rate limiting
    if resp.StatusCode == 429 {
        return nil, errors.NewRateLimitError(fmt.Sprintf("Steam API rate limit exceeded for app ID %s", appID))
    }

    // Process response...
}

// In the command handler (legacy Cobra pattern shown for reference)
func importSteamGames(cmd *cobra.Command, args []string) error {
    client := steam.NewClient()

    for _, appID := range appIDs {
        details, err := client.GetGameDetails(appID)
        if err != nil {
            // Check for rate limit errors
            if rateLimitErr, ok := err.(*errors.RateLimitError); ok {
                log.Warnf("Rate limit error: %s", rateLimitErr.Error())
                log.Info("Waiting 60 seconds before continuing...")
                time.Sleep(60 * time.Second)

                // Retry this app ID
                details, err = client.GetGameDetails(appID)
                if err != nil {
                    return fmt.Errorf("failed to get game details after retry: %w", err)
                }
            } else {
                // Handle other errors
                log.Warnf("Skipping app ID %s due to error: %v", appID, err)
                continue
            }
        }

        // Process game details...
    }

    return nil
}
```

### Creating Custom Error Types

If you need to create additional custom error types, follow the pattern established by `RateLimitError`:

```go
// Define the error type
type NotFoundError struct {
    Resource string
    ID       string
}

// Implement the error interface
func (e *NotFoundError) Error() string {
    return fmt.Sprintf("resource %s with ID %s not found", e.Resource, e.ID)
}

// Create a constructor function
func NewNotFoundError(resource, id string) *NotFoundError {
    return &NotFoundError{
        Resource: resource,
        ID:       id,
    }
}
```

## API Reference

### Types

| Type             | Description                                |
| ---------------- | ------------------------------------------ |
| `RateLimitError` | Represents a rate limit error from any API |

### Functions

| Function                                  | Description                                         |
| ----------------------------------------- | --------------------------------------------------- |
| `NewRateLimitError(message string) error` | Creates a new RateLimitError with the given message |

### Methods

| Method                        | Description                           |
| ----------------------------- | ------------------------------------- |
| `(e *RateLimitError) Error()` | Returns the error message as a string |


---

*Document created: 2025-07-20*
*Last reviewed: 2025-07-20*