package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/jbeckham/jira-tui/internal/jira"
)

// filterState tracks whether the filter bar is active and/or focused.
type filterState int

const (
	filterInactive filterState = iota // no filter bar visible
	filterFocused                     // filter bar visible, text input focused
	filterApplied                     // filter bar visible, text input blurred (confirmed)
)

// issueFilter manages client-side filtering of issues.
type issueFilter struct {
	state    filterState
	input    textinput.Model
	query    string // the confirmed or live query
	total    int    // total issues before filtering
	matched  int    // issues after filtering
	filtered []jira.Issue
}

// newIssueFilter creates an inactive filter.
func newIssueFilter() issueFilter {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.Prompt = "/ "
	ti.PromptStyle = filterPromptStyle
	ti.CharLimit = 128
	return issueFilter{
		state: filterInactive,
		input: ti,
	}
}

// activate shows the filter bar and focuses the text input.
func (f *issueFilter) activate() {
	f.state = filterFocused
	f.input.Focus()
}

// apply confirms the filter and blurs the input.
// If the query is empty, the filter is cleared instead.
func (f *issueFilter) apply(allIssues []jira.Issue, columns []string) {
	q := strings.TrimSpace(f.input.Value())
	if q == "" {
		f.clear()
		return
	}
	f.query = q
	f.state = filterApplied
	f.input.Blur()
	f.filtered = filterIssues(allIssues, columns, q)
	f.total = len(allIssues)
	f.matched = len(f.filtered)
}

// clear removes the filter entirely.
func (f *issueFilter) clear() {
	f.state = filterInactive
	f.query = ""
	f.input.SetValue("")
	f.input.Blur()
	f.filtered = nil
	f.total = 0
	f.matched = 0
}

// updateQuery live-filters as the user types.
func (f *issueFilter) updateQuery(allIssues []jira.Issue, columns []string) {
	q := strings.TrimSpace(f.input.Value())
	f.query = q
	if q == "" {
		f.filtered = allIssues
		f.total = len(allIssues)
		f.matched = len(allIssues)
	} else {
		f.filtered = filterIssues(allIssues, columns, q)
		f.total = len(allIssues)
		f.matched = len(f.filtered)
	}
}

// isActive returns true if a filter is visible (focused or applied).
func (f *issueFilter) isActive() bool {
	return f.state != filterInactive
}

// isFocused returns true if the text input has focus.
func (f *issueFilter) isFocused() bool {
	return f.state == filterFocused
}

// visibleIssues returns the filtered set, or nil if no filter is active.
func (f *issueFilter) visibleIssues(allIssues []jira.Issue) []jira.Issue {
	if f.state == filterInactive || f.query == "" {
		return allIssues
	}
	return f.filtered
}

// filterIssues returns issues where any visible field contains the query (case-insensitive).
func filterIssues(issues []jira.Issue, columns []string, query string) []jira.Issue {
	q := strings.ToLower(query)
	var result []jira.Issue
	for _, issue := range issues {
		for _, col := range columns {
			val := fieldValue(issue, col)
			if strings.Contains(strings.ToLower(val), q) {
				result = append(result, issue)
				break
			}
		}
	}
	return result
}
