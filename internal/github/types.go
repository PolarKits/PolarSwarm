package github

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Repository struct {
	Owner string
	Name  string
}

func (r Repository) String() string {
	return r.Owner + "/" + r.Name
}

func (r Repository) Validate() error {
	if strings.TrimSpace(r.Owner) == "" {
		return errors.New("repository owner is required")
	}
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("repository name is required")
	}
	return nil
}

type Issue struct {
	Repository Repository
	Number     int
	Title      string
	Body       string
	State      string
	Author     string
	HTMLURL    string
	Labels     []Label
	Comments   CommentSummary
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Label struct {
	Name        string
	Description string
	Color       string
}

type Comment struct {
	ID        int64
	Author    string
	Body      string
	HTMLURL   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CommentSummary struct {
	Count  int
	Latest *Comment
}

type Client interface {
	GetIssue(ctx context.Context, repo Repository, number int) (Issue, error)
	ListIssueLabels(ctx context.Context, repo Repository, number int) ([]Label, error)
	ListIssueComments(ctx context.Context, repo Repository, number int) ([]Comment, error)
}

type Reader interface {
	ReadIssue(ctx context.Context, repo Repository, number int) (Issue, error)
}

type IssueReader struct {
	Client Client
}

func (r IssueReader) ReadIssue(ctx context.Context, repo Repository, number int) (Issue, error) {
	if r.Client == nil {
		return Issue{}, errors.New("github reader requires a client")
	}
	if err := repo.Validate(); err != nil {
		return Issue{}, err
	}
	if number <= 0 {
		return Issue{}, fmt.Errorf("issue number must be positive: %d", number)
	}

	issue, err := r.Client.GetIssue(ctx, repo, number)
	if err != nil {
		return Issue{}, wrapReadError("read issue", repo, number, err)
	}

	labels, err := r.Client.ListIssueLabels(ctx, repo, number)
	if err != nil {
		return Issue{}, wrapReadError("read issue labels", repo, number, err)
	}

	comments, err := r.Client.ListIssueComments(ctx, repo, number)
	if err != nil {
		return Issue{}, wrapReadError("read issue comments", repo, number, err)
	}

	issue.Repository = repo
	issue.Number = number
	issue.Labels = append([]Label(nil), labels...)
	issue.Comments = summarizeComments(comments)
	return issue, nil
}

func summarizeComments(comments []Comment) CommentSummary {
	summary := CommentSummary{Count: len(comments)}
	for i := range comments {
		comment := comments[i]
		if summary.Latest == nil || comment.CreatedAt.After(summary.Latest.CreatedAt) {
			summary.Latest = &comment
		}
	}
	return summary
}

func wrapReadError(operation string, repo Repository, number int, err error) error {
	if IsNotFound(err) {
		return err
	}
	return fmt.Errorf("%s %s#%d: %w", operation, repo.String(), number, err)
}
