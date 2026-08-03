package tui

import (
	"testing"
	"time"

	"github.com/shubhxho/rss-readers/internal/cache"
	"github.com/shubhxho/rss-readers/internal/config"
	"github.com/shubhxho/rss-readers/internal/feed"
)

func newTestModel(t *testing.T) model {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	c, err := cache.Open()
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Feeds: []config.Feed{{Name: "A"}, {Name: "B"}}}
	cfg.RefreshMinutes = 15
	m := New(cfg, c)
	m.results = []feed.Result{
		{Feed: config.Feed{Name: "A"}, Items: []feed.Item{
			{FeedName: "A", Title: "a1", Published: time.Now()},
			{FeedName: "A", Title: "a2", Published: time.Now().Add(-time.Hour)},
		}},
		{Feed: config.Feed{Name: "B"}, Items: []feed.Item{
			{FeedName: "B", Title: "b1", Published: time.Now().Add(-2 * time.Hour)},
		}},
	}
	m.rebuildList()
	return m
}

func TestRebuildListBuildsIndexes(t *testing.T) {
	m := newTestModel(t)
	if len(m.allRows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(m.allRows))
	}
	if m.feedCount["A"] != 2 || m.feedCount["B"] != 1 {
		t.Fatalf("bad feed counts: %+v", m.feedCount)
	}
	if len(m.feedNames) != 2 || m.feedNames[0] != "A" || m.feedNames[1] != "B" {
		t.Fatalf("feed names not sorted: %+v", m.feedNames)
	}
	if len(m.byFeed["A"]) != 2 {
		t.Fatalf("byFeed index wrong for A: %d", len(m.byFeed["A"]))
	}
}

func TestApplyFeedFilterCounts(t *testing.T) {
	m := newTestModel(t)

	m.feedFilter = -1
	m.applyFeedFilter()
	if len(m.list.Items()) != 3 {
		t.Fatalf("all filter: want 3, got %d", len(m.list.Items()))
	}

	m.feedFilter = 0 // "A"
	m.applyFeedFilter()
	if len(m.list.Items()) != 2 {
		t.Fatalf("feed A filter: want 2, got %d", len(m.list.Items()))
	}

	m.feedFilter = 1 // "B"
	m.applyFeedFilter()
	if len(m.list.Items()) != 1 {
		t.Fatalf("feed B filter: want 1, got %d", len(m.list.Items()))
	}
}

func TestCycleFeedWraps(t *testing.T) {
	m := newTestModel(t)
	if m.feedFilter != -1 {
		t.Fatalf("initial filter should be -1 (all), got %d", m.feedFilter)
	}
	steps := []int{0, 1, -1, 0} // forward through A, B, back to all, then A
	for i, want := range steps {
		m.cycleFeed(+1)
		if m.feedFilter != want {
			t.Fatalf("cycle +1 step %d: want %d, got %d", i, want, m.feedFilter)
		}
	}
	// Reverse from A(0) wraps back to all(-1).
	m.cycleFeed(-1)
	if m.feedFilter != -1 {
		t.Fatalf("cycle -1 should reach all(-1), got %d", m.feedFilter)
	}
	m.cycleFeed(-1)
	if m.feedFilter != 1 {
		t.Fatalf("cycle -1 from all should wrap to last feed(1), got %d", m.feedFilter)
	}
}

func TestCycleFeedNoFeeds(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	c, _ := cache.Open()
	m := New(&config.Config{}, c)
	m.cycleFeed(+1) // must not panic or move off -1
	if m.feedFilter != -1 {
		t.Fatalf("no feeds: filter should stay -1, got %d", m.feedFilter)
	}
}

func TestHumanize(t *testing.T) {
	now := time.Now()
	cases := []struct {
		t    time.Time
		want string
	}{
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "5m ago"},
		{now.Add(-3 * time.Hour), "3h ago"},
		{now.Add(-2 * 24 * time.Hour), "2d ago"},
	}
	for _, c := range cases {
		if got := humanize(c.t); got != c.want {
			t.Errorf("humanize(%v) = %q, want %q", c.t, got, c.want)
		}
	}
}
