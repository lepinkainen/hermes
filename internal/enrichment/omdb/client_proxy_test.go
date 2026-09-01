package omdb

import (
	"net/http"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestResolveBaseURL(t *testing.T) {
	defer viper.Set("omdb.proxy_url", "")

	viper.Set("omdb.proxy_url", "")
	assert.Equal(t, omdbBaseURL, resolveBaseURL())

	viper.Set("omdb.proxy_url", "https://omdb-proxy.example.com/")
	assert.Equal(t, "https://omdb-proxy.example.com", resolveBaseURL())

	// Slash-only value trims to empty and must fall back to the default.
	viper.Set("omdb.proxy_url", "/")
	assert.Equal(t, omdbBaseURL, resolveBaseURL())

	// Surrounding whitespace is ignored.
	viper.Set("omdb.proxy_url", "  https://proxy.example.com  ")
	assert.Equal(t, "https://proxy.example.com", resolveBaseURL())
}

// SetTestClient must win over a configured proxy so tests never escape to a
// real endpoint.
func TestSetTestClientOverridesProxy(t *testing.T) {
	viper.Set("omdb.proxy_url", "http://configured-proxy.example.com")
	defer viper.Set("omdb.proxy_url", "")

	restore := SetTestClient("http://test-server.example", func(*http.Request) (*http.Response, error) {
		return nil, nil
	})
	assert.Equal(t, "http://test-server.example", resolveBaseURL())

	restore()
	assert.Equal(t, "http://configured-proxy.example.com", resolveBaseURL())
}
