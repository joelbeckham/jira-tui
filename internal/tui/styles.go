package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")). // bright blue
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")) // dim gray

	// Tab bar styles
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("12")).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Background(lipgloss.Color("236")).
				Padding(0, 2)

	tabBarStyle = lipgloss.NewStyle().
			MarginBottom(1)

	// Table styles
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("12")).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240"))

	tableSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("12")).
				Bold(true)

	tableCellStyle = lipgloss.NewStyle()

	// Status styles
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")) // red

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")) // green

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")) // yellow

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)
