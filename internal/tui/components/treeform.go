package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FieldType determines how a TreeField renders and edits.
type FieldType int

const (
	FieldText   FieldType = iota // Single-line text input
	FieldSelect                  // Cycle through options with left/right
	FieldToggle                  // Yes/No toggle
)

// TreeField is an editable sub-field displayed under an expanded node.
type TreeField struct {
	Label       string
	Type        FieldType
	TextValue   *string  // FieldText: pointer to the string being edited
	BoolValue   *bool    // FieldToggle: pointer to the bool being edited
	SelectValue *string  // FieldSelect: pointer to the selected value
	SelectOpts  []string // FieldSelect: available options
}

// TreeNode is a collapsible row in the tree. Space toggles its status,
// Enter expands/collapses it, and expanded nodes show their Fields inline.
type TreeNode struct {
	Label    string        // Display name, e.g. "Claude Included"
	Expanded bool          // Whether child fields are visible
	Fields   []TreeField   // Child fields shown when expanded
	OnToggle func()        // Called when user presses Space on this node
	Status   func() string // Returns current status text, e.g. "disabled"
}

// TreeForm is a bubbletea Model for editing hierarchical settings.
type TreeForm struct {
	title      string
	nodes      []TreeNode
	cursor     int  // Index into nodes
	fieldFocus int  // -1 = node row focused, 0..N = child field index
	editing    bool // True when a text field has active input
	textInput  textinput.Model
	quitting   bool
}

// NewTreeForm creates a TreeForm with the given title and nodes.
func NewTreeForm(title string, nodes []TreeNode) TreeForm {
	ti := textinput.New()
	ti.CharLimit = 200
	return TreeForm{
		title:      title,
		nodes:      nodes,
		cursor:     0,
		fieldFocus: -1,
		textInput:  ti,
	}
}

func (m TreeForm) Init() tea.Cmd {
	return nil
}

func (m TreeForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If we're editing a text field, handle text input first
		if m.editing {
			switch msg.String() {
			case "esc", "enter":
				// Commit the text value and stop editing
				node := &m.nodes[m.cursor]
				f := &node.Fields[m.fieldFocus]
				if f.Type == FieldText && f.TextValue != nil {
					*f.TextValue = m.textInput.Value()
				}
				m.editing = false
				m.textInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.fieldFocus > 0 {
				// Move up within child fields
				m.fieldFocus--
			} else if m.fieldFocus == 0 {
				// Move from first child field back to node row
				m.fieldFocus = -1
			} else {
				// Move to previous node
				if m.cursor > 0 {
					m.cursor--
					m.fieldFocus = -1
				}
			}

		case "down", "j":
			node := &m.nodes[m.cursor]
			if m.fieldFocus == -1 && node.Expanded && len(node.Fields) > 0 {
				// Move from node row into first child field
				m.fieldFocus = 0
			} else if m.fieldFocus >= 0 && m.fieldFocus < len(node.Fields)-1 {
				// Move down within child fields
				m.fieldFocus++
			} else {
				// Move to next node
				if m.cursor < len(m.nodes)-1 {
					m.cursor++
					m.fieldFocus = -1
				}
			}

		case " ":
			// Toggle node status
			if m.fieldFocus == -1 {
				node := &m.nodes[m.cursor]
				if node.OnToggle != nil {
					node.OnToggle()
				}
			} else {
				// Toggle a FieldToggle child
				node := &m.nodes[m.cursor]
				if m.fieldFocus >= 0 && m.fieldFocus < len(node.Fields) {
					f := &node.Fields[m.fieldFocus]
					if f.Type == FieldToggle && f.BoolValue != nil {
						*f.BoolValue = !*f.BoolValue
					}
				}
			}

		case "enter":
			if m.fieldFocus == -1 {
				// Expand/collapse node
				node := &m.nodes[m.cursor]
				node.Expanded = !node.Expanded
				if !node.Expanded {
					m.fieldFocus = -1
				}
			} else {
				// Start editing a text field
				node := &m.nodes[m.cursor]
				if m.fieldFocus >= 0 && m.fieldFocus < len(node.Fields) {
					f := &node.Fields[m.fieldFocus]
					if f.Type == FieldText && f.TextValue != nil {
						m.editing = true
						m.textInput.SetValue(*f.TextValue)
						m.textInput.Focus()
						return m, textinput.Blink
					}
				}
			}

		case "left", "h":
			// Cycle select option backward
			if m.fieldFocus >= 0 {
				node := &m.nodes[m.cursor]
				f := &node.Fields[m.fieldFocus]
				if f.Type == FieldSelect && f.SelectValue != nil && len(f.SelectOpts) > 0 {
					idx := selectIndex(f.SelectOpts, *f.SelectValue)
					if idx > 0 {
						*f.SelectValue = f.SelectOpts[idx-1]
					} else {
						*f.SelectValue = f.SelectOpts[len(f.SelectOpts)-1]
					}
				}
			}

		case "right", "l":
			// Cycle select option forward
			if m.fieldFocus >= 0 {
				node := &m.nodes[m.cursor]
				f := &node.Fields[m.fieldFocus]
				if f.Type == FieldSelect && f.SelectValue != nil && len(f.SelectOpts) > 0 {
					idx := selectIndex(f.SelectOpts, *f.SelectValue)
					if idx < len(f.SelectOpts)-1 {
						*f.SelectValue = f.SelectOpts[idx+1]
					} else {
						*f.SelectValue = f.SelectOpts[0]
					}
				}
			}
		}
	}
	return m, nil
}

// selectIndex finds the index of val in opts, or 0 if not found.
func selectIndex(opts []string, val string) int {
	for i, o := range opts {
		if o == val {
			return i
		}
	}
	return 0
}

func (m TreeForm) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(TitleStyle.Render(m.title))
	b.WriteString("\n")
	b.WriteString(MutedStyle.Render(strings.Repeat("─", 40)))
	b.WriteString("\n")

	activeStyle := lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true)
	statusActive := lipgloss.NewStyle().Foreground(SuccessColor)
	statusDisabled := lipgloss.NewStyle().Foreground(WarningColor)
	fieldLabelStyle := lipgloss.NewStyle().Foreground(SubtitleColor).Width(14).Align(lipgloss.Right)
	fieldFocusStyle := lipgloss.NewStyle().Foreground(PrimaryColor)

	for i, node := range m.nodes {
		isNodeFocused := i == m.cursor && m.fieldFocus == -1

		// Arrow indicator
		arrow := "▶"
		if node.Expanded {
			arrow = "▼"
		}

		// Status badge
		status := ""
		if node.Status != nil {
			s := node.Status()
			if s == "active" || s == "enabled" || s == "on" {
				status = statusActive.Render("[" + s + "]")
			} else {
				status = statusDisabled.Render("[" + s + "]")
			}
		}

		// Node line
		label := fmt.Sprintf("%s %s", arrow, node.Label)
		if isNodeFocused {
			label = activeStyle.Render(label)
		}

		// Pad label to align status badges
		padding := 30 - lipgloss.Width(node.Label)
		if padding < 2 {
			padding = 2
		}
		line := fmt.Sprintf("%s%s%s", label, strings.Repeat(" ", padding), status)
		b.WriteString(line)
		b.WriteString("\n")

		// Child fields (only if expanded)
		if node.Expanded {
			for fi, f := range node.Fields {
				isFieldFocused := i == m.cursor && m.fieldFocus == fi

				flabel := fieldLabelStyle.Render(f.Label + ":")
				var fvalue string

				switch f.Type {
				case FieldText:
					if m.editing && isFieldFocused {
						fvalue = m.textInput.View()
					} else {
						val := ""
						if f.TextValue != nil {
							val = *f.TextValue
						}
						if val == "" {
							val = MutedStyle.Render("(empty)")
						}
						if isFieldFocused {
							fvalue = fieldFocusStyle.Render(val)
						} else {
							fvalue = val
						}
					}

				case FieldSelect:
					val := ""
					if f.SelectValue != nil {
						val = *f.SelectValue
					}
					if isFieldFocused {
						fvalue = fieldFocusStyle.Render("◀ " + val + " ▶")
					} else {
						fvalue = val
					}

				case FieldToggle:
					val := false
					if f.BoolValue != nil {
						val = *f.BoolValue
					}
					display := "No"
					if val {
						display = "Yes"
					}
					if isFieldFocused {
						fvalue = fieldFocusStyle.Render("[" + display + "]")
					} else {
						fvalue = display // BUG FIX: was `val` (bool), must be `display` (string)
					}
				}

				b.WriteString(fmt.Sprintf("    %s %s\n", flabel, fvalue))
			}
		}
	}

	// Help bar
	b.WriteString("\n")
	if m.editing {
		b.WriteString(MutedStyle.Render("type to edit · enter/esc confirm"))
	} else {
		b.WriteString(MutedStyle.Render("↑↓ navigate · space toggle · enter expand · ←→ select · esc done"))
	}

	return b.String()
}

// Run starts the TreeForm as a fullscreen bubbletea program and blocks
// until the user exits. Returns nil on normal exit.
func (m TreeForm) Run() error {
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
