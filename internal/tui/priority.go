package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jbeckham/jira-tui/internal/jira"
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

// statusCategoryColor maps Jira status category keys to ANSI color codes,
// matching the detail view's statusColor function.
var statusCategoryColor = map[string]string{
	"new":           "12",  // blue
	"indeterminate": "11",  // yellow
	"done":          "10",  // green
}

// ansiColorText wraps text in ANSI foreground color using a 256-color code.
// Uses SGR 38;5 to set color and SGR 39 to reset only the foreground.
func ansiColorText(text, colorCode string) string {
	return fmt.Sprintf("\x1b[38;5;%sm%s\x1b[39m", colorCode, text)
}

// statusNameColor overrides color for specific status names,
// taking precedence over the category-based color.
var statusNameColor = map[string]string{
	"Backlog": "240", // dark gray
	"Triage":  "248", // light gray
}

// buildStatusReplacer scans issues for unique status names and their category
// keys, returning a Replacer that colorizes those names in rendered output.
func buildStatusReplacer(issues []jira.Issue) *strings.Replacer {
	seen := make(map[string]string) // status name → color code
	for _, issue := range issues {
		s := issue.Fields.Status
		if s == nil || seen[s.Name] != "" {
			continue
		}
		// Check name-level overrides first.
		if code, ok := statusNameColor[s.Name]; ok {
			seen[s.Name] = code
			continue
		}
		catKey := ""
		if s.StatusCategory != nil {
			catKey = s.StatusCategory.Key
		}
		if code, ok := statusCategoryColor[catKey]; ok {
			seen[s.Name] = code
		} else {
			seen[s.Name] = "252" // light gray default
		}
	}
	if len(seen) == 0 {
		return nil
	}
	pairs := make([]string, 0, len(seen)*2)
	for name, code := range seen {
		pairs = append(pairs, name, ansiColorText(name, code))
	}
	return strings.NewReplacer(pairs...)
}
