package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/gitstore"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

const loadSecret = "ghp_" + "AAAAAAAAAAAAAAAAAAAAAAAA"

func setupLoadHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default(home)
	cfg.Tracking.Paths = []string{"~/.myapp"}
	cfg.Security.AllowSecrets = []string{".myapp"}
	config.Save(config.DefaultPath(home), cfg)
	gitstore.Init(config.Expand(cfg.Repository.Path, home))
	os.WriteFile(filepath.Join(home, ".myapp"), []byte("token = "+loadSecret+"\n"), 0o644)
	if _, err := snapshot.Run(home, snapshot.Options{Message: "base"}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(home, ".myapp"), []byte("token = ghp_"+"BBBBBBBBBBBBBBBBBBBBBBBB\nother = 1\n"), 0o644)
	return home
}

func TestDiffFnShowsUnifiedRedacted(t *testing.T) {
	home := setupLoadHome(t)
	m, err := New(home)
	if err != nil {
		t.Fatal(err)
	}
	if m.diffFn == nil {
		t.Fatalf("diffFn must be wired")
	}
	d, err := m.diffFn(1, ".myapp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(d, "-") || !strings.Contains(d, "+") {
		t.Fatalf("want unified markers, got:\n%s", d)
	}
	if strings.Contains(d, loadSecret) || strings.Contains(d, "ghp_"+"BBBBBBBBBBBBBBBBBBBBBBBB") {
		t.Fatalf("secret leaked in tui diff:\n%s", d)
	}
}

func TestDiffFnCapsLongOutput(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default(home)
	cfg.Tracking.Paths = []string{"~/.big"}
	config.Save(config.DefaultPath(home), cfg)
	gitstore.Init(config.Expand(cfg.Repository.Path, home))
	os.WriteFile(filepath.Join(home, ".big"), []byte("v1\n"), 0o644)
	snapshot.Run(home, snapshot.Options{Message: "base"})
	var sb strings.Builder
	for i := 0; i < 2000; i++ {
		sb.WriteString("line\n")
	}
	os.WriteFile(filepath.Join(home, ".big"), []byte(sb.String()), 0o644)
	m, err := New(home)
	if err != nil {
		t.Fatal(err)
	}
	d, err := m.diffFn(1, ".big")
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(d, "\n"); n > 510 {
		t.Fatalf("diff must cap at ~500 lines, got %d", n)
	}
	if !strings.Contains(d, "truncated") {
		t.Fatalf("want truncation marker")
	}
}
