package cli

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/RakhaYandra/kage/internal/tui"
)

func newTui() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Interactive picker for changed files and restores",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := homeDir()
			m, err := tui.New(home)
			if err != nil {
				return fail(1, "%v", err)
			}
			if len(m.Snaps) == 0 {
				return fail(1, "no snapshots yet (run kage snapshot first)")
			}
			p := tea.NewProgram(m)
			if _, err := p.Run(); err != nil {
				return fail(1, "%v", err)
			}
			return nil
		},
	}
}
