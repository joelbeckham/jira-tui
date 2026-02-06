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
	issue    jira.Issue
	viewport viewport.Model
	ready    bool
	width    int
	height   int
}

func newIssueDetailView(issue jira.Issue, width, height int) issueDetailView {
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

	// Header: KEY ▸ Parent (if any)
	header := detailKeyStyle.Render(issue.Key)
	if fields.Parent != nil {
		parentLabel := fields.Parent.Key
		if fields.Parent.Fields != nil && fields.Parent.Fields.Summary != "" {
			parentLabel += " " + fields.Parent.Fields.Summary
		}
		header += detailParentStyle.Render("  ▸ " + parentLabel)
	}
	b.WriteString(header)
	b.WriteString("\n")

	// Type · Status · Priority
	var meta []string
	if fields.IssueType != nil {
		meta = append(meta, detailTypeStyle.Render(fields.IssueType.Name))
	}
	if fields.Status != nil {
		meta = append(meta, statusColor(fields.Status).Render(fields.Status.Name))
	}
	if fields.Priority != nil {
		meta = append(meta, priorityLabel(fields.Priority.Name))
	}
	if len(meta) > 0 {
		b.WriteString(strings.Join(meta, detailTypeStyle.Render(" · ")))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Summary
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(fields.Summary))
	b.WriteString("\n\n")

	// Description
	desc := extractADFText(fields.Description)
	if desc != "" {
		b.WriteString(desc)
		b.WriteString("\n")
	}

	// Fields section
	b.WriteString("\n")
	b.WriteString(renderSection("Fields", maxWidth))

	b.WriteString(renderField("Assignee", userName(fields.Assignee, "Unassigned")))
	b.WriteString(renderField("Reporter", userName(fields.Reporter, "")))
	b.WriteString(renderField("Project", namedValue(fields.Project)))
	b.WriteString(renderField("Labels", labelsValue(fields.Labels)))
	b.WriteString(renderField("Created", formatDetailDate(fields.Created)))
	b.WriteString(renderField("Updated", formatDetailDate(fields.Updated)))

	// Subtasks
	if len(fields.Subtasks) > 0 {
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

	// Linked Issues
	if len(fields.IssueLinks) > 0 {
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

	// Parent (standalone section if not shown in header)
	if fields.Parent != nil {
		b.WriteString("\n")
		b.WriteString(renderSection("Parent", maxWidth))
		parentLabel := detailKeyStyle.Render(fields.Parent.Key)
		if fields.Parent.Fields != nil && fields.Parent.Fields.Summary != "" {
			parentLabel += "  " + fields.Parent.Fields.Summary
		}
		b.WriteString("  " + parentLabel + "\n")
	}

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

// --- Helpers ---

func renderSection(label string, maxWidth int) string {
	line := strings.Repeat("─", maxWidth)
	return detailSectionStyle.Render(fmt.Sprintf("─── %s %s", label, line[:max(0, maxWidth-len(label)-5)])) + "\n"
}

func renderField(label, value string) string {
	if value == "" {
		return ""
	}
	return detailLabelStyle.Render(label) + detailValueStyle.Render(value) + "\n"
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


