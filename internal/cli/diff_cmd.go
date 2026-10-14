package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/diff"
	"github.com/RakhaYandra/kage/internal/metadata"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

func newDiff() *cobra.Command {
	var last, from, to, snap int
	var file string
	var summaryOnly bool
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare working state or two snapshots",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			var res diff.Result
			var err error
			title := "Current state vs latest snapshot"
			switch {
			case from > 0 && to > 0:
				res, err = diff.History(home, from, to, file)
				title = fmt.Sprintf("Snapshot #%d vs #%d", from, to)
			case last > 1:
				var fromID, toID int
				fromID, toID, err = lastRange(home, last)
				if err == nil {
					res, err = diff.History(home, fromID, toID, file)
					title = fmt.Sprintf("Snapshot #%d vs #%d", fromID, toID)
				}
			case snap > 0:
				res, err = diff.Working(home, snap, file)
				title = fmt.Sprintf("Current state vs Snapshot #%d", snap)
			default:
				res, err = diff.Working(home, 0, file)
			}
			if err != nil {
				return fail(1, "%v", err)
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Kage Diff")
			fmt.Fprintln(out, title)
			fmt.Fprintln(out)
			for _, f := range res.Files {
				fmt.Fprintf(out, "%-9s ~/%s\n", f.Kind, f.Path)
			}
			fmt.Fprintf(out, "\nSummary: %d modified, %d added, %d deleted\n",
				res.Modified, res.Added, res.Deleted)
			if !summaryOnly {
				for _, f := range res.Files {
					if f.Binary {
						fmt.Fprintf(out, "\n[%s] (binary, not shown)\n", f.Path)
						continue
					}
					if f.Unified != "" {
						fmt.Fprintf(out, "\n--- %s (%s) ---\n%s", f.Path, f.Kind, redactForDisplay(home, f.Unified))
						if f.Truncated {
							fmt.Fprintln(out, "[truncated]")
						}
					}
				}
			}
			if res.Added+res.Modified+res.Deleted > 0 {
				return fail(2, "changes detected")
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&last, "last", 0, "history of last N snapshots (N>=2); 1 = working vs latest")
	cmd.Flags().IntVar(&from, "from", 0, "history diff from snapshot id (needs --to)")
	cmd.Flags().IntVar(&to, "to", 0, "history diff to snapshot id")
	cmd.Flags().IntVar(&snap, "snapshot", 0, "working diff vs snapshot id")
	cmd.Flags().StringVar(&file, "file", "", "limit to one file (home-relative, e.g. .zshrc)")
	cmd.Flags().BoolVar(&summaryOnly, "summary-only", false, "print summary without content")
	return cmd
}

func lastRange(home string, n int) (int, int, error) {
	store := metadata.Open(snapshot.StateDir(home))
	list, err := store.List()
	if err != nil {
		return 0, 0, err
	}
	if len(list) < n {
		return 0, 0, fail(1, "only %d snapshots, want last %d", len(list), n)
	}
	return list[len(list)-n].ID, list[len(list)-1].ID, nil
}
