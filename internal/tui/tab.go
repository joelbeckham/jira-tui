package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/jbeckham/jira-tui/internal/config"
	"github.com/jbeckham/jira-tui/internal/jira"
)

// tabState represents the loading state of a tab.
type tabState int

const (
	tabLoading tabState = iota
	tabReady
	tabError
	tabEmpty
)

// tab holds the state for a single filter-backed tab.
type tab struct {
	config         config.TabConfig
	table          table.Model
	issues         []jira.Issue
	state          tabState
	errMsg         string
	jiraFilter     *jira.Filter // the resolved filter (contains JQL)
	columns        []string     // column names from config
	quickFilter    issueFilter  // client-side quick filter
	statusReplacer *strings.Replacer // post-render status colorizer
}

// newTab creates a tab from a TabConfig. The table is initialized empty;
// columns and rows are set once data loads and the width is known.
func newTab(cfg config.TabConfig) tab {
	t := table.New(
		table.WithFocused(true),
		table.WithHeight(10), // will be resized
	)
	s := table.DefaultStyles()
	s.Header = tableHeaderStyle
	s.Selected = tableSelectedStyle
	s.Cell = tableCellStyle
	t.SetStyles(s)

	return tab{
		config:      cfg,
		table:       t,
		state:       tabLoading,
		columns:     cfg.Columns,
		quickFilter: newIssueFilter(),
	}
}

// setSize updates the table dimensions.
func (t *tab) setSize(width, height int) {
	cols := buildColumns(t.columns, width)
	t.table.SetColumns(cols)
	t.table.SetWidth(width)
	t.table.SetHeight(height)

	// Re-render rows with new column widths if we have data
	if t.state == tabReady {
		t.table.SetRows(issuesToRows(t.issues, t.columns))
	}
}

// setIssues populates the tab with search results.
func (t *tab) setIssues(issues []jira.Issue) {
	t.issues = issues
	t.quickFilter.clear()
	t.statusReplacer = buildStatusReplacer(issues)
	if len(issues) == 0 {
		t.state = tabEmpty
	} else {
		t.state = tabReady
		t.table.SetRows(issuesToRows(issues, t.columns))
		t.table.GotoTop()
	}
}

// setError marks the tab as having an error.
func (t *tab) setError(msg string) {
	t.state = tabError
	t.errMsg = msg
}

// setLoading resets the tab to loading state.
func (t *tab) setLoading() {
	t.state = tabLoading
	t.issues = nil
}

// selectedIssue returns the issue at the cursor, or nil.
// When a quick filter is active, the cursor indexes into the filtered list.
func (t *tab) selectedIssue() *jira.Issue {
	if t.state != tabReady || len(t.issues) == 0 {
		return nil
	}
	visible := t.quickFilter.visibleIssues(t.issues)
	idx := t.table.Cursor()
	if idx >= 0 && idx < len(visible) {
		return &visible[idx]
	}
	return nil
}

// applyFilter updates the table rows based on the current quick filter.
func (t *tab) applyFilter() {
	visible := t.quickFilter.visibleIssues(t.issues)
	t.table.SetRows(issuesToRows(visible, t.columns))
	t.table.GotoTop()
}

// applyFilterKeepCursor updates table rows but preserves the cursor position.
// If the previously selected issue is still visible, the cursor stays on it.
// Otherwise the cursor stays at the same numeric index (clamped to bounds).
func (t *tab) applyFilterKeepCursor(selectedKey string) {
	visible := t.quickFilter.visibleIssues(t.issues)
	oldCursor := t.table.Cursor()
	t.table.SetRows(issuesToRows(visible, t.columns))

	// Try to find the previously selected issue by key
	for i, issue := range visible {
		if issue.Key == selectedKey {
			t.table.SetCursor(i)
			return
		}
	}

	// Fall back to same index, clamped
	if oldCursor >= len(visible) {
		oldCursor = len(visible) - 1
	}
	if oldCursor < 0 {
		oldCursor = 0
	}
	t.table.SetCursor(oldCursor)
}

// clearFilter removes the quick filter and restores the full issue list.
func (t *tab) clearFilter() {
	t.quickFilter.clear()
	t.table.SetRows(issuesToRows(t.issues, t.columns))
	t.table.GotoTop()
}

// detailBaseFields are the Jira API field names always requested so the detail
// view can render partial data immediately when opened from the list.
var detailBaseFields = []string{
	"summary", "status", "priority", "issuetype", "assignee",
	"reporter", "project", "created", "updated", "duedate",
}

// mergeSearchFields combines configured columns with the base fields needed by
// the detail view, deduplicating and mapping column names to Jira API field names.
func mergeSearchFields(columns []string) []string {
	seen := make(map[string]bool)
	var result []string
	add := func(f string) {
		// Map display column names to Jira API field names
		switch f {
		case "type":
			f = "issuetype"
		case "due_date", "due date", "due":
			f = "duedate"
		case "key":
			return // key is always returned by the API
		}
		if !seen[f] {
			seen[f] = true
			result = append(result, f)
		}
	}
	for _, f := range columns {
		add(f)
	}
	for _, f := range detailBaseFields {
		add(f)
	}
	return result
}

// issuesToRows converts issues to table rows based on the configured columns.
// Priority columns display a colored icon instead of text.
func issuesToRows(issues []jira.Issue, columns []string) []table.Row {
	rows := make([]table.Row, len(issues))
	for i, issue := range issues {
		row := make(table.Row, len(columns))
		for j, col := range columns {
			if col == "priority" && issue.Fields.Priority != nil {
				row[j] = priorityIcon(issue.Fields.Priority.Name)
			} else {
				row[j] = fieldValue(issue, col)
			}
		}
		rows[i] = row
	}
	return rows
}

// fieldValue extracts a display string for a given column name from an issue.
func fieldValue(issue jira.Issue, column string) string {
	switch column {
	case "key":
		return issue.Key
	case "summary":
		return issue.Fields.Summary
	case "status":
		if issue.Fields.Status != nil {
			return issue.Fields.Status.Name
		}
	case "priority":
		if issue.Fields.Priority != nil {
			return issue.Fields.Priority.Name
		}
	case "assignee":
		if issue.Fields.Assignee != nil {
			return issue.Fields.Assignee.DisplayName
		}
	case "reporter":
		if issue.Fields.Reporter != nil {
			return issue.Fields.Reporter.DisplayName
		}
	case "type":
		if issue.Fields.IssueType != nil {
			return issue.Fields.IssueType.Name
		}
	case "project":
		if issue.Fields.Project != nil {
			return issue.Fields.Project.Name
		}
	case "created":
		return formatDate(issue.Fields.Created)
	case "updated":
		return formatDate(issue.Fields.Updated)
	case "duedate", "due_date", "due date", "due":
		return formatDate(issue.Fields.DueDate)
	}
	return ""
}

// formatDate trims a Jira datetime to just the date portion.
func formatDate(dt string) string {
	if len(dt) >= 10 {
		return dt[:10]
	}
	return dt
}
