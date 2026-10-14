package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/config"
)

func newConfigCmd() *cobra.Command {
	cfg := &cobra.Command{Use: "config", Short: "Config operations"}
	validate := &cobra.Command{
		Use:   "validate",
		Short: "Validate config",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			c, err := config.Load(resolveConfigPath(home), home)
			if err != nil {
				return fail(2, "%v", err)
			}
			if err := config.Validate(c, home); err != nil {
				return fail(2, "%v", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "config valid")
			return nil
		},
	}
	cfg.AddCommand(validate)
	return cfg
}
