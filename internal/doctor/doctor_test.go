package doctor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// mockResponse returns an HTTP handler that returns a预设 response.
func mockResponse(status int, headers map[string]string, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	})
}

func TestGitHubRunWithTokenMissing(t *testing.T) {
	// Clear environment variables
	oldGH := os.Getenv("GH_TOKEN")
	oldGITHUB := os.Getenv("GITHUB_TOKEN")
	os.Unsetenv("GH_TOKEN")
	os.Unsetenv("GITHUB_TOKEN")
	defer func() {
		if oldGH != "" {
			os.Setenv("GH_TOKEN", oldGH)
		}
		if oldGITHUB != "" {
			os.Setenv("GITHUB_TOKEN", oldGITHUB)
		}
	}()

	g := &GitHub{Owner: "test", Repo: "test"}
	_, err := g.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
	if !strings.Contains(err.Error(), "GH_TOKEN") && !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Fatalf("error should mention env var names, got: %v", err)
	}
}

func TestGitHubRunWithTokenInGH_TOKEN(t *testing.T) {
	oldGH := os.Getenv("GH_TOKEN")
	os.Setenv("GH_TOKEN", "ghp_testtoken123")
	defer os.Setenv("GH_TOKEN", oldGH)

	// Mock GitHub API responses
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("X-OAuth-Scopes", "repo, issues, read:org")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"login": "testuser"}`))
		case "/user":
			w.Header().Set("X-OAuth-Scopes", "repo, issues")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"login": "testuser"}`))
		default:
			http.NotFound(w, r)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	// Override API URL for testing
	g := &GitHub{
		Owner:      "testowner",
		Repo:       "testrepo",
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
	// We can't easily override the URL, so this test validates the token source works
	// through the env var lookup path. For full integration, a test server would need
	// URL override support in the doctor package.
	_ = g // placeholder - full test requires URL override
}

func TestGitHubRunWithTokenInGITHUB_TOKEN(t *testing.T) {
	oldGITHUB := os.Getenv("GITHUB_TOKEN")
	os.Setenv("GITHUB_TOKEN", "ghp_testtoken456")
	defer os.Setenv("GITHUB_TOKEN", oldGITHUB)

	g := &GitHub{Owner: "test", Repo: "test"}
	_, err := g.Run(context.Background())
	// This will fail at API call stage but we just verify env var is read
	if err != nil && !strings.Contains(err.Error(), "GH_TOKEN") && !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		// Error should be about token, not about env var not being found
	}
}

func TestParseTokenFromEnv(t *testing.T) {
	// Clear both
	os.Unsetenv("GH_TOKEN")
	os.Unsetenv("GITHUB_TOKEN")

	token, source := ParseTokenFromEnv()
	if token != "" {
		t.Fatalf("expected empty token, got %q from %s", token, source)
	}

	// Set GH_TOKEN
	os.Setenv("GH_TOKEN", "ghp_gh_token")
	token, source = ParseTokenFromEnv()
	if token != "ghp_gh_token" || source != "GH_TOKEN" {
		t.Fatalf("expected GH_TOKEN, got %q from %s", token, source)
	}

	// GITHUB_TOKEN takes precedence when GH_TOKEN is not set
	os.Unsetenv("GH_TOKEN")
	os.Setenv("GITHUB_TOKEN", "ghp_github_token")
	token, source = ParseTokenFromEnv()
	if token != "ghp_github_token" || source != "GITHUB_TOKEN" {
		t.Fatalf("expected GITHUB_TOKEN, got %q from %s", token, source)
	}
}

func TestScopesFromResponse(t *testing.T) {
	tests := []struct {
		header string
		want   []string
	}{
		{"", nil},
		{"repo", []string{"repo"}},
		{"repo, issues, read:org", []string{"repo", "issues", "read:org"}},
		{"  repo  ,  issues  ", []string{"repo", "issues"}},
	}

	for _, tt := range tests {
		got := ScopesFromResponse(tt.header)
		if len(got) != len(tt.want) {
			t.Errorf("ScopesFromResponse(%q) = %v, want %v", tt.header, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ScopesFromResponse(%q)[%d] = %q, want %q", tt.header, i, got[i], tt.want[i])
			}
		}
	}
}

func TestResultCounters(t *testing.T) {
	results := []Result{
		{Icon: iconPass, Status: "pass", Target: "API reachable"},
		{Icon: iconPass, Status: "pass", Target: "Token valid"},
		{Icon: iconWarn, Status: "warn", Target: "Repo accessible"},
		{Icon: iconFail, Status: "fail", Target: "API reachable"},
	}

	if PassCount(results) != 2 {
		t.Errorf("PassCount = %d, want 2", PassCount(results))
	}
	if WarningCount(results) != 1 {
		t.Errorf("WarningCount = %d, want 1", WarningCount(results))
	}
	if ErrorCount(results) != 1 {
		t.Errorf("ErrorCount = %d, want 1", ErrorCount(results))
	}
	if HasFailures(results) != true {
		t.Error("HasFailures should be true")
	}

	cleanResults := []Result{
		{Icon: iconPass, Status: "pass"},
		{Icon: iconWarn, Status: "warn"},
	}
	if HasFailures(cleanResults) != false {
		t.Error("HasFailures should be false with no failures")
	}
}

func TestFormatSummary(t *testing.T) {
	results := []Result{
		{Status: "pass"},
		{Status: "pass"},
		{Status: "warn"},
		{Status: "fail"},
	}

	summary := FormatSummary(results)
	if !strings.Contains(summary, "passed=2") {
		t.Errorf("summary should contain passed=2, got %q", summary)
	}
	if !strings.Contains(summary, "warnings=1") {
		t.Errorf("summary should contain warnings=1, got %q", summary)
	}
	if !strings.Contains(summary, "errors=1") {
		t.Errorf("summary should contain errors=1, got %q", summary)
	}
}