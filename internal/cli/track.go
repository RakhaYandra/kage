package cli

import (
	"fmt"
	"os"
	"slices"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/config"
)

func loadCfg(home string) (config.Config, string, error) {
	cfgFile := resolveConfigPath(home)
	c, err := config.Load(cfgFile, home)
	if err != nil {
		return config.Config{}, "", fail(2, "%v", err)
	}
	return c, cfgFile, nil
}

func newTrack() *cobra.Command {
	track := &cobra.Command{Use: "track", Short: "Manage tracked paths"}

	add := &cobra.Command{
		Use:   "add <path>...",
		Short: "Track path(s)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			c, cfgFile, err := loadCfg(home)
			if err != nil {
				return err
			}
			for _, a := range args {
				n := config.Normalize(a, home)
				if _, err := os.Lstat(n); err != nil {
					return fail(1, "path not found: %s", a)
				}
				c.Tracking.Paths = append(c.Tracking.Paths, a)
				if err := config.Validate(c, home); err != nil {
					return fail(2, "%v", err)
				}
			}
			// dedupe, keep order
			seen := map[string]bool{}
			uniq := c.Tracking.Paths[:0]
			for _, p := range c.Tracking.Paths {
				if !seen[p] {
					seen[p] = true
					uniq = append(uniq, p)
				}
			}
			c.Tracking.Paths = uniq
			if err := config.Save(cfgFile, c); err != nil {
				return fail(1, "save config: %v", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "tracking updated")
			return nil
		},
	}

	remove := &cobra.Command{
		Use:   "remove <path>...",
		Short: "Untrack path(s)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			c, cfgFile, err := loadCfg(home)
			if err != nil {
				return err
			}
			for _, a := range args {
				c.Tracking.Paths = slices.DeleteFunc(c.Tracking.Paths, func(p string) bool { return p == a })
			}
			if err := config.Save(cfgFile, c); err != nil {
				return fail(1, "save config: %v", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "tracking updated")
			return nil
		},
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "List tracked paths",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			c, _, err := loadCfg(home)
			if err != nil {
				return err
			}
			for _, p := range c.Tracking.Paths {
				fmt.Fprintln(cmd.OutOrStdout(), p)
			}
			return nil
		},
	}

	track.AddCommand(add, remove, list)
	return track
}
