package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Styles defines the visual styling for the dashboard.
var Styles = struct {
	Title      lipgloss.Style
	Header     lipgloss.Style
	Issue      lipgloss.Style
	Comment    lipgloss.Style
	StatusGood lipgloss.Style
	StatusWarn lipgloss.Style
	StatusBad  lipgloss.Style
	Normal     lipgloss.Style
	Muted      lipgloss.Style
	Error      lipgloss.Style
}{
	Title: lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true),

	Header: lipgloss.NewStyle().
		Foreground(lipgloss.Color("75")).
		Bold(true),

	Issue: lipgloss.NewStyle().
		Foreground(lipgloss.Color("White")),

	Comment: lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")),

	StatusGood: lipgloss.NewStyle().
		Foreground(lipgloss.Color("76")),

	StatusWarn: lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")),

	StatusBad: lipgloss.NewStyle().
		Foreground(lipgloss.Color("204")),

	Normal: lipgloss.NewStyle().
		Foreground(lipgloss.Color("White")),

	Muted: lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")),

	Error: lipgloss.NewStyle().
		Foreground(lipgloss.Color("204")),
}

// statusIcons holds the visual indicators for each status category.
const (
	iconGood = "●"
	iconWarn = "◐"
	iconBad  = "○"
	iconNorm = "○"
)

// RenderDashboard renders the full dashboard view.
func RenderDashboard(m DashboardModel) string {
	var b strings.Builder

	// Title
	b.WriteString(Styles.Title.Render("PolarSwarm Dashboard"))
	b.WriteString(" (read-only)\n\n")

	if m.errorMsg != "" {
		b.WriteString(Styles.Error.Render(fmt.Sprintf("Error: %s", m.errorMsg)))
		b.WriteString("\n\n")
	}

	if m.loading {
		b.WriteString(Styles.Muted.Render("Loading..."))
		b.WriteString("\n")
		return b.String()
	}

	// Render issues
	if len(m.issues) == 0 {
		b.WriteString(Styles.Muted.Render("No open issues"))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString(Styles.Header.Render(fmt.Sprintf("Active Issues (%d)", len(m.issues))))
	b.WriteString("\n\n")

	for _, issue := range m.issues {
		b.WriteString(renderIssue(issue, m.Comments(issue.Number)))
	}

	return b.String()
}

// renderIssue renders a single issue with its comments.
func renderIssue(issue IssueView, comments []CommentView) string {
	var b strings.Builder

	// Issue header with status indicator
	statusIcon := getStatusIcon(issue.Status)
	b.WriteString(fmt.Sprintf("%s #%d %s\n", statusIcon, issue.Number, issue.Title))
	b.WriteString(Styles.Muted.Render(fmt.Sprintf("   State: %s | Status: %s | Agent: %s | Priority: %s | Updated: %s\n",
		issue.State,
		orUnknown(issue.Status, "none"),
		orUnknown(issue.Agent, "unassigned"),
		orUnknown(issue.Priority, "none"),
		issue.UpdatedAt.Format(time.RFC822))))

	// Render comments
	if len(comments) > 0 {
		b.WriteString(Styles.Muted.Render("   Recent comments:"))
		b.WriteString("\n")
		for _, comment := range comments {
			b.WriteString(renderComment(comment))
		}
	}

	b.WriteString("\n")
	return b.String()
}

// renderComment renders a single comment.
func renderComment(comment CommentView) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("     [%s] %s: %s\n",
		comment.CreatedAt.Format("15:04 Jan 02"),
		comment.Author,
		comment.Body))
	return b.String()
}

// getStatusIcon returns a styled status indicator based on status label.
func getStatusIcon(status string) string {
	switch {
	case strings.HasPrefix(status, "status:in-progress"):
		return Styles.StatusWarn.Render(iconWarn)
	case strings.HasPrefix(status, "status:blocked"):
		return Styles.StatusBad.Render(iconBad)
	case strings.HasPrefix(status, "status:review"),
		strings.HasPrefix(status, "status:rework"):
		return Styles.StatusWarn.Render(iconWarn)
	case strings.HasPrefix(status, "status:done"):
		return Styles.StatusGood.Render(iconGood)
	default:
		return Styles.Normal.Render(iconNorm)
	}
}

// orUnknown returns value or a default "unknown" string.
func orUnknown(value, defaultVal string) string {
	if value == "" {
		return defaultVal
	}
	return value
}