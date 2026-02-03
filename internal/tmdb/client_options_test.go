package tmdb

import (
	"net/http"
	"testing"

	"github.com/lepinkainen/hermes/internal/ratelimit"
	"github.com/stretchr/testify/require"
)

func TestClientOptionsApply(t *testing.T) {
	customHTTP := &http.Client{}
	limiter := ratelimit.New("TMDB", 2)

	client := NewClient(
		"key",
		WithBaseURL("https://example.test/"),
		WithImageBaseURL("https://images.test/"),
		WithHTTPClient(customHTTP),
		WithRetryAttempts(5),
		WithRateLimiter(limiter),
	)

	require.Equal(t, "https://example.test", client.baseURL)
	require.Equal(t, "https://images.test", client.imageBaseURL)
	require.Equal(t, customHTTP, client.httpClient)
	require.Equal(t, 5, client.retryAttempts)
	require.Equal(t, limiter, client.rateLimiter)
}
