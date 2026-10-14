// Package config defines the kage configuration schema (v1, YAML).
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version    int        `yaml:"version"`
	Repository Repository `yaml:"repository"`
	Tracking   Tracking   `yaml:"tracking"`
	Security   Security   `yaml:"security"`
	Snapshot   Snapshot   `yaml:"snapshot"`
	Watcher    Watcher    `yaml:"watcher"`
}

type Repository struct {
	Path string `yaml:"path"`
}

type Tracking struct {
	Paths   []string `yaml:"paths"`
	Exclude []string `yaml:"exclude"`
}

type Security struct {
	ProtectSecrets       bool     `yaml:"protect_secrets"`
	RedactOutput         bool     `yaml:"redact_output"`
	BlockDetectedSecrets bool     `yaml:"block_detected_secrets"`
	AllowSecrets         []string `yaml:"allow_secrets,omitempty"`
}

type Snapshot struct {
	AutoCommit          bool `yaml:"auto_commit"`
	SkipUnchanged       bool `yaml:"skip_unchanged"`
	PreservePermissions bool `yaml:"preserve_permissions"`
}

type Watcher struct {
	Enabled      bool `yaml:"enabled"`
	AutoSnapshot bool `yaml:"auto_snapshot"`
}

// Default returns the v1 default config for home.
func Default(home string) Config {
	return Config{
		Version:    1,
		Repository: Repository{Path: filepath.Join(home, ".local", "share", "kage", "repository")},
		Tracking: Tracking{
			Paths: []string{
				"~/.bashrc", "~/.zshrc", "~/.gitconfig",
				"~/.config/hypr", "~/.config/waybar", "~/.config/kitty",
			},
			Exclude: []string{
				"**/.cache/**", "**/node_modules/**",
				"**/.env", "**/.env.*", "**/id_*",
				"**/*.pem", "**/*.p12", "**/*.pfx",
				"**/credentials.json",
			},
		},
		Security: Security{ProtectSecrets: true, RedactOutput: true, BlockDetectedSecrets: true},
		Snapshot: Snapshot{AutoCommit: true, SkipUnchanged: true, PreservePermissions: true},
		Watcher:  Watcher{Enabled: false},
	}
}

// DefaultPath returns ~/.config/kage/config.yaml.
func DefaultPath(home string) string {
	return filepath.Join(home, ".config", "kage", "config.yaml")
}

// WriteDefault writes the default config if missing (0600).
func WriteDefault(path, home string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	cfg := Default(home)
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
