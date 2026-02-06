package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jbeckham/jira-tui/internal/config"
	"github.com/jbeckham/jira-tui/internal/jira"
)

// helper: create a test app with tabs
func testAppWithTabs() App {
	tabs := []config.TabConfig{
		{Label: "Sprint", FilterID: "111", Columns: []string{"key", "summary", "status"}},
		{Label: "Backlog", FilterID: "222", Columns: []string{"key", "summary"}},
	}
	app := NewApp(nil, tabs)
	return app
}

func TestAppInit(t *testing.T) {
	app := NewApp(nil, nil)
	cmd := app.Init()
	if cmd != nil {
		t.Error("Init() should return nil cmd when no client")
	}
}

func TestAppInitWithClient(t *testing.T) {
	client := jira.NewClient("https://example.atlassian.net", "test@example.com", "token")
	app := NewApp(client, nil)
	cmd := app.Init()
	if cmd == nil {
		t.Error("Init() should return a cmd when client is set")
	}
	if !app.checking {
		t.Error("expected checking=true when client is set")
	}
}

func TestAppQuitOnQ(t *testing.T) {
	app := NewApp(nil, nil)
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestAppQuitOnCtrlC(t *testing.T) {
	app := NewApp(nil, nil)
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestAppHandlesWindowSize(t *testing.T) {
	app := NewApp(nil, nil)
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	updated := model.(App)
	if updated.width != 80 || updated.height != 24 {
		t.Errorf("expected 80x24, got %dx%d", updated.width, updated.height)
	}
	if !updated.ready {
		t.Error("expected ready=true after WindowSizeMsg")
	}
}

func TestAppViewBeforeReady(t *testing.T) {
	app := NewApp(nil, nil)
	view := app.View()
	if !strings.Contains(view, "Loading") {
		t.Errorf("expected loading message, got: %s", view)
	}
}

func TestAppViewAfterReady(t *testing.T) {
	app := testAppWithTabs()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	updated := model.(App)
	view := updated.View()
	if !strings.Contains(view, "Sprint") {
		t.Errorf("expected tab label in view, got: %s", view)
	}
	if !strings.Contains(view, "quit") {
		t.Errorf("expected help text in view, got: %s", view)
	}
}

func TestAppConnStatusSuccess(t *testing.T) {
	app := NewApp(nil, nil)
	app.ready = true
	app.checking = true

	model, _ := app.Update(connStatusMsg{
		user: &jira.User{DisplayName: "Test User"},
	})
	updated := model.(App)
	if updated.checking {
		t.Error("expected checking=false after connStatusMsg")
	}
	if updated.user == nil {
		t.Fatal("expected user to be set")
	}
	if updated.user.DisplayName != "Test User" {
		t.Errorf("expected 'Test User', got %s", updated.user.DisplayName)
	}
	if !updated.connected {
		t.Error("expected connected=true after successful auth")
	}
	view := updated.View()
	if !strings.Contains(view, "Test User") {
		t.Errorf("expected user name in view, got: %s", view)
	}
}

func TestAppConnStatusError(t *testing.T) {
	app := NewApp(nil, nil)
	app.ready = true
	app.checking = true

	model, _ := app.Update(connStatusMsg{
		err: fmt.Errorf("401 Unauthorized"),
	})
	updated := model.(App)
	if updated.checking {
		t.Error("expected checking=false after connStatusMsg")
	}
	if updated.connErr == nil {
		t.Fatal("expected connErr to be set")
	}
	view := updated.View()
	if !strings.Contains(view, "Connection failed") {
		t.Errorf("expected error message, got: %s", view)
	}
}

func TestAppViewConnecting(t *testing.T) {
	app := NewApp(nil, nil)
	app.ready = true
	app.checking = true
	view := app.View()
	if !strings.Contains(view, "Connecting to Jira") {
		t.Errorf("expected connecting message, got: %s", view)
	}
}

// --- Tab switching tests ---

func TestAppTabSwitching(t *testing.T) {
	app := testAppWithTabs()
	app.ready = true

	if app.activeTab != 0 {
		t.Fatalf("expected initial activeTab=0, got %d", app.activeTab)
	}

	// Switch to tab 2
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	updated := model.(App)
	if updated.activeTab != 1 {
		t.Errorf("expected activeTab=1 after pressing 2, got %d", updated.activeTab)
	}

	// Pressing 3 (out of range) should stay at current tab
	model, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	updated = model.(App)
	if updated.activeTab != 1 {
		t.Errorf("expected activeTab=1 (3 is out of range), got %d", updated.activeTab)
	}
}

func TestAppTabBarRendering(t *testing.T) {
	app := testAppWithTabs()
	app.ready = true

	view := app.View()
	if !strings.Contains(view, "1 Sprint") {
		t.Errorf("expected '1 Sprint' in tab bar, got: %s", view)
	}
	if !strings.Contains(view, "2 Backlog") {
		t.Errorf("expected '2 Backlog' in tab bar, got: %s", view)
	}
}

func TestAppTabDataMsg(t *testing.T) {
	app := testAppWithTabs()
	app.ready = true
	// Simulate window size so tables are initialized
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app = model.(App)

	issues := []jira.Issue{
		{Key: "PROJ-1", Fields: jira.IssueFields{Summary: "First issue"}},
		{Key: "PROJ-2", Fields: jira.IssueFields{Summary: "Second issue"}},
	}

	model, _ = app.Update(tabDataMsg{
		tabIndex: 0,
		issues:   issues,
		filter:   &jira.Filter{Name: "My Filter", JQL: "project = PROJ"},
	})
	updated := model.(App)

	tab := updated.tabs[0]
	if tab.state != tabReady {
		t.Errorf("expected tabReady, got %d", tab.state)
	}
	if len(tab.issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(tab.issues))
	}
}

func TestAppTabDataError(t *testing.T) {
	app := testAppWithTabs()
	app.ready = true

	model, _ := app.Update(tabDataMsg{
		tabIndex: 0,
		err:      fmt.Errorf("filter not found"),
	})
	updated := model.(App)

	tab := updated.tabs[0]
	if tab.state != tabError {
		t.Errorf("expected tabError, got %d", tab.state)
	}
	if tab.errMsg != "filter not found" {
		t.Errorf("expected 'filter not found', got %s", tab.errMsg)
	}
}

func TestAppIssueDetailPushPop(t *testing.T) {
	app := testAppWithTabs()
	app.ready = true
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app = model.(App)

	// Load issues into tab 0
	issues := []jira.Issue{
		{Key: "PROJ-1", Fields: jira.IssueFields{Summary: "First issue"}},
	}
	model, _ = app.Update(tabDataMsg{tabIndex: 0, issues: issues})
	app = model.(App)

	// Press Enter to push detail view
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	if len(app.viewStack) != 1 {
		t.Fatalf("expected viewStack length 1, got %d", len(app.viewStack))
	}
	detail, ok := app.viewStack[0].(issueDetailView)
	if !ok {
		t.Fatalf("expected issueDetailView on stack, got %T", app.viewStack[0])
	}
	if detail.issue.Key != "PROJ-1" {
		t.Errorf("expected PROJ-1, got %s", detail.issue.Key)
	}

	// View should show the issue detail
	view := app.View()
	if !strings.Contains(view, "PROJ-1") {
		t.Errorf("expected PROJ-1 in detail view, got: %s", view)
	}
	if !strings.Contains(view, "esc: back") {
		t.Errorf("expected 'esc: back' in detail view, got: %s", view)
	}

	// Press Esc to pop
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	app = model.(App)
	if len(app.viewStack) != 0 {
		t.Errorf("expected empty viewStack after esc, got %d", len(app.viewStack))
	}
}

func TestAppTabsInitializedFromConfig(t *testing.T) {
	tabs := []config.TabConfig{
		{Label: "A", FilterID: "1"},
		{Label: "B", FilterID: "2"},
		{Label: "C", FilterID: "3"},
	}
	app := NewApp(nil, tabs)
	if len(app.tabs) != 3 {
		t.Errorf("expected 3 tabs, got %d", len(app.tabs))
	}
	if app.tabs[0].config.Label != "A" {
		t.Errorf("expected tab 0 label 'A', got '%s'", app.tabs[0].config.Label)
	}
}
