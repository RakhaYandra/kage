// Package discovery scans tracked paths into file entries.
package discovery

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/RakhaYandra/kage/internal/config"
)

// Entry is one tracked file or symlink.
type Entry struct {
	Rel           string // slash-separated, relative to home
	Abs           string
	Mode          os.FileMode
	Symlink       bool
	SymlinkTarget string
	External      bool
}

func slash(p string) string { return filepath.ToSlash(p) }

func segMatch(pat, s string) bool {
	ok, _ := path.Match(pat, s)
	return ok
}

func matchSegs(pat, p []string) bool {
	if len(pat) == 0 {
		return len(p) == 0
	}
	if pat[0] == "**" {
		for k := 0; k <= len(p); k++ {
			if matchSegs(pat[1:], p[k:]) {
				return true
			}
		}
		return false
	}
	if len(p) == 0 {
		return false
	}
	return segMatch(pat[0], p[0]) && matchSegs(pat[1:], p[1:])
}

// Match reports whether rel matches a ** glob pattern.
func Match(pattern, rel string) bool {
	pat := slash(pattern)
	rel = slash(rel)
	return matchSegs(strings.Split(pat, "/"), strings.Split(rel, "/"))
}

// Excluded reports whether rel matches any pattern.
func Excluded(patterns []string, rel string) bool {
	for _, p := range patterns {
		if Match(p, rel) {
			return true
		}
		// also try basename-anchored match for patterns like **/id_*
		if Match(p, path.Base(rel)) && strings.Contains(p, "**/") {
			return true
		}
	}
	return false
}

func relToHome(home, abs string) string {
	rel, err := filepath.Rel(home, abs)
	if err != nil || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return slash(abs)
	}
	return slash(rel)
}

// Scan expands tracked paths into entries.
func Scan(tracked, exclude []string, home string) ([]Entry, error) {
	var out []Entry
	seen := map[string]bool{}
	addFile := func(abs string, fi os.FileInfo) {
		rel := relToHome(home, abs)
		if seen[abs] || Excluded(exclude, rel) {
			return
		}
		seen[abs] = true
		e := Entry{Rel: rel, Abs: abs, Mode: fi.Mode()}
		if fi.Mode()&os.ModeSymlink != 0 {
			e.Symlink = true
			target, err := os.Readlink(abs)
			if err == nil {
				e.SymlinkTarget = target
				resolved := target
				if !filepath.IsAbs(resolved) {
					resolved = filepath.Join(filepath.Dir(abs), target)
				}
				resolved = filepath.Clean(resolved)
				r, err := filepath.Rel(home, resolved)
				if err != nil || strings.HasPrefix(r, ".."+string(os.PathSeparator)) || r == ".." {
					e.External = true
				}
			}
		}
		out = append(out, e)
	}

	for _, t := range tracked {
		abs := config.Normalize(t, home)
		fi, err := os.Lstat(abs)
		if err != nil {
			continue // missing path: skip (coverage reports it later)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			addFile(abs, fi)
			continue
		}
		if !fi.IsDir() {
			addFile(abs, fi)
			continue
		}
		_ = filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if p != abs {
					if li, err := os.Lstat(p); err == nil && li.Mode()&os.ModeSymlink != 0 {
						addFile(p, li)
						return filepath.SkipDir
					}
				}
				return nil
			}
			li, err := os.Lstat(p)
			if err != nil {
				return nil
			}
			addFile(p, li)
			return nil
		})
	}
	return out, nil
}
