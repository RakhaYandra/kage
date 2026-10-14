package diff

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

func track(t *testing.T, home, p string) {
	t.Helper()
	c, _ := config.Load(config.DefaultPath(home), home)
	c.Tracking.Paths = append(c.Tracking.Paths, p)
	config.Save(config.DefaultPath(home), c)
}

func TestHistoryDiffKinds(t *testing.T) {
	home := testHome(t)
	wf(t, home, ".zshrc", "a\n")
	wf(t, home, ".bashrc", "b\n")
	track(t, home, "~/.zshrc")
	track(t, home, "~/.bashrc")
	if _, err := snapshot.Run(home, snapshot.Options{Message: "a"}); err != nil {
		t.Fatal(err)
	}
	wf(t, home, ".zshrc", "a2\n")
	os.Remove(filepath.Join(home, ".bashrc"))
	wf(t, home, ".newrc", "n\n")
	track(t, home, "~/.newrc")
	if _, err := snapshot.Run(home, snapshot.Options{Message: "b"}); err != nil {
		t.Fatal(err)
	}
	res, err := History(home, 1, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]string{}
	for _, f := range res.Files {
		kinds[f.Path] = string(f.Kind)
	}
	if kinds[".zshrc"] != "modified" || kinds[".bashrc"] != "deleted" || kinds[".newrc"] != "added" {
		t.Fatalf("kinds=%v", kinds)
	}
	if res.Added != 1 || res.Modified != 1 || res.Deleted != 1 {
		t.Fatalf("summary=%+v", res)
	}
	found := false
	for _, f := range res.Files {
		if f.Path == ".zshrc" && len(f.Unified) > 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unified diff for .zshrc")
	}
}

func TestWorkingDiffDetectsModify(t *testing.T) {
	home := testHome(t)
	wf(t, home, ".zshrc", "v1\n")
	track(t, home, "~/.zshrc")
	if _, err := snapshot.Run(home, snapshot.Options{Message: "a"}); err != nil {
		t.Fatal(err)
	}
	wf(t, home, ".zshrc", "v2\n")
	res, err := Working(home, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Modified != 1 {
		t.Fatalf("summary=%+v", res)
	}
}

func TestBinaryFlagged(t *testing.T) {
	home := testHome(t)
	wf(t, home, ".bin", "v1\n")
	track(t, home, "~/.bin")
	if _, err := snapshot.Run(home, snapshot.Options{Message: "a"}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(home, ".bin"), []byte{0, 1, 2, 'x', 0}, 0o644)
	res, err := Working(home, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Files {
		if f.Path == ".bin" && !f.Binary {
			t.Fatalf("binary must be flagged")
		}
		if f.Path == ".bin" && len(f.Unified) != 0 {
			t.Fatalf("binary must not dump content")
		}
	}
}
