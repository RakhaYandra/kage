// Package snapshot takes config snapshots into the local git repo.
package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/discovery"
	"github.com/RakhaYandra/kage/internal/filesystem"
	"github.com/RakhaYandra/kage/internal/gitstore"
	"github.com/RakhaYandra/kage/internal/metadata"
)

// Options for Run.
type Options struct {
	Message string
	DryRun  bool
}

// Result of a snapshot.
type Result struct {
	ID       int
	Skipped  bool
	DryRun   bool
	Added    int
	Modified int
	Deleted  int
	Hash     string
	Commit   string
}

// FileMeta in the manifest.
type FileMeta struct {
	Hash    string `json:"hash"`
	Mode    uint32 `json:"mode"`
	Symlink bool   `json:"symlink,omitempty"`
	Target  string `json:"target,omitempty"`
}

// Manifest maps rel → meta.
type Manifest struct {
	Files map[string]FileMeta `json:"files"`
}

const manifestPath = ".kage/manifest.json"

// StateDir returns ~/.local/share/kage.
func StateDir(home string) string {
	return filepath.Join(home, ".local", "share", "kage")
}

func loadManifest(repo string) Manifest {
	m := Manifest{Files: map[string]FileMeta{}}
	data, err := gitstore.Show(repo, manifestPath)
	if err != nil {
		return m
	}
	_ = json.Unmarshal(data, &m)
	if m.Files == nil {
		m.Files = map[string]FileMeta{}
	}
	return m
}

func stateHash(files map[string]FileMeta) string {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		f := files[k]
		fmt.Fprintf(h, "%s\x00%s\x00%d\x00%s\n", k, f.Hash, f.Mode, f.Target)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Run takes a snapshot of all tracked files.
func Run(home string, opts Options) (Result, error) {
	cfg, err := config.Load(config.DefaultPath(home), home)
	if err != nil {
		return Result{}, err
	}
	if err := config.Validate(cfg, home); err != nil {
		return Result{}, err
	}
	repo := config.Expand(cfg.Repository.Path, home)
	if err := gitstore.Init(repo); err != nil {
		return Result{}, err
	}
	if !gitstore.IsHealthy(repo) {
		return Result{}, fmt.Errorf("repository unhealthy (run kage init --repair)")
	}

	entries, err := discovery.Scan(cfg.Tracking.Paths, cfg.Tracking.Exclude, home)
	if err != nil {
		return Result{}, err
	}
	current := map[string]FileMeta{}
	for _, e := range entries {
		if e.External {
			continue // never snapshot targets outside scope
		}
		var hash string
		var target string
		if e.Symlink {
			target = e.SymlinkTarget
			h := sha256.Sum256([]byte("link:" + target))
			hash = hex.EncodeToString(h[:])
		} else {
			hash, err = filesystem.HashFile(e.Abs)
			if err != nil {
				continue
			}
		}
		current[e.Rel] = FileMeta{Hash: hash, Mode: uint32(e.Mode.Perm()), Symlink: e.Symlink, Target: target}
	}

	prev := loadManifest(repo)
	var added, modified, deleted int
	for rel, cur := range current {
		old, ok := prev.Files[rel]
		if !ok {
			added++
		} else if old.Hash != cur.Hash {
			modified++
		}
	}
	for rel := range prev.Files {
		if _, ok := current[rel]; !ok {
			deleted++
		}
	}
	hash := stateHash(current)

	if added+modified+deleted == 0 && cfg.Snapshot.SkipUnchanged {
		return Result{Skipped: true, Hash: hash}, nil
	}

	store := metadata.Open(StateDir(home))
	nextID, err := store.NextID()
	if err != nil {
		return Result{}, err
	}
	if opts.DryRun {
		return Result{ID: nextID, DryRun: true, Added: added, Modified: modified, Deleted: deleted, Hash: hash}, nil
	}

	// write working files into repo
	for rel := range current {
		abs := filepath.Join(home, filepath.FromSlash(rel))
		// abs may be a tracked file directly; find via current only
		dst := filepath.Join(repo, filepath.FromSlash(rel))
		fi, err := os.Lstat(abs)
		if err != nil {
			continue
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			target, _ := os.Readlink(abs)
			os.Remove(dst)
			_ = os.MkdirAll(filepath.Dir(dst), 0o755)
			_ = os.Symlink(target, dst)
			continue
		}
		mode := fi.Mode()
		if !cfg.Snapshot.PreservePermissions {
			mode = 0o644
		}
		if err := filesystem.CopyFile(abs, dst, mode); err != nil {
			_ = gitstore.ResetHard(repo)
			return Result{}, fmt.Errorf("stage %s: %w", rel, err)
		}
	}
	for rel := range prev.Files {
		if _, ok := current[rel]; !ok {
			os.Remove(filepath.Join(repo, filepath.FromSlash(rel)))
		}
	}

	// detect concurrent modification during staging
	for _, e := range entries {
		if e.External || e.Symlink {
			continue
		}
		rel := e.Rel
		cur, ok := current[rel]
		if !ok {
			continue
		}
		rehash, err := filesystem.HashFile(e.Abs)
		if err != nil || rehash != cur.Hash {
			_ = gitstore.ResetHard(repo)
			return Result{}, fmt.Errorf("file changed during snapshot: %s (retry)", rel)
		}
	}

	manifest := Manifest{Files: current}
	mdata, _ := json.MarshalIndent(manifest, "", "  ")
	mpath := filepath.Join(repo, filepath.FromSlash(manifestPath))
	_ = os.MkdirAll(filepath.Dir(mpath), 0o755)
	if err := os.WriteFile(mpath, mdata, 0o644); err != nil {
		_ = gitstore.ResetHard(repo)
		return Result{}, err
	}
	if err := gitstore.AddAll(repo); err != nil {
		_ = gitstore.ResetHard(repo)
		return Result{}, err
	}
	msg := opts.Message
	if msg == "" {
		msg = fmt.Sprintf("snapshot #%d", nextID)
	}
	commitMsg := fmt.Sprintf("kage snapshot #%d: %s", nextID, msg)
	if err := gitstore.Commit(repo, commitMsg); err != nil {
		_ = gitstore.ResetHard(repo)
		return Result{}, fmt.Errorf("commit: %w", err)
	}
	commit := gitstore.Head(repo)
	rec := metadata.Snapshot{
		ID: nextID, CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Message: msg, GitCommit: commit,
		Added: added, Modified: modified, Deleted: deleted,
		Hash: hash, SchemaVersion: 1,
	}
	if err := store.Append(rec); err != nil {
		return Result{}, err
	}
	return Result{ID: nextID, Added: added, Modified: modified, Deleted: deleted, Hash: hash, Commit: commit}, nil
}

// Status compares working state to the last snapshot manifest.
type Status struct {
	Changed []string
	Drift   bool
}

// GetStatus lists working files differing from HEAD manifest.
func GetStatus(home string) (Status, error) {
	cfg, err := config.Load(config.DefaultPath(home), home)
	if err != nil {
		return Status{}, err
	}
	repo := config.Expand(cfg.Repository.Path, home)
	prev := loadManifest(repo)
	entries, err := discovery.Scan(cfg.Tracking.Paths, cfg.Tracking.Exclude, home)
	if err != nil {
		return Status{}, err
	}
	current := map[string]string{}
	for _, e := range entries {
		if e.External {
			continue
		}
		if e.Symlink {
			h := sha256.Sum256([]byte("link:" + e.SymlinkTarget))
			current[e.Rel] = hex.EncodeToString(h[:])
			continue
		}
		h, err := filesystem.HashFile(e.Abs)
		if err != nil {
			continue
		}
		current[e.Rel] = h
	}
	var changed []string
	for rel, h := range current {
		old, ok := prev.Files[rel]
		if !ok || old.Hash != h {
			changed = append(changed, rel+" (modified/added)")
		}
	}
	for rel := range prev.Files {
		if _, ok := current[rel]; !ok {
			changed = append(changed, rel+" (deleted)")
		}
	}
	sort.Strings(changed)
	_ = strings.Join(changed, "")
	return Status{Changed: changed, Drift: len(changed) > 0}, nil
}
