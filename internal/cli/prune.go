package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/metadata"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

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
			return nil
		},
	}
	cmd.Flags().IntVar(&keepFlag, "keep", 0, "keep last N records (default: snapshot.keep_last)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview without writing")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm without prompting")
	return cmd
}
