package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	cfgPath string
	noColor bool
	jsonOut bool
	verbose bool
)

// ExitError carries a process exit code across cobra.
type ExitError struct {
	Code int
	Msg  string
}

func (e *ExitError) Error() string { return e.Msg }

func fail(code int, format string, args ...any) error {
	return &ExitError{Code: code, Msg: fmt.Sprintf(format, args...)}
}

func signalCtx() (context.Context, *bool) {
	ctx, cancel := context.WithCancel(context.Background())
	fired := new(bool)
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		*fired = true
		cancel()
		<-ch
		cancel()
	}()
	return ctx, fired
}

func debugf(format string, args ...any) {
	if verbose {
		fmt.Fprintf(os.Stderr, "kage: "+format+"\n", args...)
	}
}

// New builds the root command tree.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "kage",
		Short: "Configuration state manager: snapshot, diff, restore",
		Long: `Kage tracks Linux config files, stores snapshots in a local Git repository,
shows diffs, restores selectively, and audits for secrets and drift.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "path to config.yaml")
	root.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "structured JSON output")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "diagnostic output on stderr")

	root.AddCommand(
		newInit(), newSnapshot(), newDiff(), newRestore(), newAudit(),
		newStatus(), newLog(), newTrack(), newConfigCmd(), newVersion(),
	)
	return root
}

func stubErr(name string) error {
	return fail(1, "%s: not implemented yet", name)
}

func newRestore() *cobra.Command {
	return &cobra.Command{Use: "restore", Short: "Restore files", RunE: func(cmd *cobra.Command, args []string) error { return stubErr("restore") }}
}

func newAudit() *cobra.Command {
	return &cobra.Command{Use: "audit", Short: "Audit state and security", RunE: func(cmd *cobra.Command, args []string) error { return stubErr("audit") }}
}
