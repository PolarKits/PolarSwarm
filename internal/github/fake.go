package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type FakeClient struct {
	issues   map[issueKey]Issue
	labels   map[issueKey][]Label
	comments map[issueKey][]Comment
	errs     map[string]error
}

type issueKey struct {
	repo   string
	number int
}

func NewFakeClient() *FakeClient {
	return &FakeClient{
		issues:   make(map[issueKey]Issue),
		labels:   make(map[issueKey][]Label),
		comments: make(map[issueKey][]Comment),
		errs:     make(map[string]error),
	}
}

func (c *FakeClient) AddIssue(issue Issue, comments []Comment) {
	if c.issues == nil {
		*c = *NewFakeClient()
	}
	key := makeIssueKey(issue.Repository, issue.Number)
	c.issues[key] = issue
	c.labels[key] = append([]Label(nil), issue.Labels...)
	c.comments[key] = append([]Comment(nil), comments...)
}

func (c *FakeClient) SetError(operation string, err error) {
	if c.errs == nil {
		c.errs = make(map[string]error)
	}
	c.errs[operation] = err
}

func (c *FakeClient) GetIssue(ctx context.Context, repo Repository, number int) (Issue, error) {
	if err := ctx.Err(); err != nil {
		return Issue{}, RequestError{Kind: ErrorKindNetwork, Operation: "get issue", Err: err}
	}
	if err := c.operationError("get_issue"); err != nil {
		return Issue{}, err
	}
	key := makeIssueKey(repo, number)
	issue, ok := c.issues[key]
	if !ok {
		return Issue{}, NotFoundError{Repo: repo, Number: number}
	}
	return issue, nil
}

func (c *FakeClient) ListIssueLabels(ctx context.Context, repo Repository, number int) ([]Label, error) {
	if err := ctx.Err(); err != nil {
		return nil, RequestError{Kind: ErrorKindNetwork, Operation: "list labels", Err: err}
	}
	if err := c.operationError("list_labels"); err != nil {
		return nil, err
	}
	key := makeIssueKey(repo, number)
	if _, ok := c.issues[key]; !ok {
		return nil, NotFoundError{Repo: repo, Number: number}
	}
	return append([]Label(nil), c.labels[key]...), nil
}

func (c *FakeClient) ListIssueComments(ctx context.Context, repo Repository, number int) ([]Comment, error) {
	if err := ctx.Err(); err != nil {
		return nil, RequestError{Kind: ErrorKindNetwork, Operation: "list comments", Err: err}
	}
	if err := c.operationError("list_comments"); err != nil {
		return nil, err
	}
	key := makeIssueKey(repo, number)
	if _, ok := c.issues[key]; !ok {
		return nil, NotFoundError{Repo: repo, Number: number}
	}
	comments := append([]Comment(nil), c.comments[key]...)
	sort.SliceStable(comments, func(i, j int) bool {
		return comments[i].CreatedAt.Before(comments[j].CreatedAt)
	})
	return comments, nil
}

func LoadFakeClient(path string) (*FakeClient, error) {
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read github issue fixture %q: %w", path, err)
	}

	var fixture fakeFixture
	if err := json.Unmarshal(content, &fixture); err != nil {
		return nil, fmt.Errorf("parse github issue fixture %q: %w", path, err)
	}

	client := NewFakeClient()
	for _, issue := range fixture.Issues {
		client.AddIssue(issue.Issue, issue.Comments)
	}
	return client, nil
}

func (c *FakeClient) operationError(operation string) error {
	if c == nil {
		return RequestError{Kind: ErrorKindUnknown, Operation: operation, Err: fmt.Errorf("fake client is nil")}
	}
	return c.errs[operation]
}

func makeIssueKey(repo Repository, number int) issueKey {
	return issueKey{repo: repo.String(), number: number}
}

type fakeFixture struct {
	Issues []fakeFixtureIssue `json:"issues"`
}

type fakeFixtureIssue struct {
	Issue
	Comments []Comment `json:"comments"`
}
