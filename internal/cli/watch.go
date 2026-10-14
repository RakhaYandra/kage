package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/config"
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
			ctx, _ := signalCtx()
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "watching (Ctrl-C to stop)")
			w := &watcher.Watcher{OnEvent: func(e watcher.Event) {
				appendWatchLog(home, e)
				fmt.Fprintf(out, "%s %s\n", e.Kind, e.Rel)
				if auto {
					if _, err := snapshot.Run(home, snapshot.Options{Message: "watch auto"}); err != nil {
						debugf("auto-snapshot: %v", err)
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
