// Command rss-readers is an aesthetic terminal RSS reader built on Charm's
// Bubble Tea. It fetches feeds concurrently, caches aggressively with HTTP
// conditional revalidation, and reads its subscriptions from a TOML config in
// your XDG config directory.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shubhxho/rss-readers/internal/cache"
	"github.com/shubhxho/rss-readers/internal/config"
	"github.com/shubhxho/rss-readers/internal/opml"
	"github.com/shubhxho/rss-readers/internal/tui"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "config", "path":
			cmdPath()
			return
		case "list", "ls":
			cmdList()
			return
		case "add":
			cmdAdd(args[1:])
			return
		case "rm", "remove":
			cmdRemove(args[1:])
			return
		case "import":
			cmdImport(args[1:])
			return
		case "export":
			cmdExport(args[1:])
			return
		case "-h", "--help", "help":
			printHelp()
			return
		default:
			die(fmt.Errorf("unknown command %q — run 'rss-readers help'", args[0]))
		}
	}
	runTUI()
}

func runTUI() {
	cfg := mustLoad()
	if len(cfg.Feeds) == 0 {
		die(fmt.Errorf("no feeds configured — add one with 'rss-readers add <url>'"))
	}
	c, err := cache.Open()
	if err != nil {
		die(fmt.Errorf("opening cache: %w", err))
	}
	p := tea.NewProgram(tui.New(cfg, c), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		die(err)
	}
}

func cmdPath() {
	p, err := config.Path()
	if err != nil {
		die(err)
	}
	// Ensure the file exists so `config` is a reliable "where do I edit" answer.
	mustLoad()
	fmt.Println(p)
}

func cmdList() {
	cfg := mustLoad()
	if len(cfg.Feeds) == 0 {
		fmt.Println("no feeds configured")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "CATEGORY\tNAME\tURL")
	for _, f := range cfg.Feeds {
		cat := f.Category
		if cat == "" {
			cat = "—"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", cat, f.Name, f.URL)
	}
	w.Flush()
	fmt.Printf("\n%d feeds\n", len(cfg.Feeds))
}

func cmdAdd(args []string) {
	if len(args) == 0 {
		die(fmt.Errorf("usage: rss-readers add <url> [name] [category]"))
	}
	cfg := mustLoad()
	f := config.Feed{URL: args[0]}
	if len(args) > 1 {
		f.Name = args[1]
	}
	if len(args) > 2 {
		f.Category = args[2]
	}
	if !cfg.AddFeed(f) {
		die(fmt.Errorf("already subscribed to %s", f.URL))
	}
	mustSave(cfg)
	fmt.Printf("added %s\n", f.URL)
}

func cmdRemove(args []string) {
	if len(args) == 0 {
		die(fmt.Errorf("usage: rss-readers rm <url-or-name>"))
	}
	cfg := mustLoad()
	n := cfg.RemoveFeed(args[0])
	if n == 0 {
		die(fmt.Errorf("no feed matched %q", args[0]))
	}
	mustSave(cfg)
	fmt.Printf("removed %d feed(s)\n", n)
}

func cmdImport(args []string) {
	if len(args) == 0 {
		die(fmt.Errorf("usage: rss-readers import <file.opml>"))
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		die(err)
	}
	feeds, err := opml.Parse(data)
	if err != nil {
		die(err)
	}
	cfg := mustLoad()
	added := 0
	for _, f := range feeds {
		if cfg.AddFeed(f) {
			added++
		}
	}
	mustSave(cfg)
	fmt.Printf("imported %d new feed(s) from %s (%d in file)\n", added, filepath.Base(args[0]), len(feeds))
}

func cmdExport(args []string) {
	cfg := mustLoad()
	out, err := opml.Marshal("rss-readers subscriptions", cfg.Feeds)
	if err != nil {
		die(err)
	}
	if len(args) == 0 {
		os.Stdout.Write(out)
		return
	}
	if err := os.WriteFile(args[0], out, 0o644); err != nil {
		die(err)
	}
	fmt.Printf("exported %d feeds to %s\n", len(cfg.Feeds), args[0])
}

func mustLoad() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		die(fmt.Errorf("loading config: %w", err))
	}
	return cfg
}

func mustSave(cfg *config.Config) {
	if err := cfg.Save(); err != nil {
		die(fmt.Errorf("saving config: %w", err))
	}
}

func printHelp() {
	fmt.Println(`rss-readers — a Charm-powered terminal RSS reader

usage:
  rss-readers                       launch the reader
  rss-readers add <url> [name] [category]
  rss-readers rm <url-or-name>      unsubscribe
  rss-readers list                  print subscriptions
  rss-readers import <file.opml>    import an OPML subscription list
  rss-readers export [file.opml]    export OPML (stdout if no file)
  rss-readers config                print the config file path
  rss-readers help                  show this help

keys (in the reader):
  ↑/k ↓/j   move            enter   read article
  tab       filter by feed  o       open in browser
  /         search          r       refresh
  ?         toggle help     esc     back
  q         quit

config: ~/.config/rss-readers/config.toml   (honors $XDG_CONFIG_HOME)`)
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "rss-readers:", err)
	os.Exit(1)
}
