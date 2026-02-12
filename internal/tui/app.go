package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
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

// issueTypesLoadedMsg delivers issue types for the create overlay.
type issueTypesLoadedMsg struct {
	types []jira.IssueType
	err   error
}

// issueCreatedMsg is sent after a successful issue creation.
type issueCreatedMsg struct {
	issueKey string
	err      error
}

// commentsLoadedMsg delivers comments for the detail view.
type commentsLoadedMsg struct {
	issueKey string
	comments []jira.Comment
	err      error
}

// childrenLoadedMsg delivers child issues (parent=KEY) for the detail view.
type childrenLoadedMsg struct {
	issueKey string
	children []jira.Issue
	err      error
}

// commentAddedMsg is sent after a comment is posted to the API.
type commentAddedMsg struct {
	issueKey string
	comment  *jira.Comment
	err      error
}

// --- View stack ---

// view is a stacked view that renders on top of the tab bar.
type view interface {
	// title returns a label for the view (e.g., issue key).
	title() string
}

// boolToInt returns 1 if b is true, 0 otherwise.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// startNetwork increments the inflight counter and returns the cmd.
// If this is the first in-flight request, it also starts the spinner tick.
func (a *App) startNetwork(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	wasIdle := a.inflight == 0
	a.inflight++
	if wasIdle {
		return tea.Batch(cmd, a.spinner.Tick)
	}
	return cmd
}

// clientBaseURL returns the Jira base URL from the client, or empty string.
func (a App) clientBaseURL() string {
	if a.client == nil {
		return ""
	}
	return a.client.BaseURL()
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

	defaultProject string // project key for creating issues
	createSummary  string // holds summary during multi-step create flow

	spinner  spinner.Model // activity spinner
	inflight int           // number of in-flight network requests
}

// NewApp creates a new App model.
// Pass nil client to run without Jira connection (for testing).
func NewApp(client *jira.Client, tabs []config.TabConfig, defaultProject string) App {
	t := make([]tab, len(tabs))
	for i, cfg := range tabs {
		t[i] = newTab(cfg)
	}
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	return App{
		client:         client,
		checking:       client != nil,
		tabs:           t,
		defaultProject: defaultProject,
		spinner:        s,
		inflight:       boolToInt(client != nil), // checkConnection will be in-flight
	}
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd {
	if a.client == nil {
		return nil
	}
	return tea.Batch(a.checkConnection(), a.spinner.Tick)
}

// checkConnection returns a Cmd that verifies Jira credentials.
func (a App) checkConnection() tea.Cmd {
	client := a.client
	return func() tea.Msg {
		user, err := client.GetMyself(context.Background())
		return connStatusMsg{user: user, err: err}
	}
}

// loadTab returns a Cmd that fetches issues for a tab.
// If the tab has a jql field, it searches directly with that JQL.
// If the tab has a filter_id, it fetches the filter's JQL first.
func (a App) loadTab(index int) tea.Cmd {
	if a.client == nil || index < 0 || index >= len(a.tabs) {
		return nil
	}
	client := a.client
	cfg := a.tabs[index].config

	return func() tea.Msg {
		ctx := context.Background()

		var jql string
		var filter *jira.Filter

		switch {
		case cfg.JQL != "":
			// Direct JQL — no filter fetch needed
			jql = cfg.JQL

		case cfg.FilterID != "":
			f, err := client.GetFilter(ctx, cfg.FilterID)
			if err != nil {
				return tabDataMsg{tabIndex: index, err: err}
			}
			filter = f
			jql = f.JQL

		default:
			return tabDataMsg{
				tabIndex: index,
				err:      fmt.Errorf("filter_url is not yet supported"),
			}
		}

		result, err := client.SearchIssues(ctx, jira.SearchOptions{
			JQL:        jql,
			Fields:     mergeSearchFields(cfg.Columns),
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
		a.inflight--
		a.checking = false
		if msg.err != nil {
			a.connErr = msg.err
		} else {
			a.user = msg.user
			a.connected = true
			// Load user cache (non-blocking, best effort)
			a.cachedUsers, _ = config.LoadUserCache()
			// Auth succeeded — load all tabs eagerly
			a.inflight += len(a.tabs)
			return a, tea.Batch(a.loadAllTabs(), a.spinner.Tick)
		}

	case tabDataMsg:
		a.inflight--
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
		a.inflight--
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
		a.inflight--
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
		} else if msg.issue != nil {
			// Patch updated data into tab list rows
			a.applyIssueUpdate(msg.issueKey, msg.issue)
			// Update the detail view if it's still showing this issue
			if len(a.viewStack) > 0 {
				if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
					if dv.issue.Key == msg.issueKey {
						dv.issue = *msg.issue
						dv.loading = false
						dv.buildViewport()
					}
				}
			}
		}

	case commentsLoadedMsg:
		a.inflight--
		if msg.err != nil {
			// Silently fail — comments are supplementary
			if len(a.viewStack) > 0 {
				if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
					if dv.issue.Key == msg.issueKey {
						dv.commentsLoading = false
						dv.buildViewport()
					}
				}
			}
		} else if len(a.viewStack) > 0 {
			if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
				if dv.issue.Key == msg.issueKey {
					dv.comments = msg.comments
					dv.commentsLoading = false
					dv.buildViewport()
				}
			}
		}

	case childrenLoadedMsg:
		a.inflight--
		if len(a.viewStack) > 0 {
			if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
				if dv.issue.Key == msg.issueKey {
					if msg.err == nil {
						dv.children = msg.children
					}
					dv.childrenLoading = false
					dv.buildViewport()
				}
			}
		}

	case commentAddedMsg:
		a.inflight--
		if msg.err != nil {
			a.flash = msg.err.Error()
			a.flashIsErr = true
			// Remove the optimistic placeholder (first comment) on failure
			if len(a.viewStack) > 0 {
				if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
					if dv.issue.Key == msg.issueKey && len(dv.comments) > 0 {
						dv.comments = dv.comments[1:]
						dv.buildViewport()
					}
				}
			}
		} else {
			a.flash = "Comment added"
			a.flashIsErr = false
			// Replace the optimistic placeholder with the real comment
			if len(a.viewStack) > 0 {
				if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
					if dv.issue.Key == msg.issueKey && msg.comment != nil && len(dv.comments) > 0 {
						dv.comments[0] = *msg.comment
						dv.buildViewport()
					}
				}
			}
		}

	case transitionsLoadedMsg:
		a.inflight--
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
		a.inflight--
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
		a.inflight--
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
		a.inflight--
		if msg.err != nil {
			a.flash = "Delete failed: " + msg.err.Error()
			a.flashIsErr = true
		}
		// Success is silent — the issue was already removed optimistically

	case issueTypesLoadedMsg:
		a.inflight--
		if msg.err != nil {
			a.flash = msg.err.Error()
			a.flashIsErr = true
			a.overlay = nil
			a.overlayAction = overlayActionNone
		} else {
			items := make([]selectionItem, len(msg.types))
			for i, t := range msg.types {
				items[i] = selectionItem{ID: t.ID, Label: t.Name}
			}
			a.overlay = newSelectionOverlay("Issue Type", items)
			// overlayAction was already set to overlayActionCreateType
		}

	case issueCreatedMsg:
		a.inflight--
		a.flash = ""
		if msg.err != nil {
			a.flash = msg.err.Error()
			a.flashIsErr = true
		} else {
			a.flash = "Created " + msg.issueKey
			a.flashIsErr = false
			// Push detail view for the new issue and fetch its data
			stub := jira.Issue{Key: msg.issueKey}
			dv := newIssueDetailView(stub, a.clientBaseURL(), a.width, a.height)
			a.viewStack = append(a.viewStack, &dv)
			var cmds []tea.Cmd
			cmds = append(cmds, a.cmdFetchIssue(msg.issueKey))
			cmds = append(cmds, a.cmdFetchComments(msg.issueKey))
			cmds = append(cmds, a.cmdFetchChildren(msg.issueKey))
			a.inflight += 2 // extra inflight for comments + children
			// Refresh the active tab in the background to pick up the new issue.
			// Don't call setLoading() — keep the current list visible so esc-back is instant.
			if a.connected && a.activeTab < len(a.tabs) {
				cmds = append(cmds, a.loadTab(a.activeTab))
			}
			return a, tea.Batch(cmds...)
		}

	case spinner.TickMsg:
		if a.inflight > 0 {
			var cmd tea.Cmd
			a.spinner, cmd = a.spinner.Update(msg)
			return a, cmd
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
			// Capture the dirty issue key before popping the detail view
			var dirtyKey string
			if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
				if dv.dirty {
					dirtyKey = dv.issue.Key
				}
			}
			a.viewStack = a.viewStack[:len(a.viewStack)-1]
			// If the issue was edited, refresh just that issue in the background
			if dirtyKey != "" && a.connected {
				return a, a.cmdFetchIssue(dirtyKey)
			}
			return a, nil
		}
		// Detail-view-specific hotkeys
		if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
			if key == "enter" {
				// Drill into related issue (parent / subtask / linked)
				items := dv.relatedIssues()
				if len(items) == 0 {
					a.flash = "No related issues"
					a.flashIsErr = false
					return a, nil
				}
				a.overlay = newSelectionOverlay("Related Issues", items)
				a.overlayAction = overlayActionDrillIn
				return a, nil
			}
			if key == "c" {
				// Add comment
				if a.client == nil {
					a.flash = "Not connected to Jira"
					a.flashIsErr = true
					return a, nil
				}
				a.overlay = newTextEditorOverlay("Add Comment", "", a.width, a.height)
				a.overlayIssue = dv.issue.Key
				a.overlayAction = overlayActionAddComment
				return a, nil
			}
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
			return a, a.startNetwork(a.loadTab(a.activeTab))
		}

	case "c":
		// Create new issue
		if a.client == nil {
			a.flash = "Not connected to Jira"
			a.flashIsErr = true
			return a, nil
		}
		if a.defaultProject == "" {
			a.flash = "Set default_project in config to create issues"
			a.flashIsErr = true
			return a, nil
		}
		a.overlay = newTextInputOverlay("New Issue Summary", "")
		a.overlayAction = overlayActionCreateSummary
		return a, nil

	case "enter":
		// Push issue detail onto stack and fetch full issue + comments
		if a.activeTab < len(a.tabs) {
			if issue := a.tabs[a.activeTab].selectedIssue(); issue != nil {
				dv := newIssueDetailView(*issue, a.clientBaseURL(), a.width, a.height)
				a.viewStack = append(a.viewStack, &dv)
				a.inflight += 2 // extra inflight for comments + children
				return a, tea.Batch(
					a.startNetwork(a.cmdFetchIssue(issue.Key)),
					a.cmdFetchComments(issue.Key),
					a.cmdFetchChildren(issue.Key),
				)
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
	case "enter", "down":
		// Confirm filter (or clear if empty) and return to list
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
		rendered := colorizePriorities(t.table.View())
		if t.statusReplacer != nil {
			rendered = t.statusReplacer.Replace(rendered)
		}
		parts = append(parts, rendered)
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
		parts = append(parts, helpStyle.Render("enter: related  c: comment  d: done  del: delete  q: quit"))
	} else {
		parts = append(parts, helpStyle.Render("/: filter  c: create  o: open  q: quit"))
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
	"u": true, "y": true, "o": true,
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

	case "o":
		// Open issue in default browser
		if a.client == nil {
			a.flash = "Not connected to Jira"
			a.flashIsErr = true
			return a, nil, true
		}
		url := a.client.BrowseURL(issue.Key)
		if err := openBrowser(url); err != nil {
			a.flash = "Could not open browser"
			a.flashIsErr = true
		} else {
			a.flash = "Opened " + issue.Key + " in browser"
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

// openBrowser opens a URL in the user's default browser.
// Handles native Linux, WSL, macOS, and Windows.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default: // linux, freebsd, etc.
		// WSL: /proc/version contains "microsoft" or "Microsoft"
		if data, err := os.ReadFile("/proc/version"); err == nil {
			lower := strings.ToLower(string(data))
			if strings.Contains(lower, "microsoft") {
				// Prefer wslview (from wslu), fall back to cmd.exe
				if path, err := exec.LookPath("wslview"); err == nil {
					return exec.Command(path, url).Start()
				}
				return exec.Command("cmd.exe", "/c", "start", url).Start()
			}
		}
		// Native Linux: try xdg-open, then sensible-browser
		if path, err := exec.LookPath("xdg-open"); err == nil {
			return exec.Command(path, url).Start()
		}
		if path, err := exec.LookPath("sensible-browser"); err == nil {
			return exec.Command(path, url).Start()
		}
		return fmt.Errorf("no browser opener found (install xdg-utils)")
	}
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
	overlayActionCreateSummary // step 1: enter summary
	overlayActionCreateType    // step 2: pick issue type
	overlayActionAddComment    // add comment from detail view
	overlayActionDrillIn       // drill into a related issue from detail view
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
		// Optimistic delete: remove from UI immediately, send API call in background
		// Pop detail view if it's showing the deleted issue
		if len(a.viewStack) > 0 {
			if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
				if dv.issue.Key == issueKey {
					a.viewStack = a.viewStack[:len(a.viewStack)-1]
				}
			}
		}
		// Remove from all tabs
		for ti := range a.tabs {
			for ii := range a.tabs[ti].issues {
				if a.tabs[ti].issues[ii].Key == issueKey {
					a.tabs[ti].issues = append(a.tabs[ti].issues[:ii], a.tabs[ti].issues[ii+1:]...)
					a.tabs[ti].applyFilterKeepCursor(issueKey)
					break
				}
			}
		}
		a.flash = issueKey + " deleted"
		a.flashIsErr = false
		return a, a.cmdDeleteIssue(issueKey)

	case overlayActionCreateSummary:
		summary := result.(string)
		if strings.TrimSpace(summary) == "" {
			a.flash = "Summary cannot be empty"
			a.flashIsErr = true
			return a, nil
		}
		// Store summary and move to step 2: pick issue type
		a.createSummary = summary
		a.overlayAction = overlayActionCreateType
		a.flash = "Loading issue types..."
		a.flashIsErr = false
		return a, a.cmdFetchIssueTypes()

	case overlayActionCreateType:
		item := result.(*selectionItem)
		summary := a.createSummary
		a.createSummary = ""
		a.flash = "Creating issue..."
		a.flashIsErr = false
		return a, a.cmdCreateIssue(summary, item.Label)

	case overlayActionDrillIn:
		item := result.(*selectionItem)
		stub := jira.Issue{Key: item.ID}
		dv := newIssueDetailView(stub, a.clientBaseURL(), a.width, a.height)
		a.viewStack = append(a.viewStack, &dv)
		a.inflight += 2
		return a, tea.Batch(
			a.startNetwork(a.cmdFetchIssue(item.ID)),
			a.cmdFetchComments(item.ID),
			a.cmdFetchChildren(item.ID),
		)

	case overlayActionAddComment:
		text := result.(string)
		if strings.TrimSpace(text) == "" {
			a.flash = "Comment cannot be empty"
			a.flashIsErr = true
			return a, nil
		}
		// Optimistic: prepend a placeholder comment to the detail view
		if len(a.viewStack) > 0 {
			if dv, ok := a.viewStack[len(a.viewStack)-1].(*issueDetailView); ok {
				placeholder := jira.Comment{
					Body:    makeADFDocument(text),
					Created: "just now",
				}
				dv.comments = append([]jira.Comment{placeholder}, dv.comments...)
				dv.buildViewport()
			}
		}
		a.flash = "Adding comment..."
		a.flashIsErr = false
		return a, a.startNetwork(a.cmdAddComment(issueKey, text))
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

// cmdFetchChildren searches for child issues (parent = KEY) for the detail view.
func (a App) cmdFetchChildren(issueKey string) tea.Cmd {
	if a.client == nil {
		return nil
	}
	client := a.client
	return func() tea.Msg {
		result, err := client.SearchIssues(context.Background(), jira.SearchOptions{
			JQL:        fmt.Sprintf("parent = %s ORDER BY rank ASC", issueKey),
			Fields:     []string{"summary", "status", "issuetype", "priority"},
			MaxResults: 50,
		})
		if err != nil {
			return childrenLoadedMsg{issueKey: issueKey, err: err}
		}
		return childrenLoadedMsg{issueKey: issueKey, children: result.Issues}
	}
}

// cmdFetchComments fetches comments for the detail view.
func (a App) cmdFetchComments(issueKey string) tea.Cmd {
	if a.client == nil {
		return nil
	}
	client := a.client
	return func() tea.Msg {
		comments, err := client.GetComments(context.Background(), issueKey)
		if err != nil {
			return commentsLoadedMsg{issueKey: issueKey, err: err}
		}
		return commentsLoadedMsg{issueKey: issueKey, comments: comments}
	}
}

// cmdAddComment posts a comment to a Jira issue.
func (a App) cmdAddComment(issueKey, text string) tea.Cmd {
	if a.client == nil {
		return nil
	}
	client := a.client
	body := makeADFDocument(text)
	return func() tea.Msg {
		comment, err := client.AddComment(context.Background(), issueKey, body)
		if err != nil {
			return commentAddedMsg{issueKey: issueKey, err: err}
		}
		return commentAddedMsg{issueKey: issueKey, comment: comment}
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
				// Re-apply filter and rebuild table rows, keeping cursor on the same issue
				a.tabs[ti].applyFilterKeepCursor(issueKey)
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

// cmdFetchIssueTypes fetches issue types for the default project.
func (a App) cmdFetchIssueTypes() tea.Cmd {
	if a.client == nil {
		return nil
	}
	client := a.client
	project := a.defaultProject
	return func() tea.Msg {
		types, err := client.GetProjectIssueTypes(context.Background(), project)
		if err != nil {
			return issueTypesLoadedMsg{err: fmt.Errorf("get issue types: %w", err)}
		}
		return issueTypesLoadedMsg{types: types}
	}
}

// cmdCreateIssue creates a new issue with the given summary and type.
// It auto-assigns the issue to the current user and transitions it to "To Do".
func (a App) cmdCreateIssue(summary, issueTypeName string) tea.Cmd {
	if a.client == nil {
		return nil
	}
	client := a.client
	project := a.defaultProject
	var accountID string
	if a.user != nil {
		accountID = a.user.AccountID
	}
	return func() tea.Msg {
		ctx := context.Background()
		fields := map[string]interface{}{
			"project":   map[string]interface{}{"key": project},
			"summary":   summary,
			"issuetype": map[string]interface{}{"name": issueTypeName},
		}
		if accountID != "" {
			fields["assignee"] = map[string]interface{}{"accountId": accountID}
		}
		req := jira.CreateIssueRequest{Fields: fields}
		resp, err := client.CreateIssue(ctx, req)
		if err != nil {
			return issueCreatedMsg{err: fmt.Errorf("create issue: %w", err)}
		}

		// Best-effort transition to "To Do".
		if transitions, err := client.GetTransitions(ctx, resp.Key); err == nil {
			for _, t := range transitions {
				if t.To != nil && t.To.Name == "To Do" {
					_ = client.TransitionIssue(ctx, resp.Key, t.ID)
					break
				}
			}
		}

		return issueCreatedMsg{issueKey: resp.Key}
	}
}
