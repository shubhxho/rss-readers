// Package feed fetches and parses RSS/Atom feeds concurrently, using the cache
// package for conditional HTTP revalidation. Fetching is fanned out with
// golang.org/x/sync/errgroup and bounded by a semaphore so a large feed list
// never overwhelms the network or the remote hosts.
package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync/atomic"
	"time"

	"github.com/mmcdole/gofeed"
	"golang.org/x/sync/errgroup"

	"github.com/shubhxho/rss-readers/internal/cache"
	"github.com/shubhxho/rss-readers/internal/config"
)

// itemsKind is the cache blob namespace for parsed items.
const itemsKind = "items.json"

// Item is a single article, flattened from the parsed feed.
type Item struct {
	FeedName  string
	Category  string
	Title     string
	Author    string
	Link      string
	Published time.Time
	Summary   string
	Content   string
}

// Result is the outcome of fetching one feed.
type Result struct {
	Feed      config.Feed
	Items     []Item
	FromCache bool
	Err       error
	Duration  time.Duration
}

// Fetcher fetches feeds with caching.
type Fetcher struct {
	client *http.Client
	cache  *cache.Cache
	ttl    time.Duration
	parser *gofeed.Parser
}

// New builds a Fetcher. ttl is how long a cached body stays fresh before a
// conditional revalidation request is made.
func New(c *cache.Cache, ttl time.Duration) *Fetcher {
	return &Fetcher{
		client: &http.Client{Timeout: 20 * time.Second},
		cache:  c,
		ttl:    ttl,
		parser: gofeed.NewParser(),
	}
}

// Progress is emitted as each feed finishes, for the fetching UI.
type Progress struct {
	Done  int
	Total int
	Last  Result
}

// FetchAll fetches every feed concurrently, bounded by cfg.Concurrency. The
// optional onDone callback fires once per completed feed (from a worker
// goroutine — it must be safe for concurrent use). Results come back sorted by
// feed name for stable display.
func (f *Fetcher) FetchAll(ctx context.Context, cfg *config.Config, onDone func(Progress)) []Result {
	results := make([]Result, len(cfg.Feeds))
	total := len(cfg.Feeds)
	var done int64

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(cfg.Concurrency)

	for i, fd := range cfg.Feeds {
		i, fd := i, fd // capture per iteration
		g.Go(func() error {
			start := time.Now()
			items, fromCache, err := f.fetchOne(ctx, fd)
			res := Result{
				Feed:      fd,
				Items:     items,
				FromCache: fromCache,
				Err:       err,
				Duration:  time.Since(start),
			}
			results[i] = res

			n := atomic.AddInt64(&done, 1)
			if onDone != nil {
				onDone(Progress{Done: int(n), Total: total, Last: res})
			}
			return nil // never abort siblings; per-feed errors are reported in Result
		})
	}
	_ = g.Wait()

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Feed.Name < results[j].Feed.Name
	})
	return results
}

// fetchOne performs a single conditional fetch + parse.
func (f *Fetcher) fetchOne(ctx context.Context, fd config.Feed) (items []Item, fromCache bool, err error) {
	cached, hasCached := f.cache.Get(fd.URL)

	// Serve straight from cache while still fresh — no network at all. Reuse the
	// parsed-items blob when present so we skip XML parsing entirely; a warm hit
	// costs a single JSON unmarshal instead of a full feed detect + parse.
	if hasCached && cached.Fresh(f.ttl) {
		if items, ok := f.cachedItems(fd.URL); ok {
			return items, true, nil
		}
		items, perr := f.parse(cached.Body, fd)
		if perr == nil {
			f.storeItems(fd.URL, items)
			return items, true, nil
		}
		// Corrupt cache: fall through to a network fetch.
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fd.URL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", "rss-readers/1.0 (+https://charm.sh)")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml;q=0.9, */*;q=0.8")
	if hasCached {
		if cached.ETag != "" {
			req.Header.Set("If-None-Match", cached.ETag)
		}
		if cached.LastModified != "" {
			req.Header.Set("If-Modified-Since", cached.LastModified)
		}
	}

	resp, err := f.client.Do(req)
	if err != nil {
		// Network failed — degrade gracefully to stale cache if we have it.
		if hasCached {
			if items, perr := f.parse(cached.Body, fd); perr == nil {
				return items, true, nil
			}
		}
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified && hasCached {
		f.cache.Touch(fd.URL)
		if items, ok := f.cachedItems(fd.URL); ok {
			return items, true, nil
		}
		items, perr := f.parse(cached.Body, fd)
		if perr == nil {
			f.storeItems(fd.URL, items)
		}
		return items, true, perr
	}

	if resp.StatusCode != http.StatusOK {
		if hasCached {
			if items, perr := f.parse(cached.Body, fd); perr == nil {
				return items, true, nil
			}
		}
		return nil, false, fmt.Errorf("%s: unexpected status %s", fd.URL, resp.Status)
	}

	body, err := readAllLimited(resp)
	if err != nil {
		return nil, false, err
	}

	_ = f.cache.Put(&cache.Entry{
		URL:          fd.URL,
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
		Body:         body,
	})

	items, err = f.parse(body, fd)
	if err == nil {
		f.storeItems(fd.URL, items)
	}
	return items, false, err
}

// cachedItems returns the parsed items blob for url when present.
func (f *Fetcher) cachedItems(url string) ([]Item, bool) {
	data, ok := f.cache.GetBlob(url, itemsKind)
	if !ok {
		return nil, false
	}
	var items []Item
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, false
	}
	return items, true
}

// storeItems persists parsed items so future warm reads skip XML parsing.
func (f *Fetcher) storeItems(url string, items []Item) {
	if data, err := json.Marshal(items); err == nil {
		_ = f.cache.PutBlob(url, itemsKind, data)
	}
}

func (f *Fetcher) parse(body []byte, fd config.Feed) ([]Item, error) {
	parsed, err := f.parser.ParseString(string(body))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fd.Name, err)
	}

	items := make([]Item, 0, len(parsed.Items))
	for _, it := range parsed.Items {
		published := time.Time{}
		if it.PublishedParsed != nil {
			published = *it.PublishedParsed
		} else if it.UpdatedParsed != nil {
			published = *it.UpdatedParsed
		}
		author := ""
		if it.Author != nil {
			author = it.Author.Name
		}
		items = append(items, Item{
			FeedName:  fd.Name,
			Category:  fd.Category,
			Title:     it.Title,
			Author:    author,
			Link:      it.Link,
			Published: published,
			Summary:   it.Description,
			Content:   it.Content,
		})
	}
	return items, nil
}

// Merge flattens and sorts items from many results newest-first.
func Merge(results []Result) []Item {
	var all []Item
	for _, r := range results {
		all = append(all, r.Items...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].Published.After(all[j].Published)
	})
	return all
}
