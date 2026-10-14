package cli

import (
	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/security"
)

// redactForDisplay masks secret-like values unless redact_output is off.
func redactForDisplay(home, s string) string {
	cfg, err := config.Load(resolveConfigPath(home), home)
	if err != nil {
		return s
	}
	if !cfg.Security.RedactOutput {
		return s
	}
	return security.Redact(s)
}
