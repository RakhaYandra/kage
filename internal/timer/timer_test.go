package timer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeRunner struct {
	calls [][]string
	err   error // returned for every call
}

func (f *fakeRunner) run(args ...string) error {
	f.calls = append(f.calls, args)
	return f.err
}

func TestInstallWritesUnitsAndEnables(t *testing.T) {
	dir := t.TempDir()
	fr := &fakeRunner{}
	if err := Install(dir, fr.run); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"kage-snapshot.service", "kage-snapshot.timer"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("%s missing: %v", name, err)
		}
	}
	want := [][]string{
		{"systemctl", "--user", "daemon-reload"},
		{"systemctl", "--user", "enable", "--now", "kage-snapshot.timer"},
	}
	if len(fr.calls) != len(want) {
		t.Fatalf("calls=%v", fr.calls)
	}
	for i := range want {
		if len(fr.calls[i]) != len(want[i]) {
			t.Fatalf("calls=%v", fr.calls)
		}
		for j := range want[i] {
			if fr.calls[i][j] != want[i][j] {
				t.Fatalf("calls=%v want %v", fr.calls, want)
			}
		}
	}
}

func TestUninstallDisablesAndRemoves(t *testing.T) {
	dir := t.TempDir()
	fr := &fakeRunner{}
	if err := Install(dir, fr.run); err != nil {
		t.Fatal(err)
	}
	fr.calls = nil
	if err := Uninstall(dir, fr.run); err != nil {
		t.Fatal(err)
	}
	if len(fr.calls) == 0 || fr.calls[0][len(fr.calls[0])-1] != "kage-snapshot.timer" {
		t.Fatalf("must disable timer first, got %v", fr.calls)
	}
	for _, name := range []string{"kage-snapshot.service", "kage-snapshot.timer"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("%s must be removed", name)
		}
	}
}

func TestNoSystemctlFriendlyError(t *testing.T) {
	dir := t.TempDir()
	err := Install(dir, func(args ...string) error {
		return errors.New(`executable file not found in $PATH`)
	})
	var missing *MissingSystemdError
	if !errors.As(err, &missing) {
		t.Fatalf("want MissingSystemdError, got %v", err)
	}
}
