package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/shubhxho/rss-readers/internal/feed"
)

func (m model) View() string {
	if m.width == 0 {
		return "\n  starting…"
	}
	if m.showHelp {
		return m.styles.app.Render(m.helpView())
	}
	switch m.state {
	case stateFetching:
		return m.styles.app.Render(m.fetchingView())
	case stateReading:
		return m.styles.app.Render(m.readingView())
	default:
		return m.listView()
	}
}

// helpView is a full-screen keybinding reference toggled with '?'.
func (m model) helpView() string {
	st := m.styles
	k := func(key, desc string) string {
		return "  " + st.statusKey.Render(fmt.Sprintf("%-10s", key)) + st.fetchLine.Render(desc)
	}
	lines := []string{
		st.title.Render(" keybindings "),
		"",
		st.readMeta.Render("Navigation"),
		k("↑/k ↓/j", "move up / down"),
		k("g / G", "jump to top / bottom"),
		k("enter", "read the selected article"),
		k("esc", "back / close"),
		"",
		st.readMeta.Render("Feeds"),
		k("tab", "next feed filter"),
		k("shift+tab", "previous feed filter"),
		k("/", "fuzzy search titles"),
		k("r", "refresh all feeds"),
		"",
		st.readMeta.Render("Actions"),
		k("o", "open article in browser"),
		k("?", "toggle this help"),
		k("q", "quit"),
		"",
		st.dim.Render("config: ~/.config/rss-readers/config.toml"),
	}
	box := st.fetchBox.Width(min(m.width-4, 60)).Render(strings.Join(lines, "\n"))
	return box
}

// fetchingView is the dedicated fetching page: a header, a live progress bar,
// the spinner, and a scrolling log of each feed as it resolves.
func (m model) fetchingView() string {
	st := m.styles

	header := st.title.Render(" rss-readers ") + "  " +
		st.dim.Render("fetching your feeds")

	bar := m.prog.View()
	counter := st.dim.Render(fmt.Sprintf(" %d/%d", m.done, m.tot))

	status := fmt.Sprintf("%s %s",
		st.spinner.Render(m.spinner.View()),
		st.fetchLine.Render(fmt.Sprintf("contacting %d sources…", m.tot)))

	lines := make([]string, 0, len(m.fetchLog))
	// Show the most recent results, newest at the bottom, capped to fit.
	logMax := m.height - 12
	if logMax < 3 {
		logMax = 3
	}
	start := 0
	if len(m.fetchLog) > logMax {
		start = len(m.fetchLog) - logMax
	}
	for _, r := range m.fetchLog[start:] {
		lines = append(lines, m.fetchResultLine(r))
	}

	box := st.fetchBox.Width(min(m.width-4, 76)).Render(strings.Join(lines, "\n"))

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		bar+counter,
		"",
		status,
		box,
	)
	return body
}

func (m model) fetchResultLine(r feed.Result) string {
	st := m.styles
	var badge string
	switch {
	case r.Err != nil:
		badge = st.fail.Render("✗ fail ")
	case r.FromCache:
		badge = st.cache.Render("◆ cache")
	default:
		badge = st.ok.Render("✓ live ")
	}

	name := lipgloss.NewStyle().Foreground(colText).Width(22).Render(truncate(r.Feed.Name, 22))
	detail := ""
	if r.Err != nil {
		detail = st.dim.Render(truncate(r.Err.Error(), 40))
	} else {
		detail = st.dim.Render(fmt.Sprintf("%d items · %s", len(r.Items), r.Duration.Round(time.Millisecond)))
	}
	return fmt.Sprintf("%s  %s  %s", badge, name, detail)
}

func (m model) listView() string {
	st := m.styles
	kv := func(k, v string) string { return st.statusKey.Render(k) + " " + v }
	help := st.help.Render(strings.Join([]string{
		kv("enter", "read"),
		kv("tab", "feed"),
		kv("/", "search"),
		kv("o", "browser"),
		kv("r", "refresh"),
		kv("?", "help"),
		kv("q", "quit"),
	}, "  "))
	return lipgloss.JoinVertical(lipgloss.Left, m.list.View(), help)
}

func (m model) readingView() string {
	st := m.styles
	pct := fmt.Sprintf("%3.0f%%", m.vp.ScrollPercent()*100)
	footer := st.help.Render(
		st.statusKey.Render("↑/↓") + " scroll  " +
			st.statusKey.Render("o") + " browser  " +
			st.statusKey.Render("esc") + " back  " +
			st.statusKey.Render("q") + " quit  ") +
		st.scrollPct.Render(pct)

	return lipgloss.JoinVertical(lipgloss.Left, m.vp.View(), footer)
}

// renderArticle builds the scrollable reader content for one item.
func (m model) renderArticle(it feed.Item) string {
	st := m.styles
	w := m.vp.Width
	if w <= 0 {
		w = m.width - 2
	}

	title := st.readTitle.Width(w).Render(it.Title)

	meta := []string{it.FeedName}
	if it.Author != "" {
		meta = append(meta, it.Author)
	}
	if !it.Published.IsZero() {
		meta = append(meta, it.Published.Format("Mon, 02 Jan 2006 15:04"))
	}
	metaLine := st.readMeta.Render(strings.Join(meta, "  ·  "))
	if it.Category != "" {
		metaLine += "  " + st.category.Render(it.Category)
	}

	link := st.dim.Render(it.Link)

	raw := it.Content
	if strings.TrimSpace(raw) == "" {
		raw = it.Summary
	}
	body := htmlToText(raw)
	if body == "" {
		body = "(no content provided by this feed — press o to open in browser)"
	}
	bodyBlock := st.readBody.Width(w).Render(body)

	divider := st.dim.Render(strings.Repeat("─", min(w, 60)))

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		metaLine,
		link,
		divider,
		"",
		bodyBlock,
	)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
