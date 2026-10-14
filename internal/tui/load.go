package tui

import (
	"fmt"
	"strings"

	"github.com/RakhaYandra/kage/internal/config"
	"github.com/RakhaYandra/kage/internal/diff"
	"github.com/RakhaYandra/kage/internal/metadata"
	"github.com/RakhaYandra/kage/internal/restore"
	"github.com/RakhaYandra/kage/internal/security"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

// maxDiffLines caps rendered diff output.
const maxDiffLines = 500

func redactMaybe(home, s string) string {
	if cfg, err := config.Load(config.DefaultPath(home), home); err == nil && cfg.Security.RedactOutput {
		return security.Redact(s)
	}
	return s
}

// New builds a Model wired to the real engines.
func New(home string) (Model, error) {
	st, err := snapshot.GetStatus(home)
	if err != nil {
		return Model{}, err
	}
	files := make([]string, 0, len(st.Changed))
	for _, c := range st.Changed {
		// strip " (modified/added)" suffix
		rel := c
		for i := len(c) - 1; i >= 0; i-- {
			if c[i] == ' ' && i+1 < len(c) && c[i+1] == '(' {
				rel = c[:i]
				break
			}
		}
		files = append(files, rel)
	}
	store := metadata.Open(snapshot.StateDir(home))
	list, err := store.List()
	if err != nil {
		return Model{}, err
	}
	snaps := make([]Snap, 0, len(list))
	for _, s := range list {
		snaps = append(snaps, Snap{ID: s.ID, Message: s.Message})
	}
	return Model{
		Home:  home,
		Files: files,
		Snaps: snaps,
		State: StateFiles,
		previewFn: func(snap int, file string) (string, error) {
			res, err := restore.Run(home, restore.Options{SnapshotID: snap, Files: []string{file}, DryRun: true})
			if err != nil {
				return "", err
			}
			return redactMaybe(home, res.Preview), nil
		},
		diffFn: func(snap int, file string) (string, error) {
			res, err := diff.Working(home, snap, file)
			if err != nil {
				return "", err
			}
			for _, f := range res.Files {
				if f.Path != file {
					continue
				}
				if f.Binary {
					return "(binary, not shown)", nil
				}
				u := f.Unified
				if u == "" {
					return "(no textual diff)", nil
				}
				lines := strings.Split(u, "\n")
				if len(lines) > maxDiffLines {
					lines = append(lines[:maxDiffLines],
						fmt.Sprintf("[truncated: %d lines total]", len(strings.Split(u, "\n"))))
				}
				return redactMaybe(home, strings.Join(lines, "\n")), nil
			}
			return "(no changes)", nil
		},
		applyFn: func(snap int, file string) (string, error) {
			res, err := restore.Run(home, restore.Options{SnapshotID: snap, Files: []string{file}, Yes: true})
			if err != nil {
				return "", err
			}
			out := "restored " + file
			if res.Checkpoint != "" {
				out += " (checkpoint " + res.Checkpoint + ")"
			}
			return out, nil
		},
	}, nil
}
