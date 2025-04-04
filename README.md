# Hermes

Hermes is a tool own your data, it can parse exported data from different sources and collect them in a JSON or Obsidian flavoured markdown format on your own computer

Partially ✨Vibe coded✨ with Cursor, Claude Code and Cline

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
