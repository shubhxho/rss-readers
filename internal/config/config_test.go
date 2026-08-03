package config

import "testing"

func newCfg(feeds ...Feed) *Config {
	c := &Config{Feeds: feeds}
	c.applyDefaults()
	return c
}

func TestNormalizeDedupAndSort(t *testing.T) {
	c := newCfg(
		Feed{Name: "B", URL: "https://b.com/feed", Category: "Tech"},
		Feed{Name: "A", URL: "https://a.com/feed", Category: "Tech"},
		Feed{Name: "dup", URL: "https://B.com/feed", Category: "Tech"}, // dup of B (case-insensitive)
		Feed{Name: "empty", URL: "   "},                                // dropped
	)
	c.Normalize()

	if len(c.Feeds) != 2 {
		t.Fatalf("want 2 feeds after dedup, got %d: %+v", len(c.Feeds), c.Feeds)
	}
	if c.Feeds[0].Name != "A" || c.Feeds[1].Name != "B" {
		t.Fatalf("want sorted A,B got %s,%s", c.Feeds[0].Name, c.Feeds[1].Name)
	}
}

func TestNormalizeBackfillsName(t *testing.T) {
	c := newCfg(Feed{URL: "https://www.example.com/rss"})
	c.Normalize()
	if c.Feeds[0].Name != "example.com" {
		t.Fatalf("want host-derived name example.com, got %q", c.Feeds[0].Name)
	}
}

func TestAddFeedDedup(t *testing.T) {
	c := newCfg()
	if !c.AddFeed(Feed{Name: "X", URL: "https://x.com/feed"}) {
		t.Fatal("first add should succeed")
	}
	if c.AddFeed(Feed{Name: "X2", URL: "https://x.com/feed"}) {
		t.Fatal("duplicate add should fail")
	}
	if len(c.Feeds) != 1 {
		t.Fatalf("want 1 feed, got %d", len(c.Feeds))
	}
}

func TestRemoveFeedByURLAndName(t *testing.T) {
	c := newCfg(
		Feed{Name: "Keep", URL: "https://keep.com/feed"},
		Feed{Name: "ByName", URL: "https://n.com/feed"},
		Feed{Name: "ByURL", URL: "https://u.com/feed"},
	)
	if n := c.RemoveFeed("ByName"); n != 1 {
		t.Fatalf("remove by name: want 1, got %d", n)
	}
	if n := c.RemoveFeed("https://u.com/feed"); n != 1 {
		t.Fatalf("remove by url: want 1, got %d", n)
	}
	if n := c.RemoveFeed("nope"); n != 0 {
		t.Fatalf("remove miss: want 0, got %d", n)
	}
	if len(c.Feeds) != 1 || c.Feeds[0].Name != "Keep" {
		t.Fatalf("unexpected remaining feeds: %+v", c.Feeds)
	}
}

func TestApplyDefaults(t *testing.T) {
	c := &Config{}
	c.applyDefaults()
	if c.RefreshMinutes <= 0 || c.CacheTTLMinutes <= 0 || c.Concurrency <= 0 {
		t.Fatalf("defaults not applied: %+v", c)
	}
}
