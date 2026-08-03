package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCreatesCommentedDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Feeds) == 0 {
		t.Fatal("default config should ship with feeds")
	}
	if cfg.RefreshMinutes <= 0 || cfg.Concurrency <= 0 {
		t.Fatalf("defaults not applied: %+v", cfg)
	}

	path := filepath.Join(dir, "rss-readers", "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("default file not written: %v", err)
	}
	if !strings.Contains(string(data), "#") {
		t.Fatal("default file should contain comments")
	}
}

func TestSaveReloadRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AddFeed(Feed{Name: "Custom", URL: "https://custom.example/feed", Category: "Z"}) {
		t.Fatal("add should succeed")
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	reloaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range reloaded.Feeds {
		if f.URL == "https://custom.example/feed" && f.Name == "Custom" {
			found = true
		}
	}
	if !found {
		t.Fatalf("added feed did not survive save/reload: %+v", reloaded.Feeds)
	}
}

func TestLoadNormalizesExistingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	// Hand-write a messy config: out of order, with a duplicate URL.
	body := `
refresh_minutes = 5

[[feeds]]
name = "Zeta"
url = "https://z.com/feed"

[[feeds]]
name = "Alpha"
url = "https://a.com/feed"

[[feeds]]
name = "ZetaDup"
url = "https://Z.com/feed"
`
	confDir := filepath.Join(dir, "rss-readers")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Feeds) != 2 {
		t.Fatalf("duplicate URL should be dropped, got %d feeds", len(cfg.Feeds))
	}
	if cfg.Feeds[0].Name != "Alpha" {
		t.Fatalf("feeds should be sorted by name, got first %q", cfg.Feeds[0].Name)
	}
	if cfg.RefreshMinutes != 5 {
		t.Fatalf("explicit value should be preserved, got %d", cfg.RefreshMinutes)
	}
	if cfg.CacheTTLMinutes <= 0 {
		t.Fatal("missing value should fall back to default")
	}
}

func TestDirHonorsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != "/custom/xdg/rss-readers" {
		t.Fatalf("want XDG-based dir, got %q", dir)
	}
}
