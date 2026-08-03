package cache

import (
	"testing"
	"time"
)

func tempCache(t *testing.T) *Cache {
	t.Helper()
	return &Cache{dir: t.TempDir()}
}

func TestPutGetRoundTrip(t *testing.T) {
	c := tempCache(t)
	url := "https://example.com/feed"
	if err := c.Put(&Entry{URL: url, ETag: "abc", Body: []byte("hello")}); err != nil {
		t.Fatal(err)
	}
	got, ok := c.Get(url)
	if !ok {
		t.Fatal("entry not found after Put")
	}
	if string(got.Body) != "hello" || got.ETag != "abc" {
		t.Fatalf("bad entry: %+v", got)
	}
}

func TestGetFromDiskAfterMemEvict(t *testing.T) {
	c := tempCache(t)
	url := "https://example.com/feed"
	_ = c.Put(&Entry{URL: url, Body: []byte("disk")})
	c.mem.Delete(url) // force a disk read path
	got, ok := c.Get(url)
	if !ok || string(got.Body) != "disk" {
		t.Fatalf("disk fallback failed: %v %v", ok, got)
	}
}

func TestFreshness(t *testing.T) {
	e := &Entry{FetchedAt: time.Now()}
	if !e.Fresh(time.Minute) {
		t.Fatal("just-written entry should be fresh")
	}
	old := &Entry{FetchedAt: time.Now().Add(-2 * time.Hour)}
	if old.Fresh(time.Minute) {
		t.Fatal("2h-old entry should be stale")
	}
	if (*Entry)(nil).Fresh(time.Minute) {
		t.Fatal("nil entry must not be fresh")
	}
}

func TestBlobRoundTrip(t *testing.T) {
	c := tempCache(t)
	url := "https://example.com/feed"
	if err := c.PutBlob(url, "items.json", []byte(`[1,2,3]`)); err != nil {
		t.Fatal(err)
	}
	got, ok := c.GetBlob(url, "items.json")
	if !ok || string(got) != `[1,2,3]` {
		t.Fatalf("blob round-trip failed: %v %q", ok, got)
	}
	if _, ok := c.GetBlob(url, "missing"); ok {
		t.Fatal("missing blob should report not found")
	}
}
