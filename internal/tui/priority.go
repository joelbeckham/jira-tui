package tui

import "github.com/charmbracelet/lipgloss"

// priorityDef holds the icon and color for a Jira priority level.
type priorityDef struct {
	icon  string
	color lipgloss.Color
}

// priorityMap maps priority names (case-sensitive, as returned by Jira) to their display definition.
// Icons use universally-supported Unicode characters (arrows, math symbols)
// that render correctly in all terminal fonts.
var priorityMap = map[string]priorityDef{
	"Blocked":     {icon: "⊘", color: lipgloss.Color("#FF5630")},
	"Blocker":     {icon: "⊘", color: lipgloss.Color("#FF5630")},
	"Critical":    {icon: "↑↑", color: lipgloss.Color("#FF5630")},
	"Highest":     {icon: "↑↑", color: lipgloss.Color("#FF5630")},
	"High":        {icon: "↑", color: lipgloss.Color("#FF7452")},
	"Medium":      {icon: "≡", color: lipgloss.Color("#FFAB00")},
	"Medium-Rare": {icon: "↓", color: lipgloss.Color("#6B778C")},
	"Low":         {icon: "↓↓", color: lipgloss.Color("#2684FF")},
	"Lowest":      {icon: "↓↓", color: lipgloss.Color("#2684FF")},
}

// priorityIcon returns a plain icon string for the given priority name.
// Returns the icon WITHOUT ANSI styling because the bubbles table component
// uses runewidth.Truncate internally, which mangles embedded ANSI escape codes.
// Falls back to the raw name if the priority is unknown.
func priorityIcon(name string) string {
	if def, ok := priorityMap[name]; ok {
		return def.icon
	}
	return name
}

// priorityLabel returns a colored "icon name" string for the given priority name.
// Used in the issue detail view (rendered directly, not through the table component).
// Falls back to the raw name if unknown.
func priorityLabel(name string) string {
	if def, ok := priorityMap[name]; ok {
		style := lipgloss.NewStyle().Foreground(def.color)
		return style.Render(def.icon) + " " + name
	}
	return name
}
