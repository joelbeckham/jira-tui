package tui

import (
	"testing"

	"github.com/jbeckham/jira-tui/internal/config"
	"github.com/jbeckham/jira-tui/internal/jira"
)

func TestNewTab(t *testing.T) {
	cfg := config.TabConfig{
		Label:    "Sprint",
		FilterID: "123",
		Columns:  []string{"key", "summary", "status"},
	}
	tab := newTab(cfg)

	if tab.config.Label != "Sprint" {
		t.Errorf("expected label 'Sprint', got %q", tab.config.Label)
	}
	if tab.state != tabLoading {
		t.Errorf("expected initial state tabLoading, got %d", tab.state)
	}
	if len(tab.columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(tab.columns))
	}
}

func TestTabSetIssues(t *testing.T) {
	cfg := config.TabConfig{
		Label:    "Test",
		FilterID: "1",
		Columns:  []string{"key", "summary"},
	}
	tab := newTab(cfg)
	tab.setSize(80, 20)

	issues := []jira.Issue{
		{Key: "A-1", Fields: jira.IssueFields{Summary: "First"}},
		{Key: "A-2", Fields: jira.IssueFields{Summary: "Second"}},
	}
	tab.setIssues(issues)

	if tab.state != tabReady {
		t.Errorf("expected tabReady, got %d", tab.state)
	}
	if len(tab.issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(tab.issues))
	}
}

func TestTabSetIssuesEmpty(t *testing.T) {
	cfg := config.TabConfig{
		Label:    "Empty",
		FilterID: "1",
		Columns:  []string{"key", "summary"},
	}
	tab := newTab(cfg)
	tab.setSize(80, 20)
	tab.setIssues(nil)

	if tab.state != tabEmpty {
		t.Errorf("expected tabEmpty, got %d", tab.state)
	}
}

func TestTabSetError(t *testing.T) {
	cfg := config.TabConfig{
		Label:    "Err",
		FilterID: "1",
		Columns:  []string{"key"},
	}
	tab := newTab(cfg)
	tab.setError("something broke")

	if tab.state != tabError {
		t.Errorf("expected tabError, got %d", tab.state)
	}
	if tab.errMsg != "something broke" {
		t.Errorf("expected 'something broke', got %q", tab.errMsg)
	}
}

func TestTabSetLoading(t *testing.T) {
	cfg := config.TabConfig{
		Label:    "Load",
		FilterID: "1",
		Columns:  []string{"key"},
	}
	tab := newTab(cfg)
	tab.setSize(80, 20)
	tab.setIssues([]jira.Issue{{Key: "X-1"}})
	tab.setLoading()

	if tab.state != tabLoading {
		t.Errorf("expected tabLoading, got %d", tab.state)
	}
}

func TestTabSelectedIssue(t *testing.T) {
	cfg := config.TabConfig{
		Label:    "Sel",
		FilterID: "1",
		Columns:  []string{"key", "summary"},
	}
	tab := newTab(cfg)
	tab.setSize(80, 20)

	// No issues → nil
	if got := tab.selectedIssue(); got != nil {
		t.Error("expected nil when no issues")
	}

	issues := []jira.Issue{
		{Key: "S-1", Fields: jira.IssueFields{Summary: "One"}},
		{Key: "S-2", Fields: jira.IssueFields{Summary: "Two"}},
	}
	tab.setIssues(issues)
	selected := tab.selectedIssue()
	if selected == nil {
		t.Fatal("expected selected issue, got nil")
	}
	if selected.Key != "S-1" {
		t.Errorf("expected S-1 (first row), got %s", selected.Key)
	}
}

func TestIssuesToRows(t *testing.T) {
	cols := []string{"key", "summary", "status"}
	issues := []jira.Issue{
		{
			Key: "T-1",
			Fields: jira.IssueFields{
				Summary: "Test summary",
				Status:  &jira.Status{Name: "In Progress"},
			},
		},
	}
	rows := issuesToRows(issues, cols)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0][0] != "T-1" {
		t.Errorf("expected row[0][0]='T-1', got %q", rows[0][0])
	}
	if rows[0][1] != "Test summary" {
		t.Errorf("expected row[0][1]='Test summary', got %q", rows[0][1])
	}
	if rows[0][2] != "In Progress" {
		t.Errorf("expected row[0][2]='In Progress', got %q", rows[0][2])
	}
}

func TestIssuesToRowsPriorityUsesIcon(t *testing.T) {
	cols := []string{"key", "priority"}
	issues := []jira.Issue{
		{
			Key: "P-1",
			Fields: jira.IssueFields{
				Priority: &jira.Named{Name: "High"},
			},
		},
	}
	rows := issuesToRows(issues, cols)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	// Priority column should be the plain icon, not the text name
	if rows[0][1] == "High" {
		t.Error("expected priority to use icon, but got plain text 'High'")
	}
	if rows[0][1] != "↑" {
		t.Errorf("expected priority icon ↑ in row, got %q", rows[0][1])
	}
}

func TestFieldValue(t *testing.T) {
	issue := jira.Issue{
		Key: "F-1",
		Fields: jira.IssueFields{
			Summary:  "My summary",
			Status:   &jira.Status{Name: "Done"},
			Priority: &jira.Named{Name: "High"},
			Assignee: &jira.User{DisplayName: "Alice"},
			Reporter: &jira.User{DisplayName: "Bob"},
				IssueType: &jira.Named{Name: "Bug"},
			Project:  &jira.Named{Name: "FooProj"},
		},
	}

	tests := []struct {
		col    string
		expect string
	}{
		{"key", "F-1"},
		{"summary", "My summary"},
		{"status", "Done"},
		{"priority", "High"},
		{"assignee", "Alice"},
		{"reporter", "Bob"},
		{"type", "Bug"},
		{"project", "FooProj"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		got := fieldValue(issue, tt.col)
		if got != tt.expect {
			t.Errorf("fieldValue(%q) = %q, want %q", tt.col, got, tt.expect)
		}
	}
}

func TestFieldValueNilFields(t *testing.T) {
	issue := jira.Issue{Key: "N-1", Fields: jira.IssueFields{}}

	// Nil nested fields should return empty string, not panic
	for _, col := range []string{"status", "priority", "assignee", "reporter", "type", "project"} {
		got := fieldValue(issue, col)
		if got != "" {
			t.Errorf("fieldValue(%q) with nil field = %q, want empty", col, got)
		}
	}
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"2024-01-15T10:30:00.000+0000", "2024-01-15"},
		{"2024-12-25", "2024-12-25"},
		{"", ""},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		got := formatDate(tt.input)
		if got != tt.expect {
			t.Errorf("formatDate(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

// --- Quick filter tests ---

func TestTabApplyFilter(t *testing.T) {
	cfg := config.TabConfig{
		Label:    "Filter",
		FilterID: "1",
		Columns:  []string{"key", "summary", "status"},
	}
	tab := newTab(cfg)
	tab.setSize(100, 20)

	issues := []jira.Issue{
		{Key: "F-1", Fields: jira.IssueFields{Summary: "Fix login", Status: &jira.Status{Name: "Open"}}},
		{Key: "F-2", Fields: jira.IssueFields{Summary: "Add dashboard", Status: &jira.Status{Name: "Done"}}},
		{Key: "F-3", Fields: jira.IssueFields{Summary: "Fix logout", Status: &jira.Status{Name: "Open"}}},
	}
	tab.setIssues(issues)

	// Activate filter and type "login"
	tab.quickFilter.activate()
	tab.quickFilter.input.SetValue("login")
	tab.quickFilter.apply(tab.issues, tab.columns)
	tab.applyFilter()

	// Table should now show only 1 row
	rows := tab.table.Rows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 row after filter, got %d", len(rows))
	}
	if rows[0][0] != "F-1" {
		t.Errorf("expected F-1, got %s", rows[0][0])
	}
}

func TestTabClearFilter(t *testing.T) {
	cfg := config.TabConfig{
		Label:    "Clear",
		FilterID: "1",
		Columns:  []string{"key", "summary"},
	}
	tab := newTab(cfg)
	tab.setSize(100, 20)

	issues := []jira.Issue{
		{Key: "C-1", Fields: jira.IssueFields{Summary: "One"}},
		{Key: "C-2", Fields: jira.IssueFields{Summary: "Two"}},
	}
	tab.setIssues(issues)

	// Apply a filter
	tab.quickFilter.activate()
	tab.quickFilter.input.SetValue("One")
	tab.quickFilter.apply(tab.issues, tab.columns)
	tab.applyFilter()

	if len(tab.table.Rows()) != 1 {
		t.Fatalf("expected 1 row after filter, got %d", len(tab.table.Rows()))
	}

	// Clear the filter
	tab.clearFilter()

	if len(tab.table.Rows()) != 2 {
		t.Errorf("expected 2 rows after clear, got %d", len(tab.table.Rows()))
	}
	if tab.quickFilter.isActive() {
		t.Error("expected filter to be inactive after clear")
	}
}

func TestTabSelectedIssueWithFilter(t *testing.T) {
	cfg := config.TabConfig{
		Label:    "SelF",
		FilterID: "1",
		Columns:  []string{"key", "summary"},
	}
	tab := newTab(cfg)
	tab.setSize(100, 20)

	issues := []jira.Issue{
		{Key: "S-1", Fields: jira.IssueFields{Summary: "Alpha"}},
		{Key: "S-2", Fields: jira.IssueFields{Summary: "Beta"}},
		{Key: "S-3", Fields: jira.IssueFields{Summary: "Alpha two"}},
	}
	tab.setIssues(issues)

	// Filter to "Alpha" — should match S-1 and S-3
	tab.quickFilter.activate()
	tab.quickFilter.input.SetValue("Alpha")
	tab.quickFilter.apply(tab.issues, tab.columns)
	tab.applyFilter()

	selected := tab.selectedIssue()
	if selected == nil {
		t.Fatal("expected selected issue, got nil")
	}
	if selected.Key != "S-1" {
		t.Errorf("expected S-1 as first filtered result, got %s", selected.Key)
	}
}

func TestTabSetIssuesClearsFilter(t *testing.T) {
	cfg := config.TabConfig{
		Label:    "Refresh",
		FilterID: "1",
		Columns:  []string{"key", "summary"},
	}
	tab := newTab(cfg)
	tab.setSize(100, 20)

	issues := []jira.Issue{
		{Key: "R-1", Fields: jira.IssueFields{Summary: "One"}},
		{Key: "R-2", Fields: jira.IssueFields{Summary: "Two"}},
	}
	tab.setIssues(issues)

	// Activate and apply a filter
	tab.quickFilter.activate()
	tab.quickFilter.input.SetValue("One")
	tab.quickFilter.apply(tab.issues, tab.columns)

	// Reload issues (simulates data refresh)
	newIssues := []jira.Issue{
		{Key: "R-3", Fields: jira.IssueFields{Summary: "Three"}},
	}
	tab.setIssues(newIssues)

	if tab.quickFilter.isActive() {
		t.Error("expected filter to be cleared after setIssues")
	}
	if len(tab.issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(tab.issues))
	}
}

func TestMergeSearchFields(t *testing.T) {
	t.Run("adds detail base fields", func(t *testing.T) {
		result := mergeSearchFields([]string{"key", "summary", "status"})
		// "key" should be dropped (always returned), rest merged with base fields
		want := map[string]bool{
			"summary": true, "status": true, "priority": true,
			"issuetype": true, "assignee": true, "reporter": true,
			"project": true, "created": true, "updated": true,
		}
		got := make(map[string]bool)
		for _, f := range result {
			got[f] = true
		}
		for k := range want {
			if !got[k] {
				t.Errorf("expected field %q in result", k)
			}
		}
		if got["key"] {
			t.Error("key should not be in fields (always returned by API)")
		}
	})

	t.Run("maps type to issuetype", func(t *testing.T) {
		result := mergeSearchFields([]string{"type", "summary"})
		got := make(map[string]bool)
		for _, f := range result {
			got[f] = true
		}
		if !got["issuetype"] {
			t.Error("expected 'type' to be mapped to 'issuetype'")
		}
		if got["type"] {
			t.Error("'type' should have been mapped to 'issuetype'")
		}
	})

	t.Run("deduplicates", func(t *testing.T) {
		result := mergeSearchFields([]string{"summary", "status", "priority"})
		counts := make(map[string]int)
		for _, f := range result {
			counts[f]++
		}
		for f, c := range counts {
			if c > 1 {
				t.Errorf("field %q appears %d times", f, c)
			}
		}
	})
}
