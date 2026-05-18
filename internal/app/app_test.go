package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIHelp(t *testing.T) {
	var out bytes.Buffer
	cli := CLI{Stdout: &out}

	if err := cli.Run([]string{"help"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(out.String(), "PolarSwarm") {
		t.Fatalf("help output missing product name: %q", out.String())
	}
}

func TestCLIVersion(t *testing.T) {
	var out bytes.Buffer
	cli := CLI{Stdout: &out}

	if err := cli.Run([]string{"version"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(out.String(), Version) {
		t.Fatalf("version output missing version: %q", out.String())
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	cli := CLI{Stdout: &bytes.Buffer{}}

	if err := cli.Run([]string{"missing"}); err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestCLIConfigCheck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "core.toml")
	content := `[github]
owner = "PolarKits"
repo = "PolarSwarm"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var out bytes.Buffer
	cli := CLI{Stdout: &out}

	if err := cli.Run([]string{"config", "check", "--config", path}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "config ok") || !strings.Contains(output, "dry_run=true") {
		t.Fatalf("unexpected config check output: %q", output)
	}
}

func TestCLIConfigCheckMissingPath(t *testing.T) {
	cli := CLI{Stdout: &bytes.Buffer{}}

	if err := cli.Run([]string{"config", "check", "--config"}); err == nil {
		t.Fatal("expected error for missing --config path")
	}
}

func TestCLIIssueReadFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	content := `{
  "issues": [
    {
      "repository": {"owner": "PolarKits", "name": "PolarSwarm"},
      "number": 3,
      "title": "GitHub reader",
      "state": "open",
      "labels": [{"name": "status:new"}, {"name": "area:github"}],
      "comments": [{"id": 1, "author": "alice", "body": "ready"}]
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var out bytes.Buffer
	cli := CLI{Stdout: &out}

	err := cli.Run([]string{"issue", "read", "--repo", "PolarKits/PolarSwarm", "--number", "3", "--fixture", path})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	output := out.String()
	for _, want := range []string{"issue PolarKits/PolarSwarm#3", "GitHub reader", "status:new, area:github", "comments: 1"} {
		if !strings.Contains(output, want) {
			t.Fatalf("issue read output missing %q: %q", want, output)
		}
	}
}

func TestCLIIssueReadRequiresFixture(t *testing.T) {
	cli := CLI{Stdout: &bytes.Buffer{}}

	err := cli.Run([]string{"issue", "read", "--repo", "PolarKits/PolarSwarm", "--number", "3"})
	if err == nil {
		t.Fatal("expected error for missing fixture")
	}
}

func TestCLIWritebackDryRunFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	content := `{
  "issues": [
    {
      "repository": {"owner": "PolarKits", "name": "PolarSwarm"},
      "number": 6,
      "title": "Writeback planner",
      "state": "open",
      "labels": [{"name": "status:in-progress"}, {"name": "area:github"}],
      "comments": []
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var out bytes.Buffer
	cli := CLI{Stdout: &out}

	err := cli.Run([]string{"writeback", "dry-run", "--repo", "PolarKits/PolarSwarm", "--number", "6", "--fixture", path})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"dry-run write plan for PolarKits/PolarSwarm#6",
		"```polarswarm-agent-result",
		`"role": "developer"`,
		"remove label: status:in-progress",
		"add label: status:review",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("writeback dry-run output missing %q: %q", want, output)
		}
	}
}

func TestCLIAcceptanceDryRunFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	content := `{
  "issues": [
    {
      "repository": {"owner": "PolarKits", "name": "PolarSwarm"},
      "number": 7,
      "title": "M1 acceptance",
      "state": "open",
      "labels": [{"name": "status:in-progress"}, {"name": "area:workflow"}],
      "comments": []
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var out bytes.Buffer
	cli := CLI{Stdout: &out}

	err := cli.Run([]string{"acceptance", "dry-run", "--repo", "PolarKits/PolarSwarm", "--number", "7", "--fixture", path})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"acceptance dry-run for PolarKits/PolarSwarm#7",
		"operation_id:",
		"dispatchable: yes",
		"state: in-progress -> review",
		"```polarswarm-agent-result",
		"add label: status:review",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("acceptance dry-run output missing %q: %q", want, output)
		}
	}
}

func TestCLIAcceptanceDryRunFailedDoesNotAdvanceLabel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	content := `{
  "issues": [
    {
      "repository": {"owner": "PolarKits", "name": "PolarSwarm"},
      "number": 8,
      "title": "M1 acceptance failure",
      "state": "open",
      "labels": [{"name": "status:in-progress"}],
      "comments": []
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var out bytes.Buffer
	cli := CLI{Stdout: &out}

	err := cli.Run([]string{"acceptance", "dry-run", "--repo", "PolarKits/PolarSwarm", "--number", "8", "--fixture", path, "--force-failure"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "state: in-progress -> in-progress") {
		t.Fatalf("failed acceptance output missing stable state: %q", output)
	}
	if strings.Contains(output, "add label:") || strings.Contains(output, "remove label:") {
		t.Fatalf("failed acceptance dry-run should not include label operations: %q", output)
	}
}
