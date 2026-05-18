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
