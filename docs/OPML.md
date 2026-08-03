# OPML import & export

OPML is the portable subscription-list format every RSS reader speaks.
`rss-readers` reads and writes OPML 2.0, so you can move feeds in and out.

## Import

From a local file:

```sh
rss-readers import subscriptions.opml
```

From a URL (many readers publish your list at a stable link):

```sh
rss-readers import https://example.com/my-subscriptions.opml
```

Import **merges** into your existing config, skipping any URL you are already
subscribed to. Nested `<outline>` folders are flattened, and a folder's title
becomes the `category` of the feeds inside it when they don't set one.

Output tells you how many were new:

```
imported 12 new feed(s) from subscriptions.opml (40 in file)
```

## Export

To a file:

```sh
rss-readers export my-feeds.opml
```

To stdout (pipe it anywhere):

```sh
rss-readers export | pbcopy
```

Feeds are grouped into `<outline>` folders by `category`, so the file
round-trips cleanly into other readers (and back into this one).

## Example

A minimal OPML file lives at [`examples/feeds.opml`](../examples/feeds.opml):

```sh
rss-readers import examples/feeds.opml
```
