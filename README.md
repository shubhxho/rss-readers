# rss-readers

An aesthetic terminal RSS/Atom reader built with [Charm](https://charm.sh)'s
Bubble Tea, Bubbles, and Lip Gloss. Concurrent fetching, aggressive caching, and
a config you actually own.

![status](https://img.shields.io/badge/go-1.26-00ADD8) ![tui](https://img.shields.io/badge/tui-bubbletea-ff79c6)

## Features

- **Dedicated fetching page** — a live progress bar, spinner, and a per-feed log
  showing `live` / `cache` / `fail` as each source resolves.
- **Concurrent fetching** — every feed is pulled in parallel with
  `golang.org/x/sync/errgroup`, bounded by a configurable semaphore. One slow or
  broken feed never blocks the rest.
- **Two-tier caching** — a hot in-memory `sync.Map` layer over a persistent disk
  layer, with HTTP conditional revalidation (`ETag` / `If-Modified-Since`). Fresh
  feeds skip the network entirely; unchanged feeds cost a cheap `304`. Network
  failures gracefully fall back to stale cache.
- **Aggregated reading** — all articles merged and sorted newest-first, fuzzy
  filtering (`/`), a scrollable reader view, and `o` to open in your browser.
- **Config you own** — a plain TOML file at `~/.config/rss-readers/config.toml`.

## Install

```sh
go build -o rss-readers .
./rss-readers
```

## Usage

```
rss-readers                              launch the reader
rss-readers add <url> [name] [category]  subscribe to a feed
rss-readers rm <url-or-name>             unsubscribe
rss-readers list                         print subscriptions
rss-readers import <file.opml>           import an OPML subscription list
rss-readers export [file.opml]           export OPML (stdout if no file)
rss-readers config                       print the config file path
rss-readers help                         show help
```

| key       | action              | key         | action           |
|-----------|---------------------|-------------|------------------|
| `↑/k ↓/j` | move                | `enter`     | read article     |
| `tab`     | next feed filter    | `shift+tab` | previous feed    |
| `/`       | fuzzy search        | `o`         | open in browser  |
| `r`       | refresh             | `?`         | toggle help      |
| `esc`     | back                | `q`         | quit             |

### Managing feeds

Add, remove and list without touching the file:

```sh
rss-readers add https://xkcd.com/rss.xml XKCD Comics
rss-readers list
rss-readers rm XKCD
```

Move in and out of other readers with OPML:

```sh
rss-readers import subscriptions.opml   # merges, skipping duplicates
rss-readers export my-feeds.opml        # grouped into folders by category
```

Inside the reader, `tab` cycles the list through each feed (and back to *All*),
and `/` fuzzy-searches article titles.

## Config

First launch writes a default config with a handful of feeds. Edit it to manage
your subscriptions:

```toml
refresh_minutes = 15     # background auto-refresh interval
cache_ttl_minutes = 10   # how long a cached body is served without revalidation
concurrency = 8          # max simultaneous fetches

[[feeds]]
name = "Hacker News"
url = "https://hnrss.org/frontpage"
category = "Tech"
```

- Config: `~/.config/rss-readers/config.toml` (honors `$XDG_CONFIG_HOME`)
- Cache: `~/.cache/rss-readers/` (honors `$XDG_CACHE_HOME`)

## Layout

```
main.go                    entrypoint + CLI
internal/config            TOML config load/save, defaults
internal/cache             two-tier ETag/Last-Modified cache
internal/feed              concurrent fetch + gofeed parse
internal/tui               Bubble Tea model, views, styles
```
