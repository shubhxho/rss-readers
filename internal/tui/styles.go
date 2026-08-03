package tui

import "github.com/charmbracelet/lipgloss"

// Palette — a warm charm-flavored scheme that reads well on dark terminals.
var (
	colBg      = lipgloss.Color("#1a1b26")
	colSubtle  = lipgloss.Color("#565f89")
	colText    = lipgloss.Color("#c0caf5")
	colMuted   = lipgloss.Color("#787c99")
	colPink    = lipgloss.Color("#ff79c6")
	colPurple  = lipgloss.Color("#bb9af7")
	colCyan    = lipgloss.Color("#7dcfff")
	colGreen   = lipgloss.Color("#9ece6a")
	colYellow  = lipgloss.Color("#e0af68")
	colRed     = lipgloss.Color("#f7768e")
	colBlue    = lipgloss.Color("#7aa2f7")
)

type styles struct {
	app        lipgloss.Style
	title      lipgloss.Style
	titleBar   lipgloss.Style
	statusBar  lipgloss.Style
	statusKey  lipgloss.Style
	spinner    lipgloss.Style
	fetchBox   lipgloss.Style
	fetchLine  lipgloss.Style
	ok         lipgloss.Style
	fail       lipgloss.Style
	cache      lipgloss.Style
	dim        lipgloss.Style
	category   lipgloss.Style
	readTitle  lipgloss.Style
	readMeta   lipgloss.Style
	readBody   lipgloss.Style
	help       lipgloss.Style
	scrollPct  lipgloss.Style
}

func newStyles() styles {
	return styles{
		app: lipgloss.NewStyle().Padding(0, 1),
		title: lipgloss.NewStyle().
			Bold(true).Foreground(colBg).Background(colPurple).
			Padding(0, 1),
		titleBar: lipgloss.NewStyle().Padding(0, 0, 1, 0),
		statusBar: lipgloss.NewStyle().
			Foreground(colMuted).Padding(1, 0, 0, 0),
		statusKey: lipgloss.NewStyle().Foreground(colCyan).Bold(true),
		spinner:   lipgloss.NewStyle().Foreground(colPink),
		fetchBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colPurple).
			Padding(1, 2).MarginTop(1),
		fetchLine: lipgloss.NewStyle().Foreground(colText),
		ok:        lipgloss.NewStyle().Foreground(colGreen).Bold(true),
		fail:      lipgloss.NewStyle().Foreground(colRed).Bold(true),
		cache:     lipgloss.NewStyle().Foreground(colYellow),
		dim:       lipgloss.NewStyle().Foreground(colSubtle),
		category: lipgloss.NewStyle().
			Foreground(colBg).Background(colBlue).Padding(0, 1),
		readTitle: lipgloss.NewStyle().Bold(true).Foreground(colPurple).MarginBottom(1),
		readMeta:  lipgloss.NewStyle().Foreground(colCyan),
		readBody:  lipgloss.NewStyle().Foreground(colText),
		help:      lipgloss.NewStyle().Foreground(colSubtle),
		scrollPct: lipgloss.NewStyle().Foreground(colMuted),
	}
}
