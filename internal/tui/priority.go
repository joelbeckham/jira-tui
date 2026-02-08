package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
// The icon is returned WITHOUT ANSI styling because the bubbles table component
// uses runewidth.Truncate internally, which mangles embedded ANSI escape codes.
// Colors are applied after rendering via colorizePriorities.
// Returns blank for "Not Prioritized". Falls back to the raw name if unknown.
func priorityIcon(name string) string {
	if name == "Not Prioritized" {
		return ""
	}
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

// hexToRGB parses a "#RRGGBB" hex color string into r, g, b components.
func hexToRGB(hex string) (uint8, uint8, uint8) {
	hex = strings.TrimPrefix(hex, "#")
	val, _ := strconv.ParseUint(hex, 16, 32)
	return uint8(val >> 16), uint8(val >> 8), uint8(val)
}

// ansiColorIcon wraps an icon string in raw ANSI foreground color escape codes.
// Uses SGR 38;2 (24-bit RGB) to set color and SGR 39 to reset only the
// foreground, preserving background and bold attributes from the Selected row.
func ansiColorIcon(icon, hex string) string {
	r, g, b := hexToRGB(hex)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[39m", r, g, b, icon)
}

// priorityReplacer post-processes rendered table output to colorize known
// priority icons. Longer icons (↑↑, ↓↓) are listed first so the Replacer's
// trie-based matching handles them before single-character subsets (↑, ↓).
var priorityReplacer = strings.NewReplacer(
	"⊘", ansiColorIcon("⊘", "#FF5630"),
	"↑↑", ansiColorIcon("↑↑", "#FF5630"),
	"↓↓", ansiColorIcon("↓↓", "#2684FF"),
	"↑", ansiColorIcon("↑", "#FF7452"),
	"≡", ansiColorIcon("≡", "#FFAB00"),
	"↓", ansiColorIcon("↓", "#6B778C"),
)

// colorizePriorities applies ANSI foreground colors to known priority icons
// in a rendered table string. This works around the bubbles table's use of
// runewidth.Truncate (which doesn't handle embedded ANSI codes) by applying
// colors after layout is computed.
func colorizePriorities(s string) string {
	return priorityReplacer.Replace(s)
}
