package tui

import "github.com/charmbracelet/bubbles/table"

// columnDef holds display metadata for a known Jira field column.
type columnDef struct {
	title    string
	minWidth int
	flex     bool // if true, absorbs remaining space
}

// knownColumns maps config column names to display metadata.
var knownColumns = map[string]columnDef{
	"key":      {title: "Key", minWidth: 12},
	"summary":  {title: "Summary", minWidth: 20, flex: true},
	"status":   {title: "Status", minWidth: 14},
	"priority": {title: "Priority", minWidth: 10},
	"assignee": {title: "Assignee", minWidth: 14},
	"reporter": {title: "Reporter", minWidth: 14},
	"type":     {title: "Type", minWidth: 10},
	"project":  {title: "Project", minWidth: 10},
	"created":  {title: "Created", minWidth: 12},
	"updated":  {title: "Updated", minWidth: 12},
}

// buildColumns creates bubbles table columns from config column names,
// auto-sizing to the given total width.
func buildColumns(names []string, totalWidth int) []table.Column {
	cols := make([]table.Column, len(names))
	fixedTotal := 0
	flexCount := 0

	for i, name := range names {
		def, ok := knownColumns[name]
		if !ok {
			def = columnDef{title: name, minWidth: 12}
		}
		cols[i] = table.Column{Title: def.title, Width: def.minWidth}
		if def.flex {
			flexCount++
		} else {
			fixedTotal += def.minWidth
		}
	}

	// Distribute remaining width to flex columns
	if flexCount > 0 {
		// Reserve a small gap per column for padding
		padding := len(names) * 2
		remaining := totalWidth - fixedTotal - padding
		if remaining < 0 {
			remaining = 0
		}
		perFlex := remaining / flexCount
		if perFlex < 20 {
			perFlex = 20
		}
		for i, name := range names {
			def := knownColumns[name]
			if def.flex {
				cols[i].Width = perFlex
			}
		}
	}

	return cols
}
