# Plan for Automating Goodreads Export using Chromedp

## Problem Statement

The current Goodreads data export process is manual, requiring users to visit the Goodreads website, navigate to the export page, and manually download a CSV file. This issue (GitHub Issue #1) aims to automate this process within the Hermes CLI tool.

## Initial Research and Findings

1. **Goodreads API:** The official Goodreads API was shut down in 2020 and no longer issues new developer keys. Therefore, using an official API for data export is not feasible.
2. **Existing Solutions:** Research into existing Python libraries and tools revealed that web scraping is the most common and viable approach for automating Goodreads data export. Libraries like `Selenium` and `BeautifulSoup` are frequently used for this purpose.
3. **Hermes Context:** Hermes is a Go-based CLI tool. Introducing a Python dependency for this specific task is undesirable, as it would complicate the build and deployment process. The preferred approach is to implement the web scraping logic directly in Go.

## Chosen Technology: Chromedp

After researching Go-based browser automation libraries, `chromedp` has been selected as the most suitable tool.

* `chromedp` is a popular and well-maintained library for driving browsers that support the Chrome DevTools Protocol (e.g., Chrome, Edge, Safari).
* It allows for programmatic control of a headless browser, enabling the simulation of user interactions like logging in, navigating pages, clicking buttons, and downloading files.
* Its Go-native implementation aligns with the existing Hermes codebase, avoiding external language dependencies.

## Detailed Implementation Plan

### 1. Add `chromedp` Dependency

The `chromedp` library will be added to the project's `go.mod` file. This will involve running `go get github.com/chromedp/chromedp`.

### 2. Implement Scraping Logic in a New File

A new Go file, `cmd/goodreads/automation.go`, will be created. This file will encapsulate all the web scraping logic, keeping it separate from the existing `parser.go` and `cmd.go` files, thus maintaining a clean separation of concerns.

### 3. Create a New Command for Automated Export

A new command will be added to the Hermes CLI to trigger the automated Goodreads export. This command will likely be structured as:
`hermes import goodreads --automated`

This command will require the user's Goodreads login credentials (email and password). These will be passed as command-line flags, for example:
`hermes import goodreads --automated --email "your_email@example.com" --password "your_password"`
*Security Note:* Credentials should be handled securely. The implementation will need to consider best practices for handling sensitive information, potentially by reading from environment variables or a secure configuration file rather than directly from command-line arguments in a production scenario. For the initial implementation, command-line flags will be used for simplicity, with a note about security considerations.

### 4. Implement the Automation Flow within `automation.go`

The `automation.go` file will contain the following sequence of operations using `chromedp`:

* **Initialize Headless Browser:** Start a headless Chrome instance.
* **Navigate to Login Page:** Direct the browser to the Goodreads login page (e.g., `https://www.goodreads.com/user/sign_in`).
* **Fill Login Form:** Locate the email and password input fields using CSS selectors or XPath, and programmatically fill them with the provided user credentials.
* **Submit Login Form:** Locate the login button and simulate a click to submit the form.
* **Handle Potential Redirects/Authentication:** Wait for the page to load after login, handling any potential redirects or multi-factor authentication prompts (though the latter is unlikely for a simple CSV export).
* **Navigate to Export Page:** Direct the browser to the Goodreads data export page (e.g., `https://www.goodreads.com/review/import`).
* **Trigger Export:** Locate the "Export Library" button and simulate a click.
* **Wait for Download Link:** Continuously poll or wait for the download link for the CSV file to appear on the page. This might involve waiting for a specific element to become visible or for a certain text to appear.
* **Download CSV:** Once the download link is available, programmatically click it and capture the downloaded CSV file. The file will be saved to a temporary location on the local filesystem.

### 5. Connect to Existing Logic

After the CSV file is successfully downloaded to a temporary location, the path to this file will be passed to the existing `parser.go` logic within the `goodreads` importer. This ensures that the rest of the data processing (parsing, transformation, and output to Markdown/JSON/SQLite) remains consistent with the current manual import flow.

### 6. Add Unit and Integration Tests

Comprehensive tests will be written for the new automation logic.

* **Unit Tests:** Mock `chromedp` interactions to test the logic of navigating, filling forms, and triggering downloads without actual browser interaction.
* **Integration Tests:** (If feasible and within scope) Set up a controlled environment to run `chromedp` against a test Goodreads account or a local mock server to verify the end-to-end automation flow.

### 7. Update Documentation

The `docs/importers/goodreads.md` file will be updated to reflect the new automated export capability. This will include:

* Instructions on how to use the `--automated` flag.
* Guidance on providing Goodreads credentials securely.
* Any prerequisites or setup steps required for `chromedp` (e.g., ensuring Chrome is installed).

## Expected Outcome

Upon completion, users will be able to automate the Goodreads data export process directly from the Hermes CLI, eliminating the need for manual browser interaction and streamlining their data management workflow.
