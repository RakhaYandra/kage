package snapshot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/RakhaYandra/kage/internal/config"
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
	repo := filepath.Join(home, ".local", "share", "kage", "repository")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	return home
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
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

func TestFirstSnapshotCommits(t *testing.T) {
	home := testHome(t)
	writeFile(t, filepath.Join(home, ".zshrc"), "x=1\n")
	track(t, home, "~/.zshrc")
	res, err := Run(home, Options{Message: "initial"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Skipped {
		t.Fatalf("first snapshot must not skip")
	}
	if res.Added != 1 || res.ID != 1 {
		t.Fatalf("got %+v", res)
	}
}

func TestSnapshotSkipsWhenUnchanged(t *testing.T) {
	home := testHome(t)
	writeFile(t, filepath.Join(home, ".zshrc"), "x=1\n")
	track(t, home, "~/.zshrc")
	if _, err := Run(home, Options{Message: "a"}); err != nil {
		t.Fatal(err)
	}
	res, err := Run(home, Options{Message: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped {
		t.Fatalf("unchanged must skip, got %+v", res)
	}
}

func TestSnapshotCountsModifyAddDelete(t *testing.T) {
	home := testHome(t)
	writeFile(t, filepath.Join(home, ".zshrc"), "v1\n")
	writeFile(t, filepath.Join(home, ".bashrc"), "b\n")
	track(t, home, "~/.zshrc")
	track(t, home, "~/.bashrc")
	if _, err := Run(home, Options{Message: "a"}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(home, ".zshrc"), "v2\n")
	os.Remove(filepath.Join(home, ".bashrc"))
	writeFile(t, filepath.Join(home, ".newrc"), "n\n")
	track(t, home, "~/.newrc")
	res, err := Run(home, Options{Message: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Modified != 1 || res.Deleted != 1 || res.Added != 1 {
		t.Fatalf("got %+v", res)
	}
}

func TestSnapshotDryRunWritesNothing(t *testing.T) {
	home := testHome(t)
	writeFile(t, filepath.Join(home, ".zshrc"), "v1\n")
	track(t, home, "~/.zshrc")
	res, err := Run(home, Options{Message: "dry", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun || res.Added != 1 {
		t.Fatalf("got %+v", res)
	}
	// real run after dry-run must still be ID 1
	res2, err := Run(home, Options{Message: "real"})
	if err != nil {
		t.Fatal(err)
	}
	if res2.ID != 1 {
		t.Fatalf("dry-run must not consume ID, got %+v", res2)
	}
}
