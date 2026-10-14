package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/RakhaYandra/kage/internal/metadata"
	"github.com/RakhaYandra/kage/internal/snapshot"
)

type histRow struct {
	ts   string
	line string
}

// History merges snapshots, restores and watch events chronologically.
// n <= 0 means all.
func History(home string, n int) ([]string, error) {
	var rows []histRow
	state := snapshot.StateDir(home)

	store := metadata.Open(state)
	snaps, err := store.List()
	if err != nil {
		return nil, err
	}
	for _, s := range snaps {
		rows = append(rows, histRow{s.CreatedAt,
			fmt.Sprintf("%s snapshot #%d %s (+%d ~%d -%d)", s.CreatedAt, s.ID, s.Message, s.Added, s.Modified, s.Deleted)})
	}
	readLog := func(name, render string) {
		f, err := os.Open(filepath.Join(state, name))
		if err != nil {
			return
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 256*1024), 256*1024)
		for sc.Scan() {
			var m map[string]any
			if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
				continue
			}
			ts, _ := m["ts"].(string)
			switch render {
			case "restore":
				rows = append(rows, histRow{ts, fmt.Sprintf("%s restore from %v files=%v", ts, m["commit"], m["files"])})
			case "watch":
				rows = append(rows, histRow{ts, fmt.Sprintf("%s watch %v %v", ts, m["kind"], m["rel"])})
			}
		}
	}
	readLog("restore.log", "restore")
	readLog("watch.log", "watch")

	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ts < rows[j].ts })
	if n > 0 && len(rows) > n {
		rows = rows[len(rows)-n:]
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.line)
	}
	return out, nil
}
