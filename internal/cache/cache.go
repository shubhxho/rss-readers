// Package cache implements a two-tier feed cache: a hot in-memory layer backed
// by a persistent on-disk layer. It stores raw feed bodies together with the
// HTTP validators (ETag / Last-Modified) needed to perform cheap conditional
// revalidation, so unchanged feeds cost a 304 instead of a full download.
//
// The disk layout lives under ~/.cache/rss-readers/:
//
//	<sha1(url)>.body   raw feed bytes
//	<sha1(url)>.meta   JSON metadata (etag, last-modified, fetched-at)
package cache

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry is a cached feed body plus its revalidation metadata.
type Entry struct {
	URL          string    `json:"url"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	FetchedAt    time.Time `json:"fetched_at"`
	Body         []byte    `json:"-"`
}

// Fresh reports whether the entry is younger than ttl.
func (e *Entry) Fresh(ttl time.Duration) bool {
	if e == nil {
		return false
	}
	return time.Since(e.FetchedAt) < ttl
}

// Cache is safe for concurrent use by multiple goroutines.
type Cache struct {
	dir string
	mem sync.Map // url -> *Entry
}

// Dir returns the cache directory. It honors XDG_CACHE_HOME and otherwise pins
// to ~/.cache/rss-readers on every platform.
func Dir() (string, error) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "rss-readers"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "rss-readers"), nil
}

// Open returns a cache rooted at the standard cache directory, creating it.
func Open() (*Cache, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Cache{dir: dir}, nil
}

func key(url string) string {
	sum := sha1.Sum([]byte(url))
	return hex.EncodeToString(sum[:])
}

func (c *Cache) bodyPath(url string) string { return filepath.Join(c.dir, key(url)+".body") }
func (c *Cache) metaPath(url string) string { return filepath.Join(c.dir, key(url)+".meta") }

// Get returns a cached entry, consulting the in-memory layer first and falling
// back to disk. The returned bool is false when nothing is cached.
func (c *Cache) Get(url string) (*Entry, bool) {
	if v, ok := c.mem.Load(url); ok {
		return v.(*Entry), true
	}

	meta, err := os.ReadFile(c.metaPath(url))
	if err != nil {
		return nil, false
	}
	var e Entry
	if err := json.Unmarshal(meta, &e); err != nil {
		return nil, false
	}
	body, err := os.ReadFile(c.bodyPath(url))
	if err != nil {
		return nil, false
	}
	e.Body = body

	c.mem.Store(url, &e)
	return &e, true
}

// Put stores an entry in both cache tiers. Disk writes are atomic.
func (c *Cache) Put(e *Entry) error {
	e.FetchedAt = time.Now()
	c.mem.Store(e.URL, e)

	meta, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	if err := atomicWrite(c.metaPath(e.URL), meta); err != nil {
		return err
	}
	return atomicWrite(c.bodyPath(e.URL), e.Body)
}

// Touch refreshes an entry's FetchedAt without rewriting the body, used after a
// 304 Not Modified so the freshness window slides forward.
func (c *Cache) Touch(url string) {
	e, ok := c.Get(url)
	if !ok {
		return
	}
	e.FetchedAt = time.Now()
	_ = c.Put(e)
}

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
