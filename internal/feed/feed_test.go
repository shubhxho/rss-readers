package feed

import (
	"testing"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/shubhxho/rss-readers/internal/config"
)

const rssSample = `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>First</title>
      <link>https://example.com/1</link>
      <description>hello</description>
      <pubDate>Mon, 02 Jan 2006 15:04:05 GMT</pubDate>
    </item>
    <item>
      <title>Second</title>
      <link>https://example.com/2</link>
    </item>
  </channel>
</rss>`

func TestParse(t *testing.T) {
	f := &Fetcher{parser: gofeed.NewParser()}
	fd := config.Feed{Name: "Test", Category: "Cat"}
	items, err := f.parse([]byte(rssSample), fd)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	if items[0].Title != "First" || items[0].FeedName != "Test" || items[0].Category != "Cat" {
		t.Fatalf("bad first item: %+v", items[0])
	}
	if items[0].Published.IsZero() {
		t.Fatal("first item should have a parsed date")
	}
}

func TestMergeSortsNewestFirst(t *testing.T) {
	now := time.Now()
	results := []Result{
		{Items: []Item{{Title: "old", Published: now.Add(-time.Hour)}}},
		{Items: []Item{{Title: "new", Published: now}}},
	}
	merged := Merge(results)
	if len(merged) != 2 || merged[0].Title != "new" {
		t.Fatalf("merge not newest-first: %+v", merged)
	}
}
