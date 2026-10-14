package tui

import (
	"github.com/RakhaYandra/kage/internal/metadata"
	"github.com/RakhaYandra/kage/internal/restore"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

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
			return res.Preview, nil
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
