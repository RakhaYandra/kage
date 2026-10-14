package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/gitstore"
	"github.com/RakhaYandra/kage/internal/restore"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

func TestHistoryMergesThreeSources(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default(home)
	cfg.Tracking.Paths = []string{"~/.zshrc"}
	config.Save(config.DefaultPath(home), cfg)
	gitstore.Init(config.Expand(cfg.Repository.Path, home))
	p := filepath.Join(home, ".zshrc")
	os.WriteFile(p, []byte("v1\n"), 0o644)
	snapshot.Run(home, snapshot.Options{Message: "base"})
	os.WriteFile(p, []byte("v2\n"), 0o644)
	snapshot.Run(home, snapshot.Options{Message: "second"})
	restore.Run(home, restore.Options{SnapshotID: 1, Files: []string{".zshrc"}, Yes: true})
	// fake one watch event
	os.MkdirAll(snapshot.StateDir(home), 0o700)
	os.WriteFile(filepath.Join(snapshot.StateDir(home), "watch.log"),
		[]byte("{\"ts\":\"2026-10-14T00:00:00Z\",\"rel\":\".zshrc\",\"kind\":\"modify\"}\n"), 0o600)

	lines, err := History(home, 0)
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, l := range lines {
		joined += l + "\n"
	}
	for _, want := range []string{"snapshot", "restore", "watch"} {
		if !containsStr(joined, want) {
			t.Fatalf("missing %q in:\n%s", want, joined)
		}
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
