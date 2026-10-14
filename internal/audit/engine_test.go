package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/gitstore"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

func testHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default(home)
	cfg.Tracking.Paths = []string{}
	config.Save(config.DefaultPath(home), cfg)
	if err := gitstore.Init(config.Expand(cfg.Repository.Path, home)); err != nil {
		t.Fatal(err)
	}
	return home
}

func wf(t *testing.T, home, rel, content string, mode os.FileMode) {
	t.Helper()
	p := filepath.Join(home, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte(content), mode)
}

func track(t *testing.T, home, p string) {
	t.Helper()
	c, _ := config.Load(config.DefaultPath(home), home)
	c.Tracking.Paths = append(c.Tracking.Paths, p)
	config.Save(config.DefaultPath(home), c)
}

func TestHealthyEmptyScope(t *testing.T) {
	home := testHome(t)
	rep, err := Run(home)
	if err != nil {
		t.Fatal(err)
	}
	if rep.RepoHealth != "Healthy" && rep.RepoHealth != "No snapshots yet" {
		t.Fatalf("health=%q", rep.RepoHealth)
	}
	if rep.Result != "Healthy" {
		t.Fatalf("result=%q", rep.Result)
	}
}

func TestDriftAndCoverage(t *testing.T) {
	home := testHome(t)
	wf(t, home, ".zshrc", "v1\n", 0o644)
	wf(t, home, ".bashrc", "b\n", 0o644)
	track(t, home, "~/.zshrc")
	snapshot.Run(home, snapshot.Options{Message: "a"})
	wf(t, home, ".zshrc", "v2\n", 0o644)
	rep, err := Run(home)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Drift != 1 {
		t.Fatalf("drift=%d", rep.Drift)
	}
	found := false
	for _, c := range rep.Coverage {
		if c == ".bashrc" {
			found = true
		}
	}
	if !found {
		t.Fatalf("coverage must suggest .bashrc, got %v", rep.Coverage)
	}
	if rep.Result != "Review recommended" {
		t.Fatalf("result=%q", rep.Result)
	}
}

func TestPermAndSymlinkWarnings(t *testing.T) {
	home := testHome(t)
	// un-exclude sensitive names so the perm check can see them
	c, _ := config.Load(config.DefaultPath(home), home)
	c.Tracking.Exclude = nil
	config.Save(config.DefaultPath(home), c)
	wf(t, home, ".ssh/id_mine", "key\n", 0o644) // sensitive name + open perm
	track(t, home, "~/.ssh/id_mine")
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "s"), []byte("s\n"), 0o644)
	os.MkdirAll(filepath.Join(home, ".config", "app"), 0o755)
	os.Symlink(filepath.Join(outside, "s"), filepath.Join(home, ".config", "app", "evil.conf"))
	track(t, home, "~/.config/app")
	rep, err := Run(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Perms) == 0 {
		t.Fatalf("expected perm warning")
	}
	if len(rep.Symlinks) == 0 {
		t.Fatalf("expected external symlink warning")
	}
}
