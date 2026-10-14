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

const redactSecret = "ghp_" + "abcdefghijklmnopqrstuvwx"

func setupRedactHome(t *testing.T, redact bool) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default(home)
	cfg.Tracking.Paths = []string{"~/.myapp"}
	cfg.Security.AllowSecrets = []string{".myapp"}
	cfg.Security.RedactOutput = redact
	if err := config.Save(config.DefaultPath(home), cfg); err != nil {
		t.Fatal(err)
	}
	if err := gitstore.Init(config.Expand(cfg.Repository.Path, home)); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(home, ".myapp"), []byte("token = "+redactSecret+"\n"), 0o644)
	if _, err := snapshot.Run(home, snapshot.Options{Message: "base"}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(home, ".myapp"), []byte("token = ghp_" + "ZZZZZZZZZZZZZZZZZZZZZZZZ\n"), 0o644)
	return home
}

func TestDiffRedactsSecrets(t *testing.T) {
	setupRedactHome(t, true)
	root := New()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"diff", "--file", ".myapp"})
	_ = root.Execute() // exit 2 on changes; ignore
	if got := out.String(); containsStr(got, redactSecret) || containsStr(got, "ghp_" + "ZZZZZZZZZZZZZZZZZZZZZZZZ") {
		t.Fatalf("secret leaked in diff:\n%s", got)
	}
	if got := out.String(); !containsStr(got, "****") {
		t.Fatalf("expected redaction marker in diff:\n%s", got)
	}
}

func TestDiffRawWhenRedactOff(t *testing.T) {
	setupRedactHome(t, false)
	root := New()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"diff", "--file", ".myapp"})
	_ = root.Execute()
	if got := out.String(); !containsStr(got, "ghp_" + "ZZZZZZZZZZZZZZZZZZZZZZZZ") {
		t.Fatalf("redact off must show raw content:\n%s", got)
	}
}

func TestRestorePreviewRedactsSecrets(t *testing.T) {
	setupRedactHome(t, true)
	root := New()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"restore", "--snapshot", "1", "--file", ".myapp", "--dry-run"})
	_ = root.Execute()
	if got := out.String(); containsStr(got, redactSecret) || containsStr(got, "ghp_" + "ZZZZZZZZZZZZZZZZZZZZZZZZ") {
		t.Fatalf("secret leaked in restore preview:\n%s", got)
	}
}
