package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/gitstore"
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	h, _ := os.UserHomeDir()
	return h
}

func resolveConfigPath(home string) string {
	if cfgPath != "" {
		return cfgPath
	}
	return config.DefaultPath(home)
}

func newInit() *cobra.Command {
	var repair bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize workspace, repository and config",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			cfgFile := resolveConfigPath(home)
			if repair {
				// repair needs repo path: from default (config may be unreadable)
				repo := filepath.Join(home, ".local", "share", "kage", "repository")
				if err := gitstore.Repair(repo); err != nil {
					return fail(1, "repair repository: %v", err)
				}
				if err := config.WriteDefault(cfgFile, home); err != nil {
					return fail(1, "write config: %v", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "repaired "+repo)
				return nil
			}
			if err := config.WriteDefault(cfgFile, home); err != nil {
				return fail(1, "write config: %v", err)
			}
			repo := filepath.Join(home, ".local", "share", "kage", "repository")
			if err := gitstore.Init(repo); err != nil {
				return fail(1, "init repository: %v", err)
			}
			stateDir := filepath.Join(home, ".local", "share", "kage")
			_ = os.MkdirAll(stateDir, 0o700)
			fmt.Fprintln(cmd.OutOrStdout(), "initialized "+cfgFile)
			return nil
		},
	}
	cmd.Flags().BoolVar(&repair, "repair", false, "repair broken repository")
	return cmd
}
