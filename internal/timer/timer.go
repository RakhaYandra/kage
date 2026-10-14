// Package timer installs the user-level systemd snapshot timer.
// Unit contents mirror systemd/ in the repo (see TestUnitsMatchRepo).
package timer

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ServiceUnit is systemd/kage-snapshot.service.
const ServiceUnit = `[Unit]
Description=Kage scheduled config snapshot
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
ExecStart=%h/.local/bin/kage snapshot --message "scheduled"
Nice=10
IOSchedulingClass=best-effort
IOSchedulingPriority=7
`

// TimerUnit is systemd/kage-snapshot.timer.
const TimerUnit = `[Unit]
Description=Kage daily config snapshot

[Timer]
OnCalendar=daily
Persistent=true
RandomizedDelaySec=15m

[Install]
WantedBy=timers.target
`

// MissingSystemdError means systemctl is unavailable (container/WSL/minimal).
type MissingSystemdError struct{ Op string }

func (e *MissingSystemdError) Error() string {
	return fmt.Sprintf("%s: systemctl not found (install systemd user services manually, see docs/WATCHER.md)", e.Op)
}

func wrapMissing(op string, err error) error {
	if err == nil {
		return nil
	}
	var ee *exec.Error
	if errors.As(err, &ee) || strings.Contains(err.Error(), "not found in $PATH") ||
		strings.Contains(err.Error(), "executable file not found") {
		return &MissingSystemdError{Op: op}
	}
	return fmt.Errorf("%s: %w", op, err)
}

// Runner runs an external command (systemctl by default).
type Runner func(args ...string) error

// Systemctl runs systemctl with args.
func Systemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %v: %v: %s", args, err, out)
	}
	return nil
}

// UnitDir returns ~/.config/systemd/user (or $KAGE_SYSTEMD_DIR override).
func UnitDir(home string) string {
	if d := os.Getenv("KAGE_SYSTEMD_DIR"); d != "" {
		return d
	}
	return filepath.Join(home, ".config", "systemd", "user")
}

// Install writes units and enables the timer.
func Install(dir string, run Runner) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "kage-snapshot.service"), []byte(ServiceUnit), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "kage-snapshot.timer"), []byte(TimerUnit), 0o644); err != nil {
		return err
	}
	if err := wrapMissing("daemon-reload", run("systemctl", "--user", "daemon-reload")); err != nil {
		return err
	}
	return wrapMissing("enable timer", run("systemctl", "--user", "enable", "--now", "kage-snapshot.timer"))
}

// Uninstall disables the timer and removes units.
func Uninstall(dir string, run Runner) error {
	if err := wrapMissing("disable timer", run("systemctl", "--user", "disable", "--now", "kage-snapshot.timer")); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(dir, "kage-snapshot.service"))
	_ = os.Remove(filepath.Join(dir, "kage-snapshot.timer"))
	return nil
}

// Status reports active|inactive|not-installed.
func Status(dir string, run Runner) (string, error) {
	if _, err := os.Stat(filepath.Join(dir, "kage-snapshot.timer")); os.IsNotExist(err) {
		return "not-installed", nil
	}
	if err := run("systemctl", "--user", "is-active", "--quiet", "kage-snapshot.timer"); err != nil {
		if wrapMissing("status", err) != nil {
			if _, ok := wrapMissing("status", err).(*MissingSystemdError); ok {
				return "unknown (no systemctl)", nil
			}
		}
		return "inactive", nil
	}
	return "active", nil
}
