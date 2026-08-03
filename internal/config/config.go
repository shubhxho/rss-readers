// Package config loads and persists the RSS reader configuration.
//
// The config lives at ~/.config/rss-readers/config.toml. On first run a
// sensible default with a handful of well-known feeds is written so the app is
// usable out of the box.
package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

// Feed is a single subscription.
type Feed struct {
	Name     string `toml:"name"`
	URL      string `toml:"url"`
	Category string `toml:"category,omitempty"`
}

// Config is the full user configuration.
type Config struct {
	// RefreshMinutes controls the background auto-refresh interval.
	RefreshMinutes int `toml:"refresh_minutes"`
	// CacheTTLMinutes is how long a cached feed body is considered fresh
	// before a conditional revalidation is attempted.
	CacheTTLMinutes int `toml:"cache_ttl_minutes"`
	// Concurrency caps how many feeds are fetched at once.
	Concurrency int    `toml:"concurrency"`
	Feeds       []Feed `toml:"feeds"`

	// path is where this config was loaded from; used by Save.
	path string
	mu   sync.Mutex
}

// Dir returns the configuration directory. It honors XDG_CONFIG_HOME and
// otherwise pins to ~/.config/rss-readers on every platform (including macOS,
// where the Go default would be ~/Library/Application Support).
func Dir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "rss-readers"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "rss-readers"), nil
}

// Path returns the absolute path to config.toml.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// Load reads the config, writing a default file when none exists yet.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := defaultConfig()
		cfg.path = path
		// Write a hand-formatted, commented default so the file is inviting to
		// edit by hand — Save() (used later by add/remove) emits plain TOML.
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(defaultTOML), 0o644); err != nil {
			return nil, fmt.Errorf("writing default config: %w", err)
		}
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := &Config{}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	cfg.path = path
	cfg.applyDefaults()
	cfg.Normalize()
	return cfg, nil
}

// Normalize cleans the feed list in place: it trims whitespace, drops entries
// with no URL, backfills missing names from the URL host, and removes duplicate
// URLs (keeping the first occurrence). Feeds are then sorted by category, name.
func (c *Config) Normalize() {
	seen := make(map[string]struct{}, len(c.Feeds))
	cleaned := c.Feeds[:0]
	for _, f := range c.Feeds {
		f.Name = strings.TrimSpace(f.Name)
		f.URL = strings.TrimSpace(f.URL)
		f.Category = strings.TrimSpace(f.Category)
		if f.URL == "" {
			continue
		}
		lc := strings.ToLower(f.URL)
		if _, dup := seen[lc]; dup {
			continue
		}
		seen[lc] = struct{}{}
		if f.Name == "" {
			f.Name = hostName(f.URL)
		}
		cleaned = append(cleaned, f)
	}
	c.Feeds = cleaned
	sort.SliceStable(c.Feeds, func(i, j int) bool {
		if c.Feeds[i].Category != c.Feeds[j].Category {
			return c.Feeds[i].Category < c.Feeds[j].Category
		}
		return c.Feeds[i].Name < c.Feeds[j].Name
	})
}

// AddFeed inserts a feed, returning false if the URL is already subscribed.
func (c *Config) AddFeed(f Feed) bool {
	f.URL = strings.TrimSpace(f.URL)
	if f.URL == "" {
		return false
	}
	for _, e := range c.Feeds {
		if strings.EqualFold(e.URL, f.URL) {
			return false
		}
	}
	if strings.TrimSpace(f.Name) == "" {
		f.Name = hostName(f.URL)
	}
	c.Feeds = append(c.Feeds, f)
	c.Normalize()
	return true
}

// RemoveFeed drops feeds whose URL or name matches query (case-insensitive),
// returning how many were removed.
func (c *Config) RemoveFeed(query string) int {
	query = strings.TrimSpace(query)
	kept := c.Feeds[:0]
	removed := 0
	for _, f := range c.Feeds {
		if strings.EqualFold(f.URL, query) || strings.EqualFold(f.Name, query) {
			removed++
			continue
		}
		kept = append(kept, f)
	}
	c.Feeds = kept
	return removed
}

func hostName(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return strings.TrimPrefix(u.Host, "www.")
	}
	return raw
}

// Save writes the config back to disk atomically.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.path == "" {
		p, err := Path()
		if err != nil {
			return err
		}
		c.path = p
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}

	out, err := toml.Marshal(c)
	if err != nil {
		return err
	}

	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

func (c *Config) applyDefaults() {
	if c.RefreshMinutes <= 0 {
		c.RefreshMinutes = 15
	}
	if c.CacheTTLMinutes <= 0 {
		c.CacheTTLMinutes = 10
	}
	if c.Concurrency <= 0 {
		c.Concurrency = 8
	}
}

func defaultConfig() *Config {
	c := &Config{
		RefreshMinutes:  15,
		CacheTTLMinutes: 10,
		Concurrency:     8,
		Feeds: []Feed{
			{Name: "Hacker News", URL: "https://hnrss.org/frontpage", Category: "Tech"},
			{Name: "Lobsters", URL: "https://lobste.rs/rss", Category: "Tech"},
			{Name: "The Verge", URL: "https://www.theverge.com/rss/index.xml", Category: "Tech"},
			{Name: "Ars Technica", URL: "https://feeds.arstechnica.com/arstechnica/index", Category: "Tech"},
			{Name: "Go Blog", URL: "https://go.dev/blog/feed.atom", Category: "Go"},
			{Name: "Dave Cheney", URL: "https://dave.cheney.net/feed", Category: "Go"},
			{Name: "Julia Evans", URL: "https://jvns.ca/atom.xml", Category: "Programming"},
			{Name: "Simon Willison", URL: "https://simonwillison.net/atom/everything/", Category: "Programming"},
			{Name: "NASA Breaking News", URL: "https://www.nasa.gov/feed/", Category: "Science"},
		},
	}
	c.applyDefaults()
	c.Normalize()
	return c
}

// defaultTOML is the commented file written on first run. It must stay in sync
// with defaultConfig above.
const defaultTOML = `# rss-readers configuration
# Docs: https://github.com/shubhxho/rss-readers
#
# Edit this file to manage your subscriptions, then relaunch (or press 'r').
# You can also manage feeds from the CLI:
#   rss-readers add <url> [name] [category]
#   rss-readers rm  <url-or-name>
#   rss-readers list
#   rss-readers import feeds.opml
#   rss-readers export feeds.opml

# Background auto-refresh interval, in minutes.
refresh_minutes = 15

# How long a cached feed body is served before a conditional (ETag) revalidation.
cache_ttl_minutes = 10

# Maximum number of feeds fetched simultaneously.
concurrency = 8

# --- Feeds ------------------------------------------------------------------
# Each feed is a [[feeds]] block with a name, url, and optional category.

[[feeds]]
name = "Hacker News"
url = "https://hnrss.org/frontpage"
category = "Tech"

[[feeds]]
name = "Lobsters"
url = "https://lobste.rs/rss"
category = "Tech"

[[feeds]]
name = "The Verge"
url = "https://www.theverge.com/rss/index.xml"
category = "Tech"

[[feeds]]
name = "Ars Technica"
url = "https://feeds.arstechnica.com/arstechnica/index"
category = "Tech"

[[feeds]]
name = "Go Blog"
url = "https://go.dev/blog/feed.atom"
category = "Go"

[[feeds]]
name = "Dave Cheney"
url = "https://dave.cheney.net/feed"
category = "Go"

[[feeds]]
name = "Julia Evans"
url = "https://jvns.ca/atom.xml"
category = "Programming"

[[feeds]]
name = "Simon Willison"
url = "https://simonwillison.net/atom/everything/"
category = "Programming"

[[feeds]]
name = "NASA Breaking News"
url = "https://www.nasa.gov/feed/"
category = "Science"
`
