# Hermes

Hermes is a tool to own your data. It can parse exported data from different sources and collect them in a JSON, Obsidian flavoured markdown, or SQLite/remote Datasette format on your own computer.

Partially ✨Vibe coded✨ with Cursor, Claude Code and Cline

## Key Features

- Export to Markdown, JSON, and SQLite
- Optional remote export to a Datasette instance for querying and sharing

## Documentation

Comprehensive documentation is available in the `docs/` directory:

- [Project Overview](docs/01_overview.md)
- [Installation & Setup](docs/02_installation_setup.md)
- [Architecture](docs/03_architecture.md)
- [Configuration](docs/04_configuration.md)
- [Output Formats](docs/05_output_formats.md)
- [Caching](docs/06_caching.md)
- [Logging & Error Handling](docs/07_logging_error_handling.md)

### Importers

- [Goodreads Importer](docs/importers/goodreads.md)
- [IMDb Importer](docs/importers/imdb.md)
- [Letterboxd Importer](docs/importers/letterboxd.md)
- [Steam Importer](docs/importers/steam.md)

### Utilities

- [Command Utilities](docs/utilities/cmdutil.md)
- [Configuration Utilities](docs/utilities/config.md)
- [Error Utilities](docs/utilities/errors.md)
- [File Utilities](docs/utilities/fileutil.md)

## Sources

- ✅ Imdb "Your ratings" import
  - Data enriched from OMDB
- ✅ Letterboxd using [data export](https://letterboxd.com/user/exportdata/)
  - Data enriched from OMDB
- ✅Goodreads
  - Data enriched from openlibrary
- ✅ Steam
  - Uses Steam API to fetch list of games you own (BYO Steam API key)
  - Game data enriched via Steam API

## Other

Most API data is cached locally just to be a good API citizen

- Initial Steam import might take a while, you need to restart every few hours
- OMDB has a 1k/day limit, so bigger lists may take a few days to fully process
