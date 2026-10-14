// Package diff compares snapshots and working state.
package diff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/discovery"
	"github.com/RakhaYandra/kage/internal/filesystem"
	"github.com/RakhaYandra/kage/internal/gitstore"
	"github.com/RakhaYandra/kage/internal/metadata"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

// Kind of change.
type Kind string

const (
	Added    Kind = "added"
	Modified Kind = "modified"
	Deleted  Kind = "deleted"
)

// FileDiff is one file's comparison.
type FileDiff struct {
	Path      string
	Kind      Kind
	Unified   string
	Binary    bool
	Truncated bool
}

// Result is a full comparison.
type Result struct {
	Files    []FileDiff
	Added    int
	Modified int
	Deleted  int
}

const maxFileBytes = 200 * 1024

func isBinary(b []byte) bool {
	n := len(b)
	if n > 8000 {
		n = 8000
	}
	return bytes.IndexByte(b[:n], 0) >= 0
}

func commitFor(home string, id int) (string, error) {
	store := metadata.Open(snapshot.StateDir(home))
	list, err := store.List()
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", fmt.Errorf("no snapshots yet")
	}
	if id == 0 {
		return list[len(list)-1].GitCommit, nil
	}
	for _, s := range list {
		if s.ID == id {
			return s.GitCommit, nil
		}
	}
	return "", fmt.Errorf("snapshot #%d not found", id)
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

// unified builds a minimal unified diff with 3 context lines.
func unified(oldName, newName string, a, b []byte) (string, bool) {
	if len(a) > maxFileBytes || len(b) > maxFileBytes {
		return "(content too large, truncated)", true
	}
	al := strings.Split(string(a), "\n")
	bl := strings.Split(string(b), "\n")
	if len(al) > 3000 || len(bl) > 3000 {
		return "(too many lines, truncated)", true
	}
	n, m := len(al), len(bl)
	// LCS table (suffix lengths)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if al[i] == bl[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	type op struct {
		kind string // " ", "-", "+"
		line string
	}
	var ops []op
	i, j := 0, 0
	for i < n && j < m {
		if al[i] == bl[j] {
			ops = append(ops, op{" ", al[i]})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, op{"-", al[i]})
			i++
		} else {
			ops = append(ops, op{"+", bl[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, op{"-", al[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, op{"+", bl[j]})
	}
	changed := false
	for _, o := range ops {
		if o.kind != " " {
			changed = true
		}
	}
	if !changed {
		return "", false
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", oldName, newName)
	for _, o := range ops {
		sb.WriteString(o.kind + o.line + "\n")
	}
	return sb.String(), false
}

func fileDiff(path string, oldBytes, newBytes []byte, oldOK, newOK bool) FileDiff {
	switch {
	case !oldOK && newOK:
		if isBinary(newBytes) {
			return FileDiff{Path: path, Kind: Added, Binary: true}
		}
		if len(newBytes) > maxFileBytes {
			return FileDiff{Path: path, Kind: Added, Truncated: true}
		}
		return FileDiff{Path: path, Kind: Added, Unified: "(new file, " + fmt.Sprint(len(newBytes)) + " bytes)"}
	case oldOK && !newOK:
		return FileDiff{Path: path, Kind: Deleted}
	default:
		if bytes.Equal(oldBytes, newBytes) {
			return FileDiff{}
		}
		if isBinary(oldBytes) || isBinary(newBytes) {
			return FileDiff{Path: path, Kind: Modified, Binary: true}
		}
		u, trunc := unified("a/"+path, "b/"+path, oldBytes, newBytes)
		return FileDiff{Path: path, Kind: Modified, Unified: u, Truncated: trunc}
	}
}

// History compares two snapshots.
func History(home string, fromID, toID int, fileFilter string) (Result, error) {
	cfg, err := config.Load(config.DefaultPath(home), home)
	if err != nil {
		return Result{}, err
	}
	repo := config.Expand(cfg.Repository.Path, home)
	fromCommit, err := commitFor(home, fromID)
	if err != nil {
		return Result{}, err
	}
	toCommit, err := commitFor(home, toID)
	if err != nil {
		return Result{}, err
	}
	fromM := manifestAt(repo, fromCommit)
	toM := manifestAt(repo, toCommit)
	union := map[string]bool{}
	for rel := range fromM.Files {
		union[rel] = true
	}
	for rel := range toM.Files {
		union[rel] = true
	}
	var res Result
	for rel := range union {
		if fileFilter != "" && rel != fileFilter && filepath.ToSlash(rel) != filepath.ToSlash(fileFilter) {
			continue
		}
		oldBytes, oerr := gitstore.ShowAt(repo, fromCommit, filepath.ToSlash(rel))
		newBytes, nerr := gitstore.ShowAt(repo, toCommit, filepath.ToSlash(rel))
		fd := fileDiff(rel, oldBytes, newBytes, oerr == nil, nerr == nil)
		if fd.Kind == "" {
			continue
		}
		res.Files = append(res.Files, fd)
	}
	finish(&res)
	return res, nil
}

// Working compares working files to a snapshot (0 = latest).
func Working(home string, id int, fileFilter string) (Result, error) {
	cfg, err := config.Load(config.DefaultPath(home), home)
	if err != nil {
		return Result{}, err
	}
	repo := config.Expand(cfg.Repository.Path, home)
	commit, err := commitFor(home, id)
	if err != nil {
		return Result{}, err
	}
	base := manifestAt(repo, commit)
	entries, err := discovery.Scan(cfg.Tracking.Paths, cfg.Tracking.Exclude, home)
	if err != nil {
		return Result{}, err
	}
	current := map[string]bool{}
	var res Result
	for _, e := range entries {
		if e.External {
			continue
		}
		rel := e.Rel
		if fileFilter != "" && rel != fileFilter {
			continue
		}
		current[rel] = true
		var newBytes []byte
		var newOK bool
		if e.Symlink {
			newBytes = []byte("link:" + e.SymlinkTarget)
			newOK = true
		} else {
			newBytes, err = os.ReadFile(e.Abs)
			newOK = err == nil
		}
		oldBytes, oerr := gitstore.ShowAt(repo, commit, filepath.ToSlash(rel))
		// symlinks stored as link files; compare hash-level via bytes when not symlink in repo?
		_ = filesystem.HashFile // keep import used for future hashing path
		fd := fileDiff(rel, oldBytes, newBytes, oerr == nil, newOK)
		if fd.Kind == "" {
			continue
		}
		// repo stores symlinks as symlinks; git show follows to target content — instead compare via manifest hash when symlink involved
		if e.Symlink {
			if old, ok := base.Files[rel]; ok && old.Symlink {
				if old.Target == e.SymlinkTarget {
					continue
				}
				fd = FileDiff{Path: rel, Kind: Modified, Unified: fmt.Sprintf("symlink target: %s -> %s", old.Target, e.SymlinkTarget)}
			}
		}
		res.Files = append(res.Files, fd)
	}
	for rel := range base.Files {
		if current[rel] {
			continue
		}
		if fileFilter != "" && rel != fileFilter {
			continue
		}
		res.Files = append(res.Files, FileDiff{Path: rel, Kind: Deleted})
	}
	finish(&res)
	return res, nil
}

func finish(res *Result) {
	sort.Slice(res.Files, func(i, j int) bool { return res.Files[i].Path < res.Files[j].Path })
	for _, f := range res.Files {
		switch f.Kind {
		case Added:
			res.Added++
		case Modified:
			res.Modified++
		case Deleted:
			res.Deleted++
		}
	}
}
