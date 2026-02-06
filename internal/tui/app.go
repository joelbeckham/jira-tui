package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// App is the root bubbletea model for jira-tui.
type App struct {
	width  int
	height int
	ready  bool
}

// NewApp creates a new App model.
func NewApp() App {
	return App{}
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		}
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
	}
	return a, nil
}

// View implements tea.Model.
func (a App) View() string {
	if !a.ready {
		return "Loading..."
	}

	title := titleStyle.Render("jira-tui")
	help := helpStyle.Render("q: quit")

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, title, "", help),
	)
}
