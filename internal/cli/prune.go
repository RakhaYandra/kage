package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/metadata"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

// trimLog keeps the last keep lines of path; missing file is skipped.
func trimLog(path string, keep int) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	lines := strings.Split(string(data), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) <= keep {
		return 0, nil
	}
	dropped := len(lines) - keep
	rest := strings.Join(lines[dropped:], "\n") + "\n"
	tmp := path + ".kage-tmp"
	if err := os.WriteFile(tmp, []byte(rest), 0o600); err != nil {
		return 0, err
	}
	return dropped, os.Rename(tmp, path)
}

func newPrune() *cobra.Command {
	var keep, keepFlag int
	var dryRun, yes bool
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Drop old snapshot records (git objects stay until gc)",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			cfg, err := config.Load(resolveConfigPath(home), home)
			if err != nil {
				return fail(2, "%v", err)
			}
			keep = keepFlag
			if !cmd.Flags().Changed("keep") {
				keep = cfg.Snapshot.KeepLast
			}
			if keep <= 0 {
				return fail(1, "nothing to prune (set --keep N or snapshot.keep_last)")
			}
			store := metadata.Open(snapshot.StateDir(home))
			list, err := store.List()
			if err != nil {
				return fail(1, "%v", err)
			}
			if len(list) <= keep {
				fmt.Fprintln(cmd.OutOrStdout(), "nothing to prune")
				return nil
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "will drop %d record(s), keep last %d (git objects remain)\n", len(list)-keep, keep)
			if dryRun {
				return nil
			}
			if !yes {
				fmt.Fprint(out, "proceed? [y/N] ")
				var ans string
				fmt.Scanln(&ans)
				if ans = strings.ToLower(strings.TrimSpace(ans)); ans != "y" && ans != "yes" {
					return fail(1, "aborted")
				}
			}
			dropped, err := store.Prune(keep)
			if err != nil {
				return fail(1, "%v", err)
			}
			fmt.Fprintf(out, "dropped %d record(s); run git gc in the repository to reclaim objects\n", dropped)
			state := snapshot.StateDir(home)
			for _, name := range []string{"watch.log", "restore.log"} {
				n, err := trimLog(filepath.Join(state, name), keep)
				if err != nil {
					return fail(1, "trim %s: %v", name, err)
				}
				if n > 0 {
					fmt.Fprintf(out, "trimmed %d line(s) from %s\n", n, name)
				}
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&keepFlag, "keep", 0, "keep last N records (default: snapshot.keep_last)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview without writing")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm without prompting")
	return cmd
}
