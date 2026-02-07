package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jbeckham/jira-tui/internal/jira"
)

// helper to build a minimal issue for detail tests.
func testDetailIssue() jira.Issue {
	return jira.Issue{
		Key: "TEST-42",
		Fields: jira.IssueFields{
			Summary:  "Fix the widget",
			Status:   &jira.Status{Name: "In Progress", StatusCategory: &jira.StatusCategory{Key: "indeterminate"}},
			Assignee: &jira.User{DisplayName: "Alice"},
			Reporter: &jira.User{DisplayName: "Bob"},
			Priority: &jira.Named{Name: "High"},
			IssueType: &jira.Named{Name: "Bug"},
			Project:   &jira.Named{Name: "Test Project"},
			Labels:    []string{"backend", "urgent"},
			Created:   "2025-01-15T10:30:00.000+0000",
			Updated:   "2025-07-01T14:45:00.000+0000",
		},
	}
}

func TestDetailViewTitle(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	if dv.title() != "TEST-42" {
		t.Errorf("expected title TEST-42, got %s", dv.title())
	}
}

func TestDetailViewRendersKey(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "TEST-42") {
		t.Error("expected issue key in rendered content")
	}
}

func TestDetailViewRendersSummary(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "Fix the widget") {
		t.Error("expected summary in rendered content")
	}
}

func TestDetailViewRendersStatus(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "In Progress") {
		t.Error("expected status in rendered content")
	}
}

func TestDetailViewRendersAssignee(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "Alice") {
		t.Error("expected assignee in rendered content")
	}
}

func TestDetailViewRendersLabels(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "backend, urgent") {
		t.Error("expected labels in rendered content")
	}
}

func TestDetailViewRendersReporter(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "Bob") {
		t.Error("expected reporter in rendered content")
	}
}

func TestDetailViewRendersPriority(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "High") {
		t.Error("expected priority in rendered content")
	}
}

func TestDetailViewRendersType(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "Bug") {
		t.Error("expected issue type in rendered content")
	}
}

func TestDetailViewRendersProject(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "Test Project") {
		t.Error("expected project name in rendered content")
	}
}

func TestDetailViewRendersCreatedDate(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "2025-01-15 10:30") {
		t.Error("expected formatted created date in rendered content")
	}
}

func TestDetailViewRendersDescription(t *testing.T) {
	issue := testDetailIssue()
	issue.Fields.Description = map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "The widget is broken",
					},
				},
			},
		},
	}
	dv := newIssueDetailViewReady(issue, 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "The widget is broken") {
		t.Error("expected description text in rendered content")
	}
}

func TestDetailViewRendersSubtasks(t *testing.T) {
	issue := testDetailIssue()
	issue.Fields.Subtasks = []jira.Issue{
		{
			Key: "TEST-43",
			Fields: jira.IssueFields{
				Summary: "Sub-task one",
				Status:  &jira.Status{Name: "Done", StatusCategory: &jira.StatusCategory{Key: "done"}},
			},
		},
		{
			Key: "TEST-44",
			Fields: jira.IssueFields{
				Summary: "Sub-task two",
				Status:  &jira.Status{Name: "To Do", StatusCategory: &jira.StatusCategory{Key: "new"}},
			},
		},
	}
	dv := newIssueDetailViewReady(issue, 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "TEST-43") {
		t.Error("expected subtask key TEST-43")
	}
	if !strings.Contains(content, "Sub-task one") {
		t.Error("expected subtask summary")
	}
	if !strings.Contains(content, "✓") {
		t.Error("expected done checkmark for completed subtask")
	}
	if !strings.Contains(content, "Subtasks (2)") {
		t.Error("expected subtask count header")
	}
}

func TestDetailViewRendersLinkedIssues(t *testing.T) {
	issue := testDetailIssue()
	issue.Fields.IssueLinks = []jira.IssueLink{
		{
			Type: jira.LinkType{Outward: "blocks"},
			OutwardIssue: &jira.Issue{
				Key:    "TEST-50",
				Fields: jira.IssueFields{Summary: "Blocked issue"},
			},
		},
		{
			Type: jira.LinkType{Inward: "is blocked by"},
			InwardIssue: &jira.Issue{
				Key:    "TEST-51",
				Fields: jira.IssueFields{Summary: "Blocker issue"},
			},
		},
	}
	dv := newIssueDetailViewReady(issue, 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "TEST-50") {
		t.Error("expected outward linked issue key")
	}
	if !strings.Contains(content, "blocks") {
		t.Error("expected outward link type label")
	}
	if !strings.Contains(content, "TEST-51") {
		t.Error("expected inward linked issue key")
	}
	if !strings.Contains(content, "is blocked by") {
		t.Error("expected inward link type label")
	}
}

func TestDetailViewRendersParent(t *testing.T) {
	issue := testDetailIssue()
	issue.Fields.Parent = &jira.ParentIssue{
		Key:    "TEST-10",
		Fields: &jira.IssueFields{Summary: "Parent epic"},
	}
	dv := newIssueDetailViewReady(issue, 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "TEST-10") {
		t.Error("expected parent key in rendered content")
	}
	if !strings.Contains(content, "Parent epic") {
		t.Error("expected parent summary in rendered content")
	}
}

func TestDetailViewNoLabels(t *testing.T) {
	issue := testDetailIssue()
	issue.Fields.Labels = nil
	dv := newIssueDetailViewReady(issue, 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "None") {
		t.Error("expected 'None' when no labels")
	}
}

func TestDetailViewUnassigned(t *testing.T) {
	issue := testDetailIssue()
	issue.Fields.Assignee = nil
	dv := newIssueDetailViewReady(issue, 80, 24)
	content := dv.renderContent()
	if !strings.Contains(content, "Unassigned") {
		t.Error("expected 'Unassigned' when no assignee")
	}
}

func TestDetailViewViewportScrolling(t *testing.T) {
	issue := testDetailIssue()
	// Make content long enough to need scrolling
	issue.Fields.Description = "Line one\nLine two\nLine three\nLine four"
	dv := newIssueDetailViewReady(issue, 80, 10) // small height to force scroll
	if !dv.ready {
		t.Fatal("expected viewport to be ready")
	}
	// Pressing j should scroll down (no panic, returns cmd or nil)
	cmd := dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	_ = cmd // just ensure no panic
}

func TestDetailViewSetSize(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	dv.setSize(120, 40)
	if dv.width != 120 || dv.height != 40 {
		t.Errorf("expected 120x40, got %dx%d", dv.width, dv.height)
	}
	if !dv.ready {
		t.Error("expected viewport to be ready after resize")
	}
}

func TestDetailViewViewOutput(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	output := dv.View()
	if output == "" {
		t.Error("expected non-empty view output")
	}
	if !strings.Contains(output, "TEST-42") {
		t.Error("expected issue key in view output")
	}
}

func TestFormatDetailDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2025-07-01T14:45:00.000+0000", "2025-07-01 14:45"},
		{"2025-01-15T10:30:00.000+0000", "2025-01-15 10:30"},
		{"short", "short"},
		{"", ""},
	}
	for _, tt := range tests {
		got := formatDetailDate(tt.input)
		if got != tt.expected {
			t.Errorf("formatDetailDate(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestStatusColor(t *testing.T) {
	// Just make sure it doesn't panic for various inputs
	statusColor(nil)
	statusColor(&jira.Status{})
	statusColor(&jira.Status{StatusCategory: &jira.StatusCategory{Key: "new"}})
	statusColor(&jira.Status{StatusCategory: &jira.StatusCategory{Key: "indeterminate"}})
	statusColor(&jira.Status{StatusCategory: &jira.StatusCategory{Key: "done"}})
	statusColor(&jira.Status{StatusCategory: &jira.StatusCategory{Key: "unknown"}})
}

func TestDetailViewImplementsViewInterface(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	var v view = &dv
	if v.title() != "TEST-42" {
		t.Error("expected view interface to work")
	}
}

func TestDetailViewInlineHotkeys(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	content := dv.renderContent()

	hints := []struct {
		text string
		desc string
	}{
		{"(s)", "status hint"},
		{"(p)", "priority hint"},
		{"(t)", "title hint"},
		{"(e)", "description hint"},
		{"(a,i)", "assignee hint"},
	}
	for _, h := range hints {
		if !strings.Contains(content, h.text) {
			t.Errorf("expected inline %s '%s' in detail content", h.desc, h.text)
		}
	}
}

func TestRelatedIssuesOrder(t *testing.T) {
	issue := testDetailIssue()
	issue.Fields.Parent = &jira.ParentIssue{
		Key:    "PARENT-1",
		Fields: &jira.IssueFields{Summary: "Parent task"},
	}
	issue.Fields.Subtasks = []jira.Issue{
		{Key: "SUB-1", Fields: jira.IssueFields{Summary: "Subtask one"}},
		{Key: "SUB-2", Fields: jira.IssueFields{Summary: "Subtask two"}},
	}
	issue.Fields.IssueLinks = []jira.IssueLink{
		{
			Type:         jira.LinkType{Outward: "blocks"},
			OutwardIssue: &jira.Issue{Key: "LINK-1", Fields: jira.IssueFields{Summary: "Blocked issue"}},
		},
		{
			Type:        jira.LinkType{Inward: "is blocked by"},
			InwardIssue: &jira.Issue{Key: "LINK-2", Fields: jira.IssueFields{Summary: "Blocking issue"}},
		},
	}

	dv := newIssueDetailViewReady(issue, 80, 24)
	items := dv.relatedIssues()

	if len(items) != 5 {
		t.Fatalf("expected 5 related issues, got %d", len(items))
	}

	// Order: parent, subtasks, linked
	expected := []struct {
		id   string
		desc string
	}{
		{"PARENT-1", "Parent"},
		{"SUB-1", "Subtask"},
		{"SUB-2", "Subtask"},
		{"LINK-1", "blocks"},
		{"LINK-2", "is blocked by"},
	}
	for i, exp := range expected {
		if items[i].ID != exp.id {
			t.Errorf("item %d: expected ID %q, got %q", i, exp.id, items[i].ID)
		}
		if items[i].Desc != exp.desc {
			t.Errorf("item %d: expected Desc %q, got %q", i, exp.desc, items[i].Desc)
		}
	}
}

func TestRelatedIssuesEmpty(t *testing.T) {
	dv := newIssueDetailViewReady(testDetailIssue(), 80, 24)
	items := dv.relatedIssues()
	if len(items) != 0 {
		t.Errorf("expected 0 related issues, got %d", len(items))
	}
}
