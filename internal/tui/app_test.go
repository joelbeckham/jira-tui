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
	detail, ok := app.viewStack[0].(*issueDetailView)
	if !ok {
		t.Fatalf("expected *issueDetailView on stack, got %T", app.viewStack[0])
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

// --- Quick filter tests ---

// helper: create a ready app with loaded issues on tab 0
func testAppReady() App {
	app := testAppWithTabs()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app = model.(App)

	issues := []jira.Issue{
		{Key: "PROJ-1", Fields: jira.IssueFields{Summary: "Fix login page", Status: &jira.Status{Name: "Open"}}},
		{Key: "PROJ-2", Fields: jira.IssueFields{Summary: "Update dashboard", Status: &jira.Status{Name: "Done"}}},
		{Key: "PROJ-3", Fields: jira.IssueFields{Summary: "Fix logout bug", Status: &jira.Status{Name: "Open"}}},
	}
	model, _ = app.Update(tabDataMsg{tabIndex: 0, issues: issues})
	return model.(App)
}

func TestAppSlashActivatesFilter(t *testing.T) {
	app := testAppReady()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	updated := model.(App)

	if !updated.tabs[0].quickFilter.isFocused() {
		t.Error("expected filter to be focused after pressing /")
	}
}

func TestAppSlashDoesNothingWhenNotReady(t *testing.T) {
	app := testAppWithTabs()
	app.ready = true

	// Tab is still in loading state (no issues loaded)
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	updated := model.(App)

	if updated.tabs[0].quickFilter.isFocused() {
		t.Error("expected filter NOT to be focused when tab is loading")
	}
}

func TestAppEscClearsAppliedFilter(t *testing.T) {
	app := testAppReady()

	// Activate and apply a filter
	app.tabs[0].quickFilter.activate()
	app.tabs[0].quickFilter.input.SetValue("login")
	app.tabs[0].quickFilter.apply(app.tabs[0].issues, app.tabs[0].columns)
	app.tabs[0].applyFilter()

	if !app.tabs[0].quickFilter.isActive() {
		t.Fatal("precondition: filter should be active")
	}

	// Press Esc to clear
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	updated := model.(App)

	if updated.tabs[0].quickFilter.isActive() {
		t.Error("expected filter to be cleared after Esc")
	}
}

func TestAppEscCancelsFocusedFilter(t *testing.T) {
	app := testAppReady()

	// Activate filter (focused, typing)
	app.tabs[0].quickFilter.activate()
	app.tabs[0].quickFilter.input.SetValue("something")

	// Press Esc to cancel
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	updated := model.(App)

	if updated.tabs[0].quickFilter.isActive() {
		t.Error("expected filter to be cleared after Esc during typing")
	}
}

func TestAppTabSwitchClearsFilter(t *testing.T) {
	app := testAppReady()

	// Apply a filter on tab 0
	app.tabs[0].quickFilter.activate()
	app.tabs[0].quickFilter.input.SetValue("login")
	app.tabs[0].quickFilter.apply(app.tabs[0].issues, app.tabs[0].columns)

	// Switch to tab 2
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	updated := model.(App)

	if updated.activeTab != 1 {
		t.Errorf("expected activeTab=1, got %d", updated.activeTab)
	}
	// The filter on tab 0 should be cleared
	if updated.tabs[0].quickFilter.isActive() {
		t.Error("expected filter on tab 0 to be cleared after switching")
	}
}

func TestAppQDoesNotQuitWhileFilterFocused(t *testing.T) {
	app := testAppReady()

	// Activate filter
	app.tabs[0].quickFilter.activate()

	// Press 'q' — should NOT quit, should route to text input
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Error("expected 'q' to NOT quit while filter is focused")
		}
	}
}

func TestAppCtrlCQuitsEvenWhileFilterFocused(t *testing.T) {
	app := testAppReady()

	// Activate filter
	app.tabs[0].quickFilter.activate()

	// Ctrl+C should always quit
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected quit command from ctrl+c, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg from ctrl+c, got %T", msg)
	}
}

func TestAppFilterEnterConfirms(t *testing.T) {
	app := testAppReady()

	// Activate filter and type
	app.tabs[0].quickFilter.activate()
	app.tabs[0].quickFilter.input.SetValue("login")
	// Manually trigger updateQuery since we set the value directly
	app.tabs[0].quickFilter.updateQuery(app.tabs[0].issues, app.tabs[0].columns)

	// Press Enter to confirm
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(App)

	if !updated.tabs[0].quickFilter.isActive() {
		t.Error("expected filter to still be active after Enter")
	}
	if updated.tabs[0].quickFilter.isFocused() {
		t.Error("expected filter to NOT be focused after Enter (input blurred)")
	}
}

func TestAppFilterEnterWithEmptyClears(t *testing.T) {
	app := testAppReady()

	// Activate filter but don't type anything
	app.tabs[0].quickFilter.activate()

	// Press Enter with empty input → should clear filter
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(App)

	if updated.tabs[0].quickFilter.isActive() {
		t.Error("expected filter to be cleared when pressing Enter with empty input")
	}
}

func TestAppStatusBarShowsFilterHint(t *testing.T) {
	app := testAppReady()
	view := app.View()
	if !strings.Contains(view, "/: filter") {
		t.Errorf("expected '/: filter' in status bar, got: %s", view)
	}
}

// --- Edit hotkey tests ---

func TestAppStatusBarShowsEditHints(t *testing.T) {
	app := testAppReady()
	view := app.View()

	editKeys := []string{"s: status", "p: priority", "d: done", "i: assign me", "t: title", "e: desc"}
	for _, hint := range editKeys {
		if !strings.Contains(view, hint) {
			t.Errorf("expected '%s' in list view status bar, got: %s", hint, view)
		}
	}
}

func TestAppStatusBarShowsEditHintsInDetailView(t *testing.T) {
	app := testAppReady()

	// Push detail view
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	view := app.View()
	editKeys := []string{"s: status", "p: priority", "d: done", "i: assign me", "t: title", "e: desc", "esc: back"}
	for _, hint := range editKeys {
		if !strings.Contains(view, hint) {
			t.Errorf("expected '%s' in detail view status bar, got: %s", hint, view)
		}
	}
}

func TestEditHotkeyNotConnected(t *testing.T) {
	// App with nil client
	app := testAppReady()

	// 'd' should show flash error since client is nil
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated := model.(App)

	if !updated.flashIsErr {
		t.Error("expected error flash when not connected")
	}
	if !strings.Contains(updated.flash, "Not connected") {
		t.Errorf("expected 'Not connected' flash, got: %s", updated.flash)
	}
}

func TestEditHotkeyIFromDetailView(t *testing.T) {
	// App with nil client — but we need to test the routing, not the API call
	app := testAppReady()

	// Push detail view
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	// 'i' should be handled (not passed to viewport)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	updated := model.(App)

	// With nil client, should show "Not connected"
	if !updated.flashIsErr {
		t.Error("expected error flash when client is nil")
	}
}

func TestEditHotkeySpawnsOverlays(t *testing.T) {
	app := testAppReady()
	app.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")

	t.Run("p opens priority overlay", func(t *testing.T) {
		model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
		updated := model.(App)
		if updated.overlay == nil {
			t.Fatal("expected overlay to be set")
		}
		if _, ok := updated.overlay.(*selectionOverlay); !ok {
			t.Fatalf("expected selectionOverlay, got %T", updated.overlay)
		}
		if updated.overlayIssue != "PROJ-1" {
			t.Errorf("expected overlayIssue=PROJ-1, got %s", updated.overlayIssue)
		}
		if updated.overlayAction != overlayActionPriority {
			t.Errorf("expected overlayActionPriority, got %d", updated.overlayAction)
		}
	})

	t.Run("t opens title overlay", func(t *testing.T) {
		model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
		updated := model.(App)
		if updated.overlay == nil {
			t.Fatal("expected overlay to be set")
		}
		if _, ok := updated.overlay.(*textInputOverlay); !ok {
			t.Fatalf("expected textInputOverlay, got %T", updated.overlay)
		}
		if updated.overlayAction != overlayActionTitle {
			t.Errorf("expected overlayActionTitle, got %d", updated.overlayAction)
		}
	})

	t.Run("e opens description overlay", func(t *testing.T) {
		model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
		updated := model.(App)
		if updated.overlay == nil {
			t.Fatal("expected overlay to be set")
		}
		if _, ok := updated.overlay.(*textEditorOverlay); !ok {
			t.Fatalf("expected textEditorOverlay, got %T", updated.overlay)
		}
		if updated.overlayAction != overlayActionDescription {
			t.Errorf("expected overlayActionDescription, got %d", updated.overlayAction)
		}
	})

	t.Run("delete opens confirm overlay", func(t *testing.T) {
		model, _ := app.Update(tea.KeyMsg{Type: tea.KeyDelete})
		updated := model.(App)
		if updated.overlay == nil {
			t.Fatal("expected overlay to be set")
		}
		if _, ok := updated.overlay.(*confirmOverlay); !ok {
			t.Fatalf("expected confirmOverlay, got %T", updated.overlay)
		}
		if updated.overlayAction != overlayActionDelete {
			t.Errorf("expected overlayActionDelete, got %d", updated.overlayAction)
		}
	})

	t.Run("s fires async transitions fetch", func(t *testing.T) {
		model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
		updated := model.(App)
		if updated.overlay != nil {
			t.Error("overlay should NOT be set yet (transitions loading async)")
		}
		if cmd == nil {
			t.Error("expected a cmd for async transition fetch")
		}
		if updated.overlayAction != overlayActionTransition {
			t.Errorf("expected overlayActionTransition, got %d", updated.overlayAction)
		}
	})

	t.Run("a fires async user fetch when no cache", func(t *testing.T) {
		app2 := testAppReady()
		app2.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")
		app2.cachedUsers = nil // no cache
		model, cmd := app2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
		updated := model.(App)
		if updated.overlay != nil {
			t.Error("overlay should NOT be set yet (users loading async)")
		}
		if cmd == nil {
			t.Error("expected a cmd for async user fetch")
		}
	})

	t.Run("a opens overlay when cache exists", func(t *testing.T) {
		app2 := testAppReady()
		app2.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")
		app2.cachedUsers = []config.CachedUser{
			{AccountID: "abc123", DisplayName: "Alice"},
			{AccountID: "def456", DisplayName: "Bob"},
		}
		model, _ := app2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
		updated := model.(App)
		if updated.overlay == nil {
			t.Fatal("expected overlay to be set from cache")
		}
		sel, ok := updated.overlay.(*selectionOverlay)
		if !ok {
			t.Fatalf("expected selectionOverlay, got %T", updated.overlay)
		}
		if len(sel.items) != 2 {
			t.Errorf("expected 2 items, got %d", len(sel.items))
		}
	})
}

func TestEditHotkeyClearsFlashOnNextKey(t *testing.T) {
	app := testAppReady()
	app.flash = "some old message"
	app.flashIsErr = false

	// Any keypress should clear the flash
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	updated := model.(App)

	if updated.flash != "" {
		t.Errorf("expected flash to be cleared on keypress, got: %s", updated.flash)
	}
}

func TestEditHotkeyNonEditKeyPassesThrough(t *testing.T) {
	app := testAppReady()

	// 'j' is not an edit hotkey, should pass through to table
	prevCursor := app.tabs[0].table.Cursor()
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	updated := model.(App)

	// The cursor should have moved (table handled it)
	if updated.tabs[0].table.Cursor() == prevCursor && len(updated.tabs[0].issues) > 1 {
		t.Error("expected table cursor to move on 'j' keypress")
	}
}

func TestEditHotkeyWorksFromListView(t *testing.T) {
	app := testAppReady()
	app.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")

	// 's' from list view should be handled as edit hotkey
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	updated := model.(App)

	if updated.flash == "" {
		t.Error("expected flash message from edit hotkey in list view")
	}
}

func TestEditHotkeyWorksFromDetailView(t *testing.T) {
	app := testAppReady()
	app.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")

	// Push detail view
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	// 's' from detail view should be handled
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	updated := model.(App)

	if updated.flash == "" {
		t.Error("expected flash message from edit hotkey in detail view")
	}
}

func TestIssueUpdatedMsgSuccess(t *testing.T) {
	app := testAppReady()
	updatedIssue := &jira.Issue{
		Key: "PROJ-1",
		Fields: jira.IssueFields{
			Summary: "Fix login page",
			Status:  &jira.Status{Name: "Done", StatusCategory: &jira.StatusCategory{Key: "done"}},
		},
	}

	model, _ := app.Update(issueUpdatedMsg{issueKey: "PROJ-1", issue: updatedIssue})
	updated := model.(App)

	if !strings.Contains(updated.flash, "PROJ-1 updated") {
		t.Errorf("expected 'PROJ-1 updated' flash, got: %s", updated.flash)
	}
	if updated.flashIsErr {
		t.Error("expected non-error flash for successful update")
	}

	// Issue in tab should be updated
	for _, issue := range updated.tabs[0].issues {
		if issue.Key == "PROJ-1" {
			if issue.Fields.Status == nil || issue.Fields.Status.Name != "Done" {
				t.Error("expected issue status to be updated to Done in tab data")
			}
		}
	}
}

func TestIssueUpdatedMsgError(t *testing.T) {
	app := testAppReady()

	model, _ := app.Update(issueUpdatedMsg{issueKey: "PROJ-1", err: fmt.Errorf("transition failed")})
	updated := model.(App)

	if !strings.Contains(updated.flash, "transition failed") {
		t.Errorf("expected error in flash, got: %s", updated.flash)
	}
	if !updated.flashIsErr {
		t.Error("expected error flash")
	}
}

func TestIssueUpdatedMsgUpdatesDetailView(t *testing.T) {
	app := testAppReady()

	// Push detail view for PROJ-1
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	updatedIssue := &jira.Issue{
		Key: "PROJ-1",
		Fields: jira.IssueFields{
			Summary: "Fix login page - UPDATED",
			Status:  &jira.Status{Name: "Done"},
		},
	}

	model, _ = app.Update(issueUpdatedMsg{issueKey: "PROJ-1", issue: updatedIssue})
	updated := model.(App)

	dv := updated.viewStack[0].(*issueDetailView)
	if dv.issue.Fields.Summary != "Fix login page - UPDATED" {
		t.Errorf("expected detail view issue to be updated, got: %s", dv.issue.Fields.Summary)
	}
}

func TestDetailViewUpdateIssue(t *testing.T) {
	issue := jira.Issue{Key: "TEST-1", Fields: jira.IssueFields{Summary: "Original"}}
	dv := newIssueDetailViewReady(issue, 80, 24)

	updated := jira.Issue{Key: "TEST-1", Fields: jira.IssueFields{Summary: "Updated summary"}}
	dv.updateIssue(updated)

	if dv.issue.Fields.Summary != "Updated summary" {
		t.Errorf("expected summary to be updated, got: %s", dv.issue.Fields.Summary)
	}
	// Viewport content should reflect the update
	view := dv.View()
	if !strings.Contains(view, "Updated summary") {
		t.Error("expected viewport to show updated summary")
	}
}

func TestFlashMsgSetsFlash(t *testing.T) {
	app := testAppReady()

	model, _ := app.Update(flashMsg{text: "hello", isErr: false})
	updated := model.(App)

	if updated.flash != "hello" {
		t.Errorf("expected flash 'hello', got: %s", updated.flash)
	}
	if updated.flashIsErr {
		t.Error("expected non-error flash")
	}
}

func TestFlashAppearsInStatusBar(t *testing.T) {
	app := testAppReady()
	app.flash = "Issue updated"
	app.flashIsErr = false

	view := app.View()
	if !strings.Contains(view, "Issue updated") {
		t.Errorf("expected flash in view, got: %s", view)
	}
}

// --- Overlay interaction tests ---

func TestOverlayEscCancels(t *testing.T) {
	app := testAppReady()
	app.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")

	// Open priority overlay
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	app = model.(App)
	if app.overlay == nil {
		t.Fatal("expected overlay after 'p'")
	}

	// Esc should dismiss without action
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	updated := model.(App)
	if updated.overlay != nil {
		t.Error("expected overlay to be nil after Esc")
	}
	if updated.overlayAction != overlayActionNone {
		t.Error("expected overlayAction reset to None")
	}
}

func TestOverlayRoutesKeysToOverlay(t *testing.T) {
	app := testAppReady()
	app.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")

	// Open priority overlay
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	app = model.(App)

	// 'q' should NOT quit when overlay is active (routed to overlay's filter)
	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	updated := model.(App)
	// Should still have overlay active (filter typed 'q')
	if updated.overlay == nil {
		t.Error("overlay should still be active — 'q' should be routed to overlay")
	}
	// Should NOT have a quit cmd
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Error("'q' should not quit when overlay is active")
		}
	}
}

func TestOverlayCtrlCStillQuits(t *testing.T) {
	app := testAppReady()
	app.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")

	// Open overlay
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	app = model.(App)

	// Ctrl+C should still quit
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Error("expected QuitMsg from Ctrl+C with overlay active")
	}
}

func TestTransitionsLoadedMsgOpensOverlay(t *testing.T) {
	app := testAppReady()
	app.overlayAction = overlayActionTransition
	transitions := []jira.Transition{
		{ID: "11", Name: "To Do"},
		{ID: "21", Name: "In Progress"},
		{ID: "31", Name: "Done"},
	}

	model, _ := app.Update(transitionsLoadedMsg{issueKey: "PROJ-1", transitions: transitions})
	updated := model.(App)

	if updated.overlay == nil {
		t.Fatal("expected overlay after transitions loaded")
	}
	sel, ok := updated.overlay.(*selectionOverlay)
	if !ok {
		t.Fatalf("expected selectionOverlay, got %T", updated.overlay)
	}
	if len(sel.items) != 3 {
		t.Errorf("expected 3 items, got %d", len(sel.items))
	}
}

func TestTransitionsLoadedMsgError(t *testing.T) {
	app := testAppReady()
	model, _ := app.Update(transitionsLoadedMsg{
		issueKey: "PROJ-1",
		err:      fmt.Errorf("no permission"),
	})
	updated := model.(App)
	if !updated.flashIsErr {
		t.Error("expected error flash")
	}
	if !strings.Contains(updated.flash, "no permission") {
		t.Errorf("expected error in flash, got: %s", updated.flash)
	}
}

func TestUsersLoadedMsgOpensOverlay(t *testing.T) {
	app := testAppReady()
	app.overlayAction = overlayActionAssignee
	app.overlayIssue = "PROJ-1"
	users := []config.CachedUser{
		{AccountID: "a1", DisplayName: "Alice"},
		{AccountID: "b2", DisplayName: "Bob"},
	}

	model, _ := app.Update(usersLoadedMsg{users: users})
	updated := model.(App)

	if updated.overlay == nil {
		t.Fatal("expected overlay after users loaded")
	}
	if len(updated.cachedUsers) != 2 {
		t.Errorf("expected 2 cached users, got %d", len(updated.cachedUsers))
	}
}

func TestUsersLoadedMsgError(t *testing.T) {
	app := testAppReady()
	model, _ := app.Update(usersLoadedMsg{err: fmt.Errorf("fetch failed")})
	updated := model.(App)
	if !updated.flashIsErr || !strings.Contains(updated.flash, "fetch failed") {
		t.Errorf("expected error flash, got: %s", updated.flash)
	}
}

func TestIssueDeletedMsgRemovesFromTabs(t *testing.T) {
	app := testAppReady()
	origLen := len(app.tabs[0].issues)

	model, _ := app.Update(issueDeletedMsg{issueKey: "PROJ-1"})
	updated := model.(App)

	if len(updated.tabs[0].issues) != origLen-1 {
		t.Errorf("expected %d issues after delete, got %d", origLen-1, len(updated.tabs[0].issues))
	}
	for _, issue := range updated.tabs[0].issues {
		if issue.Key == "PROJ-1" {
			t.Error("PROJ-1 should have been removed from tab")
		}
	}
	if !strings.Contains(updated.flash, "PROJ-1 deleted") {
		t.Errorf("expected deletion flash, got: %s", updated.flash)
	}
}

func TestIssueDeletedMsgPopsDetailView(t *testing.T) {
	app := testAppReady()

	// Push detail view for PROJ-1
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	if len(app.viewStack) == 0 {
		t.Fatal("expected view stack to have detail view")
	}

	model, _ = app.Update(issueDeletedMsg{issueKey: "PROJ-1"})
	updated := model.(App)

	if len(updated.viewStack) != 0 {
		t.Error("expected detail view to be popped after deleting viewed issue")
	}
}

func TestIssueDeletedMsgError(t *testing.T) {
	app := testAppReady()
	model, _ := app.Update(issueDeletedMsg{issueKey: "PROJ-1", err: fmt.Errorf("permission denied")})
	updated := model.(App)
	if !updated.flashIsErr || !strings.Contains(updated.flash, "permission denied") {
		t.Errorf("expected error flash, got: %s", updated.flash)
	}
}

func TestOverlayAppearsInView(t *testing.T) {
	app := testAppReady()
	app.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")

	// Open priority overlay
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	app = model.(App)

	view := app.View()
	if !strings.Contains(view, "Change Priority") {
		t.Errorf("expected 'Change Priority' in view when overlay is active")
	}
}

func TestHandleOverlayResultTransition(t *testing.T) {
	app := testAppReady()
	app.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")
	app.overlayIssue = "PROJ-1"
	app.overlayAction = overlayActionTransition
	app.overlay = newSelectionOverlay("Change Status", []selectionItem{{ID: "31", Label: "Done"}})

	// Simulate selecting "Done" and calling handleOverlayResult
	result := &selectionItem{ID: "31", Label: "Done"}
	model, cmd := app.handleOverlayResult(result)
	updated := model.(App)

	if updated.overlay != nil {
		t.Error("expected overlay to be cleared")
	}
	if cmd == nil {
		t.Error("expected a cmd for transition")
	}
	if !strings.Contains(updated.flash, "Transitioning") {
		t.Errorf("expected transition flash, got: %s", updated.flash)
	}
}

func TestHandleOverlayResultPriority(t *testing.T) {
	app := testAppReady()
	app.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")
	app.overlayIssue = "PROJ-1"
	app.overlayAction = overlayActionPriority
	app.overlay = newSelectionOverlay("Priority", nil)

	result := &selectionItem{ID: "2", Label: "High"}
	model, cmd := app.handleOverlayResult(result)
	updated := model.(App)

	if updated.overlay != nil {
		t.Error("expected overlay to be cleared")
	}
	if cmd == nil {
		t.Error("expected a cmd for priority update")
	}
}

func TestHandleOverlayResultTitle(t *testing.T) {
	app := testAppReady()
	app.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")
	app.overlayIssue = "PROJ-1"
	app.overlayAction = overlayActionTitle

	model, cmd := app.handleOverlayResult("New Title")
	updated := model.(App)
	if cmd == nil {
		t.Error("expected a cmd for title update")
	}
	if !strings.Contains(updated.flash, "Updating title") {
		t.Errorf("expected title flash, got: %s", updated.flash)
	}
}

func TestHandleOverlayResultDescription(t *testing.T) {
	app := testAppReady()
	app.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")
	app.overlayIssue = "PROJ-1"
	app.overlayAction = overlayActionDescription

	model, cmd := app.handleOverlayResult("Updated description")
	updated := model.(App)
	if cmd == nil {
		t.Error("expected a cmd for description update")
	}
	if !strings.Contains(updated.flash, "Updating description") {
		t.Errorf("expected description flash, got: %s", updated.flash)
	}
}

func TestHandleOverlayResultDelete(t *testing.T) {
	app := testAppReady()
	app.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")
	app.overlayIssue = "PROJ-1"
	app.overlayAction = overlayActionDelete

	model, cmd := app.handleOverlayResult(true)
	updated := model.(App)
	if cmd == nil {
		t.Error("expected a cmd for delete")
	}
	if !strings.Contains(updated.flash, "Deleting") {
		t.Errorf("expected delete flash, got: %s", updated.flash)
	}
}

func TestHandleOverlayResultCancel(t *testing.T) {
	app := testAppReady()
	app.overlayIssue = "PROJ-1"
	app.overlayAction = overlayActionPriority
	app.overlay = newSelectionOverlay("Priority", nil)

	model, cmd := app.handleOverlayResult(nil)
	updated := model.(App)
	if updated.overlay != nil {
		t.Error("expected overlay cleared on cancel")
	}
	if cmd != nil {
		t.Error("expected nil cmd on cancel")
	}
}

func TestEditHotkeyFromDetailView(t *testing.T) {
	app := testAppReady()
	app.client = jira.NewClient("https://fake.atlassian.net", "test@test.com", "token")

	// Push detail view
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	// 't' should open title overlay targeting the detail view's issue
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	updated := model.(App)
	if updated.overlay == nil {
		t.Fatal("expected overlay from detail view 't' hotkey")
	}
	if updated.overlayIssue != "PROJ-1" {
		t.Errorf("expected overlayIssue=PROJ-1, got %s", updated.overlayIssue)
	}
}
