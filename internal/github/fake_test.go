package github

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIssueReaderReadsIssueLabelsAndCommentSummary(t *testing.T) {
	repo := Repository{Owner: "PolarKits", Name: "PolarSwarm"}
	client := NewFakeClient()
	client.AddIssue(Issue{
		Repository: repo,
		Number:     3,
		Title:      "Implement reader",
		State:      "open",
		Labels: []Label{
			{Name: "status:new", Color: "0e8a16"},
			{Name: "area:github", Color: "1d76db"},
		},
	}, []Comment{
		{ID: 1, Author: "alice", Body: "first", CreatedAt: time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)},
		{ID: 2, Author: "bob", Body: "latest", CreatedAt: time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)},
	})

	issue, err := (IssueReader{Client: client}).ReadIssue(context.Background(), repo, 3)
	if err != nil {
		t.Fatalf("ReadIssue returned error: %v", err)
	}

	if issue.Title != "Implement reader" || issue.Number != 3 {
		t.Fatalf("unexpected issue: %#v", issue)
	}
	if len(issue.Labels) != 2 || issue.Labels[0].Name != "status:new" {
		t.Fatalf("unexpected labels: %#v", issue.Labels)
	}
	if issue.Comments.Count != 2 {
		t.Fatalf("unexpected comment count: %d", issue.Comments.Count)
	}
	if issue.Comments.Latest == nil || issue.Comments.Latest.Author != "bob" {
		t.Fatalf("unexpected latest comment: %#v", issue.Comments.Latest)
	}
}

func TestIssueReaderReturnsNotFound(t *testing.T) {
	repo := Repository{Owner: "PolarKits", Name: "PolarSwarm"}

	_, err := (IssueReader{Client: NewFakeClient()}).ReadIssue(context.Background(), repo, 404)
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !IsNotFound(err) {
		t.Fatalf("expected not found error, got %T: %v", err, err)
	}
}

func TestIssueReaderWrapsDiagnosticRequestErrors(t *testing.T) {
	repo := Repository{Owner: "PolarKits", Name: "PolarSwarm"}
	client := NewFakeClient()
	client.AddIssue(Issue{Repository: repo, Number: 7, Title: "x"}, nil)
	client.SetError("list_comments", RequestError{
		Kind:      ErrorKindPermission,
		Operation: "list comments",
		Err:       errors.New("403 forbidden"),
	})

	_, err := (IssueReader{Client: client}).ReadIssue(context.Background(), repo, 7)
	if err == nil {
		t.Fatal("expected permission error")
	}
	if !IsRequestKind(err, ErrorKindPermission) {
		t.Fatalf("expected permission request error, got %T: %v", err, err)
	}
}

func TestIssueReaderWrapsNetworkErrors(t *testing.T) {
	repo := Repository{Owner: "PolarKits", Name: "PolarSwarm"}
	client := NewFakeClient()
	client.SetError("get_issue", RequestError{
		Kind:      ErrorKindNetwork,
		Operation: "get issue",
		Err:       errors.New("dial tcp timeout"),
	})

	_, err := (IssueReader{Client: client}).ReadIssue(context.Background(), repo, 8)
	if err == nil {
		t.Fatal("expected network error")
	}
	if !IsRequestKind(err, ErrorKindNetwork) {
		t.Fatalf("expected network request error, got %T: %v", err, err)
	}
}

func TestLoadFakeClient(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	content := `{
  "issues": [
    {
      "repository": {"owner": "PolarKits", "name": "PolarSwarm"},
      "number": 9,
      "title": "Fixture issue",
      "state": "open",
      "labels": [{"name": "status:new"}],
      "comments": [{"id": 99, "author": "alice", "body": "ok"}]
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	client, err := LoadFakeClient(path)
	if err != nil {
		t.Fatalf("LoadFakeClient returned error: %v", err)
	}

	issue, err := (IssueReader{Client: client}).ReadIssue(context.Background(), Repository{Owner: "PolarKits", Name: "PolarSwarm"}, 9)
	if err != nil {
		t.Fatalf("ReadIssue returned error: %v", err)
	}
	if issue.Title != "Fixture issue" || issue.Comments.Count != 1 {
		t.Fatalf("unexpected fixture issue: %#v", issue)
	}
}
