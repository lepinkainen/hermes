## Datasette Integration

Hermes writes import results to a local SQLite database so you can browse and query them with Datasette. Remote Datasette connections are no longer supported.

- Default database: `hermes.db`
- Enabled by default; disable with `--datasette=false` if you only want files
- Configure path via `--datasette-dbfile` or `datasette.dbfile` in `config.yml` (or `config.yaml`)
- Cache data lives in `cache.db` and is separate from `hermes.db`

### Running Datasette

1. Install Datasette (requires Python):
   ```sh
   pip install datasette
   ```
2. After running an import, serve your database:
   ```sh
   datasette serve hermes.db
   ```
   Replace `hermes.db` if you set a custom `--datasette-dbfile`.
3. Open the URL Datasette prints to explore your data.

### Sample Config

```yaml
datasette:
  enabled: true
  dbfile: "./hermes.db"
```


---

*Document created: 2025-11-19*
*Last reviewed: 2025-11-19*