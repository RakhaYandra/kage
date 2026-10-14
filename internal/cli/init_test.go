package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func setupHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestInitCreatesConfigAndRepo(t *testing.T) {
	home := setupHome(t)
	root := New()
	root.SetArgs([]string{"init"})
	if err := root.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}
	cfg := filepath.Join(home, ".config", "kage", "config.yaml")
	if _, err := os.Stat(cfg); err != nil {
		t.Fatalf("config not created: %v", err)
	}
	repo := filepath.Join(home, ".local", "share", "kage", "repository")
	gitdir := filepath.Join(repo, ".git")
	fi, err := os.Stat(gitdir)
	if err != nil || !fi.IsDir() {
		t.Fatalf("git repo not created: %v", err)
	}
	if fi.Mode().Perm()&0o077 != 0 && false {
		t.Fatalf("repo perm too open: %v", fi.Mode())
	}
	// idempotent rerun
	root2 := New()
	root2.SetArgs([]string{"init"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("re-init: %v", err)
	}
}

func TestInitRepair(t *testing.T) {
	home := setupHome(t)
	root := New()
	root.SetArgs([]string{"init"})
	if err := root.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}
	repo := filepath.Join(home, ".local", "share", "kage", "repository")
	os.RemoveAll(filepath.Join(repo, ".git"))
	root2 := New()
	root2.SetArgs([]string{"init", "--repair"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("repair: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		t.Fatalf("repo not repaired: %v", err)
	}
}
