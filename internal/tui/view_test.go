package tui

import (
	"testing"
	"time"
)

func TestRenderComment(t *testing.T) {
	cv := CommentView{
		Author:   "testuser",
		Body:     "This is a test comment",
		CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	output := renderComment(cv)
	if output == "" {
		t.Error("renderComment should return non-empty string")
	}
}

func TestRenderIssue(t *testing.T) {
	issue := IssueView{
		Number:    42,
		Title:     "Test Issue Title",
		State:     "open",
		Status:    "status:in-progress",
		Agent:     "agent:developer",
		Priority:  "priority:high",
		UpdatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	comments := []CommentView{
		{
			Author:   "user1",
			Body:     "First comment",
			CreatedAt: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
		},
	}

	output := renderIssue(issue, comments)
	if output == "" {
		t.Error("renderIssue should return non-empty string")
	}
}

func TestRenderIssueWithNoComments(t *testing.T) {
	issue := IssueView{
		Number:    42,
		Title:     "Test Issue Title",
		State:     "open",
		Status:    "status:new",
		Agent:     "",
		Priority:  "",
		UpdatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	output := renderIssue(issue, nil)
	if output == "" {
		t.Error("renderIssue should return non-empty string even with no comments")
	}
}

func TestGetStatusIconForAllStatuses(t *testing.T) {
	statuses := []string{
		"status:pending-triage",
		"status:new",
		"status:triaged",
		"status:assigned",
		"status:in-progress",
		"status:blocked",
		"status:review",
		"status:rework",
		"status:done",
		"status:abandoned",
	}

	for _, s := range statuses {
		icon := getStatusIcon(s)
		if icon == "" {
			t.Errorf("getStatusIcon(%q) returned empty", s)
		}
	}
}