package tui

import (
	"testing"

	"github.com/jbeckham/jira-tui/internal/jira"
)

var testIssues = []jira.Issue{
	{Key: "PROJ-1", Fields: jira.IssueFields{
		Summary: "Fix login page",
		Status:  &jira.Status{Name: "In Progress"},
	}},
	{Key: "PROJ-2", Fields: jira.IssueFields{
		Summary: "Update dashboard",
		Status:  &jira.Status{Name: "Done"},
	}},
	{Key: "PROJ-3", Fields: jira.IssueFields{
		Summary: "Add search feature",
		Status:  &jira.Status{Name: "To Do"},
	}},
}

var testColumns = []string{"key", "summary", "status"}

func TestFilterIssuesMatchesSummary(t *testing.T) {
	result := filterIssues(testIssues, testColumns, "login")
	if len(result) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result))
	}
	if result[0].Key != "PROJ-1" {
		t.Errorf("expected PROJ-1, got %s", result[0].Key)
	}
}

func TestFilterIssuesCaseInsensitive(t *testing.T) {
	result := filterIssues(testIssues, testColumns, "LOGIN")
	if len(result) != 1 {
		t.Fatalf("expected 1 match (case-insensitive), got %d", len(result))
	}
}

func TestFilterIssuesMatchesKey(t *testing.T) {
	result := filterIssues(testIssues, testColumns, "PROJ-3")
	if len(result) != 1 {
		t.Fatalf("expected 1 match on key, got %d", len(result))
	}
	if result[0].Key != "PROJ-3" {
		t.Errorf("expected PROJ-3, got %s", result[0].Key)
	}
}

func TestFilterIssuesMatchesStatus(t *testing.T) {
	result := filterIssues(testIssues, testColumns, "Done")
	if len(result) != 1 {
		t.Fatalf("expected 1 match on status, got %d", len(result))
	}
	if result[0].Key != "PROJ-2" {
		t.Errorf("expected PROJ-2, got %s", result[0].Key)
	}
}

func TestFilterIssuesMultipleMatches(t *testing.T) {
	// "PROJ" appears in all keys
	result := filterIssues(testIssues, testColumns, "PROJ")
	if len(result) != 3 {
		t.Errorf("expected 3 matches, got %d", len(result))
	}
}

func TestFilterIssuesNoMatch(t *testing.T) {
	result := filterIssues(testIssues, testColumns, "zzzzz")
	if len(result) != 0 {
		t.Errorf("expected 0 matches, got %d", len(result))
	}
}

func TestFilterIssuesEmptyQuery(t *testing.T) {
	result := filterIssues(testIssues, testColumns, "")
	// strings.Contains(x, "") is always true, so all issues match.
	// Callers avoid passing empty queries; this just documents the behavior.
	if len(result) != 3 {
		t.Errorf("expected all 3 issues for empty query, got %d", len(result))
	}
}

func TestIssueFilterLifecycle(t *testing.T) {
	f := newIssueFilter()

	// Initially inactive
	if f.isActive() {
		t.Error("expected filter to be inactive initially")
	}

	// Activate
	f.activate()
	if !f.isFocused() {
		t.Error("expected filter to be focused after activate")
	}
	if !f.isActive() {
		t.Error("expected filter to be active after activate")
	}

	// Simulate typing
	f.input.SetValue("login")
	f.updateQuery(testIssues, testColumns)
	if f.matched != 1 {
		t.Errorf("expected 1 match, got %d", f.matched)
	}
	if f.total != 3 {
		t.Errorf("expected total=3, got %d", f.total)
	}

	// Apply
	f.apply(testIssues, testColumns)
	if f.isFocused() {
		t.Error("expected filter to NOT be focused after apply")
	}
	if !f.isActive() {
		t.Error("expected filter to still be active after apply")
	}
	if f.query != "login" {
		t.Errorf("expected query='login', got %q", f.query)
	}

	// Visible issues should be filtered
	visible := f.visibleIssues(testIssues)
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible issue, got %d", len(visible))
	}

	// Clear
	f.clear()
	if f.isActive() {
		t.Error("expected filter to be inactive after clear")
	}
	visible = f.visibleIssues(testIssues)
	if len(visible) != 3 {
		t.Errorf("expected all 3 issues after clear, got %d", len(visible))
	}
}

func TestIssueFilterApplyEmptyClears(t *testing.T) {
	f := newIssueFilter()
	f.activate()
	f.input.SetValue("")
	f.apply(testIssues, testColumns)

	// Applying with empty input should clear the filter
	if f.isActive() {
		t.Error("expected filter to be cleared when applying empty query")
	}
}

func TestIssueFilterVisibleIssuesWhenInactive(t *testing.T) {
	f := newIssueFilter()
	visible := f.visibleIssues(testIssues)
	if len(visible) != 3 {
		t.Errorf("expected all issues when filter inactive, got %d", len(visible))
	}
}
