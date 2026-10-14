package metadata

import (
	"testing"
)

func TestPruneKeepsLastN(t *testing.T) {
	dir := t.TempDir()
	s := Open(dir)
	for i := 1; i <= 5; i++ {
		if err := s.Append(Snapshot{ID: i, Message: "m", SchemaVersion: 1}); err != nil {
			t.Fatal(err)
		}
	}
	dropped, err := s.Prune(2)
	if err != nil {
		t.Fatal(err)
	}
	if dropped != 3 {
		t.Fatalf("dropped=%d", dropped)
	}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != 4 || list[1].ID != 5 {
		t.Fatalf("list=%+v", list)
	}
	// next ID continues from max (no reuse)
	next, _ := s.NextID()
	if next != 6 {
		t.Fatalf("next=%d", next)
	}
}

func TestPruneZeroKeepsAll(t *testing.T) {
	dir := t.TempDir()
	s := Open(dir)
	s.Append(Snapshot{ID: 1})
	dropped, err := s.Prune(0)
	if err != nil || dropped != 0 {
		t.Fatalf("dropped=%d err=%v", dropped, err)
	}
}
