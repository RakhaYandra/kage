package security

import (
	"os"
	"path/filepath"
)

// GateFile is one file to scan.
type GateFile struct {
	Rel string
	Abs string
}

// Gate scans working files; returns *BlockedError unless all findings allowed.
// allow lists home-relative paths explicitly accepted by the user.
func Gate(files []GateFile, allow []string) error {
	allowed := map[string]bool{}
	for _, a := range allow {
		allowed[toSlash(a)] = true
	}
	var blocked []Finding
	var allowedHits []string
	for _, f := range files {
		content, err := readTarget(f.Abs)
		if err != nil {
			continue
		}
		findings := ScanContent(f.Rel, content)
		if len(findings) == 0 {
			continue
		}
		if allowed[toSlash(f.Rel)] {
			allowedHits = append(allowedHits, f.Rel)
			continue
		}
		blocked = append(blocked, findings...)
	}
	if len(blocked) > 0 {
		return &BlockedError{Findings: blocked, Allowed: allowedHits}
	}
	return nil
}

func readTarget(abs string) ([]byte, error) {
	fi, err := os.Lstat(abs)
	if err != nil {
		return nil, err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(abs)
		if err != nil {
			return nil, err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(abs), target)
		}
		abs = target
	}
	return os.ReadFile(abs)
}
