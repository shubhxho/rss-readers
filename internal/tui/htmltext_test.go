package tui

import (
	"strings"
	"testing"
)

func TestHTMLToText(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"<p>Hello <b>world</b></p>", "Hello world"},
		{"a<br>b", "a\nb"},
		{"&amp;&lt;&gt;", "&<>"},
		{"", ""},
	}
	for _, c := range cases {
		got := htmlToText(c.in)
		if !strings.Contains(got, strings.TrimSpace(c.want)) && got != c.want {
			t.Errorf("htmlToText(%q) = %q, want to contain %q", c.in, got, c.want)
		}
	}
}

func TestHTMLToTextList(t *testing.T) {
	got := htmlToText("<ul><li>one</li><li>two</li></ul>")
	if !strings.Contains(got, "• one") || !strings.Contains(got, "• two") {
		t.Fatalf("list not rendered with bullets: %q", got)
	}
	if strings.Contains(got, "\n\n\n") {
		t.Fatalf("excess blank lines: %q", got)
	}
}

func TestHTMLToTextStripsTags(t *testing.T) {
	got := htmlToText(`<div class="x"><a href="/y">link</a> text</div>`)
	if strings.Contains(got, "<") || strings.Contains(got, "href") {
		t.Fatalf("tags not stripped: %q", got)
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hello", 10) != "hello" {
		t.Fatal("short string should be unchanged")
	}
	got := truncate("hello world", 5)
	if len([]rune(got)) != 5 {
		t.Fatalf("truncate length wrong: %q", got)
	}
}
