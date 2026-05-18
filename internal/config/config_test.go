package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	path := writeConfig(t, `
[github]
owner = "PolarKits"
repo = "PolarSwarm"

[workflow]
target_label = "status:new"
dry_run = false
confirm_writes = true
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.GitHub.Owner != "PolarKits" || cfg.GitHub.Repo != "PolarSwarm" {
		t.Fatalf("unexpected github config: %+v", cfg.GitHub)
	}
	if cfg.Workflow.TargetLabel != "status:new" {
		t.Fatalf("unexpected target label: %q", cfg.Workflow.TargetLabel)
	}
	if cfg.Workflow.DryRun {
		t.Fatal("expected explicit dry_run=false")
	}
	if !cfg.Workflow.ConfirmWrites {
		t.Fatal("expected explicit confirm_writes=true")
	}
}

func TestLoadConfigAcceptsRepositoryRuntimeAliases(t *testing.T) {
	path := writeConfig(t, `
[repository]
owner = "PolarKits"
name = "PolarSwarm"

[runtime]
dry_run = true
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.GitHub.Owner != "PolarKits" || cfg.GitHub.Repo != "PolarSwarm" {
		t.Fatalf("unexpected github config: %+v", cfg.GitHub)
	}
	if cfg.Workflow.TargetLabel != "status:new" {
		t.Fatalf("expected default target label, got %q", cfg.Workflow.TargetLabel)
	}
	if !cfg.Workflow.DryRun {
		t.Fatal("expected dry_run=true")
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestLoadConfigRequiresGitHubOwner(t *testing.T) {
	path := writeConfig(t, `
[github]
repo = "PolarSwarm"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "[github].owner") {
		t.Fatalf("expected owner validation error, got %v", err)
	}
}

func TestLoadConfigDefaultsDryRunTrue(t *testing.T) {
	path := writeConfig(t, `
[github]
owner = "PolarKits"
repo = "PolarSwarm"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !cfg.Workflow.DryRun {
		t.Fatal("expected dry_run to default to true")
	}
	if cfg.Workflow.ConfirmWrites {
		t.Fatal("expected confirm_writes to default to false")
	}
}

func TestSummaryDoesNotIncludeUnknownSecretFields(t *testing.T) {
	path := writeConfig(t, `
[github]
owner = "PolarKits"
repo = "PolarSwarm"
token = "ghp_should_not_appear"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	summary := cfg.Summary()
	if strings.Contains(summary, "ghp_should_not_appear") || strings.Contains(summary, "token") {
		t.Fatalf("summary leaked secret-like field: %q", summary)
	}
	if !strings.Contains(summary, "confirm_writes=false") {
		t.Fatalf("summary missing confirm_writes: %q", summary)
	}
}

func TestUnknownFieldWarning(t *testing.T) {
	path := writeConfig(t, `
[github]
owner = "PolarKits"
repo = "PolarSwarm"
unknown_field = "should_warn"
`)

	_, err := Load(path)
	if err != nil {
		t.Fatalf("Load should not error on unknown field, got: %v", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "core.toml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
