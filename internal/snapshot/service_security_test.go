package snapshot

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/security"
)

func TestSnapshotBlockedOnSecret(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default(home)
	cfg.Tracking.Paths = []string{"~/.myconf"}
	if err := config.Save(config.DefaultPath(home), cfg); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(home, ".myconf")
	os.WriteFile(p, []byte("password = hunter2hunter2\n"), 0o644)
	_, err := Run(home, Options{Message: "x"})
	var blocked *security.BlockedError
	if !errors.As(err, &blocked) {
		t.Fatalf("expected BlockedError, got %v", err)
	}
	// allowlist records explicit exception
	c, _ := config.Load(config.DefaultPath(home), home)
	c.Security.AllowSecrets = []string{".myconf"}
	config.Save(config.DefaultPath(home), c)
	res, err := Run(home, Options{Message: "x"})
	if err != nil {
		t.Fatalf("allowlisted must pass: %v", err)
	}
	if res.Skipped || res.ID != 1 {
		t.Fatalf("got %+v", res)
	}
}
