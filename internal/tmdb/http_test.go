package tmdb

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

type testHTTPDoer struct {
	calls int
}

func (t *testHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	t.calls++
	if t.calls == 1 {
		return nil, &url.Error{Err: timeoutError{}}
	}

	body := io.NopCloser(strings.NewReader(`{"status":"ok"}`))
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}, nil
}

func TestGetJSONRetriesOnTimeout(t *testing.T) {
	client := NewClient("key", WithHTTPClient(&testHTTPDoer{}), WithRetryAttempts(2), WithRateLimiter(nil))

	var payload map[string]string
	err := client.getJSON(context.Background(), "http://example.test/", &payload)
	require.NoError(t, err)
	assert.Equal(t, "ok", payload["status"])
}

func TestIsRetryable(t *testing.T) {
	retryErr := &url.Error{Err: timeoutError{}}
	assert.True(t, isRetryable(retryErr))

	connErr := &url.Error{Err: errors.New("connection reset by peer")}
	assert.True(t, isRetryable(connErr))

	nonRetryErr := &url.Error{Err: errors.New("bad request")}
	assert.False(t, isRetryable(nonRetryErr))
}

func TestBackoffDelayCaps(t *testing.T) {
	assert.Equal(t, 1*time.Second, backoffDelay(1))
	assert.Equal(t, 2*time.Second, backoffDelay(2))
	assert.Equal(t, 10*time.Second, backoffDelay(5))
}

func TestDoJSONRequestStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("oops"))
	}))
	defer server.Close()

	client := NewClient("key", WithBaseURL(server.URL), WithHTTPClient(server.Client()), WithRateLimiter(nil))

	var payload map[string]any
	err := client.doJSONRequest(context.Background(), server.URL, &payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 500")
}
