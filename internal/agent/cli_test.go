package agent

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// testableCLIRunner is a CLIRunner that records invocations for testing.
type testableCLIRunner struct {
	CLIRunner
	invocations []cliInvocation
}

type cliInvocation struct {
	req RunRequest
	cmd *exec.Cmd
}

func (r *testableCLIRunner) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	r.invocations = append(r.invocations, cliInvocation{req: req})
	return r.CLIRunner.Run(ctx, req)
}

func TestCLIRunnerRejectsEmptyCommand(t *testing.T) {
	_, err := NewCLIRunner(CLIConfig{Command: ""})
	if err == nil {
		t.Fatal("expected error for empty command")
	}
	if !strings.Contains(err.Error(), "non-empty command") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCLIRunnerRequiresValidRequest(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{Command: "echo"})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}

	// Empty role should fail validation.
	_, err = runner.Run(context.Background(), RunRequest{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "role is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCLIRunnerBuildArgs(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{
		Command: "opencode",
		Args:    []string{"--no-interactive"},
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}

	req := RunRequest{
		Role:   "developer",
		Issue:  IssueRef{Repository: "test/repo", Number: 42, Title: "Test Issue"},
		Branch: "task/42",
	}

	args := runner.buildArgs(req)

	// Default args should come first.
	if len(args) < 1 || args[0] != "--no-interactive" {
		t.Fatalf("expected default args first, got %v", args)
	}

	// Should include --prompt.
	found := false
	for i, arg := range args {
		if arg == "--prompt" && i+1 < len(args) {
			found = true
			prompt := args[i+1]
			if !strings.Contains(prompt, "developer") {
				t.Fatalf("prompt should contain role, got %s", prompt)
			}
			if !strings.Contains(prompt, "Test Issue") {
				t.Fatalf("prompt should contain title, got %s", prompt)
			}
		}
	}
	if !found {
		t.Fatal("expected --prompt argument")
	}
}

func TestCLIRunnerBuildArgsWithModel(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{Command: "opencode"})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}
	runner.model = "claude-sonnet-4-6"

	req := RunRequest{
		Role:   "developer",
		Issue:  IssueRef{Repository: "test/repo", Number: 1, Title: "T"},
		Branch: "task/1",
		Model:  "claude-opus-4-7", // Request overrides runner default.
	}

	args := runner.buildArgs(req)

	found := false
	for i, arg := range args {
		if arg == "--model" && i+1 < len(args) {
			found = true
			if args[i+1] != "claude-opus-4-7" {
				t.Fatalf("expected request model, got %s", args[i+1])
			}
		}
	}
	if !found {
		t.Fatal("expected --model argument")
	}
}

func TestCLIRunnerBuildArgsWithWorkingDirectory(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{Command: "opencode"})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}

	req := RunRequest{
		Role:           "developer",
		Issue:          IssueRef{Repository: "test/repo", Number: 1, Title: "T"},
		Branch:         "task/1",
		WorkingDirectory: "/path/to/worktree",
	}

	args := runner.buildArgs(req)

	found := false
	for i, arg := range args {
		if arg == "--dir" && i+1 < len(args) {
			found = true
			if args[i+1] != "/path/to/worktree" {
				t.Fatalf("expected working directory, got %s", args[i+1])
			}
		}
	}
	if !found {
		t.Fatal("expected --dir argument")
	}
}

func TestCLIRunnerBuildPrompt(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{Command: "echo"})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}

	req := RunRequest{
		Role:   "reviewer",
		Issue:  IssueRef{Repository: "org/project", Number: 99, Title: "Fix Bug"},
		Branch: "task/99",
	}

	prompt := runner.buildPrompt(req)

	if !strings.Contains(prompt, "reviewer") {
		t.Fatalf("prompt should contain role, got %s", prompt)
	}
	if !strings.Contains(prompt, "Fix Bug") {
		t.Fatalf("prompt should contain title, got %s", prompt)
	}
	if !strings.Contains(prompt, "org/project") {
		t.Fatalf("prompt should contain repository, got %s", prompt)
	}
	if !strings.Contains(prompt, "99") {
		t.Fatalf("prompt should contain issue number, got %s", prompt)
	}
}

func TestCLIRunnerBuildResultWithExitCode(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{Command: "opencode"})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}

	req := RunRequest{
		Role:   "developer",
		Issue:  IssueRef{Repository: "test/repo", Number: 1, Title: "T"},
		Branch: "task/1",
	}

	result := runner.buildResult(req, "", "error message", &exec.ExitError{ProcessState: nil, Stderr: []byte("error message")})

	if result.Status != StatusFailed {
		t.Fatalf("expected failed status for non-zero exit, got %s", result.Status)
	}
	if result.Success() {
		t.Fatal("result should not be successful")
	}
	if result.Error == "" {
		t.Fatal("result should have error message")
	}
}

func TestCLIRunnerBuildResultCompleted(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{Command: "opencode"})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}

	req := RunRequest{
		Role:   "developer",
		Issue:  IssueRef{Repository: "test/repo", Number: 1, Title: "T"},
		Branch: "task/1",
	}

	result := runner.buildResult(req, `{"verification":"unit test passed","confidence":0.9,"message":"ok"}`, "", nil)

	if result.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %s", result.Status)
	}
	if !result.Success() {
		t.Fatal("result should be successful")
	}
	if result.Verification != "unit test passed" {
		t.Fatalf("expected verification from JSON, got %s", result.Verification)
	}
	if result.Confidence != 0.9 {
		t.Fatalf("expected confidence from JSON, got %v", result.Confidence)
	}
	if result.Message != "ok" {
		t.Fatalf("expected message from JSON, got %s", result.Message)
	}
}

func TestCLIRunnerBuildResultDefaultsOnFailure(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{Command: "opencode"})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}

	req := RunRequest{
		Role:   "developer",
		Issue:  IssueRef{Repository: "test/repo", Number: 1, Title: "T"},
		Branch: "task/1",
	}

	result := runner.buildResult(req, "", "", &exec.ExitError{ProcessState: nil})

	// Should have default verification mentioning exit code.
	if !strings.Contains(result.Verification, "cli exit code") {
		t.Fatalf("expected default verification on failure, got %s", result.Verification)
	}
	// Should have low confidence on failure.
	if result.Confidence != 0.2 {
		t.Fatalf("expected low confidence on failure, got %v", result.Confidence)
	}
}

func TestCLIRunnerBuildResultDefaultsOnSuccess(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{Command: "opencode"})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}

	req := RunRequest{
		Role:   "developer",
		Issue:  IssueRef{Repository: "test/repo", Number: 1, Title: "T"},
		Branch: "task/1",
	}

	result := runner.buildResult(req, "raw stdout output", "", nil)

	// Should have default verification on success.
	if result.Verification != "cli completed" {
		t.Fatalf("expected default verification on success, got %s", result.Verification)
	}
	// Should have high confidence on success.
	if result.Confidence != 0.95 {
		t.Fatalf("expected high confidence on success, got %v", result.Confidence)
	}
}

func TestCLIRunnerCapabilities(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{Command: "opencode"})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}

	caps := runner.Capabilities()

	// CLIRunner does not support true streaming.
	if caps.Streaming {
		t.Error("CLIRunner should not claim streaming support")
	}
	// CLIRunner does not support structured output natively.
	if caps.StructuredOutput {
		t.Error("CLIRunner should not claim structured output")
	}
	// CLIRunner respects working directory.
	if !caps.WorkingDirectory {
		t.Error("CLIRunner should support working directory")
	}
}

func TestCLIRunnerStream(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{Command: "echo", Args: []string{"hello"}})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}

	ch, err := runner.Stream(context.Background(), RunRequest{
		Role:   "developer",
		Issue:  IssueRef{Repository: "test/repo", Number: 1, Title: "T"},
		Branch: "task/1",
	})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	event, ok := <-ch
	if !ok {
		t.Fatal("channel should have one event")
	}
	if !event.Done {
		t.Fatal("stream event should be done")
	}
	if event.Type != "done" {
		t.Fatalf("event type should be 'done', got %q", event.Type)
	}
	if !strings.Contains(event.Content, "status=") {
		t.Fatalf("event content should include status, got %s", event.Content)
	}
}

func TestCLIRunnerContextCancellation(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{Command: "sleep", Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err = runner.Run(ctx, RunRequest{
		Role:   "developer",
		Issue:  IssueRef{Repository: "test/repo", Number: 1, Title: "T"},
		Branch: "task/1",
	})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestCLIRunnerResultJSON(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{Command: "opencode"})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}

	result := runner.buildResult(RunRequest{
		Role:   "developer",
		Issue:  IssueRef{Repository: "test/repo", Number: 1, Title: "T"},
		Branch: "task/1",
	}, `{"verification":"passed","confidence":0.85,"message":"done"}`, "", nil)

	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	for _, want := range []string{`"role"`, `"issue"`, `"branch"`, `"status"`, `"verification"`, `"confidence"`} {
		if !strings.Contains(string(payload), want) {
			t.Fatalf("result JSON missing %s: %s", want, payload)
		}
	}
}

func TestCLIRunnerValidateResult(t *testing.T) {
	runner, err := NewCLIRunner(CLIConfig{Command: "opencode"})
	if err != nil {
		t.Fatalf("NewCLIRunner failed: %v", err)
	}

	result := runner.buildResult(RunRequest{
		Role:   "developer",
		Issue:  IssueRef{Repository: "test/repo", Number: 1, Title: "T"},
		Branch: "task/1",
	}, `{"verification":"passed","confidence":0.85}`, "", nil)

	if err := result.Validate(); err != nil {
		t.Fatalf("result should validate: %v", err)
	}
}