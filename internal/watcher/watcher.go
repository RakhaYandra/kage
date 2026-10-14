// Package watcher detects config changes via fsnotify with debounce.
// It never writes user files; by default it only reports. Auto-snapshot
// converges (snapshot skips when unchanged), so no feedback loop is possible.
package watcher

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/discovery"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

// Debounce coalesces editor save bursts into one event.
const Debounce = 500 * time.Millisecond

// ReconcileInterval re-scans for events missed while idle/off.
const ReconcileInterval = 30 * time.Second

// Event is one debounced filesystem change.
type Event struct {
	Rel  string // home-relative slash path
	Abs  string
	Kind string // create|modify|delete|rename
	At   time.Time
}

// Watcher monitors tracked paths.
type Watcher struct {
	OnEvent func(Event)
	// Debounce overrides Debounce; ReconcileEvery overrides ReconcileInterval (0 = default, <0 = off).
	Debounce       time.Duration
	ReconcileEvery time.Duration
}

func (w *Watcher) debounce() time.Duration {
	if w.Debounce > 0 {
		return w.Debounce
	}
	return Debounce
}

// roots returns (recursive dirs, shallow dirs) to watch.
func roots(home string, cfg config.Config) (rec, shallow []string) {
	seen := map[string]bool{}
	repo := config.Expand(cfg.Repository.Path, home)
	state := snapshot.StateDir(home)
	addRec := func(d string) {
		d = filepath.Clean(d)
		if seen[d] || d == repo || d == state {
			return
		}
		seen[d] = true
		rec = append(rec, d)
	}
	addShallow := func(d string) {
		d = filepath.Clean(d)
		if seen[d] {
			return
		}
		seen[d] = true
		shallow = append(shallow, d)
	}
	for _, t := range cfg.Tracking.Paths {
		abs := config.Normalize(t, home)
		// never watch repo/state even if user tracks them
		if abs == repo || abs == state ||
			strings.HasPrefix(abs, repo+string(os.PathSeparator)) ||
			strings.HasPrefix(abs, state+string(os.PathSeparator)) {
			continue
		}
		fi, err := os.Lstat(abs)
		if err != nil {
			continue
		}
		if fi.IsDir() && fi.Mode()&os.ModeSymlink == 0 {
			addRec(abs)
		} else {
			addShallow(filepath.Dir(abs))
		}
	}
	return rec, shallow
}

func addRecursive(fw *fsnotify.Watcher, root string) {
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			_ = fw.Add(p)
		}
		return nil
	})
}

func relOf(home, abs string) string {
	rel, err := filepath.Rel(home, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}

// Run blocks until ctx ends.
func (w *Watcher) Run(ctx context.Context, home string) error {
	cfg, err := config.Load(config.DefaultPath(home), home)
	if err != nil {
		return err
	}
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer fw.Close()
	rec, shallow := roots(home, cfg)
	for _, d := range rec {
		addRecursive(fw, d)
	}
	for _, d := range shallow {
		_ = fw.Add(d)
	}

	var mu sync.Mutex
	timers := map[string]*time.Timer{}
	pendingKind := map[string]string{}
	emit := func(abs, kind string) {
		rel := relOf(home, abs)
		// shallow-watched dirs: only report the exact tracked files, but
		// allow new dotfiles at that level (coverage signal)
		if discovery.Excluded(cfg.Tracking.Exclude, rel) {
			return
		}
		if w.OnEvent != nil {
			w.OnEvent(Event{Rel: rel, Abs: abs, Kind: kind, At: time.Now()})
		}
	}
	arm := func(abs, kind string) {
		mu.Lock()
		defer mu.Unlock()
		// latest kind wins (e.g. create→modify)
		if _, ok := pendingKind[abs]; ok {
			pendingKind[abs] = kind
			if t, ok := timers[abs]; ok {
				t.Reset(w.debounce())
			}
			return
		}
		pendingKind[abs] = kind
		timers[abs] = time.AfterFunc(w.debounce(), func() {
			mu.Lock()
			k := pendingKind[abs]
			delete(pendingKind, abs)
			delete(timers, abs)
			mu.Unlock()
			emit(abs, k)
		})
	}

	reconcileEvery := w.ReconcileEvery
	if reconcileEvery == 0 {
		reconcileEvery = ReconcileInterval
	}
	var ticker <-chan time.Time
	if reconcileEvery > 0 {
		t := time.NewTicker(reconcileEvery)
		defer t.Stop()
		ticker = t.C
	}

	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			for _, t := range timers {
				t.Stop()
			}
			mu.Unlock()
			return nil
		case err := <-fw.Errors:
			_ = err // transient; reconcile covers gaps
		case ev := <-fw.Events:
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			kind := "modify"
			switch {
			case ev.Op&fsnotify.Create != 0:
				kind = "create"
			case ev.Op&fsnotify.Remove != 0:
				kind = "delete"
			case ev.Op&fsnotify.Rename != 0:
				kind = "rename"
			}
			if kind == "create" {
				if st, err := os.Stat(ev.Name); err == nil && st.IsDir() {
					// watch new subdirs of recursive roots only
					for _, r := range rec {
						if ev.Name == r || strings.HasPrefix(ev.Name, r+string(os.PathSeparator)) {
							addRecursive(fw, ev.Name)
							break
						}
					}
				}
			}
			arm(ev.Name, kind)
		case <-ticker:
			changed, err := Reconcile(home, nil)
			if err != nil {
				continue
			}
			for _, c := range changed {
				kind := "modify"
				if strings.HasSuffix(c, "(deleted)") {
					kind = "delete"
				}
				rel := strings.TrimSpace(strings.SplitN(c, "(", 2)[0])
				arm(filepath.Join(home, filepath.FromSlash(rel)), kind)
			}
		}
	}
}
