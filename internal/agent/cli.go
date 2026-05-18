package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CLIConfig defines the configuration for a CLI backend adapter.
// This is the runtime configuration that maps a Role to a specific CLI tool.
type CLIConfig struct {
	// Command is the CLI command to execute (e.g., "opencode", "claude-code").
	Command string

	// Args are default arguments passed to the CLI tool.
	Args []string

	// Timeout is the maximum duration for a single execution.
	// Zero means no timeout.
	Timeout time.Duration
}

// CLIRunner implements BackendRunner for non-interactive CLI tools like
// OpenCode, Claude Code, Codex, and similar tools. It executes the CLI
// as a subprocess and captures stdout, stderr, and exit code.
type CLIRunner struct {
	Config CLIConfig

	// optional model override; if set, overrides the CLI default
	model string
}

// NewCLIRunner creates a CLIRunner with the given configuration.
// The config.Command must be non-empty.
func NewCLIRunner(config CLIConfig) (*CLIRunner, error) {
	if strings.TrimSpace(config.Command) == "" {
		return nil, errors.New("cli runner requires a non-empty command")
	}
	return &CLIRunner{Config: config}, nil
}

// Run executes the CLI tool synchronously and returns a RunResult.
// It captures stdout/stderr, respects timeout, and normalizes the exit code.
// Non-zero exit codes result in StatusFailed without advancing workflow state.
func (r *CLIRunner) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	if err := ctx.Err(); err != nil {
		return RunResult{}, err
	}
	if err := req.validate(); err != nil {
		return RunResult{}, err
	}

	// Build the command arguments.
	// Adapter must pass PolarSwarm computed effective config as highest priority,
	// preferring CLI flags over env vars or config files.
	args := r.buildArgs(req)

	// Determine working directory.
	workDir := req.WorkingDirectory
	if workDir == "" {
		workDir = "."
	}

	// Use CommandContext for proper timeout/cancellation support.
	cmd := exec.CommandContext(ctx, r.Config.Command, args...)
	cmd.Dir = workDir

	// Capture output.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute.
	err := cmd.Run()

	// Build result from output.
	result := r.buildResult(req, stdout.String(), stderr.String(), err)

	// Validate before returning.
	if err := result.Validate(); err != nil {
		return RunResult{}, fmt.Errorf("cli result validation failed: %w", err)
	}

	return result, nil
}

// Stream executes the CLI tool with streaming event output.
// Currently, CLI tools do not support true streaming, so this returns
// a single "done" event with the final result, similar to MockRunner.
func (r *CLIRunner) Stream(ctx context.Context, req RunRequest) (<-chan RunEvent, error) {
	result, err := r.Run(ctx, req)
	if err != nil {
		return nil, err
	}

	ch := make(chan RunEvent, 1)
	ch <- RunEvent{
		Type:    "done",
		Content: fmt.Sprintf("status=%s verification=%s confidence=%.2f", result.Status, result.Verification, result.Confidence),
		Done:    true,
	}
	close(ch)
	return ch, nil
}

// Capabilities returns CLIRunner capabilities.
// CLIRunner does not support true streaming or structured output natively,
// but respects working directory when set.
func (r *CLIRunner) Capabilities() BackendCapabilities {
	return BackendCapabilities{
		Streaming:         false, // CLI tools do not support true streaming
		StructuredOutput:  false, // Requires prompt engineering or output parsing
		WorkingDirectory:  true,  // CLI tools respect cwd
	}
}

// buildArgs constructs CLI arguments from the request.
// It formats the prompt from the Issue context and passes the model if set.
func (r *CLIRunner) buildArgs(req RunRequest) []string {
	args := make([]string, 0, len(r.Config.Args)+8)

	// Append default args first.
	args = append(args, r.Config.Args...)

	// Build prompt from issue context.
	prompt := r.buildPrompt(req)

	// Pass prompt as argument (most CLI tools accept this).
	args = append(args, "--prompt", prompt)

	// Pass model if specified in request or runner config.
	model := req.Model
	if model == "" {
		model = r.model
	}
	if model != "" {
		args = append(args, "--model", model)
	}

	// Pass working directory as explicit flag to override project config.
	if req.WorkingDirectory != "" {
		args = append(args, "--dir", req.WorkingDirectory)
	}

	return args
}

// buildPrompt formats the issue context as a prompt for the CLI tool.
func (r *CLIRunner) buildPrompt(req RunRequest) string {
	// Format: Role directive with issue context.
	var sb strings.Builder
	sb.WriteString(req.Role)
	sb.WriteString(": ")
	sb.WriteString(req.Issue.Title)
	sb.WriteString(" (")
	sb.WriteString(req.Issue.Repository)
	sb.WriteString("#")
	sb.WriteString(fmt.Sprintf("%d", req.Issue.Number))
	sb.WriteString(")")
	return sb.String()
}

// buildResult constructs a RunResult from CLI execution output.
// It parses stdout for structured result and normalizes exit codes.
func (r *CLIRunner) buildResult(req RunRequest, stdout, stderr string, err error) RunResult {
	var exitCode int
	var exitError string

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
			exitError = stderr
		} else {
			exitCode = -1
			exitError = err.Error()
		}
	}

	// Determine status based on exit code.
	// Non-zero exit codes result in StatusFailed.
	status := StatusCompleted
	if exitCode != 0 {
		status = StatusFailed
	}

	// Parse structured output from stdout if available.
	verification := ""
	confidence := 0.0

	var structured struct {
		Verification string  `json:"verification"`
		Confidence   float64 `json:"confidence"`
		Message      string  `json:"message"`
	}

	if err := json.Unmarshal([]byte(stdout), &structured); err == nil {
		verification = structured.Verification
		confidence = structured.Confidence
	}

	// Apply defaults.
	if verification == "" {
		if status == StatusFailed {
			verification = fmt.Sprintf("cli exit code %d", exitCode)
		} else {
			verification = "cli completed"
		}
	}
	if confidence == 0 {
		if status == StatusFailed {
			confidence = 0.2
		} else {
			confidence = 0.95
		}
	}

	result := RunResult{
		Role:         req.Role,
		Issue:        req.Issue,
		Branch:       req.Branch,
		Status:       status,
		Verification: verification,
		Confidence:   confidence,
	}

	if status == StatusCompleted {
		if structured.Message != "" {
			result.Message = structured.Message
		} else if stdout != "" {
			result.Message = stdout
		}
	} else {
		result.Error = exitError
		if result.Error == "" {
			result.Error = fmt.Sprintf("cli exited with code %d", exitCode)
		}
	}

	return result
}