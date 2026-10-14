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
	result  string
	err     string

	previewFn func(snap int, file string) (string, error)
	applyFn   func(snap int, file string) (string, error)
}

func (m Model) Init() tea.Cmd { return nil }

var title = lipgloss.NewStyle().Bold(true).Render

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m.move(-1), nil
		case 'j':
			return m.move(1), nil
		case 'y':
			if m.State == StatePreview {
				return m.confirm(), nil
			}
		}
	}
	switch km.Type {
	case tea.KeyUp:
		return m.move(-1), nil
	case tea.KeyDown:
		return m.move(1), nil
	case tea.KeyEnter:
		return m.enter(), nil
	}
	return m, nil
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
		sb.WriteString("\nenter: snapshots · q: quit\n")
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
	}
	if m.err != "" {
		sb.WriteString("\nerror: " + m.err + "\n")
	}
	return sb.String()
}
