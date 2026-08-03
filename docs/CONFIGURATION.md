# Configuration

`rss-readers` reads a single TOML file. On first run a commented default is
written for you.

- **Location:** `~/.config/rss-readers/config.toml`
- **Override:** set `$XDG_CONFIG_HOME` to move the whole `rss-readers/` directory.
- **Print the path:** `rss-readers config`

## Top-level keys

| key                 | type | default | meaning                                                        |
|---------------------|------|---------|----------------------------------------------------------------|
| `refresh_minutes`   | int  | `15`    | Background auto-refresh interval while the reader is open.      |
| `cache_ttl_minutes` | int  | `10`    | How long a cached body is served before a conditional refetch. |
| `concurrency`       | int  | `8`     | Maximum feeds fetched simultaneously.                          |

Any value ≤ 0 falls back to its default.

## Feeds

Each subscription is a `[[feeds]]` block:

```toml
[[feeds]]
name = "Hacker News"      # display name (optional — derived from the URL host if omitted)
url = "https://hnrss.org/frontpage"
category = "Tech"         # optional; groups the feed in the sidebar and OPML export
```

Feeds are **normalized** on load:

- surrounding whitespace is trimmed,
- entries with no `url` are dropped,
- a missing `name` is filled from the URL host,
- duplicate URLs are removed (case-insensitive, first wins),
- the list is sorted by `category`, then `name`.

## Managing feeds without editing the file

```sh
rss-readers add https://xkcd.com/rss.xml XKCD Comics
rss-readers rm  XKCD                 # by name or URL
rss-readers list                     # table by category
```

Edits made by `add`/`rm` rewrite the file as plain TOML (comments are not
preserved). Hand-edit the file directly if you want to keep comments.

## Caching

- **Config-independent.** The cache lives at `~/.cache/rss-readers/`
  (override with `$XDG_CACHE_HOME`).
- Three artifacts per feed: the raw body, HTTP validators (`ETag` /
  `Last-Modified`), and the **parsed items** as JSON.
- A warm start reuses the parsed-items JSON and skips XML parsing entirely, so
  launching against a warm cache is effectively instant.
- Delete the directory to reset; nothing else depends on it.
