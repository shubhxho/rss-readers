// Package tui implements the Bubble Tea interface for the RSS reader: a
// concurrent fetching page, an aggregated article list, and a scrollable
// reader view.
package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/shubhxho/rss-readers/internal/cache"
	"github.com/shubhxho/rss-readers/internal/config"
	"github.com/shubhxho/rss-readers/internal/feed"
)

type state int

const (
	stateFetching state = iota
	stateList
	stateReading
)

// ---- messages -------------------------------------------------------------

type fetchEventMsg struct{ p feed.Progress }
type fetchDoneMsg struct{ results []feed.Result }
type refreshTickMsg struct{}

// fetchPipe carries streamed fetch events from worker goroutines to the UI.
// Held by pointer so it survives Bubble Tea's value-copy of the model.
type fetchPipe struct {
	events chan tea.Msg
}

// ---- list item ------------------------------------------------------------

type listItem struct{ it feed.Item }

func (l listItem) Title() string { return l.it.Title }
func (l listItem) Description() string {
	when := "—"
	if !l.it.Published.IsZero() {
		when = humanize(l.it.Published)
	}
	desc := fmt.Sprintf("%s · %s", l.it.FeedName, when)
	if l.it.Author != "" {
		desc += " · " + l.it.Author
	}
	return desc
}
func (l listItem) FilterValue() string { return l.it.Title + " " + l.it.FeedName }

// ---- model ----------------------------------------------------------------

type model struct {
	cfg     *config.Config
	fetcher *feed.Fetcher
	pipe    *fetchPipe

	state   state
	styles  styles
	spinner spinner.Model
	prog    progress.Model
	list    list.Model
	vp      viewport.Model

	width, height int
	sidebarW      int

	results   []feed.Result
	fetchLog  []feed.Result // completed feeds in finish order, for the fetch page
	done, tot int

	allItems   []feed.Item            // every merged item, newest-first
	allRows    []list.Item            // prebuilt list rows for "all feeds"
	byFeed     map[string][]list.Item // feed name -> prebuilt rows (O(1) filter)
	feedCount  map[string]int         // feed name -> item count, for the sidebar
	feedNames  []string               // unique feed names, for the tab filter
	feedFilter int                    // active feed filter index; -1 == all

	current  feed.Item // article being read
	err      error
	ready    bool
	showHelp bool
}

// New constructs the root model.
func New(cfg *config.Config, c *cache.Cache) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	pr := progress.New(
		progress.WithScaledGradient("#bb9af7", "#7dcfff"),
		progress.WithoutPercentage(),
	)

	st := newStyles()
	sp.Style = st.spinner

	de := list.NewDefaultDelegate()
	styleDelegate(&de, st)

	l := list.New(nil, de, 0, 0)
	l.Title = "Articles"
	l.SetShowStatusBar(true)
	l.SetShowHelp(false)
	l.Styles.Title = st.title
	l.StatusMessageLifetime = 3 * time.Second

	return model{
		cfg:        cfg,
		fetcher:    feed.New(c, time.Duration(cfg.CacheTTLMinutes)*time.Minute),
		pipe:       &fetchPipe{events: make(chan tea.Msg, len(cfg.Feeds)+4)},
		state:      stateFetching,
		styles:     st,
		spinner:    sp,
		prog:       pr,
		list:       l,
		tot:        len(cfg.Feeds),
		feedFilter: -1,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startFetch(),
		m.waitEvent(),
		tea.EnterAltScreen,
	)
}

// startFetch fans out the concurrent fetch and streams events into the pipe.
func (m model) startFetch() tea.Cmd {
	pipe := m.pipe
	cfg := m.cfg
	f := m.fetcher
	return func() tea.Msg {
		go func() {
			res := f.FetchAll(context.Background(), cfg, func(p feed.Progress) {
				pipe.events <- fetchEventMsg{p: p}
			})
			pipe.events <- fetchDoneMsg{results: res}
		}()
		return nil
	}
}

// waitEvent blocks for the next streamed fetch event.
func (m model) waitEvent() tea.Cmd {
	pipe := m.pipe
	return func() tea.Msg { return <-pipe.events }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case fetchEventMsg:
		m.done, m.tot = msg.p.Done, msg.p.Total
		m.fetchLog = append(m.fetchLog, msg.p.Last)
		var cmds []tea.Cmd
		if m.state == stateFetching {
			cmds = append(cmds, m.prog.SetPercent(float64(m.done)/float64(max(m.tot, 1))))
		}
		cmds = append(cmds, m.waitEvent()) // keep listening
		return m, tea.Batch(cmds...)

	case fetchDoneMsg:
		m.results = msg.results
		m.rebuildList()
		if m.state == stateFetching {
			m.state = stateList
			m.layout()
		}
		return m, m.scheduleRefresh()

	case refreshTickMsg:
		// Background auto-refresh: re-run the fetch, staying on the current view.
		m.fetchLog = m.fetchLog[:0]
		m.done = 0
		return m, tea.Batch(m.startFetch(), m.waitEvent())

	case progress.FrameMsg:
		pm, cmd := m.prog.Update(msg)
		m.prog = pm.(progress.Model)
		return m, cmd

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	// Delegate to the active child component.
	switch m.state {
	case stateList:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case stateReading:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While filtering in the list, let the list own all keys.
	if m.state == stateList && m.list.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Help):
		if m.state != stateFetching {
			m.showHelp = !m.showHelp
			return m, nil
		}

	case key.Matches(msg, keys.NextFeed):
		if m.state == stateList {
			m.cycleFeed(+1)
			return m, nil
		}

	case key.Matches(msg, keys.PrevFeed):
		if m.state == stateList {
			m.cycleFeed(-1)
			return m, nil
		}

	case key.Matches(msg, keys.Refresh):
		if m.state != stateFetching {
			m.fetchLog = m.fetchLog[:0]
			m.done = 0
			m.list.NewStatusMessage("refreshing…")
			return m, tea.Batch(m.startFetch(), m.waitEvent())
		}

	case key.Matches(msg, keys.Back):
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		if m.state == stateReading {
			m.state = stateList
			return m, nil
		}

	case key.Matches(msg, keys.Read):
		if m.state == stateList {
			if it, ok := m.list.SelectedItem().(listItem); ok {
				m.current = it.it
				m.state = stateReading
				m.layout()
				m.vp.SetContent(m.renderArticle(it.it))
				m.vp.GotoTop()
				return m, nil
			}
		}

	case key.Matches(msg, keys.Open):
		if m.state == stateList {
			if it, ok := m.list.SelectedItem().(listItem); ok {
				return m, openBrowser(it.it.Link)
			}
		} else if m.state == stateReading {
			return m, openBrowser(m.current.Link)
		}
	}

	// Pass through to the active component.
	switch m.state {
	case stateList:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case stateReading:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

// ---- layout & list --------------------------------------------------------

func (m *model) layout() {
	if m.width == 0 {
		return
	}
	h := m.width - 2
	inner := m.height - 4

	// Reserve a feed sidebar on wide terminals; collapse it when cramped.
	m.sidebarW = 0
	if m.width >= 84 {
		m.sidebarW = 24
	}
	m.prog.Width = min(60, h)
	m.list.SetSize(h-m.sidebarW, m.height-2)

	if !m.ready {
		m.vp = viewport.New(h, inner)
		m.ready = true
	} else {
		m.vp.Width = h
		m.vp.Height = inner
	}
	if m.state == stateReading && m.current.Title != "" {
		m.vp.SetContent(m.renderArticle(m.current))
	}
}

// rebuildList precomputes every derived view once — the flat "all" rows and a
// per-feed row index — so switching filters later is an O(1) map lookup rather
// than a rescan of the full item set.
func (m *model) rebuildList() {
	m.allItems = feed.Merge(m.results)

	m.allRows = make([]list.Item, 0, len(m.allItems))
	m.byFeed = make(map[string][]list.Item, len(m.results))
	m.feedCount = make(map[string]int, len(m.results))
	for _, it := range m.allItems {
		row := listItem{it: it}
		m.allRows = append(m.allRows, row)
		m.byFeed[it.FeedName] = append(m.byFeed[it.FeedName], row)
		m.feedCount[it.FeedName]++
	}

	names := make([]string, 0, len(m.byFeed))
	for name := range m.byFeed {
		names = append(names, name)
	}
	sort.Strings(names)
	m.feedNames = names
	if m.feedFilter >= len(names) {
		m.feedFilter = -1
	}

	ok, cached, failed := 0, 0, 0
	for _, r := range m.results {
		switch {
		case r.Err != nil:
			failed++
		case r.FromCache:
			cached++
		default:
			ok++
		}
	}
	m.applyFeedFilter()
	m.list.NewStatusMessage(fmt.Sprintf("%d fetched · %d cached · %d failed", ok, cached, failed))
}

// applyFeedFilter swaps in the prebuilt rows for the active scope. No scanning.
func (m *model) applyFeedFilter() {
	var rows []list.Item
	var scope string
	if m.feedFilter < 0 || m.feedFilter >= len(m.feedNames) {
		rows = m.allRows
		scope = fmt.Sprintf("All · %d feeds", len(m.feedNames))
	} else {
		name := m.feedNames[m.feedFilter]
		rows = m.byFeed[name]
		scope = fmt.Sprintf("%s [%d/%d]", name, m.feedFilter+1, len(m.feedNames))
	}
	m.list.SetItems(rows)
	m.list.Title = fmt.Sprintf("%s · %d items", scope, len(rows))
}

// cycleFeed advances the feed filter; delta of +1/-1 with wraparound through an
// "all feeds" sentinel state.
func (m *model) cycleFeed(delta int) {
	n := len(m.feedNames)
	if n == 0 {
		return
	}
	// States: -1 (all), 0..n-1. Map to 0..n then back.
	cur := m.feedFilter + 1 // 0==all
	cur = (cur + delta + (n + 1)) % (n + 1)
	m.feedFilter = cur - 1
	m.applyFeedFilter()
}

// ---- commands -------------------------------------------------------------

type errMsg struct{ err error }

func (m model) scheduleRefresh() tea.Cmd {
	d := time.Duration(m.cfg.RefreshMinutes) * time.Minute
	if d <= 0 {
		return nil
	}
	return tea.Tick(d, func(time.Time) tea.Msg { return refreshTickMsg{} })
}

func openBrowser(url string) tea.Cmd {
	return func() tea.Msg {
		if url == "" {
			return nil
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		if err := cmd.Start(); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}

// ---- helpers --------------------------------------------------------------

func humanize(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

func styleDelegate(d *list.DefaultDelegate, st styles) {
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(colPink).BorderForeground(colPink)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		Foreground(colPurple).BorderForeground(colPink)
	d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(colText)
	d.Styles.NormalDesc = d.Styles.NormalDesc.Foreground(colSubtle)
	d.Styles.DimmedTitle = d.Styles.DimmedTitle.Foreground(colMuted)
}
