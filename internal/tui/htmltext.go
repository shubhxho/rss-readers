package tui

import (
	"html"
	"regexp"
	"strings"
)

var (
	reTag      = regexp.MustCompile(`(?s)<[^>]*>`)
	reBlock    = regexp.MustCompile(`(?i)</(p|div|br|h[1-6]|tr)\s*>`)
	reBR       = regexp.MustCompile(`(?i)<br\s*/?>`)
	reListItem = regexp.MustCompile(`(?i)<li[^>]*>`)
	reSpaces   = regexp.MustCompile(`[ \t]{2,}`)
	reNewlines = regexp.MustCompile(`\n{3,}`)
)

// htmlToText converts feed HTML into readable plain text, preserving rough
// paragraph and list structure. It is deliberately dependency-free: feeds are
// messy but we only need something legible in a terminal, not a full renderer.
func htmlToText(s string) string {
	if s == "" {
		return ""
	}
	s = reBR.ReplaceAllString(s, "\n")
	s = reListItem.ReplaceAllString(s, "\n  • ")
	s = reBlock.ReplaceAllString(s, "\n")
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)

	s = reSpaces.ReplaceAllString(s, " ")
	s = reNewlines.ReplaceAllString(s, "\n\n")

	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		lines[i] = strings.TrimRight(ln, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
