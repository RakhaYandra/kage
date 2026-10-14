// Package security detects secrets, redacts output, and gates snapshots.
package security

import (
	"bytes"
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Finding is one detected secret indicator.
type Finding struct {
	Path string
	Rule string
}

// BlockedError stops a snapshot until secrets are handled (exit 3).
type BlockedError struct {
	Findings []Finding
	Allowed  []string
}

func (e *BlockedError) Error() string {
	var names []string
	for _, f := range e.Findings {
		names = append(names, f.Path+" ["+f.Rule+"]")
	}
	return fmt.Sprintf("secret detected in %d file(s): %s (remove, exclude, or allow via security.allow_secrets)",
		len(names), strings.Join(names, ", "))
}

type rule struct {
	name string
	re   *regexp.Regexp
}

var contentRules = []rule{
	{"private-key", regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`)},
	{"aws-access-key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"github-token", regexp.MustCompile(`(ghp_[A-Za-z0-9]{20,}|gho_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|glpat-[A-Za-z0-9_-]{10,})`)},
	{"slack-token", regexp.MustCompile(`xox[bap]-[A-Za-z0-9-]{10,}`)},
	{"google-api-key", regexp.MustCompile(`AIza[0-9A-Za-z_-]{20,}`)},
	{"stripe-key", regexp.MustCompile(`sk-live-[A-Za-z0-9]{10,}`)},
	{"credential-assign", regexp.MustCompile(`(?i)(password|passwd|secret|api[_-]?key|auth[_-]?token)\s*[:=]\s*\S+`)},
}

var nameRules = []struct {
	name  string
	match func(base string) bool
}{
	{"private-key-file", func(b string) bool {
		return strings.HasPrefix(b, "id_") || strings.HasSuffix(b, ".pem") ||
			strings.HasSuffix(b, ".p12") || strings.HasSuffix(b, ".pfx")
	}},
	{"env-file", func(b string) bool {
		return b == ".env" || strings.HasPrefix(b, ".env.")
	}},
	{"credentials-file", func(b string) bool {
		return b == "credentials.json" || strings.Contains(b, "credential")
	}},
}

// ScanContent checks bytes; binary (NUL) content is skipped for content rules
// but the filename is still evaluated by callers via SensitiveName.
func ScanContent(rel string, content []byte) []Finding {
	var out []Finding
	base := path.Base(toSlash(rel))
	for _, nr := range nameRules {
		if nr.match(base) {
			out = append(out, Finding{Path: rel, Rule: nr.name})
		}
	}
	if bytes.IndexByte(content, 0) >= 0 {
		return out
	}
	if len(content) > 1024*1024 {
		content = content[:1024*1024]
	}
	text := string(content)
	for _, r := range contentRules {
		if r.re.MatchString(text) {
			out = append(out, Finding{Path: rel, Rule: r.name})
		}
	}
	return out
}

// SensitiveName reports whether the filename alone is sensitive.
func SensitiveName(rel string) bool {
	base := path.Base(toSlash(rel))
	for _, nr := range nameRules {
		if nr.match(base) {
			return true
		}
	}
	return false
}

// Redact masks secret-like values in display text.
func Redact(s string) string {
	out := s
	for _, r := range contentRules {
		out = r.re.ReplaceAllString(out, "****")
	}
	return out
}

func toSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}
