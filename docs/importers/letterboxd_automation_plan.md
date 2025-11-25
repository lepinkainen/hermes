# Letterboxd Automated Export Plan

Purpose: design a headless/browser automation flow that mirrors the Goodreads exporter to fetch the Letterboxd data export zip and feed `watched.csv` into the importer. Reference dataset: `tmp/letterboxd-lepinkainen-2025-11-25-22-30-utc/` (contains `watched.csv`, `diary.csv`, `ratings.csv`, etc.).

## Manual Flow Baseline (Login → Export → Download)
1. Sign in: open `https://letterboxd.com/sign-in/`. The form uses `form.js-sign-in-form` posting to `/user/login.do`, with inputs `#field-username` (type=text, autocomplete=username) and `#field-password` (type=password). Both ship with `disabled` until page JS runs. Wait for `disabled` to clear (or remove it via JS) before sending credentials. Submit via the form’s primary button inside `.standalone-flow-form` (text: “Sign in”).
2. Confirm login: wait for navigation to complete and detect a logged-in indicator such as `body[data-person-role]` not equal to `guest`, or a profile menu element (e.g., `.navaccount` or `[data-track-action="profile"]`). Abort if redirected back to `/sign-in/`.
3. Navigate to export page: go directly to `https://letterboxd.com/settings/data/`. If not authenticated, this URL redirects back to sign-in—handle that by retrying step 1.
4. Request export: on the data page, look for an action labeled “Export your data” / “Request data export.” Likely selectors: a `button` or `input[type=submit]` containing “Export”, or a form targeting `/settings/data/` or `/data-export`. Click to start the export job; the page may show a status message like “Your data export is being prepared.”
5. Wait for ready download: the page should present a download link for a zip (pattern `letterboxd-<username>-YYYY-MM-DD-HH-MM-utc.zip`). If the link is not immediately available, reload/poll the page until an `a[href$=".zip"]` appears.
6. Download and inspect: save the zip, then unzip to a workspace directory (e.g., `tmp/letterboxd-<timestamp>/`). Primary target file is `watched.csv` with columns `Date`, `Name`, `Year`, `Letterboxd URI` (ID is the final path segment of the URI). Use this file for the importer pipeline.

## Automation Design (Chromedp, mirroring Goodreads flow)
- Inputs/config: add automation options under `letterboxd.automation` (email, password, download_dir, headful, timeout) plus `--automated` / `--headful` CLI flags paralleling Goodreads.
- Browser setup: reuse the Goodreads helpers (exec allocator, `browser.SetDownloadBehavior`, temp download dir) but update file patterns for `.zip`. Default headless; allow `--headful` for debugging selectors.
- Login routine:
  - Navigate to sign-in URL and wait for `#field-username` and `#field-password` to be present and enabled. If they stay disabled for >2s, run a JS snippet to remove `disabled` before `SendKeys`.
  - Fill username/email and password, click the submit button within `.js-sign-in-form`, and wait for a logged-in indicator (profile menu check or absence of `/sign-in/` in `Location`).
  - Consider a short `chromedp.Sleep` after submit to allow Cloudflare/recaptcha scripts to complete; surface a clear error if login fails.
- Export trigger:
  - Navigate to `https://letterboxd.com/settings/data/` after login.
  - Wait for an export button using a selector set such as `//button[contains(., 'Export')]`, `//input[@type='submit' and contains(@value, 'Export')]`, or a form action containing `data`/`export`.
  - Click the button; if the page responds with a flash message instead of a link, reload after a short delay to check for the generated download.
- Polling for download link:
  - Implement a `waitForExportLink` that evaluates DOM for `a[href$='.zip']` and captures the absolute URL. Also check any table/listing of previous exports in case a ready link already exists.
  - If no link appears, reload every few seconds (with jitter) until timeout; log HTML snippets when selectors fail (matching Goodreads diagnostics).
- Download handling:
  - Keep downloads in a temporary dir first; accept filenames matching `letterboxd-*.zip` without `.crdownload`.
  - Move the completed zip into `exports/` (e.g., `exports/letterboxd-export.zip` or timestamped filename) and return its path.
  - Unzip into `tmp/letterboxd-<timestamp>/` and point the importer at `<unzip>/watched.csv`. Preserve the original zip for repeatable runs.
- Import integration:
  - Wire `--automated` to run the automation step, then set `csvFile` to the downloaded `watched.csv` before calling existing parsing/enrichment.
  - Add a `--dry-run` option that executes automation only (no parsing) to debug selectors, matching the Goodreads UX.
  - Respect existing flags (`--skip-enrich`, `--overwrite`, `--json-output`) and reuse logging patterns from `cmd/goodreads/automation.go`.

## Data Notes From Sample Export
- Export folder name format seen: `letterboxd-<username>-YYYY-MM-DD-HH-MM-utc`.
- `watched.csv` headers: `Date`, `Name`, `Year`, `Letterboxd URI`; rows contain watched dates and films (e.g., `Captain Marvel`, `2019`, `https://boxd.it/9vSA`). Letterboxd ID derives from the URI’s final segment.
- Other useful files in the zip: `diary.csv`, `ratings.csv`, `reviews.csv`, `watchlist.csv`, `likes/films.csv`, `lists/*.csv`, `orphaned/*`, `deleted/*`. Keep unzip location stable so future features can consume them.

## Verification Checklist (post-implementation)
- Headless run downloads a `.zip` to the chosen directory without manual clicks; log shows successful login and export link discovery.
- Unzip produces `watched.csv` at the expected path and parsing succeeds on the sample export.
- `--headful` run allows selector debugging; `--dry-run` exits after download/unzip without invoking the importer.
- Add tests that mock chromedp runner functions (as in Goodreads automation tests) to cover login, export-link polling, and download file detection.
