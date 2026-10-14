package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/gitstore"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

func setupWatchHome(t *testing.T, enabled bool) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default(home)
	cfg.Tracking.Paths = []string{"~/.config/app"}
	if enabled {
		cfg.Watcher.Enabled = true
	}
	if err := config.Save(config.DefaultPath(home), cfg); err != nil {
		t.Fatal(err)
	}
	if err := gitstore.Init(config.Expand(cfg.Repository.Path, home)); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(home, ".config", "app"), 0o755)
	os.WriteFile(filepath.Join(home, ".config", "app", "a.conf"), []byte("a\n"), 0o644)
	if _, err := snapshot.Run(home, snapshot.Options{Message: "base"}); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestWatchOnceExitsZero(t *testing.T) {
	setupWatchHome(t, false)
	root := New()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"watch", "--once"})
	if err := root.Execute(); err != nil {
		t.Fatalf("watch --once: %v", err)
	}
}

func TestWatchRequiresEnabled(t *testing.T) {
	setupWatchHome(t, false)
	root := New()
	root.SetArgs([]string{"watch"})
	err := root.Execute()
	if err == nil {
		t.Fatalf("disabled watcher must refuse long-run")
	}
	if ee, ok := err.(*ExitError); !ok || ee.Code != 1 {
		t.Fatalf("want exit 1, got %v", err)
	}
}

func TestStatusShowsWatcher(t *testing.T) {
	setupWatchHome(t, true)
	root := New()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"status"})
	_ = root.Execute() // may exit 2 on drift; ignore
	if got := out.String(); !containsStr(got, "watcher:") {
		t.Fatalf("status must show watcher line, got %q", got)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
