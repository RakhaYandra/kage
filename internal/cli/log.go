package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/metadata"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

func newLog() *cobra.Command {
	var n int
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show snapshot history",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			store := metadata.Open(snapshot.StateDir(home))
			list, err := store.List()
			if err != nil {
				return fail(1, "%v", err)
			}
			if n > 0 && len(list) > n {
				list = list[len(list)-n:]
			}
			out := cmd.OutOrStdout()
			if jsonOut {
				for _, s := range list {
					fmt.Fprintf(out, "{\"id\":%d,\"created_at\":\"%s\",\"message\":%q,\"commit\":%q,\"added\":%d,\"modified\":%d,\"deleted\":%d}\n",
						s.ID, s.CreatedAt, s.Message, s.GitCommit, s.Added, s.Modified, s.Deleted)
				}
				return nil
			}
			for _, s := range list {
				short := s.GitCommit
				if len(short) > 7 {
					short = short[:7]
				}
				fmt.Fprintf(out, "#%d %s %s [%s] +%d ~%d -%d\n",
					s.ID, s.CreatedAt, s.Message, short, s.Added, s.Modified, s.Deleted)
			}
			_ = metadata.Snapshot{}
			return nil
		},
	}
	cmd.Flags().IntVarP(&n, "last", "n", 0, "show last N snapshots")
	return cmd
}
