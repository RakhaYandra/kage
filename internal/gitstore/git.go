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

// Output runs git and returns stdout.
func Output(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
	)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %v: %s", args, ee.Stderr)
		}
		return "", fmt.Errorf("git %v: %w", args, err)
	}
	return string(out), nil
}

// AddAll stages everything (including deletions).
func AddAll(repo string) error { return run(repo, "add", "-A") }

// Commit creates a commit; returns nil error only on success.
func Commit(repo, message string) error {
	return run(repo, "-c", "user.name=kage", "-c", "user.email=kage@localhost",
		"commit", "-q", "-m", message)
}

// Head returns HEAD commit hash, or "" if none.
func Head(repo string) string {
	out, err := Output(repo, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	if len(out) > 7 {
		return out[:len(out)-1]
	}
	return out
}

// Show returns file content at HEAD (err if missing).
func Show(repo, path string) ([]byte, error) {
	cmd := exec.Command("git", "show", "HEAD:"+path)
	cmd.Dir = repo
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
	)
	return cmd.Output()
}

// ResetHard aborts a failed snapshot.
func ResetHard(repo string) error { return run(repo, "reset", "-q", "--hard", "HEAD") }

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
