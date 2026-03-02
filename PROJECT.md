# Hermes — Project Purpose

## What it is

Hermes is a personal data ownership tool. It imports exported data from external services (Goodreads, IMDb, Letterboxd, Steam) and converts it into portable, locally-owned formats: Obsidian-compatible Markdown, JSON, and SQLite/Datasette.

The core philosophy: your consumption history (books read, films watched, games played) should live on your own machine in open formats, not locked inside proprietary platforms.

## Current capabilities

- **Importers**: Goodreads, IMDb, Letterboxd, Steam — each parses source exports and enriches data via third-party APIs (OMDB, OpenLibrary, Steam API, TMDB)
- **Output formats**: Markdown with YAML frontmatter (Obsidian-ready), JSON, SQLite with optional remote Datasette sync
- **Enhance command**: Enriches existing Markdown notes with TMDB metadata without re-importing from source
- **Diff command**: Compares data across sources (e.g. IMDb vs Letterboxd watch history) - manual review required
- **Automated exports**: Browser automation (via Rod) for services that don't offer direct file exports (Goodreads, Letterboxd)
- **Caching**: All API responses cached locally in SQLite to respect rate limits and avoid redundant calls

## Direction

The project is expanding toward:

- More importers for additional services
- Richer enrichment (more metadata sources, better content generation)
- Better cross-source analysis and deduplication (the diff command is early work in this direction)
- Exporting/syncing data to the same platforms it imports from, giving users the ability own their data but still keep them in sync on different services
