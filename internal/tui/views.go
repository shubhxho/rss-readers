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
	hints := st.help.Render(strings.Join([]string{
		kv("enter", "read"),
		kv("tab", "feed"),
		kv("/", "search"),
		kv("o", "browser"),
		kv("r", "refresh"),
		kv("?", "help"),
		kv("q", "quit"),
	}, "  "))

	// Right-align a freshness stamp on the footer.
	footer := hints
	if !m.lastRefresh.IsZero() {
		stamp := st.dim.Render("updated " + humanize(m.lastRefresh))
		gap := m.width - lipgloss.Width(hints) - lipgloss.Width(stamp) - 2
		if gap > 1 {
			footer = hints + strings.Repeat(" ", gap) + stamp
		}
	}

	body := m.list.View()
	if len(m.allItems) == 0 {
		msg := "No articles yet."
		if m.failed > 0 {
			msg = fmt.Sprintf("All %d feeds failed — check your network or edit ~/.config/rss-readers/config.toml", m.failed)
		}
		body = lipgloss.Place(lipgloss.Width(body), m.height-2,
			lipgloss.Center, lipgloss.Center, st.dim.Render(msg))
	}
	if m.sidebarW > 0 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.renderSidebar(), body)
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, footer)
}

// renderSidebar draws the feed column with per-feed item counts, the active
// filter highlighted. "All" sits at the top as the reset target.
func (m model) renderSidebar() string {
	st := m.styles
	w := m.sidebarW - 3 // account for border + padding
	lines := []string{st.sidebarTitle.Render("Feeds")}

	total := len(m.allItems)
	lines = append(lines, m.sidebarRow("All", total, m.feedFilter < 0, w))
	for i, name := range m.feedNames {
		lines = append(lines, m.sidebarRow(name, m.feedCount[name], i == m.feedFilter, w))
	}

	col := strings.Join(lines, "\n")
	return st.sidebar.Height(m.height - 2).Render(col)
}

func (m model) sidebarRow(name string, count int, active bool, w int) string {
	st := m.styles
	cnt := fmt.Sprintf("%d", count)
	label := truncate(name, w-len(cnt)-1)
	pad := w - lipgloss.Width(label) - len(cnt)
	if pad < 1 {
		pad = 1
	}
	line := label + strings.Repeat(" ", pad) + cnt
	if active {
		return st.sidebarActive.Width(w).Render(line)
	}
	return st.sidebarItem.Render(label) + strings.Repeat(" ", pad) + st.sidebarCount.Render(cnt)
}

func (m model) readingView() string {
	st := m.styles
	pct := fmt.Sprintf("%3.0f%%", m.vp.ScrollPercent()*100)
	footer := st.help.Render(
		st.statusKey.Render("↑/↓")+" scroll  "+
			st.statusKey.Render("o")+" browser  "+
			st.statusKey.Render("esc")+" back  "+
			st.statusKey.Render("q")+" quit  ") +
		st.scrollPct.Render(pct)

	return lipgloss.JoinVertical(lipgloss.Left, m.vp.View(), footer)
}

// renderArticle builds the scrollable reader content for one item. Body text is
// capped to a comfortable reading measure and centered in the viewport rather
// than stretched edge-to-edge, which is far easier to read on wide terminals.
func (m model) renderArticle(it feed.Item) string {
	st := m.styles
	vw := m.vp.Width
	if vw <= 0 {
		vw = m.width - 2
	}
	// Reading measure: ~72–90 columns is the readable sweet spot.
	w := min(vw, 90)
	indent := (vw - w) / 2
	if indent < 0 {
		indent = 0
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

	link := st.readLink.Render(it.Link)

	raw := it.Content
	if strings.TrimSpace(raw) == "" {
		raw = it.Summary
	}
	body := htmlToText(raw)
	if body == "" {
		body = "(no content provided by this feed — press o to open in browser)"
	}
	bodyBlock := st.readBody.Width(w).Render(body)

	divider := st.dim.Render(strings.Repeat("─", min(w, 72)))

	block := lipgloss.JoinVertical(lipgloss.Left,
		title,
		metaLine,
		link,
		divider,
		"",
		bodyBlock,
	)
	if indent > 0 {
		block = lipgloss.NewStyle().MarginLeft(indent).Render(block)
	}
	return block
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
