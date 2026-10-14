package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/audit"
)

func newAudit() *cobra.Command {
	var onlySecurity, onlyCoverage bool
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit state drift, coverage and security (read-only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			rep, err := audit.Run(home)
			if err != nil {
				return fail(1, "%v", err)
			}
			out := cmd.OutOrStdout()
			if jsonOut {
				fmt.Fprintf(out, "{\"drift\":%d,\"coverage\":%d,\"secrets\":%d,\"perms\":%d,\"symlinks\":%d,\"repo\":%q,\"result\":%q}\n",
					rep.Drift, len(rep.Coverage), len(rep.Secrets), len(rep.Perms), len(rep.Symlinks), rep.RepoHealth, rep.Result)
			} else {
				fmt.Fprintln(out, "Kage Audit")
				fmt.Fprintln(out)
				fmt.Fprintf(out, "State drift       %d files changed\n", rep.Drift)
				fmt.Fprintf(out, "Coverage          %d suggested paths not tracked\n", len(rep.Coverage))
				fmt.Fprintf(out, "Permissions       %d warnings\n", len(rep.Perms))
				fmt.Fprintf(out, "Secret exposure   %d findings (indicators, not proof)\n", len(rep.Secrets))
				fmt.Fprintf(out, "Repository        %s\n", rep.RepoHealth)
				if onlySecurity {
					for _, s := range rep.Secrets {
						fmt.Fprintf(out, "  secret: %s [%s]\n", s.Path, s.Rule)
					}
					for _, p := range rep.Perms {
						fmt.Fprintf(out, "  perm: %s\n", p)
					}
				} else if onlyCoverage {
					for _, c := range rep.Coverage {
						fmt.Fprintf(out, "  untracked: ~/%s\n", c)
					}
				} else {
					for _, c := range rep.Coverage {
						fmt.Fprintf(out, "  untracked: ~/%s\n", c)
					}
					for _, p := range rep.Perms {
						fmt.Fprintf(out, "  perm: %s\n", p)
					}
					for _, s := range rep.Symlinks {
						fmt.Fprintf(out, "  symlink: %s\n", s)
					}
				}
				fmt.Fprintf(out, "\nResult: %s\n", rep.Result)
			}
			switch rep.Result {
			case "Action required":
				return fail(3, "audit: action required")
			case "Review recommended":
				return fail(2, "audit: review recommended")
			default:
				return nil
			}
		},
	}
	cmd.Flags().BoolVar(&onlySecurity, "security", false, "show only security findings")
	cmd.Flags().BoolVar(&onlyCoverage, "coverage", false, "show only coverage suggestions")
	return cmd
}
