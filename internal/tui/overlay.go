package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// overlay is a transient input capture that floats on top of any view.
// When done() returns true, the overlay is dismissed.
// result is nil if aborted, or contains the user's selection/input.
type overlay interface {
	Update(tea.Msg) (overlay, tea.Cmd)
	View(width, height int) string
	done() (bool, interface{})
}

// --- Styles ---

var (
	overlayBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("12")).
				Padding(1, 2)

	overlayTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("12")).
				MarginBottom(1)

	overlayHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				MarginTop(1)

	overlayItemStyle = lipgloss.NewStyle()

	overlaySelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("12")).
				Bold(true)

	overlayFilterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))
)

// --- Selection List Overlay ---

// selectionItem is a single option in the selection list.
type selectionItem struct {
	ID      string
	Label   string
	Desc    string // optional secondary text (used for filtering)
	Display string // optional pre-rendered label (overrides Label+Desc for display)
	Icon    string // optional pre-rendered icon (rendered outside of highlight)
}

// selectionOverlay is a filterable selection list.
type selectionOverlay struct {
	title    string
	items    []selectionItem
	filtered []int // indices into items
	cursor   int
	filter   textinput.Model
	isDone   bool
	result   interface{} // *selectionItem or nil
}

func newSelectionOverlay(title string, items []selectionItem) *selectionOverlay {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = 100
	ti.Focus()

	s := &selectionOverlay{
		title:  title,
		items:  items,
		filter: ti,
	}
	s.applyFilter()
	return s
}

func (s *selectionOverlay) applyFilter() {
	query := strings.ToLower(s.filter.Value())
	s.filtered = nil
	for i, item := range s.items {
		if query == "" || strings.Contains(strings.ToLower(item.Label), query) ||
			strings.Contains(strings.ToLower(item.Desc), query) {
			s.filtered = append(s.filtered, i)
		}
	}
	if s.cursor >= len(s.filtered) {
		s.cursor = max(0, len(s.filtered)-1)
	}
}

func (s *selectionOverlay) Update(msg tea.Msg) (overlay, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			s.isDone = true
			s.result = nil
			return s, nil
		case "enter":
			if len(s.filtered) > 0 && s.cursor < len(s.filtered) {
				idx := s.filtered[s.cursor]
				s.result = &s.items[idx]
			}
			s.isDone = true
			return s, nil
		case "up", "ctrl+p":
			if s.cursor > 0 {
				s.cursor--
			}
			return s, nil
		case "down", "ctrl+n":
			if s.cursor < len(s.filtered)-1 {
				s.cursor++
			}
			return s, nil
		}
	}

	// Forward to text input for filtering
	var cmd tea.Cmd
	s.filter, cmd = s.filter.Update(msg)
	s.applyFilter()
	return s, cmd
}

func (s *selectionOverlay) View(width, height int) string {
	var b strings.Builder

	b.WriteString(overlayTitleStyle.Render(s.title))
	b.WriteString("\n")
	b.WriteString(s.filter.View())
	b.WriteString("\n\n")

	// Show up to maxVisible items, capped so the overlay never fills the screen
	maxVisible := height - 12
	if maxVisible > 15 {
		maxVisible = 15
	}
	if maxVisible < 3 {
		maxVisible = 3
	}

	start := 0
	if s.cursor >= maxVisible {
		start = s.cursor - maxVisible + 1
	}

	for i := start; i < len(s.filtered) && i < start+maxVisible; i++ {
		idx := s.filtered[i]
		item := s.items[idx]
		var line string
		if item.Display != "" {
			line = item.Display
		} else {
			line = item.Label
			if item.Desc != "" {
				line += overlayFilterStyle.Render("  " + item.Desc)
			}
		}
		if i == s.cursor {
			if item.Icon != "" {
				b.WriteString(item.Icon + " " + overlaySelectedStyle.Render(line))
			} else {
				b.WriteString(overlaySelectedStyle.Render("> " + line))
			}
		} else {
			if item.Icon != "" {
				b.WriteString(item.Icon + " " + line)
			} else {
				b.WriteString("  " + line)
			}
		}
		b.WriteString("\n")
	}

	if len(s.filtered) == 0 {
		b.WriteString(overlayFilterStyle.Render("  No matches"))
		b.WriteString("\n")
	}

	b.WriteString(overlayHintStyle.Render("↑/↓: navigate  enter: select  esc: cancel"))

	boxWidth := width - 10
	if boxWidth < 30 {
		boxWidth = 30
	}
	if boxWidth > 70 {
		boxWidth = 70
	}

	content := overlayBorderStyle.Width(boxWidth).Render(b.String())

	// Center the overlay
	return lipgloss.Place(width, height-2, lipgloss.Center, lipgloss.Center, content)
}

func (s *selectionOverlay) done() (bool, interface{}) {
	return s.isDone, s.result
}

// --- Text Input Overlay ---

// textInputOverlay is a single-line text input.
type textInputOverlay struct {
	title  string
	input  textinput.Model
	isDone bool
	result interface{} // string or nil
}

func newTextInputOverlay(title, initial string) *textInputOverlay {
	ti := textinput.New()
	ti.SetValue(initial)
	ti.CharLimit = 500
	ti.Width = 60
	ti.Focus()

	return &textInputOverlay{
		title: title,
		input: ti,
	}
}

func (t *textInputOverlay) Update(msg tea.Msg) (overlay, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			t.isDone = true
			t.result = nil
			return t, nil
		case "enter":
			t.isDone = true
			t.result = t.input.Value()
			return t, nil
		}
	}

	var cmd tea.Cmd
	t.input, cmd = t.input.Update(msg)
	return t, cmd
}

func (t *textInputOverlay) View(width, height int) string {
	var b strings.Builder

	b.WriteString(overlayTitleStyle.Render(t.title))
	b.WriteString("\n")
	b.WriteString(t.input.View())
	b.WriteString("\n")
	b.WriteString(overlayHintStyle.Render("enter: save  esc: cancel"))

	boxWidth := width - 10
	if boxWidth < 30 {
		boxWidth = 30
	}
	if boxWidth > 70 {
		boxWidth = 70
	}

	content := overlayBorderStyle.Width(boxWidth).Render(b.String())
	return lipgloss.Place(width, height-2, lipgloss.Center, lipgloss.Center, content)
}

func (t *textInputOverlay) done() (bool, interface{}) {
	return t.isDone, t.result
}

// --- Text Editor Overlay ---

// textEditorOverlay is a multi-line text editor (for description).
type textEditorOverlay struct {
	title  string
	editor textarea.Model
	isDone bool
	result interface{} // string or nil
}

func newTextEditorOverlay(title, initial string, width, height int) *textEditorOverlay {
	ta := textarea.New()
	ta.SetValue(initial)
	ta.SetWidth(min(width-14, 70))
	ta.SetHeight(max(height-12, 5))
	ta.Focus()
	// Allow Enter for newlines — Ctrl+Enter will save
	ta.KeyMap.InsertNewline.SetKeys("enter")

	return &textEditorOverlay{
		title:  title,
		editor: ta,
	}
}

func (e *textEditorOverlay) Update(msg tea.Msg) (overlay, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			e.isDone = true
			e.result = nil
			return e, nil
		case "ctrl+s":
			e.isDone = true
			e.result = e.editor.Value()
			return e, nil
		}
	}

	var cmd tea.Cmd
	e.editor, cmd = e.editor.Update(msg)
	return e, cmd
}

func (e *textEditorOverlay) View(width, height int) string {
	var b strings.Builder

	b.WriteString(overlayTitleStyle.Render(e.title))
	b.WriteString("\n")
	b.WriteString(e.editor.View())
	b.WriteString("\n")
	b.WriteString(overlayHintStyle.Render("ctrl+s: save  esc: cancel"))

	boxWidth := width - 10
	if boxWidth < 30 {
		boxWidth = 30
	}
	if boxWidth > 75 {
		boxWidth = 75
	}

	content := overlayBorderStyle.Width(boxWidth).Render(b.String())
	return lipgloss.Place(width, height-2, lipgloss.Center, lipgloss.Center, content)
}

func (e *textEditorOverlay) done() (bool, interface{}) {
	return e.isDone, e.result
}

// --- Confirmation Overlay ---

// confirmOverlay shows a y/n confirmation prompt.
type confirmOverlay struct {
	message string
	isDone  bool
	result  interface{} // bool (true=confirmed) or nil
}

func newConfirmOverlay(message string) *confirmOverlay {
	return &confirmOverlay{message: message}
}

func (c *confirmOverlay) Update(msg tea.Msg) (overlay, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "y", "Y":
			c.isDone = true
			c.result = true
			return c, nil
		case "n", "N", "esc":
			c.isDone = true
			c.result = nil
			return c, nil
		}
	}
	return c, nil
}

func (c *confirmOverlay) View(width, height int) string {
	content := overlayBorderStyle.Render(
		fmt.Sprintf("%s\n\n%s",
			overlayTitleStyle.Render(c.message),
			overlayHintStyle.Render("y: confirm  n/esc: cancel"),
		),
	)
	return lipgloss.Place(width, height-2, lipgloss.Center, lipgloss.Center, content)
}

func (c *confirmOverlay) done() (bool, interface{}) {
	return c.isDone, c.result
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
