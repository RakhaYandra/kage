// Package gitstore wraps the Git CLI as versioning backend.
package gitstore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func run(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %v: %v: %s", args, err, out)
	}
	return nil
}

// Init creates the repo if missing with 0700 perms.
func Init(repo string) error {
	if err := os.MkdirAll(repo, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(repo, 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(repo, ".git")); err == nil {
		return nil
	}
	if err := run(repo, "init", "-q"); err != nil {
		return err
	}
	_ = run(repo, "config", "user.name", "kage")
	_ = run(repo, "config", "user.email", "kage@localhost")
	_ = run(repo, "config", "commit.gpgsign", "false")
	return os.Chmod(repo, 0o700)
}

// IsHealthy reports whether repo looks like a git repo.
func IsHealthy(repo string) bool {
	fi, err := os.Stat(filepath.Join(repo, ".git"))
	return err == nil && fi.IsDir()
}

// Repair re-inits a broken repo directory.
func Repair(repo string) error {
	if IsHealthy(repo) {
		return nil
	}
	os.RemoveAll(filepath.Join(repo, ".git"))
	if err := os.MkdirAll(repo, 0o700); err != nil {
		return err
	}
	return Init(repo)
}
