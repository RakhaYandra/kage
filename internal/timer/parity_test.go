package timer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestUnitsMatchRepo(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repo := filepath.Join(filepath.Dir(file), "..", "..", "systemd")
	svc, err := os.ReadFile(filepath.Join(repo, "kage-snapshot.service"))
	if err != nil {
		t.Fatal(err)
	}
	tmr, err := os.ReadFile(filepath.Join(repo, "kage-snapshot.timer"))
	if err != nil {
		t.Fatal(err)
	}
	if string(svc) != ServiceUnit {
		t.Errorf("service unit drifted from systemd/")
	}
	if string(tmr) != TimerUnit {
		t.Errorf("timer unit drifted from systemd/")
	}
}
