# Hermes

Hermes is a tool own your data, it can parse exported data from different sources and collect them in a JSON or Obsidian flavoured markdown format on your own computer

## Sources

- ✅ Imdb "Your ratings" import
  - Data enriched from OMDB
- ✅Goodreads
  - Fetching covers (coming up)
- ✅ Steam
  - Uses Steam API to fetch list of games you own (BYO Steam API key)
  - Game data enriched via Steam API
- Letterboxd (as soon as their API opens up)

Most API data is cached locally just to be a good API citizen
  - Initial Steam import might take a while, you need to restart every few hours
  - OMDB has a 1k/day limit, so bigger lists may take a few days
