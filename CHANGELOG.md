# Changelog

All notable changes to this project are documented here.

## [0.2.0]

### Added
- **Feed sidebar** in the reader: every feed with its item count, the active
  filter highlighted. Collapses automatically on narrow terminals.
- **Parsed-item cache.** Alongside the raw body and HTTP validators, parsed
  items are stored as JSON. A warm start reuses them and skips XML parsing
  entirely — launching against a warm cache is effectively instant (~4ms vs a
  ~1.3s cold fetch).
- **O(1) feed filtering.** The reader precomputes flat and per-feed row indexes
  once, so `tab` switches feeds with a map lookup instead of a rescan.
- **OPML import from a URL**, not just a local file.
- **Help overlay** toggled with `?`.
- **Docs:** `docs/CONFIGURATION.md`, `docs/OPML.md`, and a sample
  `examples/feeds.opml`.
- **Tests** across config, cache, feed, opml, and the HTML-to-text renderer.

## [0.1.0]

### Added
- Charm Bubble Tea TUI: dedicated fetching page (live progress + per-feed
  status), aggregated newest-first article list, fuzzy search, scrollable
  reader, open-in-browser.
- Concurrent fetching with `errgroup` and a bounded semaphore.
- Two-tier cache (in-memory `sync.Map` over disk) with HTTP conditional
  revalidation (`ETag` / `If-Modified-Since`) and stale-cache fallback.
- TOML config in `~/.config/rss-readers` with a commented default.
- CLI: `add`, `rm`, `list`, `import`, `export`, `config`, `help`.
