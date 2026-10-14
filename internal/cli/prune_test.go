package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/gitstore"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

func TestPruneTrimsLogs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default(home)
	cfg.Tracking.Paths = []string{"~/.zshrc"}
	config.Save(config.DefaultPath(home), cfg)
	gitstore.Init(config.Expand(cfg.Repository.Path, home))
	os.WriteFile(filepath.Join(home, ".zshrc"), []byte("a\n"), 0o644)
	snapshot.Run(home, snapshot.Options{Message: "one"})
	os.WriteFile(filepath.Join(home, ".zshrc"), []byte("b\n"), 0o644)
	snapshot.Run(home, snapshot.Options{Message: "two"})
	state := snapshot.StateDir(home)
	for _, name := range []string{"watch.log", "restore.log"} {
		var sb strings.Builder
		for i := 0; i < 10; i++ {
			sb.WriteString("{\"ts\":\"2026-01-01T00:00:00Z\",\"i\":" + string(rune('0'+i)) + "}\n")
		}
		os.WriteFile(filepath.Join(state, name), []byte(sb.String()), 0o600)
	}

	root := New()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"prune", "--keep", "1", "--yes"})
	if err := root.Execute(); err != nil {
		t.Fatalf("prune: %v", err)
	}
	for _, name := range []string{"watch.log", "restore.log"} {
		data, _ := os.ReadFile(filepath.Join(state, name))
		if n := strings.Count(string(data), "\n"); n != 1 {
			t.Fatalf("%s lines=%d, want 1", name, n)
		}
	}
}

func TestPruneSkipsMissingLogs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default(home)
	cfg.Tracking.Paths = []string{"~/.zshrc"}
	config.Save(config.DefaultPath(home), cfg)
	gitstore.Init(config.Expand(cfg.Repository.Path, home))
	os.WriteFile(filepath.Join(home, ".zshrc"), []byte("a\n"), 0o644)
	snapshot.Run(home, snapshot.Options{Message: "one"})
	os.WriteFile(filepath.Join(home, ".zshrc"), []byte("b\n"), 0o644)
	snapshot.Run(home, snapshot.Options{Message: "two"})

	root := New()
	root.SetArgs([]string{"prune", "--keep", "1", "--yes"})
	if err := root.Execute(); err != nil {
		t.Fatalf("prune without logs: %v", err)
	}
}
