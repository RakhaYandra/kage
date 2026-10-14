package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRejectsTraversal(t *testing.T) {
	home := t.TempDir()
	bad := []string{"../etc/passwd", "~/../etc/shadow", "/etc/passwd"}
	for _, p := range bad {
		c := Default(home)
		c.Tracking.Paths = []string{p}
		if err := Validate(c, home); err == nil {
			t.Errorf("path %q must be rejected", p)
		}
	}
}

func TestValidateDefaultOK(t *testing.T) {
	home := t.TempDir()
	c := Default(home)
	if err := Validate(c, home); err != nil {
		t.Fatalf("default must validate: %v", err)
	}
	found := false
	for _, e := range c.Tracking.Exclude {
		if e == "**/id_*" {
			found = true
		}
	}
	if !found {
		t.Fatalf("default exclude must contain **/id_*")
	}
}

func TestValidateSchedule(t *testing.T) {
	home := t.TempDir()
	for _, s := range []string{"", "daily", "hourly"} {
		c := Default(home)
		c.Snapshot.Schedule = s
		if err := Validate(c, home); err != nil {
			t.Errorf("schedule %q must validate: %v", s, err)
		}
	}
	c := Default(home)
	c.Snapshot.Schedule = "minutely"
	if err := Validate(c, home); err == nil {
		t.Errorf("schedule minutely must be rejected")
	}
}

func TestValidateKeepLast(t *testing.T) {
	home := t.TempDir()
	c := Default(home)
	c.Snapshot.KeepLast = -1
	if err := Validate(c, home); err == nil {
		t.Errorf("keep_last -1 must be rejected")
	}
	c.Snapshot.KeepLast = 30
	if err := Validate(c, home); err != nil {
		t.Errorf("keep_last 30 must validate: %v", err)
	}
}

func TestValidateQuietPeriod(t *testing.T) {
	home := t.TempDir()
	c := Default(home)
	c.Watcher.QuietPeriod = "bogus"
	if err := Validate(c, home); err == nil {
		t.Errorf("quiet_period bogus must be rejected")
	}
	c.Watcher.QuietPeriod = "10m"
	if err := Validate(c, home); err != nil {
		t.Errorf("quiet_period 10m must validate: %v", err)
	}
}

func TestLoadRoundTrip(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, "config.yaml")
	if err := WriteDefault(p, home); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p, home)
	if err != nil {
		t.Fatal(err)
	}
	if c.Version != 1 {
		t.Fatalf("version=%d", c.Version)
	}
	_ = os.Remove(p)
}
