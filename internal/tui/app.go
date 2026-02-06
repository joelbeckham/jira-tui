package tui

import (
	"context"
	"fmt"
	"strings"

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

// --- View stack ---

// view is a stacked view that renders on top of the tab bar.
type view interface {
	// title returns a label for the view (e.g., issue key).
	title() string
}

// issueDetailView is a stub detail view for a single issue.
type issueDetailView struct {
	issue jira.Issue
}

func (v issueDetailView) title() string {
	return v.issue.Key
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

	case connStatusMsg:
		a.checking = false
		if msg.err != nil {
			a.connErr = msg.err
		} else {
			a.user = msg.user
			a.connected = true
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

	case tea.KeyMsg:
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

	// If a view is on the stack, handle stack-specific keys
	if len(a.viewStack) > 0 {
		switch key {
		case "q":
			return a, tea.Quit
		case "esc":
			a.viewStack = a.viewStack[:len(a.viewStack)-1]
			return a, nil
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
		// Push issue detail onto stack
		if a.activeTab < len(a.tabs) {
			if issue := a.tabs[a.activeTab].selectedIssue(); issue != nil {
				a.viewStack = append(a.viewStack, issueDetailView{issue: *issue})
				return a, nil
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
	if len(a.viewStack) > 0 {
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
	case issueDetailView:
		// Stub: show issue key and summary
		var b strings.Builder
		b.WriteString(titleStyle.Render(v.issue.Key))
		b.WriteString("\n")
		b.WriteString(v.issue.Fields.Summary)
		b.WriteString("\n\n")
		if v.issue.Fields.Status != nil {
			b.WriteString(fmt.Sprintf("Status: %s\n", v.issue.Fields.Status.Name))
		}
		if v.issue.Fields.Assignee != nil {
			b.WriteString(fmt.Sprintf("Assignee: %s\n", v.issue.Fields.Assignee.DisplayName))
		}
		if v.issue.Fields.Priority != nil {
			b.WriteString(fmt.Sprintf("Priority: %s\n", priorityLabel(v.issue.Fields.Priority.Name)))
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("esc: back"))
		return b.String()
	}
	return ""
}

// renderStatusBar draws the bottom help/status line.
func (a App) renderStatusBar() string {
	var parts []string

	if a.user != nil {
		parts = append(parts, successStyle.Render(a.user.DisplayName))
	}

	if len(a.viewStack) > 0 {
		parts = append(parts, helpStyle.Render("esc: back  q: quit"))
	} else {
		parts = append(parts, helpStyle.Render("j/k: navigate  enter: open  /: filter  r: refresh  1-9: tabs  q: quit"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		strings.Join(parts, helpStyle.Render("  │  ")),
	)
}
