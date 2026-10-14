package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/timer"
)

func newTimer() *cobra.Command {
	tc := &cobra.Command{Use: "timer", Short: "Manage the scheduled snapshot timer"}
	install := &cobra.Command{
		Use:   "install",
		Short: "Install and enable the user timer",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			if err := timer.Install(timer.UnitDir(home), timer.Systemctl); err != nil {
				var missing *timer.MissingSystemdError
				if errors.As(err, &missing) {
					return fail(1, "%v", missing)
				}
				return fail(1, "%v", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "timer installed (daily)")
			return nil
		},
	}
	uninstall := &cobra.Command{
		Use:   "uninstall",
		Short: "Disable and remove the user timer",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			if err := timer.Uninstall(timer.UnitDir(home), timer.Systemctl); err != nil {
				return fail(1, "%v", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "timer removed")
			return nil
		},
	}
	status := &cobra.Command{
		Use:   "status",
		Short: "Show timer state (read-only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			st, err := timer.Status(timer.UnitDir(home), timer.Systemctl)
			if err != nil {
				return fail(1, "%v", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "timer: "+st)
			return nil
		},
	}
	tc.AddCommand(install, uninstall, status)
	return tc
}
