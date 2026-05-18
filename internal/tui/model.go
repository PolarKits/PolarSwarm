// Package tui provides a read-only TUI dashboard for PolarSwarm status monitoring.
// It displays active issues, their states, and recent comment summaries using Bubble Tea.
// All operations are read-only - no write operations to GitHub or other external systems.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// DashboardModel is the Bubble Tea model for the read-only dashboard.
type DashboardModel struct {
	issues    []IssueView
	comments  map[int][]CommentView // keyed by issue number
	errorMsg  string
	loading   bool
	owner     string
	repo      string
	token     string
}

// IssueView represents a compact view of an issue for the dashboard.
type IssueView struct {
	Number    int
	Title     string
	State     string
	Status    string // status:* label value
	Agent     string // agent:* label value
	Priority  string // priority:* label value
	UpdatedAt time.Time
}

// CommentView represents a compact view of a comment for the dashboard.
type CommentView struct {
	Author    string
	Body      string
	CreatedAt time.Time
}

// NewDashboardModel creates a new dashboard model with the given configuration.
func NewDashboardModel(owner, repo, token string) DashboardModel {
	return DashboardModel{
		issues:    []IssueView{},
		comments:  make(map[int][]CommentView),
		owner:     owner,
		repo:      repo,
		token:     token,
		loading:   true,
	}
}

// Init initializes the dashboard model. It performs no write operations.
func (m DashboardModel) Init(ctx context.Context) error {
	m.loading = true
	return nil
}

// Update loads issues and comments from GitHub (read-only).
// Returns updated model and error if loading fails.
func (m *DashboardModel) Update(ctx context.Context) error {
	if m.token == "" {
		m.errorMsg = "GitHub token not configured (GH_TOKEN or GITHUB_TOKEN)"
		m.loading = false
		return nil
	}
	if m.owner == "" || m.repo == "" {
		m.errorMsg = "Repository not configured (owner/repo required)"
		m.loading = false
		return nil
	}

	// Create GitHub client for read-only operations
	client := NewClient(m.token)

	// Fetch open issues (read-only)
	issues, err := client.ListOpenIssues(ctx, m.owner, m.repo)
	if err != nil {
		m.errorMsg = fmt.Sprintf("Failed to load issues: %v", err)
		m.loading = false
		return nil
	}

	// Build issue views
	m.issues = make([]IssueView, 0, len(issues))
	for _, issue := range issues {
		iv := IssueView{
			Number:    issue.Number,
			Title:    issue.Title,
			State:    issue.State,
			UpdatedAt: issue.UpdatedAt,
		}
		// Extract labels
		for _, label := range issue.Labels {
			if strings.HasPrefix(label.Name, "status:") {
				iv.Status = label.Name
			} else if strings.HasPrefix(label.Name, "agent:") {
				iv.Agent = label.Name
			} else if strings.HasPrefix(label.Name, "priority:") {
				iv.Priority = label.Name
			}
		}
		m.issues = append(m.issues, iv)
	}

	// Fetch recent comments for each issue (read-only)
	m.comments = make(map[int][]CommentView)
	for _, issue := range issues {
		comments, err := client.ListIssueComments(ctx, m.owner, m.repo, issue.Number)
		if err != nil {
			continue // skip issues with comment fetch errors
		}
		// Limit to 3 most recent comments
		recent := comments
		if len(recent) > 3 {
			recent = recent[len(recent)-3:]
		}
		cvs := make([]CommentView, 0, len(recent))
		for _, c := range recent {
			author := c.Author
			if author == "" {
				author = c.Login
			}
			cvs = append(cvs, CommentView{
				Author:   author,
				Body:     truncate(c.Body, 100),
				CreatedAt: c.CreatedAt,
			})
		}
		m.comments[issue.Number] = cvs
	}

	m.loading = false
	m.errorMsg = ""
	return nil
}

// Error returns the current error message if any.
func (m DashboardModel) Error() string {
	return m.errorMsg
}

// Loading returns whether the dashboard is still loading.
func (m DashboardModel) Loading() bool {
	return m.loading
}

// Issues returns the list of issues.
func (m DashboardModel) Issues() []IssueView {
	return m.issues
}

// Comments returns comments for a given issue number.
func (m DashboardModel) Comments(issueNumber int) []CommentView {
	return m.comments[issueNumber]
}

// truncate truncates a string to at most n characters, adding "..." if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}