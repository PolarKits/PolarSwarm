package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Issue represents a GitHub issue for the TUI dashboard.
type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Body      string    `json:"body"`
	HTMLURL   string    `json:"html_url"`
	Labels    []Label   `json:"labels"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Label represents a GitHub label.
type Label struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// Comment represents a GitHub issue comment.
type Comment struct {
	ID        int64     `json:"id"`
	Author    string    `json:"user"`
	Login     string    `json:"login"` // fallback for user login
	Body      string    `json:"body"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Client is a read-only GitHub API client for the TUI dashboard.
// It only performs GET requests - no write operations.
type Client struct {
	token  string
	client *http.Client
}

// NewClient creates a new read-only GitHub client.
func NewClient(token string) *Client {
	return &Client{
		token: token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ListOpenIssues fetches all open issues from the repository (read-only GET).
func (c *Client) ListOpenIssues(ctx context.Context, owner, repo string) ([]Issue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=open&per_page=100", owner, repo)

	var allIssues []Issue
	page := 1
	for {
		pageURL := fmt.Sprintf("%s&page=%d", url, page)
		req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
		if err != nil {
			return nil, err
		}
		c.setAuth(req)

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		if err := resp.Body.Close(); err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status: %s", resp.Status)
		}

		var issues []Issue
		if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
			return nil, err
		}
		if len(issues) == 0 {
			break
		}
		allIssues = append(allIssues, issues...)

		// Check for next page
		if resp.Header.Get("Link") == "" || !strings.Contains(resp.Header.Get("Link"), "next") {
			break
		}
		page++
	}
	return allIssues, nil
}

// ListIssueComments fetches comments for an issue (read-only GET).
func (c *Client) ListIssueComments(ctx context.Context, owner, repo string, issueNumber int) ([]Comment, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/comments?per_page=100", owner, repo, issueNumber)

	var allComments []Comment
	page := 1
	for {
		pageURL := fmt.Sprintf("%s&page=%d", url, page)
		req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
		if err != nil {
			return nil, err
		}
		c.setAuth(req)

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		if err := resp.Body.Close(); err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status: %s", resp.Status)
		}

		var comments []Comment
		if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
			return nil, err
		}
		if len(comments) == 0 {
			break
		}
		allComments = append(allComments, comments...)

		// Check for next page
		if resp.Header.Get("Link") == "" || !strings.Contains(resp.Header.Get("Link"), "next") {
			break
		}
		page++
	}
	return allComments, nil
}

// setAuth sets authentication headers for the request.
func (c *Client) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
}