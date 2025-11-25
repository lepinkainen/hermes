# Plan: Extract Goodreads Automation into a Standalone Module

## Goals
- Publish the Goodreads export automation as an independent Go module and small CLI suitable for its own GitHub project.
- Keep the public API stable for Hermes: `AutomateGoodreadsExport(ctx, opts)` with `AutomationOptions`.
- Preserve behaviour (chromedp-driven login/export/download) while clarifying configuration, logging, and platform expectations.

## Proposed Repository Layout
- `go.mod` with module path like `github.com/<org>/goodreads-automation`; depend on `chromedp` and standard library only.
- `pkg/automation`: core library (extracted logic from `cmd/goodreads/automation.go`), keeping helpers private.
- `cmd/goodreads-export`: thin CLI over the library; flags/envs for email, password, download dir, headless, timeout.
- `docs/`: usage, configuration, troubleshooting (Chromium install, cache/temp dirs, common chromedp errors).
- `examples/`: minimal library example invoking `AutomateGoodreadsExport`.
- CI: lint/test (e.g., `golangci-lint`, `go test ./...`), optional release workflow for binaries.
- LICENSE + README with quickstart, security notice about credentials, and browser requirements.

## API and Behaviour
- Maintain `AutomationOptions` fields: email, password, download dir, headless, timeout; optional logger hook (default noop or stdlib).
- Public entry point remains `AutomateGoodreadsExport(ctx, opts) (string, error)` returning the final CSV path.
- Keep chromedp allocator/context builders overridable for tests; expose injection points as functional options or test-only vars.
- Document defaults: headless on by default, timeout default (match Hermes `defaultAutomationTimeout` semantics), export filename `goodreads_library_export.csv`.
- Preserve download-move semantics: temp dir creation when unspecified, move/copy to target dir, reuse existing export link when present.

## CLI Deliverable
- Build as an independent executable (`goodreads-export`) so it can be installed and run without embedding in other apps; default behaviour saves the CSV to a local directory (current working directory or `--download-dir` if provided).
- Flags: `--email`, `--password`, `--download-dir`, `--timeout`, `--headless/--no-headless`.
- Env fallbacks: `GOODREADS_EMAIL`, `GOODREADS_PASSWORD`, `GOODREADS_DOWNLOAD_DIR`, `GOODREADS_TIMEOUT`.
- Output: log progress to stdout/stderr, print final CSV path on success; exit codes map errors cleanly.
- Provide Homebrew-friendly build or Go install instructions; release binaries for macOS/Linux (optional Windows).

## Testing Strategy
- Unit tests for directory prep/move, option defaults, selector-wait helpers (where deterministic).
- Integration test with a fake chromedp runner to simulate flows; gate real browser/E2E tests behind a build tag (e.g., `chromedp_e2e`) and environment guard for credentials.
- Add lint/static checks; ensure race detector passes where applicable.

## Migration Steps for Hermes
- Add dependency on the new module; replace internal `downloadGoodreadsCSV` assignment with imported function.
- Keep Hermes-specific timeout default (`defaultAutomationTimeout`) and flag wiring unchanged.
- Update `cmd/goodreads` tests to stub the imported function instead of package-level vars.
- Refresh Hermes docs to note the external module and how to update it when the API changes.

## Release and Maintenance
- Initial release v0.1.0 after CI green; tag and publish binaries if desired.
- Enable dependabot/renovate for `chromedp` updates; document manual verification steps when selectors/UI change.
- Provide contribution guidelines and a minimal changelog to track API adjustments.

## Open Questions
- Module name ownership: which GitHub org/user should host it?
- Minimum Go version target (likely Go 1.21 or repo standard).
- Do we need packaged secrets guidance (e.g., keychain, env files) or will callers supply creds at runtime only?
