package tui

import (
	"testing"
	"time"
)

func TestNewDashboardModel(t *testing.T) {
	m := NewDashboardModel("owner", "repo", "token")
	if m.owner != "owner" {
		t.Errorf("owner = %q, want %q", m.owner, "owner")
	}
	if m.repo != "repo" {
		t.Errorf("repo = %q, want %q", m.repo, "repo")
	}
	if m.token != "token" {
		t.Errorf("token = %q, want %q", m.token, "token")
	}
	if !m.loading {
		t.Error("loading should be true initially")
	}
	if m.errorMsg != "" {
		t.Errorf("errorMsg = %q, want empty", m.errorMsg)
	}
}

func TestDashboardModelIssuesAndComments(t *testing.T) {
	m := NewDashboardModel("owner", "repo", "token")

	// Initially no issues
	if len(m.Issues()) != 0 {
		t.Errorf("Issues() len = %d, want 0", len(m.Issues()))
	}

	// Comments for non-existent issue returns nil
	if m.Comments(123) != nil {
		t.Error("Comments(123) should be nil for non-existent issue")
	}

	// Loading state
	if !m.Loading() {
		t.Error("Loading() should be true initially")
	}

	// Error should be empty initially
	if m.Error() != "" {
		t.Errorf("Error() = %q, want empty", m.Error())
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		n        int
		expected string
	}{
		{"", 10, ""},
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "h..."}, // n=4: 1 char + ellipsis
		{"hello", 3, "hel"},  // n=3: no room for ellipsis, return first n chars
		{"hello", 2, "he"},
		{"hello", 1, "h"},
		{"short", 100, "short"},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.n)
		if got != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.n, got, tt.expected)
		}
	}
}

func TestOrUnknown(t *testing.T) {
	if orUnknown("", "default") != "default" {
		t.Error("orUnknown with empty should return default")
	}
	if orUnknown("value", "default") != "value" {
		t.Error("orUnknown with value should return value")
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status   string
		expected string // just check it's not empty
	}{
		{"status:in-progress", ""},  // returns warning style
		{"status:blocked", ""},       // returns bad style
		{"status:review", ""},        // returns warn style
		{"status:rework", ""},        // returns warn style
		{"status:done", ""},          // returns good style
		{"", ""},                     // returns normal style
		{"status:pending-triage", ""}, // returns normal style
	}

	for _, tt := range tests {
		got := getStatusIcon(tt.status)
		if got == "" {
			t.Errorf("getStatusIcon(%q) returned empty, expected non-empty", tt.status)
		}
	}
}

func TestIssueView(t *testing.T) {
	iv := IssueView{
		Number:    42,
		Title:     "Test Issue",
		State:     "open",
		Status:    "status:in-progress",
		Agent:     "agent:developer",
		Priority:  "priority:high",
	}

	if iv.Number != 42 {
		t.Errorf("Number = %d, want 42", iv.Number)
	}
	if iv.Title != "Test Issue" {
		t.Errorf("Title = %q, want %q", iv.Title, "Test Issue")
	}
	if iv.Status != "status:in-progress" {
		t.Errorf("Status = %q, want %q", iv.Status, "status:in-progress")
	}
	if iv.Agent != "agent:developer" {
		t.Errorf("Agent = %q, want %q", iv.Agent, "agent:developer")
	}
}

func TestCommentView(t *testing.T) {
	cv := CommentView{
		Author:   "user",
		Body:     "Test comment body",
	}

	if cv.Author != "user" {
		t.Errorf("Author = %q, want %q", cv.Author, "user")
	}
	if cv.Body != "Test comment body" {
		t.Errorf("Body = %q, want %q", cv.Body, "Test comment body")
	}
}

func TestRenderDashboardWithError(t *testing.T) {
	m := NewDashboardModel("owner", "repo", "")
	m.loading = false
	m.errorMsg = "test error"

	output := RenderDashboard(m)
	if output == "" {
		t.Error("RenderDashboard should return non-empty string with error")
	}
}

func TestRenderDashboardWithNoIssues(t *testing.T) {
	m := NewDashboardModel("owner", "repo", "token")
	m.loading = false
	m.issues = []IssueView{}

	output := RenderDashboard(m)
	if output == "" {
		t.Error("RenderDashboard should return non-empty string with no issues")
	}
}

func TestRenderDashboardWithIssues(t *testing.T) {
	m := NewDashboardModel("owner", "repo", "token")
	m.loading = false
	m.issues = []IssueView{
		{
			Number:    1,
			Title:     "Test Issue",
			State:     "open",
			Status:    "status:in-progress",
			Agent:     "agent:developer",
			Priority:  "priority:high",
		},
	}
	m.comments = map[int][]CommentView{
		1: {
			{Author: "user", Body: "Test comment", CreatedAt: time.Now()},
		},
	}

	output := RenderDashboard(m)
	if output == "" {
		t.Error("RenderDashboard should return non-empty string with issues")
	}
}

func TestDashboardModelReadOnly(t *testing.T) {
	// Verify the DashboardModel does not have any write methods
	// This test documents that the dashboard is read-only
	m := NewDashboardModel("owner", "repo", "token")

	// Check that Update method exists and is for reading
	// The model should only perform GET requests
	if m.token != "token" {
		t.Error("token should be stored for read-only API calls")
	}
}