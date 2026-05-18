package app

import (
	"bytes"
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
