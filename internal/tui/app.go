package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jbeckham/jira-tui/internal/jira"
)

// connStatusMsg is sent when the startup auth check completes.
type connStatusMsg struct {
	user *jira.User
	err  error
}

// App is the root bubbletea model for jira-tui.
type App struct {
	width  int
	height int
	ready  bool

	client   *jira.Client
	user     *jira.User
	connErr  error
	checking bool
}

// NewApp creates a new App model. Pass nil client to run without Jira connection.
func NewApp(client *jira.Client) App {
	return App{
		client:   client,
		checking: client != nil,
	}
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd {
	if a.client == nil {
		return nil
	}
	return a.checkConnection()
}

// checkConnection returns a Cmd that verifies Jira credentials.
func (a App) checkConnection() tea.Cmd {
	client := a.client
	return func() tea.Msg {
		user, err := client.GetMyself(context.Background())
		return connStatusMsg{user: user, err: err}
	}
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
	case connStatusMsg:
		a.checking = false
		if msg.err != nil {
			a.connErr = msg.err
		} else {
			a.user = msg.user
		}
	}
	return a, nil
}

// View implements tea.Model.
func (a App) View() string {
	if !a.ready {
		return "Loading..."
	}

	title := titleStyle.Render("jira-tui")

	var status string
	switch {
	case a.checking:
		status = helpStyle.Render("Connecting to Jira...")
	case a.connErr != nil:
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(
			fmt.Sprintf("Connection failed: %v", a.connErr),
		)
	case a.user != nil:
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(
			fmt.Sprintf("Connected as %s", a.user.DisplayName),
		)
	}

	help := helpStyle.Render("q: quit")

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, title, "", status, "", help),
	)
}
