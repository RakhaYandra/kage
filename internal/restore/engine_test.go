package restore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

func testHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default(home)
	cfg.Tracking.Paths = []string{}
	if err := config.Save(config.DefaultPath(home), cfg); err != nil {
		t.Fatal(err)
	}
	return home
}

func wf(t *testing.T, home, rel, content string) {
	t.Helper()
	p := filepath.Join(home, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte(content), 0o644)
}

func rf(t *testing.T, home, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(home, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func track(t *testing.T, home, p string) {
	t.Helper()
	c, _ := config.Load(config.DefaultPath(home), home)
	c.Tracking.Paths = append(c.Tracking.Paths, p)
	config.Save(config.DefaultPath(home), c)
}

func TestRestoreSingleFileKeepsOthers(t *testing.T) {
	home := testHome(t)
	wf(t, home, ".zshrc", "v1\n")
	wf(t, home, ".bashrc", "b1\n")
	track(t, home, "~/.zshrc")
	track(t, home, "~/.bashrc")
	if _, err := snapshot.Run(home, snapshot.Options{Message: "a"}); err != nil {
		t.Fatal(err)
	}
	wf(t, home, ".zshrc", "v2\n")
	wf(t, home, ".bashrc", "b2\n")
	if _, err := snapshot.Run(home, snapshot.Options{Message: "b"}); err != nil {
		t.Fatal(err)
	}
	res, err := Run(home, Options{SnapshotID: 1, Files: []string{".zshrc"}, Yes: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Restored) != 1 {
		t.Fatalf("res=%+v", res)
	}
	if got := rf(t, home, ".zshrc"); got != "v1\n" {
		t.Fatalf("zshrc=%q", got)
	}
	if got := rf(t, home, ".bashrc"); got != "b2\n" {
		t.Fatalf("bashrc must be untouched, got %q", got)
	}
}

func TestRestoreDryRunWritesNothing(t *testing.T) {
	home := testHome(t)
	wf(t, home, ".zshrc", "v1\n")
	track(t, home, "~/.zshrc")
	snap, err := snapshot.Run(home, snapshot.Options{Message: "a"})
	if err != nil {
		t.Fatal(err)
	}
	_ = snap
	wf(t, home, ".zshrc", "v2\n")
	res, err := Run(home, Options{SnapshotID: 1, Files: []string{".zshrc"}, DryRun: true, Yes: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun {
		t.Fatalf("res=%+v", res)
	}
	if got := rf(t, home, ".zshrc"); got != "v2\n" {
		t.Fatalf("dry-run must not write, got %q", got)
	}
}

func TestRestoreConflictRefused(t *testing.T) {
	home := testHome(t)
	wf(t, home, ".zshrc", "v1\n")
	track(t, home, "~/.zshrc")
	snapshot.Run(home, snapshot.Options{Message: "a"})
	wf(t, home, ".zshrc", "v2\n")
	snapshot.Run(home, snapshot.Options{Message: "b"})
	wf(t, home, ".zshrc", "v3-dirty-uncommitted\n")
	_, err := Run(home, Options{SnapshotID: 1, Files: []string{".zshrc"}, Yes: true})
	if err == nil {
		t.Fatalf("expected conflict error")
	}
	if got := rf(t, home, ".zshrc"); got != "v3-dirty-uncommitted\n" {
		t.Fatalf("conflict must not write, got %q", got)
	}
}

func TestRestoreRejectsTraversal(t *testing.T) {
	home := testHome(t)
	wf(t, home, ".zshrc", "v1\n")
	track(t, home, "~/.zshrc")
	snapshot.Run(home, snapshot.Options{Message: "a"})
	_, err := Run(home, Options{SnapshotID: 1, Files: []string{"../etc/passwd"}, Yes: true})
	if err == nil {
		t.Fatalf("expected traversal error")
	}
}
