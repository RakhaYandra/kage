package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanRespectsExclude(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".zshrc"), "x=1\n")
	write(t, filepath.Join(home, ".env"), "SECRET=1\n")
	write(t, filepath.Join(home, ".ssh", "id_rsa"), "key\n")
	entries, err := Scan([]string{home + "/.zshrc", home + "/.env"}, []string{"**/.env", "**/id_*"}, home)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, e := range entries {
		seen[e.Rel] = true
	}
	if !seen[".zshrc"] {
		t.Errorf("expected .zshrc tracked, got %v", seen)
	}
	if seen[".env"] {
		t.Errorf(".env must be excluded")
	}
}

func TestScanExternalSymlinkFlagged(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	write(t, filepath.Join(outside, "secret"), "s\n")
	write(t, filepath.Join(home, ".config", "app", "real.conf"), "a\n")
	link := filepath.Join(home, ".config", "app", "evil.conf")
	os.MkdirAll(filepath.Dir(link), 0o755)
	if err := os.Symlink(filepath.Join(outside, "secret"), link); err != nil {
		t.Fatal(err)
	}
	entries, err := Scan([]string{home + "/.config/app"}, nil, home)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if e.Symlink && e.External {
			found = true
		}
	}
	if !found {
		t.Errorf("external symlink must be flagged External")
	}
}

func TestScanDirRecursive(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".config", "hypr", "hyprland.conf"), "x\n")
	write(t, filepath.Join(home, ".config", "hypr", "sub", "extra.conf"), "y\n")
	entries, err := Scan([]string{home + "/.config/hypr"}, nil, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 files, got %d", len(entries))
	}
}
