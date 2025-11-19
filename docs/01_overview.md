# Hermes Project Overview

## Purpose

Hermes is a tool designed to help users own their data by parsing exported data from different sources and collecting them in JSON or Obsidian-flavored Markdown format on their local computer. The project aims to give users control over their digital content consumption history across various platforms.

## Core Philosophy

The core philosophy of Hermes is data ownership and portability. In an era where our digital consumption is tracked across numerous platforms, Hermes allows users to:

1. **Consolidate personal data** from various services into a unified, locally-stored collection
2. **Enrich basic export data** with additional metadata from public APIs
3. **Format data consistently** for easy searching, analysis, and integration with tools like Obsidian
4. **Maintain privacy** by keeping all data local rather than in third-party services

## Workflow

The general workflow of Hermes follows these steps:

1. **Parse** - Import data from service-specific exports (CSV files, API responses)
2. **Enrich** - Enhance basic data with additional metadata from public APIs
3. **Format** - Convert the enriched data to standardized JSON and Markdown formats
4. **Store** - Save the formatted data to local directories

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│             │    │             │    │             │    │             │
│    Parse    │───►│   Enrich    │───►│   Format    │───►│    Store    │
│             │    │             │    │             │    │             │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

## Supported Data Sources

Hermes currently supports the following data sources:

- **IMDb** – Imports movie and TV ratings, optionally enriched with OMDB and TMDB metadata
- **Letterboxd** – Imports diary entries and ratings, with OMDB/TMDB enrichment and cover downloads
- **Goodreads** – Imports book ratings/reviews, enriched with OpenLibrary metadata and covers
- **Steam** – Imports your Steam library with store metadata and cover art
- **Enhance** – Re-processes existing Markdown notes to inject TMDB metadata/content when TMDB IDs are present or discoverable

Future planned importers include:

- Audible (audiobooks)
- Netflix (viewing history)
- Untappd (beer check-ins)

## Output Formats

Hermes generates two types of output:

1. **JSON** - Structured data format suitable for programmatic access and further processing
2. **Markdown** - Human-readable format with Obsidian-compatible frontmatter, suitable for knowledge management systems

## Technical Foundation

Hermes is built with:

- **Go** - For performance, simplicity, and cross-platform compatibility
- **Kong** - For command-line interface and configuration management
- **Viper** - For layered configuration files, environment variables, and overrides
- **Public APIs** - For data enrichment (TMDB, OMDB, OpenLibrary, Steam)
- **Local caching** - To respect API rate limits and improve performance

## Getting Started

See the [Installation & Setup](02_installation_setup.md) guide to get started with Hermes.
