package opml

import (
	"testing"

	"github.com/shubhxho/rss-readers/internal/config"
)

const sample = `<?xml version="1.0"?>
<opml version="2.0">
  <head><title>test</title></head>
  <body>
    <outline text="Tech">
      <outline text="Hacker News" type="rss" xmlUrl="https://hnrss.org/frontpage"/>
      <outline text="Lobsters" title="Lobsters" type="rss" xmlUrl="https://lobste.rs/rss"/>
    </outline>
    <outline text="Loose" type="rss" xmlUrl="https://example.com/feed"/>
  </body>
</opml>`

func TestParseFlattensAndInheritsCategory(t *testing.T) {
	feeds, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(feeds) != 3 {
		t.Fatalf("want 3 feeds, got %d", len(feeds))
	}
	// Feeds inside the Tech folder inherit its text as category.
	byName := map[string]config.Feed{}
	for _, f := range feeds {
		byName[f.Name] = f
	}
	if byName["Hacker News"].Category != "Tech" {
		t.Fatalf("want inherited category Tech, got %q", byName["Hacker News"].Category)
	}
	if byName["Hacker News"].URL != "https://hnrss.org/frontpage" {
		t.Fatalf("wrong url: %q", byName["Hacker News"].URL)
	}
}

func TestRoundTrip(t *testing.T) {
	in := []config.Feed{
		{Name: "A", URL: "https://a.com/feed", Category: "Tech"},
		{Name: "B", URL: "https://b.com/feed", Category: "Go"},
		{Name: "C", URL: "https://c.com/feed"}, // Uncategorized
	}
	data, err := Marshal("subs", in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(in) {
		t.Fatalf("round-trip lost feeds: want %d got %d", len(in), len(out))
	}
	got := map[string]string{}
	for _, f := range out {
		got[f.Name] = f.URL
	}
	for _, f := range in {
		if got[f.Name] != f.URL {
			t.Fatalf("round-trip url mismatch for %s: %q", f.Name, got[f.Name])
		}
	}
}
