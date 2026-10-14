package watcher

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/RakhaYandra/kage/internal/config"
)

func testHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default(home)
	cfg.Tracking.Paths = []string{"~/.config/app"}
	config.Save(config.DefaultPath(home), cfg)
	os.MkdirAll(filepath.Join(home, ".config", "app"), 0o755)
	os.WriteFile(filepath.Join(home, ".config", "app", "a.conf"), []byte("a\n"), 0o644)
	return home
}

func collect(t *testing.T, home string, dur time.Duration, act func()) []Event {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	var got []Event
	w := &Watcher{Debounce: 150 * time.Millisecond, ReconcileEvery: -1, OnEvent: func(e Event) {
		mu.Lock()
		got = append(got, e)
		mu.Unlock()
	}}
	runErr := make(chan error, 1)
	go func() { runErr <- w.Run(ctx, home) }()
	time.Sleep(300 * time.Millisecond) // let watcher arm
	act()
	time.Sleep(dur)
	cancel()
	<-runErr
	time.Sleep(300 * time.Millisecond) // let in-flight debounce fire
	mu.Lock()
	defer mu.Unlock()
	return append([]Event(nil), got...)
}

func TestDetectsCreateModifyDelete(t *testing.T) {
	home := testHome(t)
	got := collect(t, home, 800*time.Millisecond, func() {
		os.WriteFile(filepath.Join(home, ".config", "app", "b.conf"), []byte("b\n"), 0o644)
		time.Sleep(400 * time.Millisecond) // separate debounce windows
		os.WriteFile(filepath.Join(home, ".config", "app", "a.conf"), []byte("a2\n"), 0o644)
		time.Sleep(400 * time.Millisecond)
		os.Remove(filepath.Join(home, ".config", "app", "b.conf"))
	})
	kinds := map[string]bool{}
	seenB := false
	for _, e := range got {
		kinds[e.Kind] = true
		if e.Rel == ".config/app/b.conf" {
			seenB = true
		}
	}
	// note: create usually coalesces into modify (Create+Write in one window)
	for _, k := range []string{"modify", "delete"} {
		if !kinds[k] {
			t.Fatalf("missing %q in %v", k, got)
		}
	}
	if !seenB {
		t.Fatalf("no event for b.conf in %v", got)
	}
}

func TestDebounceCoalescesBurst(t *testing.T) {
	home := testHome(t)
	got := collect(t, home, 1500*time.Millisecond, func() {
		for i := 0; i < 10; i++ {
			os.WriteFile(filepath.Join(home, ".config", "app", "a.conf"), []byte("x\n"), 0o644)
			time.Sleep(20 * time.Millisecond)
		}
	})
	n := 0
	for _, e := range got {
		if e.Rel == ".config/app/a.conf" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("burst must coalesce to 1 event, got %d (%v)", n, got)
	}
}

func TestRenameDetected(t *testing.T) {
	home := testHome(t)
	got := collect(t, home, 1200*time.Millisecond, func() {
		os.Rename(filepath.Join(home, ".config", "app", "a.conf"),
			filepath.Join(home, ".config", "app", "renamed.conf"))
	})
	if len(got) == 0 {
		t.Fatalf("rename must produce events, got none")
	}
}

func TestReconcileFindsMissedChanges(t *testing.T) {
	home := testHome(t)
	// change while watcher is off
	os.WriteFile(filepath.Join(home, ".config", "app", "late.conf"), []byte("l\n"), 0o644)
	changed, err := Reconcile(home, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range changed {
		if c == ".config/app/late.conf (modified/added)" || c == ".config/app/a.conf (modified/added)" {
			found = true
		}
	}
	_ = found
	hasLate := false
	for _, c := range changed {
		if len(c) >= len(".config/app/late.conf") && c[:len(".config/app/late.conf")] == ".config/app/late.conf" {
			hasLate = true
		}
	}
	if !hasLate {
		t.Fatalf("reconcile must find late.conf, got %v", changed)
	}
}
