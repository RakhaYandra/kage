// Package restore performs selective restores with checkpoint and conflict checks.
package restore

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
	"github.com/RakhaYandra/kage/internal/snapshot"
)

// Options for Run.
type Options struct {
	SnapshotID int
	Commit     string
	Files      []string
	DryRun     bool
	Yes        bool // unused by engine (CLI prompts); kept for API parity
}

// Result of a restore.
type Result struct {
	Restored   []string
	Skipped    []string
	Conflicts  []string
	DryRun     bool
	Checkpoint string
	Preview    string
}

// ConflictError signals unresolved conflicts (exit 3).
type ConflictError struct {
	Files []string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict: %d file(s) changed since snapshot (%s); snapshot current state first",
		len(e.Files), strings.Join(e.Files, ", "))
}

func targetCommit(home string, snapshotID int, commit string, repo string) (string, error) {
	if commit != "" {
		if !gitstore.Exists(repo, commit) {
			return "", fmt.Errorf("commit %q not found", commit)
		}
		return commit, nil
	}
	store := metadata.Open(snapshot.StateDir(home))
	list, err := store.List()
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", fmt.Errorf("no snapshots yet")
	}
	if snapshotID == 0 {
		return list[len(list)-1].GitCommit, nil
	}
	for _, s := range list {
		if s.ID == snapshotID {
			return s.GitCommit, nil
		}
	}
	return "", fmt.Errorf("snapshot #%d not found", snapshotID)
}

func latestCommit(home string) (string, error) {
	store := metadata.Open(snapshot.StateDir(home))
	list, err := store.List()
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", fmt.Errorf("no snapshots yet")
	}
	return list[len(list)-1].GitCommit, nil
}

func manifestAt(repo, commit string) snapshot.Manifest {
	m := snapshot.Manifest{Files: map[string]snapshot.FileMeta{}}
	data, err := gitstore.ShowAt(repo, commit, ".kage/manifest.json")
	if err != nil {
		return m
	}
	_ = json.Unmarshal(data, &m)
	if m.Files == nil {
		m.Files = map[string]snapshot.FileMeta{}
	}
	return m
}

// normalizeFile maps user input to home-relative slash path.
func normalizeFile(home, f string) (string, error) {
	var abs string
	if strings.HasPrefix(f, "~/") || f == "~" {
		abs = config.Expand(f, home)
	} else if filepath.IsAbs(f) {
		abs = filepath.Clean(f)
	} else {
		abs = filepath.Clean(filepath.Join(home, filepath.FromSlash(f)))
	}
	rel, err := filepath.Rel(home, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("outside scope: %q", f)
	}
	return filepath.ToSlash(rel), nil
}

func workingBytes(home, rel string) ([]byte, bool) {
	abs := filepath.Join(home, filepath.FromSlash(rel))
	fi, err := os.Lstat(abs)
	if err != nil {
		return nil, false
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(abs)
		if err != nil {
			return nil, false
		}
		return []byte("link:" + target), true
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return nil, false
	}
	return b, true
}

func atBytes(repo, commit, rel string) ([]byte, bool) {
	b, err := gitstore.ShowAt(repo, commit, rel)
	if err != nil {
		return nil, false
	}
	return b, true
}

func same(a []byte, aOK bool, b []byte, bOK bool) bool {
	if aOK != bOK {
		return false
	}
	if !aOK {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func previewLines(b []byte, ok bool) string {
	if !ok {
		return "(absent)"
	}
	if len(b) > 2000 {
		b = b[:2000]
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) > 12 {
		lines = lines[:12]
		return strings.Join(lines, "\n") + "\n..."
	}
	return strings.Join(lines, "\n")
}

// Run restores selected files from a snapshot.
func Run(home string, opts Options) (Result, error) {
	cfg, err := config.Load(config.DefaultPath(home), home)
	if err != nil {
		return Result{}, err
	}
	repo := config.Expand(cfg.Repository.Path, home)
	if !gitstore.IsHealthy(repo) {
		return Result{}, fmt.Errorf("repository unhealthy (run kage init --repair)")
	}
	commit, err := targetCommit(home, opts.SnapshotID, opts.Commit, repo)
	if err != nil {
		return Result{}, err
	}
	latest, err := latestCommit(home)
	if err != nil {
		latest = commit
	}

	var rels []string
	if len(opts.Files) > 0 {
		for _, f := range opts.Files {
			rel, err := normalizeFile(home, f)
			if err != nil {
				return Result{}, err
			}
			rels = append(rels, rel)
		}
	} else {
		// default: all files differing between working and target
		targetM := manifestAt(repo, commit)
		entries, _ := discovery.Scan(cfg.Tracking.Paths, cfg.Tracking.Exclude, home)
		seen := map[string]bool{}
		for _, e := range entries {
			if e.External {
				continue
			}
			seen[e.Rel] = true
		}
		for rel := range targetM.Files {
			seen[rel] = true
		}
		for rel := range seen {
			wb, wok := workingBytes(home, rel)
			tb, tok := atBytes(repo, commit, rel)
			if !same(wb, wok, tb, tok) {
				rels = append(rels, rel)
			}
		}
		sort.Strings(rels)
	}

	targetM := manifestAt(repo, commit)
	var planned, skipped, conflicts []string
	var previews []string
	for _, rel := range rels {
		tb, tok := atBytes(repo, commit, rel)
		if _, inManifest := targetM.Files[rel]; !inManifest && !tok {
			skipped = append(skipped, rel+" (not in snapshot)")
			continue
		}
		wb, wok := workingBytes(home, rel)
		lb, lok := atBytes(repo, latest, rel)
		if commit != latest && !same(wb, wok, lb, lok) && !same(wb, wok, tb, tok) {
			conflicts = append(conflicts, rel)
			continue
		}
		planned = append(planned, rel)
		action := "overwrite"
		if !wok {
			action = "create"
		} else if !tok {
			action = "delete"
		}
		previews = append(previews, fmt.Sprintf("~/%s [%s]\n--- current ---\n%s\n--- snapshot ---\n%s",
			rel, action, previewLines(wb, wok), previewLines(tb, tok)))
	}
	if len(conflicts) > 0 {
		return Result{Conflicts: conflicts, Skipped: skipped, Preview: strings.Join(previews, "\n\n")},
			&ConflictError{Files: conflicts}
	}

	preview := strings.Join(previews, "\n\n")
	if opts.DryRun {
		return Result{Restored: planned, Skipped: skipped, DryRun: true, Preview: preview}, nil
	}

	// checkpoint current state (skip if clean vs HEAD)
	checkpoint := ""
	if head := gitstore.Head(repo); head != "" {
		entries, _ := discovery.Scan(cfg.Tracking.Paths, cfg.Tracking.Exclude, home)
		if dirty := checkpointNeeded(repo, head, entries, home); dirty {
			cp, err := makeCheckpoint(repo, home, cfg, entries)
			if err != nil {
				return Result{}, fmt.Errorf("checkpoint: %w", err)
			}
			checkpoint = cp
		}
	}

	var restored []string
	for _, rel := range planned {
		meta, inManifest := targetM.Files[rel]
		abs := filepath.Join(home, filepath.FromSlash(rel))
		tb, tok := atBytes(repo, commit, rel)
		if !tok && !inManifest {
			continue
		}
		if !tok {
			// restore deletion
			os.Remove(abs)
			restored = append(restored, rel)
			continue
		}
		if meta.Symlink {
			os.Remove(abs)
			_ = os.MkdirAll(filepath.Dir(abs), 0o755)
			if err := os.Symlink(meta.Target, abs); err != nil {
				return Result{}, fmt.Errorf("restore %s: %w", rel, err)
			}
			restored = append(restored, rel)
			continue
		}
		// symlink in repo (non-manifest path?) — git stores links; bytes are target content; write regular file
		os.Remove(abs)
		_ = os.MkdirAll(filepath.Dir(abs), 0o755)
		tmp := abs + ".kage-tmp"
		mode := os.FileMode(0o644)
		if cfg.Snapshot.PreservePermissions && meta.Mode != 0 {
			mode = os.FileMode(meta.Mode).Perm()
		}
		if err := os.WriteFile(tmp, tb, 0o600); err != nil {
			return Result{}, fmt.Errorf("restore %s: %w", rel, err)
		}
		_ = os.Chmod(tmp, mode)
		if err := os.Rename(tmp, abs); err != nil {
			os.Remove(tmp)
			return Result{}, fmt.Errorf("restore %s: %w", rel, err)
		}
		restored = append(restored, rel)
	}

	appendRestoreLog(home, commit, opts, restored, skipped)
	return Result{Restored: restored, Skipped: skipped, Checkpoint: checkpoint, Preview: preview}, nil
}

func checkpointNeeded(repo, head string, entries []discovery.Entry, home string) bool {
	base := manifestAt(repo, head)
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
	if len(current) != len(base.Files) {
		return true
	}
	for rel, h := range current {
		old, ok := base.Files[rel]
		if !ok || old.Hash != h {
			return true
		}
	}
	return false
}

func makeCheckpoint(repo, home string, cfg config.Config, entries []discovery.Entry) (string, error) {
	files := map[string]snapshot.FileMeta{}
	for _, e := range entries {
		if e.External {
			continue
		}
		var hash, target string
		if e.Symlink {
			target = e.SymlinkTarget
			h := sha256.Sum256([]byte("link:" + target))
			hash = hex.EncodeToString(h[:])
		} else {
			h, err := filesystem.HashFile(e.Abs)
			if err != nil {
				continue
			}
			hash = h
		}
		files[e.Rel] = snapshot.FileMeta{Hash: hash, Mode: uint32(e.Mode.Perm()), Symlink: e.Symlink, Target: target}
		// stage into repo
		dst := filepath.Join(repo, filepath.FromSlash(e.Rel))
		if e.Symlink {
			os.Remove(dst)
			_ = os.MkdirAll(filepath.Dir(dst), 0o755)
			_ = os.Symlink(target, dst)
			continue
		}
		if err := filesystem.CopyFile(e.Abs, dst, e.Mode); err != nil {
			return "", err
		}
	}
	// remove repo files absent from working (relative to HEAD manifest)
	head := gitstore.Head(repo)
	base := manifestAt(repo, head)
	for rel := range base.Files {
		if _, ok := files[rel]; !ok {
			os.Remove(filepath.Join(repo, filepath.FromSlash(rel)))
		}
	}
	mdata, _ := json.MarshalIndent(snapshot.Manifest{Files: files}, "", "  ")
	_ = os.MkdirAll(filepath.Join(repo, ".kage"), 0o755)
	_ = os.WriteFile(filepath.Join(repo, ".kage", "manifest.json"), mdata, 0o644)
	if err := gitstore.AddAll(repo); err != nil {
		return "", err
	}
	msg := fmt.Sprintf("kage checkpoint pre-restore %s", time.Now().UTC().Format(time.RFC3339))
	if err := gitstore.Commit(repo, msg); err != nil {
		return "", err
	}
	return gitstore.Head(repo), nil
}

func appendRestoreLog(home, commit string, opts Options, restored, skipped []string) {
	dir := snapshot.StateDir(home)
	_ = os.MkdirAll(dir, 0o700)
	rec := map[string]any{
		"ts": time.Now().UTC().Format(time.RFC3339), "commit": commit,
		"snapshot": opts.SnapshotID, "files": restored, "skipped": skipped,
	}
	data, _ := json.Marshal(rec)
	f, err := os.OpenFile(filepath.Join(dir, "restore.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(data, '\n'))
}
