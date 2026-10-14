package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/metadata"
	"github.com/RakhaYandra/kage/internal/snapshot"
	"github.com/RakhaYandra/kage/internal/watcher"
)

// watchLogPath is ~/.local/share/kage/watch.log.
func watchLogPath(home string) string {
	return filepath.Join(snapshot.StateDir(home), "watch.log")
}

func appendWatchLog(home string, e watcher.Event) {
	_ = os.MkdirAll(snapshot.StateDir(home), 0o700)
	data, _ := json.Marshal(map[string]string{
		"ts": e.At.UTC().Format(time.RFC3339), "rel": e.Rel, "kind": e.Kind,
	})
	f, err := os.OpenFile(watchLogPath(home), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(data, '\n'))
}

func newWatch() *cobra.Command {
	var once, autoSnapshot bool
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch tracked paths for changes (opt-in)",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			cfg, err := config.Load(resolveConfigPath(home), home)
			if err != nil {
				return fail(2, "%v", err)
			}
			if once {
				changed, err := watcher.Reconcile(home, nil)
				if err != nil {
					return fail(1, "%v", err)
				}
				out := cmd.OutOrStdout()
				if len(changed) == 0 {
					fmt.Fprintln(out, "clean")
					return nil
				}
				for _, c := range changed {
					fmt.Fprintln(out, c)
				}
				return fail(2, "drift: %d files changed", len(changed))
			}
			if !cfg.Watcher.Enabled {
				return fail(1, "watcher disabled (set watcher.enabled: true in config)")
			}
			auto := cfg.Watcher.AutoSnapshot || autoSnapshot
			quiet, err := config.QuietDuration(cfg)
			if err != nil {
				return fail(2, "%v", err)
			}
			ctx, _ := signalCtx()
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "watching (Ctrl-C to stop)")
			var mu sync.Mutex // debounce timers fire concurrently; keep git ops serial
			st := &autoState{quiet: quiet}
			w := &watcher.Watcher{OnEvent: func(e watcher.Event) {
				mu.Lock()
				defer mu.Unlock()
				appendWatchLog(home, e)
				fmt.Fprintf(out, "%s %s\n", e.Kind, e.Rel)
				if auto {
					if !st.allow(time.Now(), lastSnapshotAt(home)) {
						return
					}
					if _, err := snapshot.Run(home, snapshot.Options{Message: "watch auto"}); err != nil {
						if st.shouldLogFail(time.Now()) {
							debugf("auto-snapshot: %v", err)
						}
					}
				}
			}}
			return w.Run(ctx, home)
		},
	}
	cmd.Flags().BoolVar(&once, "once", false, "reconcile once and exit")
	cmd.Flags().BoolVar(&autoSnapshot, "auto-snapshot", false, "snapshot on change (overrides config off)")
	return cmd
}

// autoState gates auto-snapshots (quiet period) and failure logs (1/min).
type autoState struct {
	quiet       time.Duration
	lastFailLog time.Time
}

// allow reports whether an auto-snapshot may run given the newest known
// snapshot timestamp (zero = none yet).
func (s *autoState) allow(now, lastSnap time.Time) bool {
	if lastSnap.IsZero() || s.quiet <= 0 {
		return true
	}
	return now.Sub(lastSnap) >= s.quiet
}

// shouldLogFail rate-limits failure logging to one line per minute.
func (s *autoState) shouldLogFail(now time.Time) bool {
	if now.Sub(s.lastFailLog) < time.Minute {
		return false
	}
	s.lastFailLog = now
	return true
}

// lastSnapshotAt returns the newest snapshot timestamp (zero if none).
func lastSnapshotAt(home string) time.Time {
	list, err := metadata.Open(snapshot.StateDir(home)).List()
	if err != nil || len(list) == 0 {
		return time.Time{}
	}
	ts, _ := time.Parse(time.RFC3339, list[len(list)-1].CreatedAt)
	return ts
}

// watcherLine reports "watcher: ..." for status.
func watcherLine(home string) string {
	cfg, err := config.Load(resolveConfigPath(home), home)
	if err != nil {
		return "watcher: unknown"
	}
	if !cfg.Watcher.Enabled {
		return "watcher: disabled"
	}
	if fi, err := os.Stat(watchLogPath(home)); err == nil && time.Since(fi.ModTime()) < time.Minute {
		return "watcher: active"
	}
	return "watcher: enabled, idle"
}
