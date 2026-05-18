# Test Suite

## Network policy

**No test that runs by default may connect to the public internet.** This is a hard rule.

Every test reachable via `task test` / `go test ./...` must either:

1. Use `httptest.NewServer` for HTTP-shaped dependencies, OR
2. Stub the HTTP indirection seam exposed by the production code (package var pattern), OR
3. Be guarded by a build tag so it does not run unless explicitly requested.

If a code path makes a real outbound request during the default suite, that is a regression — fix it by adding a seam, not by relying on the network being reachable.

## Why

- The default suite must pass offline, in CI, on a plane, behind a corporate proxy.
- External APIs (OMDB, Steam Store, OpenLibrary, archive.org, TMDB, Google Books, BookBrainz, ISBNdb) rate-limit, change schemas, or simply go down. Tests that depend on them are flaky by construction.
- A 30-second TCP timeout per offender turns a 5-second suite into a 5-minute suite.

## Categories

### Default ("always-on") tests

- Run by: `task test`, `go test ./...`
- May not touch the network.
- Use `httptest.Server` or seam-swapping (see "Seam pattern" below).

### Integration tests

- Marked: `//go:build integration` at top of file.
- Run by: `go test -tags=integration ./...`
- Currently: `cmd/goodreads/import_e2e_test.go`, `cmd/letterboxd/import_e2e_test.go`.
- These still must not call the live internet. They exercise full import flows end-to-end with file fixtures and seam-stubbed HTTP. Treat the tag as "slow / multi-component," not "live."

### CI-only tests

- Marked: `//go:build ci`
- Run by: `task test-ci` (`go test -tags=ci -cover ./...`)
- Reserved for tests that need a specific CI environment. None currently exist.

### Live tests

- **None exist by default in this repo.**
- If you need to add one (e.g. to verify the real OMDB schema hasn't changed), it must:
  - Live behind its own build tag: `//go:build live`.
  - Skip itself when its API key env var is missing: `if os.Getenv("OMDB_API_KEY") == "" { t.Skip(...) }`.
  - Document the invocation in this file.
- Invocation (when one exists): `go test -tags=live -run TestLive ./cmd/imdb/...`
- Existing `import_e2e_test.go` in `cmd/imdb/` has a `t.Skip("OMDB API key not configured...")` guard around one assertion block — this is fine because the skip path is the default. Do not add new tests that would silently start hitting the network when an env var is set.

## Seam pattern

Production functions that issue HTTP requests expose a package-level var so tests can swap the transport without mocking interfaces.

Canonical example — `cmd/imdb/omdb.go`:

```go
var (
    omdbBaseURL = "http://www.omdbapi.com"
    omdbHTTPGet = func(url string) (*http.Response, error) { return http.Get(url) }
    omdbHTTPDo  = func(req *http.Request) (*http.Response, error) {
        return http.DefaultClient.Do(req)
    }
)
```

Test setup (e.g. `cmd/imdb/omdb_test.go`):

```go
server := httptest.NewServer(http.HandlerFunc(...))
defer server.Close()

origBase := omdbBaseURL
origGet  := omdbHTTPGet
defer func() { omdbBaseURL = origBase; omdbHTTPGet = origGet }()
omdbBaseURL = server.URL
omdbHTTPGet = server.Client().Get
```

To discover existing seams: `grep -rn '^var (\|HTTPClient\|HTTPGet\|HTTPDo\|BaseURL\|WithBaseURL' --include='*.go' cmd internal`. Two styles are in use:

- Package vars (preferred when multiple functions share an endpoint): `cmd/imdb/omdb.go`, `cmd/letterboxd/omdb.go`, `internal/enrichment/omdb/client.go`, `internal/enrichment/steam_lookup.go`, `cmd/goodreads/googlebooks.go`, `cmd/goodreads/openlibrary.go`.
- `WithBaseURL` overloads (when one function has a hardcoded URL): `cmd/steam/steam.go`.
- Exported `Set…HTTPClient(c) (restore func())` swap helpers (when the var lives in `internal/` but tests sit in a different package): `internal/fileutil/cover.go`.

### Choosing between patterns

- **Single function with hardcoded URL**: extract a `…WithBaseURL` overload (Steam pattern).
- **Multiple functions sharing the same base URL**: use package vars (`omdbBaseURL` + `omdbHTTPDo`).
- **Shared HTTP client in an internal package other test packages need**: export a `Set…HTTPClient(c) (restore func())` swap helper rather than exporting the var directly.

## Cover image fakes

`cmd/goodreads/markdown_test.go` installs a `RoundTripper` that returns canned bytes for `covers.openlibrary.org` and 404 for everything else, so it can exercise both the download-success and URL-fallback branches deterministically. Use the same shape for any new test that needs the markdown writer to hit `fileutil.DownloadCover`.

```go
fileutil.SetCoverHTTPClient(&http.Client{Transport: fakeCoverRoundTripper{}})
```

## Adding a new HTTP-using importer

When you add a new data source:

1. Put the base URL in a package var, not a const.
2. Wrap the HTTP call in a package-var function (`xxxHTTPDo` / `xxxHTTPGet`) or accept a `*http.Client`.
3. Write the test against `httptest.NewServer`. Swap the seam in `t.Cleanup` / `defer`.
4. If the importer hits more than one host, give each host its own seam.
5. Do not call `t.Skip` based on connectivity. If the test cannot run offline, it does not belong in the default suite — gate it with a `//go:build live` tag instead.

## Running the suite

```bash
task test                          # default: no network, race-detected, with coverage
go test ./...                      # same, without coverage harness
go test -tags=integration ./...    # include integration_e2e tests
go test -tags=ci -cover ./...      # CI variant
go test -tags=live ./...           # only if a live test exists (none today)
```

## Verifying a test is offline

Quick sanity check before merging a new test:

```bash
# Block egress and run only the test you added
sudo pfctl -e 2>/dev/null   # macOS — or use any sandbox/network namespace tool
go test -run TestYourNewThing ./your/pkg/...
```

If it fails, the test has a hidden dependency on the network. Add a seam.
