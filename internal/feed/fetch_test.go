package feed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shubhxho/rss-readers/internal/cache"
	"github.com/shubhxho/rss-readers/internal/config"
)

// newTestFetcher builds a Fetcher backed by a throwaway on-disk cache.
func newTestFetcher(t *testing.T, ttl time.Duration) *Fetcher {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	c, err := cache.Open()
	if err != nil {
		t.Fatal(err)
	}
	return New(c, ttl)
}

// rssServer serves rssSample with an ETag and honors conditional requests.
func rssServer(hits *int32, etag string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		_, _ = w.Write([]byte(rssSample))
	}))
}

func TestFetchOneLiveThenServedFromCache(t *testing.T) {
	var hits int32
	srv := rssServer(&hits, "v1")
	defer srv.Close()

	f := newTestFetcher(t, time.Hour) // long TTL: second read must not touch network
	fd := config.Feed{Name: "T", URL: srv.URL}
	ctx := context.Background()

	items, fromCache, err := f.fetchOne(ctx, fd)
	if err != nil {
		t.Fatal(err)
	}
	if fromCache {
		t.Fatal("first fetch should be live, not cached")
	}
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}

	// Fresh within TTL — no HTTP request, served from the parsed-items cache.
	items2, fromCache2, err := f.fetchOne(ctx, fd)
	if err != nil {
		t.Fatal(err)
	}
	if !fromCache2 {
		t.Fatal("second fetch should be served from cache")
	}
	if len(items2) != 2 {
		t.Fatalf("cached read lost items: %d", len(items2))
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("want exactly 1 network hit, got %d", got)
	}
}

func TestConditionalRevalidation304(t *testing.T) {
	var hits int32
	srv := rssServer(&hits, "v1")
	defer srv.Close()

	f := newTestFetcher(t, 0) // TTL 0: always revalidate
	fd := config.Feed{Name: "T", URL: srv.URL}
	ctx := context.Background()

	if _, _, err := f.fetchOne(ctx, fd); err != nil {
		t.Fatal(err)
	}
	// Second call is stale, revalidates, receives 304, serves cached items.
	items, fromCache, err := f.fetchOne(ctx, fd)
	if err != nil {
		t.Fatal(err)
	}
	if !fromCache {
		t.Fatal("304 should be reported as a cache hit")
	}
	if len(items) != 2 {
		t.Fatalf("want 2 items after 304, got %d", len(items))
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("want 2 network hits (live + revalidate), got %d", got)
	}
}

func TestNon200FallsBackToStaleCache(t *testing.T) {
	var hits int32
	fail := int32(0)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if atomic.LoadInt32(&fail) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("ETag", "v1")
		_, _ = w.Write([]byte(rssSample))
	}))
	defer srv.Close()

	f := newTestFetcher(t, 0)
	fd := config.Feed{Name: "T", URL: srv.URL}
	ctx := context.Background()

	if _, _, err := f.fetchOne(ctx, fd); err != nil {
		t.Fatal(err)
	}
	atomic.StoreInt32(&fail, 1) // now the server 500s

	items, fromCache, err := f.fetchOne(ctx, fd)
	if err != nil {
		t.Fatalf("should fall back to cache on 500, got error: %v", err)
	}
	if !fromCache || len(items) != 2 {
		t.Fatalf("want cached items on 500, fromCache=%v items=%d", fromCache, len(items))
	}
}

func TestFetchAllIsolatesErrors(t *testing.T) {
	var hits int32
	good := rssServer(&hits, "v1")
	defer good.Close()

	bad := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	badURL := bad.URL
	bad.Close() // guarantee a connection error for this URL

	f := newTestFetcher(t, time.Hour)
	cfg := &config.Config{
		Concurrency: 4,
		Feeds: []config.Feed{
			{Name: "Good", URL: good.URL},
			{Name: "Bad", URL: badURL},
		},
	}

	results := f.FetchAll(context.Background(), cfg, nil)
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	byName := map[string]Result{}
	for _, r := range results {
		byName[r.Feed.Name] = r
	}
	if byName["Good"].Err != nil || len(byName["Good"].Items) != 2 {
		t.Fatalf("good feed should succeed: %+v", byName["Good"])
	}
	if byName["Bad"].Err == nil {
		t.Fatal("bad feed should report an error without aborting the good one")
	}
}

func TestFetchAllReportsProgress(t *testing.T) {
	var hits int32
	srv := rssServer(&hits, "v1")
	defer srv.Close()

	f := newTestFetcher(t, time.Hour)
	cfg := &config.Config{
		Concurrency: 2,
		Feeds: []config.Feed{
			{Name: "A", URL: srv.URL},
			{Name: "B", URL: srv.URL},
			{Name: "C", URL: srv.URL},
		},
	}

	var count int32
	last := int32(0)
	f.FetchAll(context.Background(), cfg, func(p Progress) {
		atomic.AddInt32(&count, 1)
		atomic.StoreInt32(&last, int32(p.Total))
		if p.Done < 1 || p.Done > p.Total {
			t.Errorf("bad progress Done=%d Total=%d", p.Done, p.Total)
		}
	})
	if atomic.LoadInt32(&count) != 3 {
		t.Fatalf("want 3 progress callbacks, got %d", count)
	}
	if atomic.LoadInt32(&last) != 3 {
		t.Fatalf("want Total=3, got %d", last)
	}
}
