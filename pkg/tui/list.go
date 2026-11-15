package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type listModel struct {
	title  string
	items  []string
	hint   string
	keys   map[string]bool
	cursor int
	key    string
}

func (m listModel) Init() tea.Cmd { return nil }

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch s := k.String(); s {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case "esc", "q", "ctrl+c":
		m.key = "esc"
		return m, tea.Quit
	default:
		if m.keys[s] {
			m.key = s
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m listModel) View() string {
	var b strings.Builder
	b.WriteString(m.title + "\n\n")
	for i, it := range m.items {
		cur := "  "
		if i == m.cursor {
			cur = "> "
		}
		b.WriteString(cur + it + "\n")
	}
	b.WriteString("\n" + m.hint + "\n")
	return b.String()
}

// RunList shows an interactive list. It returns the key that ended the
// session ("esc" to quit, otherwise one of keys) and the highlighted index.
func RunList(title string, items []string, keys []string, hint string) (key string, index int, err error) {
	set := map[string]bool{}
	for _, k := range keys {
		set[k] = true
	}
	final, err := tea.NewProgram(listModel{title: title, items: items, keys: set, hint: hint}).Run()
	if err != nil {
		return "", 0, fmt.Errorf("running list: %w", err)
	}
	m := final.(listModel)
	return m.key, m.cursor, nil
}
