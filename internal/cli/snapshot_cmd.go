package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/snapshot"
)

func newSnapshot() *cobra.Command {
	var message string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Take a snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			res, err := snapshot.Run(home, snapshot.Options{Message: message, DryRun: dryRun})
			if err != nil {
				return fail(1, "%v", err)
			}
			if res.Skipped {
				fmt.Fprintln(cmd.OutOrStdout(), "no changes, skipped")
				return nil
			}
			if jsonOut {
				fmt.Fprintf(cmd.OutOrStdout(), "{\"id\":%d,\"added\":%d,\"modified\":%d,\"deleted\":%d,\"hash\":\"%s\"}\n",
					res.ID, res.Added, res.Modified, res.Deleted, res.Hash)
			} else if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "dry-run: %d added, %d modified, %d deleted\n", res.Added, res.Modified, res.Deleted)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "snapshot #%d: %d added, %d modified, %d deleted\n", res.ID, res.Added, res.Modified, res.Deleted)
			}
			if dryRun && res.Added+res.Modified+res.Deleted > 0 {
				return fail(2, "changes detected (dry-run)")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "snapshot message")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview without writing")
	return cmd
}
