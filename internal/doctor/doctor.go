// Package doctor provides health check commands for PolarSwarm.
// It performs read-only checks on configuration, GitHub connectivity,
// labels, workflows, store, and worktrees without executing any write operations.
package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// StandardLabel represents a label in the standard label set per PolarSwarm.md.
type StandardLabel struct {
	Name        string
	Color       string
	Description string
}

// StandardLabels returns the complete set of 35 standard labels defined in PolarSwarm.md.
// These are grouped by category: status (10), agent (12), decision (3), priority (4), effort (6).
func StandardLabels() []StandardLabel {
	return []StandardLabel{
		// Status labels (mutually exclusive per Issue) - 10 labels
		{Name: "status:pending-triage", Color: "ededed", Description: "External submission, pending authorized user review"},
		{Name: "status:new", Color: "0075ca", Description: "Passed gate, pending Orchestrator review"},
		{Name: "status:triaged", Color: "7057ff", Description: "Reviewed, entering Backlog"},
		{Name: "status:assigned", Color: "cfd3d7", Description: "Assigned to Agent, pending start"},
		{Name: "status:in-progress", Color: "fbca04", Description: "Agent is executing"},
		{Name: "status:blocked", Color: "d73a4a", Description: "Waiting for dependency resolution"},
		{Name: "status:review", Color: "0e8a16", Description: "Execution complete, pending review"},
		{Name: "status:rework", Color: "f39c12", Description: "Review failed, rework assigned"},
		{Name: "status:done", Color: "2ea44f", Description: "Completed and accepted"},
		{Name: "status:abandoned", Color: "e4e4e4", Description: "Cancelled / will not fix"},

		// Agent labels (current responsible Role) - 12 labels
		{Name: "agent:orchestrator", Color: "f9d0c4", Description: "Orchestrator holds this"},
		{Name: "agent:architect", Color: "c5def5", Description: "Architect holds this"},
		{Name: "agent:developer", Color: "bfd4f2", Description: "Developer holds this"},
		{Name: "agent:reviewer", Color: "d4c5f9", Description: "Reviewer holds this"},
		{Name: "agent:security", Color: "f1c40f", Description: "Security Agent holds this"},
		{Name: "agent:documenter", Color: "c2e0c6", Description: "Documenter Agent holds this"},
		{Name: "agent:debugger", Color: "e67e22", Description: "Debugger Agent holds this"},
		{Name: "agent:merger", Color: "5319e7", Description: "Merge expert Agent holds this"},
		{Name: "agent:tester", Color: "1abc9c", Description: "Test Agent holds this"},
		{Name: "agent:devops", Color: "95a5a6", Description: "DevOps Agent holds this"},
		{Name: "agent:compliance", Color: "8e44ad", Description: "Compliance Agent holds this"},
		{Name: "agent:human", Color: "ffffff", Description: "Requires human intervention"},

		// Decision labels (can coexist with status) - 3 labels
		{Name: "decision:approved", Color: "0e8a16", Description: "Approved"},
		{Name: "decision:rejected", Color: "d73a4a", Description: "Rejected"},
		{Name: "decision:needs-info", Color: "e4e669", Description: "Needs additional information"},

		// Priority labels - 4 labels
		{Name: "priority:critical", Color: "d73a4a", Description: "Critical priority"},
		{Name: "priority:high", Color: "e4e669", Description: "High priority"},
		{Name: "priority:normal", Color: "0075ca", Description: "Normal priority"},
		{Name: "priority:low", Color: "cfd3d7", Description: "Low priority"},

		// Effort labels (Fibonacci) - 6 labels
		{Name: "effort:1", Color: "009800", Description: "Effort 1 story point"},
		{Name: "effort:2", Color: "009800", Description: "Effort 2 story points"},
		{Name: "effort:3", Color: "ffac32", Description: "Effort 3 story points"},
		{Name: "effort:5", Color: "ffac32", Description: "Effort 5 story points"},
		{Name: "effort:8", Color: "d73a4a", Description: "Effort 8 story points"},
		{Name: "effort:13", Color: "d73a4a", Description: "Effort 13 story points"},
	}
}

// Labels represents a labels health check.
type Labels struct {
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

// LabelCheckResult represents the result of a labels health check.
type LabelCheckResult struct {
	Present   int      // number of standard labels present
	Total     int      // total number of standard labels
	Missing   []string // names of missing standard labels
	Mismatches []LabelMismatch // labels with color/description mismatches
}

// LabelMismatch represents a label that exists but has different color or description.
type LabelMismatch struct {
	Name        string
	Field       string // "color" or "description"
	Expected    string
	Actual      string
}

// Run executes the labels health check and writes results to Output.
// It compares the repository's actual labels against the standard label set.
// Returns a LabelCheckResult and error (only on check failure, not individual mismatches).
func (l *Labels) Run(ctx context.Context) (*LabelCheckResult, error) {
	out := l.Output
	if out == nil {
		out = os.Stdout
	}

	// Check for token in environment if not provided
	token := l.Token
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	if token == "" {
		fmt.Fprintf(out, "[ labels ]  %s  Token missing   Set GH_TOKEN or GITHUB_TOKEN environment variable\n", iconFail)
		return nil, fmt.Errorf("github token missing: set GH_TOKEN or GITHUB_TOKEN environment variable")
	}

	if l.Owner == "" || l.Repo == "" {
		fmt.Fprintf(out, "[ labels ]  %s  Owner/Repo not configured\n", iconWarn)
		return nil, fmt.Errorf("owner/repo not configured")
	}

	// Fetch repository labels
	actualLabels, err := l.fetchRepoLabels(ctx, token)
	if err != nil {
		fmt.Fprintf(out, "[ labels ]  %s  Failed to fetch labels: %v\n", iconFail, err)
		return nil, err
	}

	// Build map of actual labels by name
	actualMap := make(map[string]Label)
	for _, label := range actualLabels {
		actualMap[label.Name] = label
	}

	standard := StandardLabels()
	result := &LabelCheckResult{
		Present:    0,
		Total:      len(standard),
		Missing:    []string{},
		Mismatches: []LabelMismatch{},
	}

	// Group standard labels by prefix for output
	statusLabels := filterByPrefix(standard, "status:")
	agentLabels := filterByPrefix(standard, "agent:")
	decisionLabels := filterByPrefix(standard, "decision:")
	priorityLabels := filterByPrefix(standard, "priority:")
	effortLabels := filterByPrefix(standard, "effort:")

	// Check each category
	result.Present, result.Missing = checkCategory(statusLabels, actualMap, out, "status")
	result.Present += checkCategoryCount(agentLabels, actualMap, out, "agent")
	result.Present += checkCategoryCount(decisionLabels, actualMap, out, "decision")
	result.Present += checkCategoryCount(priorityLabels, actualMap, out, "priority")
	result.Present += checkCategoryCount(effortLabels, actualMap, out, "effort")

	// Check for mismatches (color/description differences)
	for _, stdLabel := range standard {
		if actual, ok := actualMap[stdLabel.Name]; ok {
			if actual.Color != stdLabel.Color {
				result.Mismatches = append(result.Mismatches, LabelMismatch{
					Name:     stdLabel.Name,
					Field:    "color",
					Expected: stdLabel.Color,
					Actual:   actual.Color,
				})
			}
			if actual.Description != stdLabel.Description {
				result.Mismatches = append(result.Mismatches, LabelMismatch{
					Name:     stdLabel.Name,
					Field:    "description",
					Expected: stdLabel.Description,
					Actual:   actual.Description,
				})
			}
		}
	}

	// Report mismatches
	if len(result.Mismatches) > 0 {
		fmt.Fprintf(out, "[ labels ]  %s  %d label(s) have color/description mismatches\n", iconWarn, len(result.Mismatches))
		for _, m := range result.Mismatches {
			fmt.Fprintf(out, "         %s %s: expected %q, got %q\n", m.Name, m.Field, m.Expected, m.Actual)
		}
	}

	return result, nil
}

// fetchRepoLabels retrieves all labels from the repository.
func (l *Labels) fetchRepoLabels(ctx context.Context, token string) ([]Label, error) {
	client := l.httpClient()
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/labels?per_page=100", l.Owner, l.Repo)

	var allLabels []Label
	page := 1
	for {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Key", token)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status: %s", resp.Status)
		}

		var labels []Label
		if err := json.NewDecoder(resp.Body).Decode(&labels); err != nil {
			return nil, err
		}

		if len(labels) == 0 {
			break
		}
		allLabels = append(allLabels, labels...)

		// Check for next page
		if resp.Header.Get("Link") == "" || !strings.Contains(resp.Header.Get("Link"), "next") {
			break
		}
		page++
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/labels?per_page=100&page=%d", l.Owner, l.Repo, page)
	}

	return allLabels, nil
}

func filterByPrefix(labels []StandardLabel, prefix string) []StandardLabel {
	var result []StandardLabel
	for _, l := range labels {
		if strings.HasPrefix(l.Name, prefix) {
			result = append(result, l)
		}
	}
	return result
}

func checkCategory(labels []StandardLabel, actualMap map[string]Label, out io.Writer, category string) (present int, missing []string) {
	for _, stdLabel := range labels {
		if _, ok := actualMap[stdLabel.Name]; ok {
			present++
		} else {
			missing = append(missing, stdLabel.Name)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(out, "[ labels ]  %s  %s:*         %d/%d present — missing: %s\n", iconFail, category, present, len(labels), strings.Join(missing, ", "))
		fmt.Fprintf(out, "         run `polarswarm init` to create standard labels\n")
	} else {
		fmt.Fprintf(out, "[ labels ]  %s  %s:*         %d/%d present\n", iconPass, category, present, len(labels))
	}
	return present, missing
}

func checkCategoryCount(labels []StandardLabel, actualMap map[string]Label, out io.Writer, category string) int {
	present := 0
	for _, stdLabel := range labels {
		if _, ok := actualMap[stdLabel.Name]; ok {
			present++
		}
	}
	if present == len(labels) {
		fmt.Fprintf(out, "[ labels ]  %s  %s:*         %d/%d present\n", iconPass, category, present, len(labels))
	} else {
		fmt.Fprintf(out, "[ labels ]  %s  %s:*         %d/%d present\n", iconWarn, category, present, len(labels))
	}
	return present
}

// Label represents a GitHub label with its properties.
type Label struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

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

func (l *Labels) httpClient() *http.Client {
	if l.HTTPClient != nil {
		return l.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
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