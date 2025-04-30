# Steam Importer

This document describes the Steam importer in Hermes, which processes Steam library data and converts it to JSON and Markdown formats.

## Overview

The Steam importer fetches games from a user's Steam library using the Steam Web API and enriches them with additional metadata from the Steam Store API. It then generates structured JSON and Markdown files for each game in your library.

## Data Source

### Steam Web API

The Steam importer uses two primary data sources:

1. **Steam Web API** - To fetch the list of games owned by a user

   - Requires a Steam API key and the user's Steam ID
   - Provides basic information about each game (app ID, name, playtime)

2. **Steam Store API** - To fetch detailed information about each game
   - Provides rich metadata including descriptions, screenshots, developers, publishers, genres, etc.
   - Does not require authentication but has rate limits

### Data Enrichment

The importer enriches the basic Steam library data with additional information from the Steam Store API, including:

- Detailed descriptions
- Header images
- Screenshots
- Developers and publishers
- Release dates
- Categories and genres
- Metacritic scores (when available)

## Usage

### Command-Line Usage

```bash
./hermes steam --steamid your_steam_id --apikey your_steam_api_key
```

### Configuration

In your `config.yaml` file:

```yaml
steam:
  steamid: "your_steam_id"
  apikey: "your_steam_api_key"
  output:
    markdown: "./markdown/steam"
    json: "./json/steam.json"
```

### Command-Line Options

- `--steamid`, `-s`: Steam ID of the user (required if not in config)
- `--apikey`, `-k`: Steam API key (required if not in config)
- `--output-dir`: Directory for Markdown output (default: `./markdown/steam`)
- `--write-json`: Enable JSON output
- `--json-output`: Path for JSON output file (default: `./json/steam.json`)
- `--overwrite`: Overwrite existing files (default: false)

## Output Format

### Markdown Files

Each game is saved as a separate Markdown file with YAML frontmatter containing metadata. The filename is derived from the game title.

Example Markdown output:

```markdown
---
title: "Portal 2"
type: game
playtime: 720
duration: 12h 0m
release_date: "2011-04-18"
cover: "https://cdn.akamai.steamstatic.com/steam/apps/620/header.jpg"
developers:
  - "Valve"
publishers:
  - "Valve"
categories:
  - "Single-player"
  - "Co-op"
  - "Steam Achievements"
  - "Steam Workshop"
genres:
  - "Action"
  - "Adventure"
  - "Puzzle"
tags:
  - steam/game
metacritic_score: 95
metacritic_url: "https://www.metacritic.com/game/pc/portal-2"
---

# Portal 2

![](https://cdn.akamai.steamstatic.com/steam/apps/620/header.jpg)

> [!summary] Description
> Portal 2 draws from the award-winning formula of innovative gameplay, story, and music that earned the original Portal over 70 industry accolades and created a cult following.

> [!info] Game Details
>
> - **Playtime**: 720 minutes (12h 0m)
> - **Developers**: Valve
> - **Publishers**: Valve
> - **Release Date**: 2011-04-18
> - **Categories**: Single-player, Co-op, Steam Achievements, Steam Workshop
> - **Genres**: Action, Adventure, Puzzle
> - **Metacritic Score**: 95
> - **Metacritic URL**: [View on Metacritic](https://www.metacritic.com/game/pc/portal-2)

## Screenshots

![](https://cdn.akamai.steamstatic.com/steam/apps/620/ss_1.jpg)
![](https://cdn.akamai.steamstatic.com/steam/apps/620/ss_2.jpg)
```

### JSON Output

All games are saved in a single JSON file as an array of objects.

Example JSON output:

```json
[
  {
    "appid": 620,
    "name": "Portal 2",
    "playtime_forever": 720,
    "playtime_2weeks": 0,
    "last_played": "2023-04-15T00:00:00Z",
    "details_fetched": true,
    "detailed_description": "Portal 2 draws from the award-winning formula of innovative gameplay, story, and music that earned the original Portal over 70 industry accolades and created a cult following.",
    "short_description": "The highly anticipated sequel to 2007's Game of the Year, Portal 2 is a hilariously mind-bending adventure that challenges you to use wits over weaponry in a funhouse of diabolical science.",
    "header_image": "https://cdn.akamai.steamstatic.com/steam/apps/620/header.jpg",
    "screenshots": [
      {
        "id": 1,
        "path_full": "https://cdn.akamai.steamstatic.com/steam/apps/620/ss_1.jpg"
      },
      {
        "id": 2,
        "path_full": "https://cdn.akamai.steamstatic.com/steam/apps/620/ss_2.jpg"
      }
    ],
    "developers": ["Valve"],
    "publishers": ["Valve"],
    "release_date": {
      "coming_soon": false,
      "date": "2011-04-18"
    },
    "categories": [
      {
        "id": 2,
        "description": "Single-player"
      },
      {
        "id": 9,
        "description": "Co-op"
      }
    ],
    "genres": [
      {
        "id": "1",
        "description": "Action"
      },
      {
        "id": "25",
        "description": "Adventure"
      },
      {
        "id": "4",
        "description": "Puzzle"
      }
    ],
    "metacritic": {
      "score": 95,
      "url": "https://www.metacritic.com/game/pc/portal-2"
    }
  }
]
```

## Caching

The Steam importer implements caching for Steam Store API responses to:

1. Respect API rate limits
2. Improve performance for subsequent imports
3. Allow for offline processing of previously fetched data

Cache files are stored in:

- `cache/steam/`: Game data cached by app ID

The importer first checks the cache before making API requests. If the data is found in the cache, it uses that instead of making a new API request.

## Implementation Details

### Steam Web API Integration

The importer uses the Steam Web API to fetch the list of games owned by a user:

1. Makes a request to the `IPlayerService/GetOwnedGames` endpoint
2. Includes parameters for the Steam ID, API key, and additional options
3. Parses the JSON response to extract the list of games

### Steam Store API Integration

For each game in the user's library, the importer:

1. Checks if data is already cached
2. If not found, fetches detailed information from the Steam Store API
3. Extracts additional metadata (description, images, developers, publishers, etc.)
4. Caches the response for future use

The importer handles rate limiting by detecting when the Steam Store API limit is reached and stopping further requests with an appropriate error message.

### Output Generation

The importer generates:

1. One Markdown file per game, with a filename derived from the title
2. A single JSON file containing all games (if JSON output is enabled)

## Troubleshooting

### API Rate Limits

The Steam Store API has rate limits that may cause the importer to fail if you have a large library. If you hit a rate limit, the importer will detect this and stop with an error message. You can resume the import later, and it will continue from where it left off thanks to caching.

### Missing or Incorrect Data

If you notice missing or incorrect data in the output:

1. Check if the game is still available on the Steam Store (some games may be removed)
2. Verify that the Steam Store API has data for that game
3. Check if the game is region-locked, which might prevent accessing its data

### Finding Your Steam ID

To find your Steam ID:

1. Go to your Steam profile page
2. Look at the URL, which should be in the format: `https://steamcommunity.com/id/[custom_url]` or `https://steamcommunity.com/profiles/[steam_id]`
3. If you have a custom URL, you can use a Steam ID finder tool to convert it to a Steam ID

### Obtaining a Steam API Key

To use the Steam importer, you need a Steam API key:

1. Go to [Steam Web API Key](https://steamcommunity.com/dev/apikey)
2. Sign in with your Steam account
3. Enter a domain name (can be any domain you own, or localhost for personal use)
4. Agree to the terms and click "Register"
5. Add the key to your `config.yaml` file or use the `--apikey` flag

## Rate Limiting Considerations

The Steam Store API has rate limits that are not officially documented. If you encounter a "429 Too Many Requests" error, the importer will stop with a rate limit error message. The cache system helps mitigate this issue by reducing the number of API calls needed for subsequent imports.

Some tips for handling rate limits:

1. Run the importer in smaller batches if you have a large library
2. Wait a few minutes before retrying after hitting a rate limit
3. Ensure you're using the cache system effectively by not clearing the cache between runs
4. Consider running imports during off-peak hours when the Steam API might be less busy
