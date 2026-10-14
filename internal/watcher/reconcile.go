package watcher

import (
	"github.com/RakhaYandra/kage/internal/snapshot"
)

// Reconcile lists working files differing from the last snapshot,
// catching changes made while the watcher was off or events were dropped.
// last is reserved for incremental state (nil = full scan).
func Reconcile(home string, last map[string]string) ([]string, error) {
	st, err := snapshot.GetStatus(home)
	if err != nil {
		return nil, err
	}
	return st.Changed, nil
}
