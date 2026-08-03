// Package opml reads and writes OPML 2.0 subscription lists, the interchange
// format every RSS reader speaks. It lets users import an existing feed list or
// export theirs to move between apps.
package opml

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/shubhxho/rss-readers/internal/config"
)

type opml struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    head     `xml:"head"`
	Body    body     `xml:"body"`
}

type head struct {
	Title string `xml:"title"`
}

type body struct {
	Outlines []outline `xml:"outline"`
}

type outline struct {
	Text     string    `xml:"text,attr"`
	Title    string    `xml:"title,attr,omitempty"`
	Type     string    `xml:"type,attr,omitempty"`
	XMLURL   string    `xml:"xmlUrl,attr,omitempty"`
	Category string    `xml:"category,attr,omitempty"`
	Outlines []outline `xml:"outline"`
}

// Parse extracts feeds from OPML bytes. Nested outlines (folders) are flattened,
// with the parent outline's text used as the feed category when none is set.
func Parse(data []byte) ([]config.Feed, error) {
	var doc opml
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing OPML: %w", err)
	}
	var feeds []config.Feed
	var walk func(outlines []outline, parent string)
	walk = func(outlines []outline, parent string) {
		for _, o := range outlines {
			if url := strings.TrimSpace(o.XMLURL); url != "" {
				name := firstNonEmpty(o.Title, o.Text, url)
				cat := firstNonEmpty(o.Category, parent)
				feeds = append(feeds, config.Feed{Name: name, URL: url, Category: cat})
			}
			if len(o.Outlines) > 0 {
				group := firstNonEmpty(o.Text, o.Title, parent)
				walk(o.Outlines, group)
			}
		}
	}
	walk(doc.Body.Outlines, "")
	return feeds, nil
}

// Marshal renders feeds as an OPML 2.0 document, grouping them into folders by
// category so the export round-trips cleanly into other readers.
func Marshal(title string, feeds []config.Feed) ([]byte, error) {
	byCat := map[string][]config.Feed{}
	var order []string
	for _, f := range feeds {
		cat := f.Category
		if cat == "" {
			cat = "Uncategorized"
		}
		if _, seen := byCat[cat]; !seen {
			order = append(order, cat)
		}
		byCat[cat] = append(byCat[cat], f)
	}

	doc := opml{Version: "2.0", Head: head{Title: title}}
	for _, cat := range order {
		folder := outline{Text: cat, Title: cat}
		for _, f := range byCat[cat] {
			folder.Outlines = append(folder.Outlines, outline{
				Text: f.Name, Title: f.Name, Type: "rss", XMLURL: f.URL, Category: f.Category,
			})
		}
		doc.Body.Outlines = append(doc.Body.Outlines, folder)
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
