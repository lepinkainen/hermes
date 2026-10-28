package steam

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lepinkainen/hermes/internal/httputil"
)

// The Steam storefront (store.steampowered.com) must see a browser UA, while
// the Steam Web API (api.steampowered.com) identifies as hermes.
func TestUserAgent_StorefrontUsesBrowserUA(t *testing.T) {
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.UserAgent()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"123":{"success":true,"data":{"name":"Game"}}}`))
	}))
	defer server.Close()

	if _, err := getGameDetailsWithBaseURL(123, server.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen != httputil.BrowserUserAgent {
		t.Fatalf("storefront UA = %q, want browser UA", seen)
	}
}

func TestUserAgent_WebAPIUsesHermesUA(t *testing.T) {
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.UserAgent()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":{"game_count":0,"games":[]}}`))
	}))
	defer server.Close()

	if _, err := importSteamGamesWithBaseURL("1", "key", server.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(seen, "hermes/") {
		t.Fatalf("web API UA = %q, want hermes UA", seen)
	}
	if strings.HasPrefix(seen, "Go-http-client") {
		t.Fatalf("default Go UA leaked: %q", seen)
	}
}
