// Package doctor provides health check commands for PolarSwarm.
// It performs read-only checks on configuration, GitHub connectivity,
// labels, workflows, store, and worktrees without executing any write operations.
package doctor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GitHub checks the health of GitHub token, repository access, and API reachability.
// It performs only read operations (GET requests) and never executes any writes.
type GitHub struct {
	// Token is the GitHub personal access token. Can also be set via GH_TOKEN or GITHUB_TOKEN env var.
	Token string
	// Owner is the repository owner (organization or user).
	Owner string
	// Repo is the repository name.
	Repo string
	// HTTPClient is used for API requests. If nil, a default client is used.
	HTTPClient *http.Client
	// Output writer. If nil, os.Stdout is used.
	Output io.Writer
}

// Result represents a single check result with a status icon and description.
type Result struct {
	Icon   string // ✓ for pass, ✗ for fail, ⚠ for warning
	Status string // "pass", "fail", "warn"
	Target string // e.g., "API reachable", "Token valid"
	Detail string // e.g., "200 OK (124ms)" or "403 forbidden"
}

// Run executes all github health checks and writes results to Output.
// Returns an error only if the check cannot run at all (e.g., no token provided),
// not for individual check failures which are reported in the results.
func (g *GitHub) Run(ctx context.Context) ([]Result, error) {
	out := g.Output
	if out == nil {
		out = os.Stdout
	}

	// Check for token in environment if not provided
	token := g.Token
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	if token == "" {
		fmt.Fprintf(out, "[ github ]  %s  Token missing   Set GH_TOKEN or GITHUB_TOKEN environment variable\n", iconFail)
		return nil, fmt.Errorf("github token missing: set GH_TOKEN or GITHUB_TOKEN environment variable")
	}

	results := make([]Result, 0, 4)

	// 1. API reachability check
	result := g.checkAPIReachability(ctx, token)
	results = append(results, result)
	fmt.Fprintf(out, "[ github ]  %s  %-18s%s\n", result.Icon, result.Target, result.Detail)

	// 2. Token validity check
	result = g.checkTokenValidity(ctx, token)
	results = append(results, result)
	fmt.Fprintf(out, "[ github ]  %s  %-18s%s\n", result.Icon, result.Target, result.Detail)

	// 3. Repository access check
	result = g.checkRepoAccess(ctx, token)
	results = append(results, result)
	fmt.Fprintf(out, "[ github ]  %s  %-18s%s\n", result.Icon, result.Target, result.Detail)

	return results, nil
}

// checkAPIReachability verifies GitHub API is reachable.
func (g *GitHub) checkAPIReachability(ctx context.Context, token string) Result {
	client := g.httpClient()
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com", nil)
	if err != nil {
		return Result{Icon: iconFail, Status: "fail", Target: "API reachable", Detail: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	start := time.Now()
	resp, err := client.Do(req)
	ms := time.Since(start).Milliseconds()

	if err != nil {
		return Result{Icon: iconFail, Status: "fail", Target: "API reachable", Detail: fmt.Sprintf("error: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return Result{Icon: iconPass, Status: "pass", Target: "API reachable", Detail: fmt.Sprintf("200 OK (%dms)", ms)}
	}
	return Result{Icon: iconFail, Status: "fail", Target: "API reachable", Detail: fmt.Sprintf("%s (%dms)", resp.Status, ms)}
}

// checkTokenValidity verifies the token is valid by calling /user endpoint.
func (g *GitHub) checkTokenValidity(ctx context.Context, token string) Result {
	client := g.httpClient()
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return Result{Icon: iconFail, Status: "fail", Target: "Token valid", Detail: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	start := time.Now()
	resp, err := client.Do(req)
	ms := time.Since(start).Milliseconds()

	if err != nil {
		return Result{Icon: iconFail, Status: "fail", Target: "Token valid", Detail: fmt.Sprintf("error: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Extract scopes from response header
		scopes := resp.Header.Get("X-OAuth-Scopes")
		scopeStr := ""
		if scopes != "" {
			scopeStr = fmt.Sprintf(" scopes: %s", scopes)
		}
		return Result{Icon: iconPass, Status: "pass", Target: "Token valid", Detail: fmt.Sprintf("valid%s (%dms)", scopeStr, ms)}
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return Result{Icon: iconFail, Status: "fail", Target: "Token valid", Detail: fmt.Sprintf("unauthorized (%dms)", ms)}
	}
	return Result{Icon: iconFail, Status: "fail", Target: "Token valid", Detail: fmt.Sprintf("%s (%dms)", resp.Status, ms)}
}

// checkRepoAccess verifies the configured repository is accessible with current token.
func (g *GitHub) checkRepoAccess(ctx context.Context, token string) Result {
	if g.Owner == "" || g.Repo == "" {
		return Result{Icon: iconWarn, Status: "warn", Target: "Repo accessible", Detail: "owner/repo not configured"}
	}

	client := g.httpClient()
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", g.Owner, g.Repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return Result{Icon: iconFail, Status: "fail", Target: "Repo accessible", Detail: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	start := time.Now()
	resp, err := client.Do(req)
	ms := time.Since(start).Milliseconds()

	if err != nil {
		return Result{Icon: iconFail, Status: "fail", Target: "Repo accessible", Detail: fmt.Sprintf("error: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return Result{Icon: iconPass, Status: "pass", Target: "Repo accessible", Detail: fmt.Sprintf("%s/%s accessible (%dms)", g.Owner, g.Repo, ms)}
	}
	if resp.StatusCode == http.StatusNotFound {
		return Result{Icon: iconFail, Status: "fail", Target: "Repo accessible", Detail: fmt.Sprintf("%s/%s not found", g.Owner, g.Repo)}
	}
	if resp.StatusCode == http.StatusForbidden {
		return Result{Icon: iconWarn, Status: "warn", Target: "Repo accessible", Detail: fmt.Sprintf("permission denied for %s/%s", g.Owner, g.Repo)}
	}
	return Result{Icon: iconFail, Status: "fail", Target: "Repo accessible", Detail: fmt.Sprintf("%s (%dms)", resp.Status, ms)}
}

func (g *GitHub) httpClient() *http.Client {
	if g.HTTPClient != nil {
		return g.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

const (
	iconPass = "✓"
	iconFail = "✗"
	iconWarn = "⚠"
)

// ErrorCount returns the number of failed checks in results.
func ErrorCount(results []Result) int {
	count := 0
	for _, r := range results {
		if r.Status == "fail" {
			count++
		}
	}
	return count
}

// WarningCount returns the number of warning checks in results.
func WarningCount(results []Result) int {
	count := 0
	for _, r := range results {
		if r.Status == "warn" {
			count++
		}
	}
	return count
}

// PassCount returns the number of passed checks in results.
func PassCount(results []Result) int {
	count := 0
	for _, r := range results {
		if r.Status == "pass" {
			count++
		}
	}
	return count
}

// FormatSummary returns a one-line summary string for the results.
func FormatSummary(results []Result) string {
	pass := PassCount(results)
	warn := WarningCount(results)
	fail := ErrorCount(results)
	return fmt.Sprintf("passed=%d warnings=%d errors=%d", pass, warn, fail)
}

// HasFailures returns true if any check failed.
func HasFailures(results []Result) bool {
	return ErrorCount(results) > 0
}

// ParseTokenFromEnv checks for GitHub token in environment variables.
// Returns the token value and the name of the env var that was found.
func ParseTokenFromEnv() (token string, source string) {
	if token = os.Getenv("GH_TOKEN"); token != "" {
		return token, "GH_TOKEN"
	}
	if token = os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, "GITHUB_TOKEN"
	}
	return "", ""
}

// ScopesFromResponse extracts OAuth scopes from a GitHub API response header.
func ScopesFromResponse(header string) []string {
	if header == "" {
		return nil
	}
	scopes := strings.Split(header, ", ")
	result := make([]string, 0, len(scopes))
	for _, s := range scopes {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}