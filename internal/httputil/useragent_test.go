package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newEchoServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Seen-UA", r.UserAgent())
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func captureUA(t *testing.T, client *http.Client, srv *httptest.Server, preset string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, http.NoBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if preset != "" {
		req.Header.Set("User-Agent", preset)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.Header.Get("X-Seen-UA")
}

func TestUserAgentFormat(t *testing.T) {
	SetVersion("1.2.3")
	t.Cleanup(func() { SetVersion("dev") })

	got := UserAgent()
	want := "hermes/1.2.3 (+https://github.com/lepinkainen/hermes)"
	if got != want {
		t.Fatalf("UserAgent() = %q, want %q", got, want)
	}

	SetVersion("")
	if !strings.HasPrefix(UserAgent(), "hermes/dev ") {
		t.Fatalf("empty version should fall back to dev, got %q", UserAgent())
	}
}

func TestNewClientStampsHermesUA(t *testing.T) {
	SetVersion("9.9.9")
	t.Cleanup(func() { SetVersion("dev") })
	srv := newEchoServer(t)

	got := captureUA(t, NewClient(5*time.Second), srv, "")
	if got != "hermes/9.9.9 (+https://github.com/lepinkainen/hermes)" {
		t.Fatalf("unexpected UA %q", got)
	}
}

func TestNewBrowserClientStampsBrowserUA(t *testing.T) {
	srv := newEchoServer(t)

	got := captureUA(t, NewBrowserClient(5*time.Second), srv, "")
	if got != BrowserUserAgent {
		t.Fatalf("UA = %q, want %q", got, BrowserUserAgent)
	}
	if !strings.HasPrefix(got, "Mozilla/5.0") {
		t.Fatalf("browser UA should look like a browser: %q", got)
	}
}

func TestTransportHonoursExplicitUA(t *testing.T) {
	srv := newEchoServer(t)

	// A hermes-UA client with an explicit per-request browser UA keeps it.
	got := captureUA(t, NewClient(5*time.Second), srv, BrowserUserAgent)
	if got != BrowserUserAgent {
		t.Fatalf("explicit UA was overridden: %q", got)
	}
}

func TestTransportDoesNotMutateOriginalRequest(t *testing.T) {
	srv := newEchoServer(t)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := NewTransport(nil).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if req.Header.Get("User-Agent") != "" {
		t.Fatalf("RoundTrip mutated caller's request headers: %q", req.Header.Get("User-Agent"))
	}
}

func TestZeroValueTransportDefaultsToHermes(t *testing.T) {
	srv := newEchoServer(t)
	client := &http.Client{Transport: &Transport{}}
	got := captureUA(t, client, srv, "")
	if !strings.HasPrefix(got, "hermes/") {
		t.Fatalf("zero-value Transport should default to hermes UA, got %q", got)
	}
}
