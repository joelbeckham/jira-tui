package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jbeckham/jira-tui/internal/config"
	"github.com/jbeckham/jira-tui/internal/jira"
)

// --- Messages ---

// connStatusMsg is sent when the startup auth check completes.
type connStatusMsg struct {
	user *jira.User
	err  error
}

// tabDataMsg delivers fetched issues (or an error) for a specific tab index.
type tabDataMsg struct {
	tabIndex int
	issues   []jira.Issue
	filter   *jira.Filter
	err      error
}

// issueUpdatedMsg is sent after a successful issue edit (status, assignee, etc.).
// The handler uses issueKey to update both the tab data and the detail view.
type issueUpdatedMsg struct {
	issueKey string
	issue    *jira.Issue // refreshed issue from API
	err      error
}

// flashMsg sets a temporary status message.
type flashMsg struct {
	text  string
	isErr bool
}

// issueDetailMsg delivers a fully-fetched issue for the detail view.
type issueDetailMsg struct {
	issueKey string
	issue    *jira.Issue
	err      error
}

// transitionsLoadedMsg delivers available transitions for the status overlay.
type transitionsLoadedMsg struct {
	issueKey    string
	transitions []jira.Transition
	err         error
}

// usersLoadedMsg delivers the user list for the assignee overlay.
type usersLoadedMsg struct {
	users []config.CachedUser
	err   error
}

// prioritiesLoadedMsg delivers the priority list for the priority overlay.
type prioritiesLoadedMsg struct {
	issues    string // issue key the overlay targets
	priorities []jira.Priority
	err        error
}

// issueDeletedMsg is sent after a successful issue deletion.
type issueDeletedMsg struct {
	issueKey string
	err      error
}

// --- View stack ---

// view is a stacked view that renders on top of the tab bar.
type view interface {
	// title returns a label for the view (e.g., issue key).
	title() string
}



// --- App model ---

// App is the root bubbletea model for jira-tui.
type App struct {
	width  int
	height int
	ready  bool

	client    *jira.Client
	user      *jira.User
	connErr   error
	checking  bool
	connected bool

	tabs      []tab
	activeTab int
	viewStack []view

	overlay       overlay       // active overlay (nil = none)
	overlayIssue  string        // issue key the overlay is targeting
	overlayAction overlayAction // which edit action the overlay is for

	flash      string // transient status message
	flashIsErr bool   // true if the flash is an error

	cachedUsers      []config.CachedUser // loaded at startup from user cache
	cachedPriorities []jira.Priority     // loaded on first use from API
}

// NewApp creates a new App model.
// Pass nil client to run without Jira connection (for testing).
func NewApp(client *jira.Client, tabs []config.TabConfig) App {
	t := make([]tab, len(tabs))
	for i, cfg := range tabs {
		t[i] = newTab(cfg)
	}
	return App{
		client:   client,
		checking: client != nil,
		tabs:     t,
	}
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd {
	if a.client == nil {
		return nil
	}
	return a.checkConnection()
}

// checkConnection returns a Cmd that verifies Jira credentials.
func (a App) checkConnection() tea.Cmd {
	client := a.client
	return func() tea.Msg {
		user, err := client.GetMyself(context.Background())
		return connStatusMsg{user: user, err: err}
	}
}

// loadTab returns a Cmd that fetches filter JQL then searches for issues.
func (a App) loadTab(index int) tea.Cmd {
	if a.client == nil || index < 0 || index >= len(a.tabs) {
		return nil
	}
	client := a.client
	cfg := a.tabs[index].config

	return func() tea.Msg {
		ctx := context.Background()

		filterID := cfg.FilterID
		// If only filter_url is provided, try to extract filter ID from it
		// For now, we require filter_id
		if filterID == "" {
			return tabDataMsg{
				tabIndex: index,
				err:      fmt.Errorf("filter_id is required (filter_url not yet supported)"),
			}
		}

		filter, err := client.GetFilter(ctx, filterID)
		if err != nil {
			return tabDataMsg{tabIndex: index, err: err}
		}

		result, err := client.SearchIssues(ctx, jira.SearchOptions{
			JQL:        filter.JQL,
			Fields:     cfg.Columns,
			MaxResults: 50,
		})
		if err != nil {
			return tabDataMsg{tabIndex: index, filter: filter, err: err}
		}

		return tabDataMsg{
			tabIndex: index,
			filter:   filter,
			issues:   result.Issues,
		}
	}
}

// loadAllTabs returns Cmds that load every tab in parallel.
func (a App) loadAllTabs() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(a.tabs))
	for i := range a.tabs {
		cmds = append(cmds, a.loadTab(i))
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		// Resize all tab tables
		tableH := a.tableHeight()
		for i := range a.tabs {
			a.tabs[i].setSize(a.width, tableH)
		}
		// Resize detail view if on stack
		if len(a.viewStack) > 0 {
			if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
				dv.setSize(a.width, a.height)
			}
		}

	case connStatusMsg:
		a.checking = false
		if msg.err != nil {
			a.connErr = msg.err
		} else {
			a.user = msg.user
			a.connected = true
			// Load user cache (non-blocking, best effort)
			a.cachedUsers, _ = config.LoadUserCache()
			// Auth succeeded — load all tabs eagerly
			return a, a.loadAllTabs()
		}

	case tabDataMsg:
		if msg.tabIndex >= 0 && msg.tabIndex < len(a.tabs) {
			tab := &a.tabs[msg.tabIndex]
			if msg.filter != nil {
				tab.jiraFilter = msg.filter
			}
			if msg.err != nil {
				tab.setError(msg.err.Error())
			} else {
				tab.setIssues(msg.issues)
			}
		}

	case issueUpdatedMsg:
		a.flash = ""
		if msg.err != nil {
			a.flash = msg.err.Error()
			a.flashIsErr = true
		} else if msg.issue != nil {
			a.applyIssueUpdate(msg.issueKey, msg.issue)
			a.flash = msg.issueKey + " updated"
			a.flashIsErr = false
		}

	case flashMsg:
		a.flash = msg.text
		a.flashIsErr = msg.isErr

	case issueDetailMsg:
		if msg.err != nil {
			a.flash = fmt.Sprintf("Failed to load %s: %v", msg.issueKey, msg.err)
			a.flashIsErr = true
			// Still show what we have from the search result
			if len(a.viewStack) > 0 {
				if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
					if dv.issue.Key == msg.issueKey {
						dv.loading = false
						dv.buildViewport()
					}
				}
			}
		} else if msg.issue != nil && len(a.viewStack) > 0 {
			if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
				if dv.issue.Key == msg.issueKey {
					dv.issue = *msg.issue
					dv.loading = false
					dv.buildViewport()
				}
			}
		}

	case transitionsLoadedMsg:
		if msg.err != nil {
			a.flash = msg.err.Error()
			a.flashIsErr = true
		} else {
			items := make([]selectionItem, len(msg.transitions))
			for i, t := range msg.transitions {
				items[i] = selectionItem{ID: t.ID, Label: t.Name}
			}
			a.overlay = newSelectionOverlay("Change Status", items)
			a.overlayIssue = msg.issueKey
			// overlayAction was already set to overlayActionTransition by handleEditHotkey
		}

	case prioritiesLoadedMsg:
		if msg.err != nil {
			a.flash = msg.err.Error()
			a.flashIsErr = true
		} else {
			a.cachedPriorities = msg.priorities
			items := make([]selectionItem, len(msg.priorities))
			for i, p := range msg.priorities {
				items[i] = selectionItem{ID: p.ID, Label: p.Name}
			}
			a.overlay = newSelectionOverlay("Change Priority", items)
			a.overlayIssue = msg.issues
			// overlayAction was already set to overlayActionPriority by handleEditHotkey
		}

	case usersLoadedMsg:
		if msg.err != nil {
			a.flash = msg.err.Error()
			a.flashIsErr = true
		} else {
			a.cachedUsers = msg.users
			items := make([]selectionItem, len(msg.users))
			for i, u := range msg.users {
				items[i] = selectionItem{ID: u.AccountID, Label: u.DisplayName, Desc: u.Email}
			}
			a.overlay = newSelectionOverlay("Assign To", items)
			// overlayIssue and overlayAction were already set by handleEditHotkey
		}

	case issueDeletedMsg:
		a.flash = ""
		if msg.err != nil {
			a.flash = msg.err.Error()
			a.flashIsErr = true
		} else {
			// Pop detail view if it's showing the deleted issue
			if len(a.viewStack) > 0 {
				if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
					if dv.issue.Key == msg.issueKey {
						a.viewStack = a.viewStack[:len(a.viewStack)-1]
					}
				}
			}
			// Remove from all tabs
			for ti := range a.tabs {
				for ii := range a.tabs[ti].issues {
					if a.tabs[ti].issues[ii].Key == msg.issueKey {
						a.tabs[ti].issues = append(a.tabs[ti].issues[:ii], a.tabs[ti].issues[ii+1:]...)
						a.tabs[ti].applyFilter()
						break
					}
				}
			}
			a.flash = msg.issueKey + " deleted"
			a.flashIsErr = false
		}

	case tea.KeyMsg:
		a.flash = "" // clear flash on any keypress
		return a.handleKey(msg)
	}
	return a, nil
}

// handleKey processes key input.
func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys always work
	switch key {
	case "ctrl+c":
		return a, tea.Quit
	}

	// If an overlay is active, route ALL keys to it
	if a.overlay != nil {
		var cmd tea.Cmd
		a.overlay, cmd = a.overlay.Update(msg)
		if isDone, result := a.overlay.done(); isDone {
			return a.handleOverlayResult(result)
		}
		return a, cmd
	}

	// If a view is on the stack, handle stack-specific keys
	if len(a.viewStack) > 0 {
		switch key {
		case "q":
			return a, tea.Quit
		case "esc":
			a.viewStack = a.viewStack[:len(a.viewStack)-1]
			return a, nil
		}
		// Edit hotkeys on the detail view's issue
		if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
			if model, cmd, handled := a.handleEditHotkey(msg, &dv.issue); handled {
				return model, cmd
			}
			// Delegate remaining keys to viewport (j/k scrolling, etc.)
			cmd := dv.Update(msg)
			return a, cmd
		}
		return a, nil
	}

	// If filter input is focused, route keypresses to the text input
	if a.activeTab < len(a.tabs) && a.tabs[a.activeTab].quickFilter.isFocused() {
		return a.handleFilterKey(msg)
	}

	// Tab-level keys (no stack views open, filter not focused)
	switch key {
	case "q":
		return a, tea.Quit

	case "esc":
		// If a filter is applied, clear it
		if a.activeTab < len(a.tabs) && a.tabs[a.activeTab].quickFilter.isActive() {
			a.tabs[a.activeTab].clearFilter()
			return a, nil
		}
		return a, nil

	case "/":
		// Activate filter input
		if a.activeTab < len(a.tabs) && a.tabs[a.activeTab].state == tabReady {
			a.tabs[a.activeTab].quickFilter.activate()
			return a, a.tabs[a.activeTab].quickFilter.input.Focus()
		}

	case "r":
		// Refresh active tab
		if a.connected && a.activeTab < len(a.tabs) {
			a.tabs[a.activeTab].setLoading()
			return a, a.loadTab(a.activeTab)
		}

	case "enter":
		// Push issue detail onto stack and fetch full issue
		if a.activeTab < len(a.tabs) {
			if issue := a.tabs[a.activeTab].selectedIssue(); issue != nil {
				dv := newIssueDetailView(*issue, a.width, a.height)
				a.viewStack = append(a.viewStack, &dv)
				return a, a.cmdFetchIssue(issue.Key)
			}
		}

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(key[0]-'0') - 1
		if idx < len(a.tabs) {
			// Clear filter when switching tabs
			if a.activeTab < len(a.tabs) {
				a.tabs[a.activeTab].clearFilter()
			}
			a.activeTab = idx
			return a, nil
		}

	default:
		// Edit hotkeys on the selected issue in the list
		if a.activeTab < len(a.tabs) && a.tabs[a.activeTab].state == tabReady {
			if issue := a.tabs[a.activeTab].selectedIssue(); issue != nil {
				if model, cmd, handled := a.handleEditHotkey(msg, issue); handled {
					return model, cmd
				}
			}
		}
		// Delegate to table for j/k/up/down scrolling
		if a.activeTab < len(a.tabs) && a.tabs[a.activeTab].state == tabReady {
			var cmd tea.Cmd
			a.tabs[a.activeTab].table, cmd = a.tabs[a.activeTab].table.Update(msg)
			return a, cmd
		}
	}

	return a, nil
}

// handleFilterKey routes keypresses when the filter input is focused.
func (a App) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tab := &a.tabs[a.activeTab]
	key := msg.String()

	switch key {
	case "enter":
		// Confirm filter (or clear if empty)
		tab.quickFilter.apply(tab.issues, tab.columns)
		tab.applyFilter()
		return a, nil

	case "esc":
		// Cancel filter entirely
		tab.clearFilter()
		return a, nil
	}

	// Forward to text input
	var cmd tea.Cmd
	tab.quickFilter.input, cmd = tab.quickFilter.input.Update(msg)

	// Live filter as user types
	tab.quickFilter.updateQuery(tab.issues, tab.columns)
	tab.applyFilter()

	return a, cmd
}

// tableHeight returns the height available for the issue table.
func (a App) tableHeight() int {
	// Reserve: tab bar (1) + margin (1) + status/help line (1) + margin (1)
	h := a.height - 4
	// If the active tab has a filter bar visible, reserve 1 more line
	if a.activeTab < len(a.tabs) && a.tabs[a.activeTab].quickFilter.isActive() {
		h--
	}
	if h < 3 {
		h = 3
	}
	return h
}

// --- View ---

// View implements tea.Model.
func (a App) View() string {
	if !a.ready {
		return "Loading..."
	}

	var sections []string

	// Tab bar
	sections = append(sections, a.renderTabBar())

	// Main content area
	if a.overlay != nil {
		// Render the underlying view then overlay on top
		if len(a.viewStack) > 0 {
			sections = append(sections, a.overlay.View(a.width, a.height-2))
		} else {
			sections = append(sections, a.overlay.View(a.width, a.height-2))
		}
	} else if len(a.viewStack) > 0 {
		sections = append(sections, a.renderStackView())
	} else if a.checking {
		sections = append(sections, loadingStyle.Render("Connecting to Jira..."))
	} else if a.connErr != nil {
		sections = append(sections, errorStyle.Render(
			fmt.Sprintf("Connection failed: %v", a.connErr),
		))
	} else if len(a.tabs) > 0 {
		sections = append(sections, a.renderActiveTab())
	}

	// Status bar
	sections = append(sections, a.renderStatusBar())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderTabBar draws the tab strip across the top.
func (a App) renderTabBar() string {
	if len(a.tabs) == 0 {
		return ""
	}

	var tabs []string
	for i, t := range a.tabs {
		label := fmt.Sprintf(" %d %s ", i+1, t.config.Label)
		if i == a.activeTab {
			tabs = append(tabs, activeTabStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(label))
		}
	}
	return tabBarStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
}

// renderActiveTab draws the content of the currently active tab.
func (a App) renderActiveTab() string {
	if a.activeTab >= len(a.tabs) {
		return ""
	}
	t := &a.tabs[a.activeTab]

	var parts []string

	// Filter bar (if active)
	if t.quickFilter.isActive() {
		parts = append(parts, a.renderFilterBar(t))
	}

	switch t.state {
	case tabLoading:
		parts = append(parts, loadingStyle.Render("Loading issues..."))
	case tabError:
		parts = append(parts, errorStyle.Render(fmt.Sprintf("Error: %s", t.errMsg)))
	case tabEmpty:
		parts = append(parts, emptyStyle.Render("No issues found"))
	case tabReady:
		parts = append(parts, t.table.View())
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderFilterBar draws the quick filter bar for a tab.
func (a App) renderFilterBar(t *tab) string {
	var bar string
	if t.quickFilter.isFocused() {
		bar = t.quickFilter.input.View()
	} else {
		// Show confirmed filter text dimmed
		bar = filterPromptStyle.Render("/ ") + helpStyle.Render(t.quickFilter.query)
	}

	// Append match count
	count := filterCountStyle.Render(
		fmt.Sprintf("  %d of %d issues", t.quickFilter.matched, t.quickFilter.total),
	)

	return filterBarStyle.Render(bar + count)
}

// renderStackView draws the top view on the stack.
func (a App) renderStackView() string {
	if len(a.viewStack) == 0 {
		return ""
	}
	top := a.viewStack[len(a.viewStack)-1]

	switch v := top.(type) {
	case *issueDetailView:
		return v.View()
	}
	return ""
}

// renderStatusBar draws the bottom help/status line.
func (a App) renderStatusBar() string {
	var parts []string

	if a.user != nil {
		parts = append(parts, successStyle.Render(a.user.DisplayName))
	}

	// Flash message (transient feedback)
	if a.flash != "" {
		if a.flashIsErr {
			parts = append(parts, errorStyle.Render(a.flash))
		} else {
			parts = append(parts, successStyle.Render(a.flash))
		}
	}

	if len(a.viewStack) > 0 {
		parts = append(parts, helpStyle.Render("j/k: scroll  esc: back  q: quit"))
	} else {
		parts = append(parts, helpStyle.Render("j/k: navigate  enter: open  /: filter  r: refresh  1-9: tabs  q: quit"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		strings.Join(parts, helpStyle.Render("  │  ")),
	)
}

// --- Edit hotkeys ---

// editHotkeys is the set of keys that trigger issue editing actions.
var editHotkeys = map[string]bool{
	"s": true, "p": true, "d": true, "e": true,
	"t": true, "i": true, "a": true, "delete": true,
	"u": true, "y": true,
}

// handleEditHotkey processes edit hotkeys (s/p/d/e/t/i/a/del) for the given
// target issue. Returns (model, cmd, true) if the key was handled, or
// (model, nil, false) if it wasn't an edit hotkey.
func (a App) handleEditHotkey(msg tea.KeyMsg, issue *jira.Issue) (tea.Model, tea.Cmd, bool) {
	key := msg.String()
	if !editHotkeys[key] {
		return a, nil, false
	}

	// Clipboard hotkeys don't require a Jira connection.
	switch key {
	case "y":
		// Yank (copy) issue key to clipboard
		if err := clipboard.WriteAll(issue.Key); err != nil {
			a.flash = "Clipboard unavailable"
			a.flashIsErr = true
		} else {
			a.flash = "Copied " + issue.Key
			a.flashIsErr = false
		}
		return a, nil, true

	case "u":
		// Copy issue URL to clipboard
		if a.client == nil {
			a.flash = "Not connected to Jira"
			a.flashIsErr = true
			return a, nil, true
		}
		url := a.client.BrowseURL(issue.Key)
		if err := clipboard.WriteAll(url); err != nil {
			a.flash = "Clipboard unavailable"
			a.flashIsErr = true
		} else {
			a.flash = "Copied URL"
			a.flashIsErr = false
		}
		return a, nil, true
	}

	if a.client == nil {
		a.flash = "Not connected to Jira"
		a.flashIsErr = true
		return a, nil, true
	}

	switch key {
	case "d":
		// Mark as done — find the "done" category transition and execute immediately
		a.flash = "Marking " + issue.Key + " as done..."
		a.flashIsErr = false
		return a, a.cmdMarkDone(issue.Key), true

	case "i":
		// Assign to me
		if a.user == nil {
			a.flash = "Not logged in"
			a.flashIsErr = true
			return a, nil, true
		}
		a.flash = "Assigning " + issue.Key + " to you..."
		a.flashIsErr = false
		return a, a.cmdAssignToMe(issue.Key, a.user), true

	case "s":
		// Status — async fetch transitions, then show selection overlay
		a.overlayIssue = issue.Key
		a.overlayAction = overlayActionTransition
		a.flash = "Loading transitions..."
		a.flashIsErr = false
		return a, a.cmdFetchTransitions(issue.Key), true

	case "p":
		// Priority — show selection overlay with priorities (cached or fetch)
		a.overlayIssue = issue.Key
		a.overlayAction = overlayActionPriority
		if len(a.cachedPriorities) > 0 {
			items := make([]selectionItem, len(a.cachedPriorities))
			for i, p := range a.cachedPriorities {
				items[i] = selectionItem{ID: p.ID, Label: p.Name}
			}
			a.overlay = newSelectionOverlay("Change Priority", items)
			return a, nil, true
		}
		// No cache — fetch priorities from API
		a.flash = "Loading priorities..."
		a.flashIsErr = false
		return a, a.cmdFetchPriorities(issue.Key), true

	case "a":
		// Assignee — show selection overlay with cached users (or fetch them)
		a.overlayIssue = issue.Key
		a.overlayAction = overlayActionAssignee
		if len(a.cachedUsers) > 0 {
			items := make([]selectionItem, len(a.cachedUsers))
			for i, u := range a.cachedUsers {
				items[i] = selectionItem{ID: u.AccountID, Label: u.DisplayName, Desc: u.Email}
			}
			a.overlay = newSelectionOverlay("Assign To", items)
			return a, nil, true
		}
		// No cache — fetch users from API
		a.flash = "Loading users..."
		a.flashIsErr = false
		return a, a.cmdFetchAndCacheUsers(), true

	case "t":
		// Title — text input overlay pre-filled with current summary
		a.overlay = newTextInputOverlay("Edit Title", issue.Fields.Summary)
		a.overlayIssue = issue.Key
		a.overlayAction = overlayActionTitle
		return a, nil, true

	case "e":
		// Description — text editor overlay pre-filled with current description
		desc := extractADFText(issue.Fields.Description)
		a.overlay = newTextEditorOverlay("Edit Description", desc, a.width, a.height)
		a.overlayIssue = issue.Key
		a.overlayAction = overlayActionDescription
		return a, nil, true

	case "delete":
		// Delete — confirmation overlay
		a.overlay = newConfirmOverlay(fmt.Sprintf("Delete %s? This cannot be undone.", issue.Key))
		a.overlayIssue = issue.Key
		a.overlayAction = overlayActionDelete
		return a, nil, true
	}

	return a, nil, false
}

// overlayAction identifies which edit action the overlay result maps to.
type overlayAction int

const (
	overlayActionNone overlayAction = iota
	overlayActionTransition
	overlayActionPriority
	overlayActionAssignee
	overlayActionTitle
	overlayActionDescription
	overlayActionDelete
)

// handleOverlayResult processes the result of a completed overlay and dispatches
// the appropriate API call. Called when overlay.done() returns true.
func (a App) handleOverlayResult(result interface{}) (tea.Model, tea.Cmd) {
	issueKey := a.overlayIssue
	action := a.overlayAction
	a.overlay = nil
	a.overlayIssue = ""
	a.overlayAction = overlayActionNone

	if result == nil {
		// User cancelled
		return a, nil
	}

	switch action {
	case overlayActionTransition:
		item := result.(*selectionItem)
		a.flash = "Transitioning " + issueKey + "..."
		a.flashIsErr = false
		return a, a.cmdTransitionIssue(issueKey, item.ID)

	case overlayActionPriority:
		item := result.(*selectionItem)
		a.flash = "Setting priority on " + issueKey + "..."
		a.flashIsErr = false
		return a, a.cmdUpdateField(issueKey, map[string]interface{}{
			"priority": map[string]interface{}{"id": item.ID},
		})

	case overlayActionAssignee:
		item := result.(*selectionItem)
		a.flash = "Assigning " + issueKey + "..."
		a.flashIsErr = false
		return a, a.cmdUpdateField(issueKey, map[string]interface{}{
			"assignee": map[string]interface{}{"accountId": item.ID},
		})

	case overlayActionTitle:
		newTitle := result.(string)
		a.flash = "Updating title of " + issueKey + "..."
		a.flashIsErr = false
		return a, a.cmdUpdateField(issueKey, map[string]interface{}{
			"summary": newTitle,
		})

	case overlayActionDescription:
		newDesc := result.(string)
		a.flash = "Updating description of " + issueKey + "..."
		a.flashIsErr = false
		return a, a.cmdUpdateField(issueKey, map[string]interface{}{
			"description": makeADFDocument(newDesc),
		})

	case overlayActionDelete:
		a.flash = "Deleting " + issueKey + "..."
		a.flashIsErr = false
		return a, a.cmdDeleteIssue(issueKey)
	}

	return a, nil
}

// cmdFetchIssue fetches the full issue details for the detail view.
func (a App) cmdFetchIssue(issueKey string) tea.Cmd {
	if a.client == nil {
		return nil
	}
	client := a.client
	return func() tea.Msg {
		issue, err := client.GetIssue(context.Background(), issueKey)
		if err != nil {
			return issueDetailMsg{issueKey: issueKey, err: err}
		}
		return issueDetailMsg{issueKey: issueKey, issue: issue}
	}
}

// cmdMarkDone fetches transitions, finds the "done" category, and executes it.
func (a App) cmdMarkDone(issueKey string) tea.Cmd {
	client := a.client
	return func() tea.Msg {
		ctx := context.Background()

		transitions, err := client.GetTransitions(ctx, issueKey)
		if err != nil {
			return issueUpdatedMsg{issueKey: issueKey, err: fmt.Errorf("get transitions: %w", err)}
		}

		// Find a transition whose target status category is "done"
		var doneTransition *jira.Transition
		for i, t := range transitions {
			if t.To != nil && t.To.StatusCategory != nil && t.To.StatusCategory.Key == "done" {
				doneTransition = &transitions[i]
				break
			}
		}
		if doneTransition == nil {
			return issueUpdatedMsg{issueKey: issueKey, err: fmt.Errorf("no 'done' transition available for %s", issueKey)}
		}

		if err := client.TransitionIssue(ctx, issueKey, doneTransition.ID); err != nil {
			return issueUpdatedMsg{issueKey: issueKey, err: fmt.Errorf("transition: %w", err)}
		}

		// Re-fetch the issue to get the updated state
		issue, err := client.GetIssue(ctx, issueKey)
		if err != nil {
			return issueUpdatedMsg{issueKey: issueKey, err: fmt.Errorf("refresh: %w", err)}
		}
		return issueUpdatedMsg{issueKey: issueKey, issue: issue}
	}
}

// cmdAssignToMe assigns the issue to the current user and re-fetches it.
func (a App) cmdAssignToMe(issueKey string, user *jira.User) tea.Cmd {
	client := a.client
	return func() tea.Msg {
		ctx := context.Background()

		if err := client.AssignIssue(ctx, issueKey, user.AccountID); err != nil {
			return issueUpdatedMsg{issueKey: issueKey, err: fmt.Errorf("assign: %w", err)}
		}

		issue, err := client.GetIssue(ctx, issueKey)
		if err != nil {
			return issueUpdatedMsg{issueKey: issueKey, err: fmt.Errorf("refresh: %w", err)}
		}
		return issueUpdatedMsg{issueKey: issueKey, issue: issue}
	}
}

// applyIssueUpdate updates the issue in both the tab data and the detail view.
func (a *App) applyIssueUpdate(issueKey string, updated *jira.Issue) {
	// Update in all tabs
	for ti := range a.tabs {
		for ii := range a.tabs[ti].issues {
			if a.tabs[ti].issues[ii].Key == issueKey {
				a.tabs[ti].issues[ii] = *updated
				// Re-apply filter and rebuild table rows
				a.tabs[ti].applyFilter()
				break
			}
		}
	}

	// Update the detail view if it's on the stack
	if len(a.viewStack) > 0 {
		if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
			if dv.issue.Key == issueKey {
				dv.updateIssue(*updated)
			}
		}
	}
}

// --- Overlay command functions ---

// cmdFetchTransitions fetches available transitions for an issue.
func (a App) cmdFetchTransitions(issueKey string) tea.Cmd {
	if a.client == nil {
		return nil
	}
	client := a.client
	return func() tea.Msg {
		transitions, err := client.GetTransitions(context.Background(), issueKey)
		if err != nil {
			return transitionsLoadedMsg{issueKey: issueKey, err: fmt.Errorf("get transitions: %w", err)}
		}
		return transitionsLoadedMsg{issueKey: issueKey, transitions: transitions}
	}
}

// cmdFetchPriorities fetches available priorities from the Jira instance.
func (a App) cmdFetchPriorities(issueKey string) tea.Cmd {
	if a.client == nil {
		return nil
	}
	client := a.client
	return func() tea.Msg {
		priorities, err := client.GetPriorities(context.Background())
		if err != nil {
			return prioritiesLoadedMsg{issues: issueKey, err: fmt.Errorf("get priorities: %w", err)}
		}
		return prioritiesLoadedMsg{issues: issueKey, priorities: priorities}
	}
}

// cmdTransitionIssue executes a transition then re-fetches the issue.
func (a App) cmdTransitionIssue(issueKey, transitionID string) tea.Cmd {
	client := a.client
	return func() tea.Msg {
		ctx := context.Background()
		if err := client.TransitionIssue(ctx, issueKey, transitionID); err != nil {
			return issueUpdatedMsg{issueKey: issueKey, err: fmt.Errorf("transition: %w", err)}
		}
		issue, err := client.GetIssue(ctx, issueKey)
		if err != nil {
			return issueUpdatedMsg{issueKey: issueKey, err: fmt.Errorf("refresh: %w", err)}
		}
		return issueUpdatedMsg{issueKey: issueKey, issue: issue}
	}
}

// cmdUpdateField updates one or more fields on an issue then re-fetches it.
func (a App) cmdUpdateField(issueKey string, fields map[string]interface{}) tea.Cmd {
	client := a.client
	return func() tea.Msg {
		ctx := context.Background()
		if err := client.UpdateIssue(ctx, issueKey, fields); err != nil {
			return issueUpdatedMsg{issueKey: issueKey, err: fmt.Errorf("update: %w", err)}
		}
		issue, err := client.GetIssue(ctx, issueKey)
		if err != nil {
			return issueUpdatedMsg{issueKey: issueKey, err: fmt.Errorf("refresh: %w", err)}
		}
		return issueUpdatedMsg{issueKey: issueKey, issue: issue}
	}
}

// cmdDeleteIssue deletes an issue from Jira.
func (a App) cmdDeleteIssue(issueKey string) tea.Cmd {
	client := a.client
	return func() tea.Msg {
		if err := client.DeleteIssue(context.Background(), issueKey, false); err != nil {
			return issueDeletedMsg{issueKey: issueKey, err: fmt.Errorf("delete: %w", err)}
		}
		return issueDeletedMsg{issueKey: issueKey}
	}
}

// cmdFetchAndCacheUsers fetches all users from Jira and saves them to the cache.
func (a App) cmdFetchAndCacheUsers() tea.Cmd {
	client := a.client
	return func() tea.Msg {
		ctx := context.Background()
		users, err := client.SearchAllUsers(ctx)
		if err != nil {
			return usersLoadedMsg{err: fmt.Errorf("fetch users: %w", err)}
		}

		cached := make([]config.CachedUser, len(users))
		for i, u := range users {
			cached[i] = config.CachedUser{
				AccountID:   u.AccountID,
				DisplayName: u.DisplayName,
				Email:       u.Email,
			}
		}

		// Best-effort save to disk cache
		_ = config.SaveUserCache(cached)

		return usersLoadedMsg{users: cached}
	}
}
