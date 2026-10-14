// Package metadata keeps snapshot records separate from the git repo.
package metadata

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

// Snapshot is the query-friendly record of one snapshot.
type Snapshot struct {
	ID            int    `json:"id"`
	CreatedAt     string `json:"created_at"`
	Message       string `json:"message"`
	GitCommit     string `json:"git_commit"`
	Added         int    `json:"added"`
	Modified      int    `json:"modified"`
	Deleted       int    `json:"deleted"`
	Hash          string `json:"hash"`
	SchemaVersion int    `json:"schema_version"`
}

// Store is a JSONL file of snapshots.
type Store struct {
	Path string
}

// Open returns the store at <stateDir>/snapshots.jsonl.
func Open(stateDir string) Store {
	return Store{Path: filepath.Join(stateDir, "snapshots.jsonl")}
}

// List reads all records (missing file → empty).
func (s Store) List() ([]Snapshot, error) {
	f, err := os.Open(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []Snapshot
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		var snap Snapshot
		if err := json.Unmarshal(sc.Bytes(), &snap); err != nil {
			continue
		}
		out = append(out, snap)
	}
	return out, sc.Err()
}

// NextID returns max+1.
func (s Store) NextID() (int, error) {
	list, err := s.List()
	if err != nil {
		return 0, err
	}
	max := 0
	for _, sn := range list {
		if sn.ID > max {
			max = sn.ID
		}
	}
	return max + 1, nil
}

// Append adds one record (0600, parent 0700).
func (s Store) Append(sn Snapshot) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(sn)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}
