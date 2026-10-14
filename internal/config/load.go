package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultQuietPeriod applies when watcher.quiet_period is unset.
const DefaultQuietPeriod = 5 * time.Minute

// Expand turns ~/... into home/... and cleans the result.
func Expand(p, home string) string {
	if strings.HasPrefix(p, "~/") {
		p = filepath.Join(home, strings.TrimPrefix(p, "~/"))
	} else if p == "~" {
		p = home
	}
	return filepath.Clean(p)
}

// Normalize resolves a tracked path against home.
func Normalize(p, home string) string {
	if strings.HasPrefix(p, "~/") || p == "~" {
		return Expand(p, home)
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(home, p))
}

// Validate checks schema version and scope safety.
func Validate(c Config, home string) error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported version %d (want 1)", c.Version)
	}
	home = filepath.Clean(home)
	repo := Expand(c.Repository.Path, home)
	if !filepath.IsAbs(repo) {
		return fmt.Errorf("repository.path must be absolute: %q", c.Repository.Path)
	}
	for _, p := range c.Tracking.Paths {
		if strings.TrimSpace(p) == "" {
			return fmt.Errorf("empty tracking path")
		}
		n := Normalize(p, home)
		if n != home && !strings.HasPrefix(n, home+string(os.PathSeparator)) {
			return fmt.Errorf("path outside home scope: %q", p)
		}
	}
	for _, e := range c.Tracking.Exclude {
		if strings.TrimSpace(e) == "" {
			return fmt.Errorf("empty exclude pattern")
		}
	}
	switch c.Snapshot.Schedule {
	case "", "daily", "hourly":
	default:
		return fmt.Errorf("snapshot.schedule must be daily|hourly, got %q", c.Snapshot.Schedule)
	}
	if c.Snapshot.KeepLast < 0 {
		return fmt.Errorf("snapshot.keep_last must be >= 0")
	}
	if _, err := QuietDuration(c); err != nil {
		return err
	}
	return nil
}

// QuietDuration parses watcher.quiet_period ("" = DefaultQuietPeriod).
func QuietDuration(c Config) (time.Duration, error) {
	if strings.TrimSpace(c.Watcher.QuietPeriod) == "" {
		return DefaultQuietPeriod, nil
	}
	d, err := time.ParseDuration(c.Watcher.QuietPeriod)
	if err != nil {
		return 0, fmt.Errorf("watcher.quiet_period: %w", err)
	}
	if d < 0 {
		return 0, fmt.Errorf("watcher.quiet_period must be >= 0")
	}
	return d, nil
}

// Load reads and validates a config file.
func Load(path, home string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := Validate(c, home); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Save writes config with 0600.
func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(&c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
