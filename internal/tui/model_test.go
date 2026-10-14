package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

var errTestDiff = errors.New("no snapshot")

func testModel() Model {
	return Model{
		Home:  "/tmp/fake-home",
		State: StateFiles,
		Files: []string{".zshrc", ".config/hypr/hyprland.conf"},
		Snaps: []Snap{{ID: 1, Message: "one"}, {ID: 2, Message: "two"}},
		previewFn: func(snap int, file string) (string, error) {
			return "preview " + file, nil
		},
		applyFn: func(snap int, file string) (string, error) {
			return "checkpoint abc", nil
		},
	}
}

func key(s string) tea.Msg {
	switch s {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "q":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	case "y":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	case "d":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	}
	return nil
}

func diffModel() Model {
	m := testModel()
	m.diffFn = func(snap int, file string) (string, error) {
		return "-old\n+new\nline3\nline4\nline5", nil
	}
	return m
}

func TestDiffKeyShowsDiffState(t *testing.T) {
	m := diffModel()
	m2, _ := m.Update(key("d"))
	mm := m2.(Model)
	if mm.State != StateDiff {
		t.Fatalf("state=%q", mm.State)
	}
	if mm.diff == "" {
		t.Fatalf("diff content must load")
	}
}

func TestDiffScrollClamped(t *testing.T) {
	m := diffModel()
	m.height = 2
	m2, _ := m.Update(key("d"))
	mm := m2.(Model)
	m3, _ := mm.Update(key("down"))
	if m3.(Model).scroll != 1 {
		t.Fatalf("scroll=%d", m3.(Model).scroll)
	}
	// scroll past end clamps (5 lines, height 2 -> max 3)
	for i := 0; i < 10; i++ {
		m4, _ := mm.Update(key("down"))
		mm = m4.(Model)
	}
	if mm.scroll != 3 {
		t.Fatalf("scroll must clamp at 3, got %d", mm.scroll)
	}
	m5, _ := mm.Update(key("up"))
	if m5.(Model).scroll != 2 {
		t.Fatalf("scroll up=%d", m5.(Model).scroll)
	}
}

func TestDiffErrorStays(t *testing.T) {
	m := testModel()
	m.diffFn = func(snap int, file string) (string, error) {
		return "", errTestDiff
	}
	m2, _ := m.Update(key("d"))
	mm := m2.(Model)
	if mm.State != StateFiles {
		t.Fatalf("state=%q, must stay on error", mm.State)
	}
	if mm.err == "" {
		t.Fatalf("err must be set")
	}
}

func TestDiffEscBack(t *testing.T) {
	m := diffModel()
	m2, _ := m.Update(key("d"))
	m3, _ := m2.(Model).Update(key("esc"))
	if m3.(Model).State != StateFiles {
		t.Fatalf("state=%q", m3.(Model).State)
	}
}

func TestNavigateFiles(t *testing.T) {
	m := testModel()
	m2, _ := m.Update(key("down"))
	mm := m2.(Model)
	if mm.Cursor != 1 {
		t.Fatalf("cursor=%d", mm.Cursor)
	}
	m3, _ := mm.Update(key("up"))
	if m3.(Model).Cursor != 0 {
		t.Fatalf("cursor back=%d", m3.(Model).Cursor)
	}
}

func TestEnterShowsDiffPreview(t *testing.T) {
	m := testModel()
	m2, _ := m.Update(key("enter"))
	mm := m2.(Model)
	if mm.State != StateSnap {
		t.Fatalf("state=%q", mm.State)
	}
}

func TestRestoreNeedsConfirm(t *testing.T) {
	applied := false
	m := testModel()
	m.applyFn = func(snap int, file string) (string, error) {
		applied = true
		return "", nil
	}
	// files -> snaps -> preview -> (no apply yet)
	m2, _ := m.Update(key("enter"))
	m3, _ := m2.(Model).Update(key("enter"))
	mm := m3.(Model)
	if mm.State != StatePreview {
		t.Fatalf("state=%q", mm.State)
	}
	if applied {
		t.Fatalf("apply must not run before confirm")
	}
	m4, _ := mm.Update(key("y"))
	if !applied {
		t.Fatalf("y must apply")
	}
	if m4.(Model).State != StateDone {
		t.Fatalf("state=%q", m4.(Model).State)
	}
}

func TestQuit(t *testing.T) {
	m := testModel()
	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatalf("q must quit")
	}
}

func TestEscGoesBack(t *testing.T) {
	m := testModel()
	m2, _ := m.Update(key("enter"))
	m3, _ := m2.(Model).Update(key("esc"))
	if m3.(Model).State != StateFiles {
		t.Fatalf("state=%q", m3.(Model).State)
	}
}
