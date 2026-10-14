package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/restore"
)

func newRestore() *cobra.Command {
	var snap int
	var commit string
	var files []string
	var dryRun, yes, nonInteractive bool
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore files from a snapshot (with preview + checkpoint)",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			// preview first (dry-run has no side effects)
			preview, err := restore.Run(home, restore.Options{
				SnapshotID: snap, Commit: commit, Files: files, DryRun: true,
			})
			var ce *restore.ConflictError
			if errors.As(err, &ce) {
				return fail(3, "%v", ce)
			}
			if err != nil {
				return fail(1, "%v", err)
			}
			out := cmd.OutOrStdout()
			if preview.Preview != "" {
				fmt.Fprintln(out, redactForDisplay(home, preview.Preview))
				fmt.Fprintln(out)
			}
			if len(preview.Restored) == 0 {
				fmt.Fprintln(out, "nothing to restore")
				return nil
			}
			note := ""
			if len(preview.Skipped) > 0 {
				note = fmt.Sprintf(" (%d skipped)", len(preview.Skipped))
			}
			fmt.Fprintf(out, "will restore %d file(s)%s\n", len(preview.Restored), note)
			if dryRun {
				return nil
			}
			if !yes && !nonInteractive {
				fmt.Fprint(out, "proceed? [y/N] ")
				var ans string
				fmt.Scanln(&ans)
				ans = strings.ToLower(strings.TrimSpace(ans))
				if ans != "y" && ans != "yes" {
					return fail(1, "aborted")
				}
			} else if nonInteractive && !yes {
				return fail(1, "refusing destructive restore without --yes in non-interactive mode")
			}
			res, err := restore.Run(home, restore.Options{
				SnapshotID: snap, Commit: commit, Files: files, Yes: true,
			})
			if errors.As(err, &ce) {
				return fail(3, "%v", ce)
			}
			if err != nil {
				return fail(1, "%v", err)
			}
			fmt.Fprintf(out, "restored %d file(s)", len(res.Restored))
			if res.Checkpoint != "" {
				short := res.Checkpoint
				if len(short) > 7 {
					short = short[:7]
				}
				fmt.Fprintf(out, " (checkpoint %s; undo: kage restore --commit %s)", short, res.Checkpoint)
			}
			fmt.Fprintln(out)
			return nil
		},
	}
	cmd.Flags().IntVar(&snap, "snapshot", 0, "snapshot id (default: latest)")
	cmd.Flags().StringVar(&commit, "commit", "", "restore from explicit commit (e.g. checkpoint hash)")
	cmd.Flags().StringArrayVar(&files, "file", nil, "file to restore (repeatable, home-relative)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview without writing")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm destructive restore")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "fail instead of prompting")
	return cmd
}
