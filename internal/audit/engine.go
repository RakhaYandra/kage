// Package audit checks state drift, coverage, secrets, permissions and repo health.
package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/discovery"
	"github.com/RakhaYandra/kage/internal/gitstore"
	"github.com/RakhaYandra/kage/internal/metadata"
	"github.com/RakhaYandra/kage/internal/security"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

// Report is the audit result.
type Report struct {
	Drift            int
	DriftFiles       []string
	Coverage         []string
	Secrets          []security.Finding
	Perms            []string
	Symlinks         []string
	RepoHealth       string
	RestoreReadiness []string
	Result           string
}

var coverageCandidates = []string{
	"~/.bashrc", "~/.zshrc", "~/.gitconfig",
	"~/.config/hypr", "~/.config/waybar", "~/.config/kitty",
	"~/.config/alacritty", "~/.config/Code/User", "~/.config/opencode",
}

// Run performs a read-only audit.
func Run(home string) (Report, error) {
	var rep Report
	cfg, err := config.Load(config.DefaultPath(home), home)
	if err != nil {
		return rep, err
	}
	repo := config.Expand(cfg.Repository.Path, home)

	// repo health
	store := metadata.Open(snapshot.StateDir(home))
	snaps, _ := store.List()
	switch {
	case !gitstore.IsHealthy(repo):
		rep.RepoHealth = "Broken (run kage init --repair)"
	case len(snaps) == 0:
		rep.RepoHealth = "No snapshots yet"
	default:
		rep.RepoHealth = "Healthy"
	}

	entries, err := discovery.Scan(cfg.Tracking.Paths, cfg.Tracking.Exclude, home)
	if err != nil {
		return rep, err
	}

	// drift (working vs latest manifest)
	if st, err := snapshot.GetStatus(home); err == nil && len(snaps) > 0 {
		rep.Drift = len(st.Changed)
		rep.DriftFiles = st.Changed
	}

	// coverage: candidates present but untracked and not excluded
	tracked := make([]string, 0, len(cfg.Tracking.Paths))
	for _, t := range cfg.Tracking.Paths {
		tracked = append(tracked, config.Normalize(t, home))
	}
	for _, c := range coverageCandidates {
		abs := config.Normalize(c, home)
		if _, err := os.Lstat(abs); err != nil {
			continue
		}
		covered := false
		for _, t := range tracked {
			if abs == t || strings.HasPrefix(abs, t+string(os.PathSeparator)) {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		rel, _ := filepath.Rel(home, abs)
		if discovery.Excluded(cfg.Tracking.Exclude, filepath.ToSlash(rel)) {
			continue
		}
		rep.Coverage = append(rep.Coverage, filepath.ToSlash(rel))
	}
	sort.Strings(rep.Coverage)

	// secrets + perms + symlinks over working entries
	for _, e := range entries {
		if e.External {
			rep.Symlinks = append(rep.Symlinks,
				fmt.Sprintf("%s -> outside scope (%s)", e.Rel, security.Redact(e.SymlinkTarget)))
			continue
		}
		content, err := readEntry(e.Abs)
		if err != nil {
			continue
		}
		for _, f := range security.ScanContent(e.Rel, content) {
			// content-rule findings count as secrets; name-only findings are
			// reported once and also feed the permissions check below
			rep.Secrets = append(rep.Secrets, f)
		}
		if security.SensitiveName(e.Rel) && e.Mode.Perm()&0o044 != 0 {
			rep.Perms = append(rep.Perms,
				fmt.Sprintf("open permissions %o on %s", e.Mode.Perm(), e.Rel))
		}
	}
	// repo dir perms
	if fi, err := os.Stat(repo); err == nil && fi.IsDir() && fi.Mode().Perm()&0o077 != 0 {
		rep.Perms = append(rep.Perms, fmt.Sprintf("repository permissions too open (%o)", fi.Mode().Perm()))
	}

	// restore readiness
	if rep.Drift > 0 {
		rep.RestoreReadiness = append(rep.RestoreReadiness,
			"uncommitted changes present: snapshot first before restoring old versions")
	}
	if rep.RepoHealth != "Healthy" && rep.RepoHealth != "No snapshots yet" {
		rep.RestoreReadiness = append(rep.RestoreReadiness, "repository broken: restore unavailable")
	}
	if len(rep.Secrets) > 0 {
		rep.RestoreReadiness = append(rep.RestoreReadiness, "secret indicators present: review before sharing snapshots")
	}

	switch {
	case rep.RepoHealth == "Broken (run kage init --repair)" || len(rep.Secrets) > 0:
		rep.Result = "Action required"
	case rep.Drift > 0 || len(rep.Coverage) > 0 || len(rep.Perms) > 0 || len(rep.Symlinks) > 0 || len(rep.RestoreReadiness) > 0:
		rep.Result = "Review recommended"
	default:
		rep.Result = "Healthy"
	}
	return rep, nil
}

func readEntry(abs string) ([]byte, error) {
	fi, err := os.Lstat(abs)
	if err != nil {
		return nil, err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(abs)
		if err != nil {
			return nil, err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(abs), target)
		}
		abs = target
	}
	return os.ReadFile(abs)
}
