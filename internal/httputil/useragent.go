// Package httputil provides the shared outbound HTTP plumbing for hermes,
// most importantly a consistent User-Agent on every request.
//
// Two identities are used, chosen per host:
//
//   - UserAgent() — "hermes/<version> (+https://github.com/lepinkainen/hermes)".
//     Used for documented APIs where a descriptive, contact-bearing UA is the
//     polite default and is how operators reach us about abuse: TMDB API,
//     OMDB, OpenLibrary (their API docs explicitly ask for a contact UA),
//     Google Books, BookBrainz (asks for a contact UA, like MusicBrainz),
//     Finna, ISBNdb, RAWG and the Steam Web API (api.steampowered.com).
//
//   - BrowserUserAgent — a mainstream desktop Chrome UA. Used for consumer web
//     endpoints that are not published APIs and are known to rate-limit,
//     degrade or block non-browser clients: Steam storefront endpoints
//     (store.steampowered.com/api/appdetails and /api/storesearch), the TMDB
//     image CDN (image.tmdb.org) and cover image downloads from arbitrary
//     hosts (OpenLibrary covers CDN, Steam CDN, etc.).
//
// Out of scope: go-rod browser automation drives a real Chrome and already
// sends a browser UA.
package httputil

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// BrowserUserAgent is a mainstream desktop Chrome User-Agent for consumer
// web endpoints that are picky about non-browser clients.
const BrowserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

const projectURL = "https://github.com/lepinkainen/hermes"

var (
	versionMu sync.RWMutex
	version   = "dev"
)

// SetVersion records the build version used in UserAgent(). It is called
// from cmd.SetVersionInfo with the value injected at link time.
func SetVersion(v string) {
	versionMu.Lock()
	defer versionMu.Unlock()
	if v == "" {
		v = "dev"
	}
	version = v
}

// UserAgent returns the hermes-identifying User-Agent string, e.g.
// "hermes/1.2.3 (+https://github.com/lepinkainen/hermes)".
func UserAgent() string {
	versionMu.RLock()
	defer versionMu.RUnlock()
	return fmt.Sprintf("hermes/%s (+%s)", version, projectURL)
}

// Transport is an http.RoundTripper that stamps a User-Agent header on every
// request that does not already carry one, then delegates to Base.
type Transport struct {
	// Base is the underlying RoundTripper. nil means http.DefaultTransport.
	Base http.RoundTripper
	// UserAgent yields the header value to stamp. nil means the hermes UA,
	// resolved at request time so SetVersion is honoured.
	UserAgent func() string
}

// NewTransport returns a Transport that stamps the hermes User-Agent.
func NewTransport(base http.RoundTripper) *Transport {
	return &Transport{Base: base, UserAgent: UserAgent}
}

// NewBrowserTransport returns a Transport that stamps BrowserUserAgent.
func NewBrowserTransport(base http.RoundTripper) *Transport {
	return &Transport{Base: base, UserAgent: func() string { return BrowserUserAgent }}
}

// RoundTrip implements http.RoundTripper. The incoming request is cloned
// before modification, as the RoundTripper contract requires. An explicit
// per-request User-Agent is left untouched.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	if req.Header.Get("User-Agent") != "" {
		return base.RoundTrip(req)
	}
	ua := UserAgent
	if t.UserAgent != nil {
		ua = t.UserAgent
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("User-Agent", ua())
	return base.RoundTrip(clone)
}

// NewClient returns an http.Client with the given timeout whose requests
// identify as hermes. A zero timeout means no timeout.
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: NewTransport(nil)}
}

// NewBrowserClient returns an http.Client with the given timeout whose
// requests present BrowserUserAgent. A zero timeout means no timeout.
func NewBrowserClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: NewBrowserTransport(nil)}
}
