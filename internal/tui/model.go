// Package tui provides an interactive picker for changed files and restores.
// Restore always goes through preview + explicit confirm; the engine's
// checkpoint and conflict checks still apply.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// States.
const (
	StateFiles   = "files"
	StateSnap    = "snapshots"
	StatePreview = "preview"
	StateDone    = "done"
	StateDiff    = "diff"
)

// Snap is one snapshot option.
type Snap struct {
	ID      int
	Message string
}

// Model is the TUI state.
type Model struct {
	Home   string
	Files  []string
	Snaps  []Snap
	State  string
	Cursor int

	selFile int
	selSnap int
	preview string
	diff    string
	result  string
	err     string

	scroll int
	height int // visible diff lines; 0 = default

	previewFn func(snap int, file string) (string, error)
	applyFn   func(snap int, file string) (string, error)
	diffFn    func(snap int, file string) (string, error)
}

// visibleHeight returns the diff window height.
func (m Model) visibleHeight() int {
	if m.height > 0 {
		return m.height
	}
	return 20
}

// diffLines splits the loaded diff for rendering.
func (m Model) diffLines() []string {
	if m.diff == "" {
		return nil
	}
	return strings.Split(m.diff, "\n")
}

func (m Model) maxScroll() int {
	n := len(m.diffLines()) - m.visibleHeight()
	if n < 0 {
		return 0
	}
	return n
}

func (m Model) Init() tea.Cmd { return nil }

var title = lipgloss.NewStyle().Bold(true).Render

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.height = ws.Height - 6
		if m.height < 5 {
			m.height = 5
		}
		if m.scroll > m.maxScroll() {
			m.scroll = m.maxScroll()
		}
		return m, nil
	}
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch km.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		return m.back(), nil
	}
	if km.Type == tea.KeyRunes && len(km.Runes) == 1 {
		switch km.Runes[0] {
		case 'q':
			return m, tea.Quit
		case 'k':
			return m.up(), nil
		case 'j':
			return m.down(), nil
		case 'y':
			if m.State == StatePreview {
				return m.confirm(), nil
			}
		case 'd':
			if m.State == StateFiles {
				return m.showDiff(), nil
			}
		}
	}
	switch km.Type {
	case tea.KeyUp:
		return m.up(), nil
	case tea.KeyDown:
		return m.down(), nil
	case tea.KeyEnter:
		return m.enter(), nil
	}
	return m, nil
}

// up moves the list cursor, or scrolls the diff view.
func (m Model) up() Model {
	if m.State == StateDiff {
		m.scroll--
		if m.scroll < 0 {
			m.scroll = 0
		}
		return m
	}
	return m.move(-1)
}

// down moves the list cursor, or scrolls the diff view.
func (m Model) down() Model {
	if m.State == StateDiff {
		m.scroll++
		if max := m.maxScroll(); m.scroll > max {
			m.scroll = max
		}
		return m
	}
	return m.move(1)
}

// showDiff loads the full unified diff for the selected file.
func (m Model) showDiff() Model {
	if len(m.Files) == 0 {
		return m
	}
	m.selFile = m.Cursor
	if m.diffFn == nil {
		m.err = "diff unavailable"
		return m
	}
	snap := 0
	if len(m.Snaps) > 0 {
		snap = m.Snaps[len(m.Snaps)-1].ID
	}
	d, err := m.diffFn(snap, m.Files[m.selFile])
	if err != nil {
		m.err = err.Error()
		return m
	}
	m.diff = d
	m.scroll = 0
	m.State = StateDiff
	return m
}

func (m Model) count() int {
	switch m.State {
	case StateFiles:
		return len(m.Files)
	case StateSnap:
		return len(m.Snaps)
	default:
		return 0
	}
}

func (m Model) move(d int) Model {
	n := m.count()
	if n == 0 {
		return m
	}
	m.Cursor += d
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	if m.Cursor >= n {
		m.Cursor = n - 1
	}
	return m
}

func (m Model) back() Model {
	m.Cursor = 0
	m.err = ""
	switch m.State {
	case StateSnap:
		m.State = StateFiles
	case StatePreview:
		m.State = StateSnap
	case StateDiff:
		m.State = StateFiles
		m.scroll = 0
	}
	return m
}

func (m Model) enter() Model {
	switch m.State {
	case StateFiles:
		if len(m.Files) == 0 {
			return m
		}
		m.selFile = m.Cursor
		m.Cursor = 0
		m.State = StateSnap
	case StateSnap:
		if len(m.Snaps) == 0 {
			m.err = "no snapshots"
			return m
		}
		m.selSnap = m.Cursor
		m.Cursor = 0
		p, err := m.previewFn(m.Snaps[m.selSnap].ID, m.Files[m.selFile])
		if err != nil {
			m.err = err.Error()
			return m
		}
		m.preview = p
		m.State = StatePreview
	}
	return m
}

func (m Model) confirm() Model {
	out, err := m.applyFn(m.Snaps[m.selSnap].ID, m.Files[m.selFile])
	if err != nil {
		m.err = err.Error()
		return m
	}
	m.result = out
	m.State = StateDone
	return m
}

func (m Model) View() string {
	var sb strings.Builder
	switch m.State {
	case StateFiles:
		sb.WriteString(title("Kage — changed files") + "\n\n")
		if len(m.Files) == 0 {
			sb.WriteString("clean (nothing changed)\n")
			break
		}
		for i, f := range m.Files {
			mark := "  "
			if i == m.Cursor {
				mark = "> "
			}
			sb.WriteString(fmt.Sprintf("%s%s\n", mark, f))
		}
		sb.WriteString("\nenter: snapshots · d: diff · q: quit\n")
	case StateSnap:
		sb.WriteString(title(fmt.Sprintf("Restore %s from?", m.Files[m.selFile])) + "\n\n")
		for i, s := range m.Snaps {
			mark := "  "
			if i == m.Cursor {
				mark = "> "
			}
			sb.WriteString(fmt.Sprintf("%s#%d %s\n", mark, s.ID, s.Message))
		}
		sb.WriteString("\nenter: preview · esc: back\n")
	case StatePreview:
		sb.WriteString(title("Preview (nothing written yet)") + "\n\n")
		sb.WriteString(m.preview + "\n")
		sb.WriteString("\ny: restore · esc: back\n")
	case StateDone:
		sb.WriteString(title("Restored") + "\n\n" + m.result + "\n")
		sb.WriteString("\nq: quit\n")
	case StateDiff:
		sb.WriteString(title(fmt.Sprintf("Diff %s", m.Files[m.selFile])) + "\n\n")
		lines := m.diffLines()
		h := m.visibleHeight()
		end := m.scroll + h
		if end > len(lines) {
			end = len(lines)
		}
		for _, l := range lines[m.scroll:end] {
			sb.WriteString(l + "\n")
		}
		sb.WriteString(fmt.Sprintf("\nline %d-%d/%d · j/k scroll · esc: back\n", m.scroll+1, end, len(lines)))
	}
	if m.err != "" {
		sb.WriteString("\nerror: " + m.err + "\n")
	}
	return sb.String()
}
