package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/snapshot"
)

func newStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show working state vs last snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			st, err := snapshot.GetStatus(home)
			if err != nil {
				return fail(1, "%v", err)
			}
			if !st.Drift {
				fmt.Fprintln(cmd.OutOrStdout(), "clean")
				return nil
			}
			for _, c := range st.Changed {
				fmt.Fprintln(cmd.OutOrStdout(), c)
			}
			return fail(2, "drift: %d files changed", len(st.Changed))
		},
	}
}
