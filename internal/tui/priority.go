package tui

import "github.com/charmbracelet/lipgloss"

// priorityDef holds the icon and color for a Jira priority level.
type priorityDef struct {
	icon  string
	color lipgloss.Color
}

// priorityMap maps priority names (case-sensitive, as returned by Jira) to their display definition.
var priorityMap = map[string]priorityDef{
	"Blocked":     {icon: "⊘", color: lipgloss.Color("#FF5630")},
	"Blocker":     {icon: "⊘", color: lipgloss.Color("#FF5630")},
	"Critical":    {icon: "⏶⏶", color: lipgloss.Color("#FF5630")},
	"Highest":     {icon: "⏶⏶", color: lipgloss.Color("#FF5630")},
	"High":        {icon: "⏶", color: lipgloss.Color("#FF7452")},
	"Medium":      {icon: "≡", color: lipgloss.Color("#FFAB00")},
	"Medium-Rare": {icon: "⏷", color: lipgloss.Color("#6B778C")},
	"Low":         {icon: "⏷⏷", color: lipgloss.Color("#2684FF")},
	"Lowest":      {icon: "⏷⏷", color: lipgloss.Color("#2684FF")},
}

// priorityIcon returns a colored icon string for the given priority name.
// Used in the issue list (table) view. Falls back to the raw name if unknown.
func priorityIcon(name string) string {
	if def, ok := priorityMap[name]; ok {
		return lipgloss.NewStyle().Foreground(def.color).Render(def.icon)
	}
	return name
}

// priorityLabel returns a colored "icon name" string for the given priority name.
// Used in the issue detail view. Falls back to the raw name if unknown.
func priorityLabel(name string) string {
	if def, ok := priorityMap[name]; ok {
		style := lipgloss.NewStyle().Foreground(def.color)
		return style.Render(def.icon) + " " + name
	}
	return name
}
