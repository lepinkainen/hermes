# Hermes-Datasette Integration: Implementation Plan

This document outlines the steps required to integrate Datasette as an optional storage backend for the Hermes project. The existing functionality of exporting to local JSON and Markdown files will be preserved.

## Phase 1: Configuration and Setup

This phase involves setting up the necessary configuration options in both the `config.yaml` file and the command-line interface to control the new Datasette feature.

- [x] **Update `config.yaml` Structure:**
  - [x] Add a new top-level `datasette` section to the default `config.yaml`.
  - [x] This section should include the following keys:
    - `enabled`: `false` (boolean, enables/disables all Datasette functionality)
    - `mode`: `"local"` (string, can be `local` or `remote`)
    - `dbfile`: `"./hermes.db"` (string, path for the local SQLite database file)
    - `remote_url`: `""` (string, base URL for the remote Datasette instance, e.g., `https://my-datasette.com`)
    - `api_token`: `""` (string, bearer token for authenticating with the remote API)
- [x] **Update `cmd/root.go` for CLI Flags:**
  - [x] Add new persistent flags to the `rootCmd` to mirror the `config.yaml` settings. This allows overriding the config file from the command line.
  - [x] `--datasette`: A boolean flag to enable Datasette output (`enabled`).
  - [x] `--datasette-mode`: A string flag for the mode (`local` or `remote`).
  - [x] `--datasette-dbfile`: A string flag for the local database file path.
  - [x] `--datasette-url`: A string flag for the remote instance URL.
  - [x] `--datasette-token`: A string flag for the remote API token.
  - [x] Ensure Viper binds these new flags to the corresponding configuration keys.

## Phase 2: Core Database and API Logic

This phase focuses on creating the foundational packages for interacting with a local SQLite database and a remote Datasette API.

- [x] **Create a New Internal Package: `internal/datastore`**
  - [x] Create a new directory: `internal/datastore`.
  - [x] This package will house the logic for both local and remote data storage.
- [x] **Implement Local SQLite Logic (`internal/datastore/sqlite.go`)**
  - [x] Add a Go dependency for a SQLite driver (e.g., `go get modernc.org/sqlite`).
  - [x] Define a `Store` interface with methods like `Connect()`, `CreateTable(schema string)`, `BatchInsert(table string, records []map[string]any)`, and `Close()`.
  - [x] Create a `SQLiteStore` struct that implements the `Store` interface.
  - [x] The `Connect` method should open a connection to the SQLite file specified in the configuration.
  - [x] The `CreateTable` method should execute a `CREATE TABLE IF NOT EXISTS` SQL statement.
  - [x] The `BatchInsert` method should efficiently insert multiple records into a specified table using a single transaction for performance. It should handle converting slices of structs (like `[]imdb.MovieSeen`) into `[]map[string]any`.
- [x] **Implement Remote API Client (`internal/datastore/client.go`)**
  - [x] Create a `DatasetteClient` struct.
  - [x] The client should be initialized with the remote URL and API token from the configuration.
  - [x] Implement a `BatchInsert(database, table string, records []map[string]any)` method.
  - [x] This method should construct the correct API endpoint URL (e.g., `{remote_url}/-/insert/{database}/{table}`).
  - [x] It must create a JSON payload from the `records` slice. Refer to the `datasette-insert` plugin documentation for the exact payload structure.
  - [x] It must send the payload as an HTTP POST request.
  - [x] It must include the `Authorization: Bearer <token>` header in the request.
  - [x] It must handle HTTP status codes and API error responses gracefully.

## Phase 3: Integrate an Importer with Datasette Logic

This phase details how to modify an existing importer to use the new datastore functionality. We will use the **IMDb importer** as the primary example.

- [ ] **Modify the Importer's Main Function (`cmd/imdb/parser.go`)**
  - [ ] At the end of the `ParseImdb` function, after all movies have been processed and enriched, add a new section for Datasette output.
  - [ ] Add a check for `viper.GetBool("datasette.enabled")`.
- [ ] **Implement `local` Mode Integration**
  - [ ] Inside the `if datasette.enabled` block, check if `viper.GetString("datasette.mode")` is `local`.
  - [ ] If true:
    - [ ] Instantiate the `SQLiteStore` from `internal/datastore`.
    - [ ] Connect to the database file specified by `viper.GetString("datasette.dbfile")`.
    - [ ] Define the SQL schema for an `imdb_movies` table based on the fields in the `imdb.MovieSeen` struct. Ensure data types are correct (e.g., `TEXT`, `INTEGER`, `REAL`). Use the `ImdbId` as the `PRIMARY KEY`.
    - [ ] Call `store.CreateTable()` with the schema.
    - [ ] Convert the `[]MovieSeen` slice into a slice of `map[string]any`.
    - [ ] Call `store.BatchInsert("imdb_movies", records)`.
    - [ ] Close the database connection.
    - [ ] Add logging to indicate success or failure.
- [ ] **Implement `remote` Mode Integration**
  - [ ] Check if `viper.GetString("datasette.mode")` is `remote`.
  - [ ] If true:
    - [ ] Instantiate the `DatasetteClient` from `internal/datastore`.
    - [ ] Convert the `[]MovieSeen` slice into a slice of `map[string]any`.
    - [ ] Call `client.BatchInsert("hermes", "imdb_movies", records)`. The database name (`hermes`) is a convention here; it should match the remote setup.
    - [ ] Add logging to indicate success or failure of the API call.
- [ ] **Refactor Other Importers**
  - [ ] Once the pattern is established with the IMDb importer, apply the same logic to `goodreads`, `letterboxd`, and `steam` importers, creating tables like `goodreads_books`, `letterboxd_movies`, and `steam_games`respectively.

## Phase 4: Documentation

Update the project documentation to reflect the new feature.

- [ ] **Create New Documentation File (`docs/datasette_integration.md`)**
  - [ ] Write a new document explaining the feature.
  - [ ] Detail how to configure Hermes for both `local` and `remote` Datasette modes.
  - [ ] Provide instructions for the user on how to install and run Datasette locally to serve the generated `hermes.db` file.
  - [ ] Provide instructions for setting up a remote Datasette instance with the `datasette-insert` plugin and generating an API token.
- [ ] **Update Existing Documentation**
  - [ ] **`README.md`**: Add Datasette to the list of key features.
  - [ ] **`docs/02_installation_setup.md`**: Add a section on optional Datasette setup.
  - [ ] **`docs/03_architecture.md`**: Update the architecture diagram and description to include the new `datastore` component and the optional Datasette backend.
  - [ ] **`docs/04_configuration.md`**: Add details for the new `datasette` section in the `config.yaml` and the new CLI flags.

## Phase 5: Testing

Add tests to ensure the new functionality is reliable.

- [ ] **Test `internal/datastore/sqlite.go`**
  - [ ] Write unit tests for the `SQLiteStore`.
  - [ ] Use an in-memory SQLite database for testing to avoid creating files on disk (e.g., by using the `file::memory:?cache=shared` DSN).
  - [ ] Test table creation, single inserts, and batch inserts.
- [ ] **Test `internal/datastore/client.go`**
  - [ ] Write unit tests for the `DatasetteClient`.
  - [ ] Use a mock HTTP server (from `net/http/httptest`) to simulate the Datasette API.
  - [ ] Test successful batch inserts.
  - [ ] Test API error handling (e.g., for 403 Forbidden, 500 Internal Server Error).
