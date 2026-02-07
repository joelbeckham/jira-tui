package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jbeckham/jira-tui/internal/jira"
)

// Ensure issueDetailView implements the view interface.
var _ view = (*issueDetailView)(nil)

// --- Styles for detail view ---

var (
	detailKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12"))

	detailTypeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	detailStatusStyle = lipgloss.NewStyle().
				Bold(true)

	detailSectionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				MarginTop(1)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Width(14)

	detailValueStyle = lipgloss.NewStyle()

	detailHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	detailSubtaskDone = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10")) // green ✓

	detailSubtaskOpen = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")) // dim ·

	detailLinkTypeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")). // yellow
				Width(20)

	detailParentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))
)

// statusColor returns a lipgloss style colored by status category.
func statusColor(status *jira.Status) lipgloss.Style {
	s := detailStatusStyle
	if status == nil || status.StatusCategory == nil {
		return s
	}
	switch status.StatusCategory.Key {
	case "new":
		return s.Foreground(lipgloss.Color("12")) // blue
	case "indeterminate":
		return s.Foreground(lipgloss.Color("11")) // yellow
	case "done":
		return s.Foreground(lipgloss.Color("10")) // green
	default:
		return s.Foreground(lipgloss.Color("252")) // light gray
	}
}

// issueDetailView is the full detail view for a single issue.
type issueDetailView struct {
	issue           jira.Issue
	viewport        viewport.Model
	ready           bool
	loading         bool // true while the full issue fetch is in-flight
	dirty           bool // true if the issue was edited while this view was open
	comments        []jira.Comment
	commentsLoading bool
	width           int
	height          int
}

func newIssueDetailView(issue jira.Issue, width, height int) issueDetailView {
	v := issueDetailView{
		issue:           issue,
		width:           width,
		height:          height,
		loading:         true,
		commentsLoading: true,
	}
	v.buildViewport()
	return v
}

func newIssueDetailViewReady(issue jira.Issue, width, height int) issueDetailView {
	v := issueDetailView{
		issue:  issue,
		width:  width,
		height: height,
	}
	v.buildViewport()
	return v
}

func (v issueDetailView) title() string {
	return v.issue.Key
}

// buildViewport creates the viewport with rendered content.
func (v *issueDetailView) buildViewport() {
	content := v.renderContent()

	// Height available for the viewport: total height minus tab bar (2) and status bar (1)
	vpHeight := v.height - 3
	if vpHeight < 3 {
		vpHeight = 3
	}

	vp := viewport.New(v.width, vpHeight)
	vp.SetContent(content)
	// Use j/k for scrolling
	vp.KeyMap.Up.SetKeys("up", "k")
	vp.KeyMap.Down.SetKeys("down", "j")
	v.viewport = vp
	v.ready = true
}

// renderContent builds the full detail text.
func (v *issueDetailView) renderContent() string {
	issue := v.issue
	fields := issue.Fields
	maxWidth := v.width - 2 // small margin
	if maxWidth < 20 {
		maxWidth = 20
	}

	var b strings.Builder

	// Header: KEY(k) ▸ Parent (if any)
	header := detailKeyStyle.Render(issue.Key) + detailHintStyle.Render("(y)")
	if fields.Parent != nil {
		parentLabel := fields.Parent.Key
		if fields.Parent.Fields != nil && fields.Parent.Fields.Summary != "" {
			parentLabel += " " + fields.Parent.Fields.Summary
		}
		header += detailParentStyle.Render("  ▸ " + parentLabel)
	}
	b.WriteString(header)
	b.WriteString("\n")

	// Type · Status(s) · Priority(p)
	var meta []string
	if fields.IssueType != nil {
		meta = append(meta, detailTypeStyle.Render(fields.IssueType.Name))
	}
	if fields.Status != nil {
		meta = append(meta, statusColor(fields.Status).Render(fields.Status.Name)+detailHintStyle.Render("(s)"))
	}
	if fields.Priority != nil {
		meta = append(meta, priorityLabel(fields.Priority.Name)+detailHintStyle.Render("(p)"))
	}
	if len(meta) > 0 {
		b.WriteString(strings.Join(meta, detailTypeStyle.Render(" · ")))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Summary (t)
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(fields.Summary))
	b.WriteString("  " + detailHintStyle.Render("(t)"))
	b.WriteString("\n\n")

	// Description (e)
	if v.loading {
		b.WriteString(detailSectionStyle.Render("Description") + " " + detailHintStyle.Render("(e)") + "\n")
		b.WriteString(detailTypeStyle.Render("Loading…") + "\n")
	} else {
		desc := extractADFText(fields.Description)
		if desc != "" {
			b.WriteString(detailSectionStyle.Render("Description") + " " + detailHintStyle.Render("(e)") + "\n")
			b.WriteString(desc)
			b.WriteString("\n")
		} else {
			b.WriteString(detailSectionStyle.Render("Description") + " " + detailHintStyle.Render("(e)") + "\n")
			b.WriteString(detailTypeStyle.Render("No description") + "\n")
		}
	}

	// Fields section
	b.WriteString("\n")
	b.WriteString(renderSection("Fields", maxWidth))

	b.WriteString(renderFieldHint("Assignee", userName(fields.Assignee, "Unassigned"), "a,i"))
	b.WriteString(renderField("Reporter", userName(fields.Reporter, "")))
	b.WriteString(renderField("Project", namedValue(fields.Project)))
	if v.loading {
		b.WriteString(renderField("Labels", "Loading…"))
	} else {
		b.WriteString(renderField("Labels", labelsValue(fields.Labels)))
	}
	b.WriteString(renderField("Created", formatDetailDate(fields.Created)))
	b.WriteString(renderField("Updated", formatDetailDate(fields.Updated)))

	// Subtasks (only available from full fetch)
	if v.loading {
		// skip — subtask data not yet available
	} else if len(fields.Subtasks) > 0 {
		b.WriteString("\n")
		b.WriteString(renderSection(fmt.Sprintf("Subtasks (%d)", len(fields.Subtasks)), maxWidth))
		for _, sub := range fields.Subtasks {
			icon := detailSubtaskOpen.Render("·")
			if sub.Fields.Status != nil && sub.Fields.Status.StatusCategory != nil &&
				sub.Fields.Status.StatusCategory.Key == "done" {
				icon = detailSubtaskDone.Render("✓")
			}
			b.WriteString(fmt.Sprintf("  %s %s  %s\n",
				icon,
				detailKeyStyle.Render(sub.Key),
				sub.Fields.Summary,
			))
		}
	}

	// Linked Issues (only available from full fetch)
	if v.loading {
		// skip — link data not yet available
	} else if len(fields.IssueLinks) > 0 {
		b.WriteString("\n")
		b.WriteString(renderSection(fmt.Sprintf("Linked Issues (%d)", len(fields.IssueLinks)), maxWidth))
		for _, link := range fields.IssueLinks {
			if link.OutwardIssue != nil {
				b.WriteString(fmt.Sprintf("  %s %s  %s\n",
					detailLinkTypeStyle.Render(link.Type.Outward),
					detailKeyStyle.Render(link.OutwardIssue.Key),
					link.OutwardIssue.Fields.Summary,
				))
			}
			if link.InwardIssue != nil {
				b.WriteString(fmt.Sprintf("  %s %s  %s\n",
					detailLinkTypeStyle.Render(link.Type.Inward),
					detailKeyStyle.Render(link.InwardIssue.Key),
					link.InwardIssue.Fields.Summary,
				))
			}
		}
	}

	// Parent (standalone section if not shown in header, only from full fetch)
	if !v.loading && fields.Parent != nil {
		b.WriteString("\n")
		b.WriteString(renderSection("Parent", maxWidth))
		parentLabel := detailKeyStyle.Render(fields.Parent.Key)
		if fields.Parent.Fields != nil && fields.Parent.Fields.Summary != "" {
			parentLabel += "  " + fields.Parent.Fields.Summary
		}
		b.WriteString("  " + parentLabel + "\n")
	}

	// Comments
	if v.commentsLoading {
		b.WriteString("\n")
		b.WriteString(renderSection("Comments", maxWidth))
		b.WriteString(detailTypeStyle.Render("  Loading…") + "\n")
	} else if len(v.comments) > 0 {
		b.WriteString("\n")
		b.WriteString(renderSection(fmt.Sprintf("Comments (%d)", len(v.comments)), maxWidth))
		for i, c := range v.comments {
			author := "Unknown"
			if c.Author != nil {
				author = c.Author.DisplayName
			}
			date := formatDetailDate(c.Created)
			b.WriteString(fmt.Sprintf("  %s  %s\n",
				lipgloss.NewStyle().Bold(true).Render(author),
				detailTypeStyle.Render(date),
			))
			body := extractADFText(c.Body)
			if body != "" {
				// Indent comment body
				for _, line := range strings.Split(body, "\n") {
					b.WriteString("  " + line + "\n")
				}
			}
			if i < len(v.comments)-1 {
				b.WriteString("\n")
			}
		}
	}

	// Actions hint line
	b.WriteString("\n")
	b.WriteString(detailHintStyle.Render("c: comment  d: mark done  u: copy url  del: delete") + "\n")

	return b.String()
}

// Update processes key events for the detail view's viewport.
func (v *issueDetailView) Update(msg tea.Msg) tea.Cmd {
	if !v.ready {
		return nil
	}
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return cmd
}

// View renders the detail view viewport.
func (v *issueDetailView) View() string {
	if !v.ready {
		return loadingStyle.Render("Loading...")
	}
	return v.viewport.View()
}

// setSize updates the viewport dimensions.
func (v *issueDetailView) setSize(width, height int) {
	v.width = width
	v.height = height
	if v.ready {
		v.buildViewport()
	}
}

// updateIssue replaces the displayed issue and rebuilds the viewport content.
func (v *issueDetailView) updateIssue(issue jira.Issue) {
	v.issue = issue
	v.dirty = true
	if v.ready {
		v.buildViewport()
	}
}

// --- Helpers ---

func renderSection(label string, maxWidth int) string {
	// "─── Label ─────────"
	// prefix "─── " = 4 display cols, " " after label = 1
	remaining := maxWidth - 4 - len(label) - 1
	if remaining < 0 {
		remaining = 0
	}
	tail := strings.Repeat("─", remaining)
	return detailSectionStyle.Render(fmt.Sprintf("─── %s %s", label, tail)) + "\n"
}

func renderField(label, value string) string {
	if value == "" {
		return ""
	}
	return detailLabelStyle.Render(label) + detailValueStyle.Render(value) + "\n"
}

func renderFieldHint(label, value, hint string) string {
	if value == "" {
		return ""
	}
	return detailLabelStyle.Render(label+detailHintStyle.Render("("+hint+")")) + detailValueStyle.Render(value) + "\n"
}

func userName(user *jira.User, fallback string) string {
	if user == nil {
		return fallback
	}
	return user.DisplayName
}

func namedValue(n *jira.Named) string {
	if n == nil {
		return ""
	}
	return n.Name
}

func labelsValue(labels []string) string {
	if len(labels) == 0 {
		return "None"
	}
	return strings.Join(labels, ", ")
}

func formatDetailDate(s string) string {
	if s == "" {
		return ""
	}
	// Jira dates are ISO 8601: "2025-07-01T10:23:45.000+0000"
	// Show just "2025-07-01 10:23"
	if len(s) >= 16 {
		return s[:10] + " " + s[11:16]
	}
	return s
}


