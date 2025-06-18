# Hermes Datasette Integration

Hermes can export your imported data to a SQLite database and optionally to a remote Datasette instance. This enables powerful querying, sharing, and visualization of your media data.

## Features

- Export to local SQLite database (`hermes.db` by default)
- Optional remote export to a Datasette instance with the `datasette-insert` plugin
- CLI and config file control for all Datasette options

## Configuration

### config.yaml

Add or edit the `datasette` section in your `config.yaml`:

```yaml
datasette:
  enabled: true
  mode: "local" # "local" or "remote"
  dbfile: "./hermes.db" # Path to SQLite file (for local mode)
  remote_url: "" # Remote Datasette URL (for remote mode)
  api_token: "" # API token for remote insert (for remote mode)
```

### CLI Flags

All options can be overridden with CLI flags:

- `--datasette` (enable)
- `--datasette-mode` (local/remote)
- `--datasette-dbfile` (SQLite file)
- `--datasette-url` (remote URL)
- `--datasette-token` (API token)

## Local Mode Usage

1. Set `mode: "local"` in config or use `--datasette-mode local`.
2. Run any importer (e.g. `hermes imdb import ...`).
3. The database file (default: `hermes.db`) will be created/updated.
4. To explore your data, install [Datasette](https://datasette.io/) and run:
   ```sh
   datasette serve hermes.db
   ```
5. Open the provided URL in your browser to query and browse your data.

## Remote Mode Usage

1. Set up a remote Datasette instance with the `datasette-insert` plugin.
2. Generate an API token for your user.
3. Set `mode: "remote"`, `remote_url`, and `api_token` in config or use CLI flags.
4. Run any importer. Data will be sent to the remote instance.

### Example Remote Setup

- See [datasette-insert plugin docs](https://github.com/simonw/datasette-insert) for setup and API details.

## Table Schemas

- `imdb_movies`: All fields from IMDb imports
- `goodreads_books`: All fields from Goodreads imports
- `letterboxd_movies`: All fields from Letterboxd imports
- `steam_games`: All fields from Steam imports

## Troubleshooting

- Ensure the SQLite file is writable (local mode)
- For remote mode, ensure the API token is valid and the remote instance is reachable
- Check logs for error messages

## More Information

- [Datasette documentation](https://docs.datasette.io/)
- [datasette-insert plugin](https://github.com/simonw/datasette-insert)
