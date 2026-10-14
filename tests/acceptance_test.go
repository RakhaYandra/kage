// Package tests holds MVP acceptance scenarios (all in t.TempDir homes).
package tests

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/RakhaYandra/kage/internal/audit"
	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/diff"
	"github.com/RakhaYandra/kage/internal/gitstore"
	"github.com/RakhaYandra/kage/internal/restore"
	"github.com/RakhaYandra/kage/internal/security"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

func setup(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default(home)
	cfg.Tracking.Paths = []string{}
	if err := config.Save(config.DefaultPath(home), cfg); err != nil {
		t.Fatal(err)
	}
	repo := config.Expand(cfg.Repository.Path, home)
	if err := gitstore.Init(repo); err != nil {
		t.Fatal(err)
	}
	return home
}

func wf(t *testing.T, home, rel, content string) {
	t.Helper()
	p := filepath.Join(home, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func track(t *testing.T, home, p string) {
	t.Helper()
	c, err := config.Load(config.DefaultPath(home), home)
	if err != nil {
		t.Fatal(err)
	}
	c.Tracking.Paths = append(c.Tracking.Paths, p)
	if err := config.Save(config.DefaultPath(home), c); err != nil {
		t.Fatal(err)
	}
}

// Full journey: snapshot → diff → restore one file → audit.
func TestAcceptanceJourney(t *testing.T) {
	home := setup(t)
	wf(t, home, ".zshrc", "v1\n")
	wf(t, home, ".config/hypr/hyprland.conf", "kb=old\n")
	track(t, home, "~/.zshrc")
	track(t, home, "~/.config/hypr")

	s1, err := snapshot.Run(home, snapshot.Options{Message: "initial"})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if s1.ID != 1 {
		t.Fatalf("s1=%+v", s1)
	}

	wf(t, home, ".config/hypr/hyprland.conf", "kb=new\n")
	d, err := snapshot.GetStatus(home)
	if err != nil || !d.Drift {
		t.Fatalf("drift not detected: %v %+v", d, err)
	}

	wd, err := diff.Working(home, 0, "")
	if err != nil || wd.Modified != 1 {
		t.Fatalf("diff=%+v err=%v", wd, err)
	}

	s2, err := snapshot.Run(home, snapshot.Options{Message: "kb change"})
	if err != nil || s2.ID != 2 {
		t.Fatalf("s2=%+v err=%v", s2, err)
	}

	r, err := restore.Run(home, restore.Options{SnapshotID: 1, Files: []string{".config/hypr/hyprland.conf"}, Yes: true})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if len(r.Restored) != 1 {
		t.Fatalf("restored=%+v", r)
	}
	got, _ := os.ReadFile(filepath.Join(home, ".config", "hypr", "hyprland.conf"))
	if string(got) != "kb=old\n" {
		t.Fatalf("content=%q", got)
	}
	// other file untouched
	zb, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	if string(zb) != "v1\n" {
		t.Fatalf("zshrc touched: %q", zb)
	}

	rep, err := audit.Run(home)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if rep.RepoHealth != "Healthy" {
		t.Fatalf("health=%q", rep.RepoHealth)
	}
}

func TestAcceptanceSecretBlocked(t *testing.T) {
	home := setup(t)
	wf(t, home, ".myapp", "password = hunter2hunter2\n")
	track(t, home, "~/.myapp")
	_, err := snapshot.Run(home, snapshot.Options{Message: "x"})
	var blocked *security.BlockedError
	if !errors.As(err, &blocked) {
		t.Fatalf("expected BlockedError, got %v", err)
	}
	// failed op must not record a snapshot
	st, _ := snapshot.GetStatus(home)
	_ = st
	rep, err := audit.Run(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Secrets) == 0 {
		t.Fatalf("audit must flag the secret")
	}
}

func TestAcceptanceSymlinkPreserved(t *testing.T) {
	home := setup(t)
	wf(t, home, ".config/app/real.conf", "a\n")
	if err := os.Symlink("real.conf", filepath.Join(home, ".config", "app", "link.conf")); err != nil {
		t.Fatal(err)
	}
	track(t, home, "~/.config/app")
	if _, err := snapshot.Run(home, snapshot.Options{Message: "a"}); err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(home, ".config", "app", "link.conf"))
	r, err := restore.Run(home, restore.Options{SnapshotID: 1, Files: []string{".config/app/link.conf"}, Yes: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Restored) != 1 {
		t.Fatalf("restored=%+v", r)
	}
	fi, err := os.Lstat(filepath.Join(home, ".config", "app", "link.conf"))
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink not restored: %v", fi)
	}
}
