package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// BackendRunner is the unified interface for all Agent execution backends.
// Both mock backends and real CLI backends (opencode, claude-code, etc.)
// implement this interface, allowing the Orchestrator to remain decoupled
// from specific backend implementations.
type BackendRunner interface {
	// Run executes the request synchronously and returns a stable result.
	Run(ctx context.Context, req RunRequest) (RunResult, error)

	// Stream executes the request with streaming event output.
	// Returns a channel that yields RunEvent values until the context
	// is cancelled or the execution completes.
	Stream(ctx context.Context, req RunRequest) (<-chan RunEvent, error)

	// Capabilities returns the backend's supported features and limitations.
	Capabilities() BackendCapabilities
}

// BackendCapabilities describes what features a BackendRunner supports.
type BackendCapabilities struct {
	// Streaming indicates whether the backend supports Stream().
	Streaming bool

	// StructuredOutput indicates whether the backend natively supports
	// structured (JSON) output without prompt engineering.
	StructuredOutput bool

	// WorkingDirectory indicates whether the backend respects the
	// working directory passed in RunRequest.WorkingDirectory.
	WorkingDirectory bool
}

// RunRequest is the input passed to BackendRunner.Run.
type RunRequest struct {
	// Role is the agent role (e.g., "developer", "reviewer", "tester:unit").
	Role string

	// Issue contains the stable issue identity for context.
	Issue IssueRef

	// Branch is the git branch the agent should work on.
	Branch string

	// WorkingDirectory is the path to the worktree directory.
	// Empty string means use the current working directory.
	WorkingDirectory string

	// Model is the model identifier to use (e.g., "claude-sonnet-4-6").
	// Empty string means use the backend's default.
	Model string

	// ForceFailure is only used by test/mock backends to simulate failures.
	ForceFailure bool
}

// RunEvent represents a streaming event from a BackendRunner execution.
type RunEvent struct {
	// Type classifies the event (e.g., "message", "error", "done").
	Type string

	// Content is the event payload. For "message" events, this is
	// the text output. For "error" events, this is the error message.
	Content string

	// Done indicates whether this is the final event.
	Done bool
}

// RunResult is the stable result payload returned by BackendRunner.Run.
// This is the T006 format that polarswarm-agent-result renders.
type RunResult struct {
	Role         string   `json:"role"`
	Issue        IssueRef `json:"issue"`
	Branch       string   `json:"branch"`
	Status       Status   `json:"status"`
	Verification string   `json:"verification"`
	Confidence   float64  `json:"confidence"`
	Message      string   `json:"message,omitempty"`
	Error        string   `json:"error,omitempty"`
	// BackendID is the identifier of the backend that executed this result.
	// Examples: "opencode", "claude-code", "mock", "api:anthropic"
	BackendID string `json:"backend_id,omitempty"`
	// ModelID is the model or command identifier used by the backend.
	// Examples: "claude-sonnet-4-6", "o4-mini", "anthropic:claude-sonnet-4-6"
	ModelID string `json:"model_id,omitempty"`
	// DurationMs is the wall-clock execution time in milliseconds.
	DurationMs int64 `json:"duration_ms,omitempty"`
}

// Success reports whether the result may be treated as a successful Agent run.
func (r RunResult) Success() bool {
	return r.Status == StatusCompleted
}

// Validate checks that a RunResult contains the required stable fields.
func (r RunResult) Validate() error {
	if strings.TrimSpace(r.Role) == "" {
		return errors.New("agent result role is required")
	}
	if err := r.Issue.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.Branch) == "" {
		return errors.New("agent result branch is required")
	}
	if !r.Status.IsValid() {
		return fmt.Errorf("invalid agent result status %q", r.Status)
	}
	if strings.TrimSpace(r.Verification) == "" {
		return errors.New("agent result verification is required")
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		return fmt.Errorf("agent result confidence must be between 0 and 1: %v", r.Confidence)
	}
	return nil
}

// Status is the Agent execution outcome.
type Status string

const (
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// IsValid reports whether status is a known Agent result status.
func (s Status) IsValid() bool {
	switch s {
	case StatusCompleted, StatusFailed:
		return true
	default:
		return false
	}
}

// IssueRef is the stable issue identity included in Agent requests and results.
type IssueRef struct {
	Repository string `json:"repository"`
	Number     int    `json:"number"`
	Title      string `json:"title,omitempty"`
	URL        string `json:"url,omitempty"`
}

// Runner executes an Agent request and returns a stable result payload.
// Deprecated: Use BackendRunner instead. This interface is retained for
// backward compatibility with existing code.
type Runner interface {
	Run(ctx context.Context, req Request) (Result, error)
}

// Request describes the work assigned to an Agent.
// Deprecated: Use RunRequest instead.
type Request struct {
	Role         string   `json:"role"`
	Issue        IssueRef `json:"issue"`
	Branch       string   `json:"branch"`
	ForceFailure bool     `json:"force_failure,omitempty"`
}

// Result is the payload T006 can render as polarswarm-agent-result.
// Deprecated: Use RunResult instead.
type Result struct {
	Role         string   `json:"role"`
	Issue        IssueRef `json:"issue"`
	Branch       string   `json:"branch"`
	Status       Status   `json:"status"`
	Verification string   `json:"verification"`
	Confidence   float64  `json:"confidence"`
	Message      string   `json:"message,omitempty"`
	Error        string   `json:"error,omitempty"`
	// BackendID is the identifier of the backend that executed this result.
	BackendID string `json:"backend_id,omitempty"`
	// ModelID is the model or command identifier used by the backend.
	ModelID string `json:"model_id,omitempty"`
	// DurationMs is the wall-clock execution time in milliseconds.
	DurationMs int64 `json:"duration_ms,omitempty"`
}

// Success reports whether the result may be treated as a successful Agent run.
// Deprecated: Use RunResult.Success instead.
func (r Result) Success() bool {
	return r.Status == StatusCompleted
}

// Validate checks that a Result contains the required stable fields.
// Deprecated: Use RunResult.Validate instead.
func (r Result) Validate() error {
	if strings.TrimSpace(r.Role) == "" {
		return errors.New("agent result role is required")
	}
	if err := r.Issue.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.Branch) == "" {
		return errors.New("agent result branch is required")
	}
	if !r.Status.IsValid() {
		return fmt.Errorf("invalid agent result status %q", r.Status)
	}
	if strings.TrimSpace(r.Verification) == "" {
		return errors.New("agent result verification is required")
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		return fmt.Errorf("agent result confidence must be between 0 and 1: %v", r.Confidence)
	}
	return nil
}

// MockRunner returns deterministic Agent results without calling an external LLM.
// It implements the BackendRunner interface for testing and dry-run scenarios.
type MockRunner struct {
	Status       Status
	Verification string
	Confidence   float64
	Message      string
	Error        string
}

// Run returns a completed or failed mock result for req.
func (r MockRunner) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	if err := ctx.Err(); err != nil {
		return RunResult{}, err
	}
	if err := req.validate(); err != nil {
		return RunResult{}, err
	}

	status := r.Status
	if status == "" {
		status = StatusCompleted
	}
	if req.ForceFailure {
		status = StatusFailed
	}
	if !status.IsValid() {
		return RunResult{}, fmt.Errorf("invalid mock agent status %q", status)
	}

	result := RunResult{
		Role:         req.Role,
		Issue:        req.Issue,
		Branch:       req.Branch,
		Status:       status,
		Verification: r.Verification,
		Confidence:   r.Confidence,
		Message:      r.Message,
		Error:        r.Error,
	}

	if result.Verification == "" {
		result.Verification = defaultVerification(status)
	}
	if result.Confidence == 0 {
		result.Confidence = defaultConfidence(status)
	}
	if status == StatusFailed && result.Error == "" {
		result.Error = "mock agent forced failure"
	}
	if status == StatusCompleted && result.Message == "" {
		result.Message = "mock agent completed"
	}

	return result, nil
}

// Stream returns a single Done event with the result from Run.
// MockRunner does not support true streaming.
func (r MockRunner) Stream(ctx context.Context, req RunRequest) (<-chan RunEvent, error) {
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

// Capabilities returns MockRunner capabilities.
// MockRunner supports streaming (emulated), structured output (via Result),
// but does not use working directory.
func (r MockRunner) Capabilities() BackendCapabilities {
	return BackendCapabilities{
		Streaming:         true,
		StructuredOutput:  true,
		WorkingDirectory:  false,
	}
}

func (r Request) validate() error {
	if strings.TrimSpace(r.Role) == "" {
		return errors.New("agent request role is required")
	}
	if err := r.Issue.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.Branch) == "" {
		return errors.New("agent request branch is required")
	}
	return nil
}

func (r RunRequest) validate() error {
	if strings.TrimSpace(r.Role) == "" {
		return errors.New("agent request role is required")
	}
	if err := r.Issue.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.Branch) == "" {
		return errors.New("agent request branch is required")
	}
	return nil
}

func (i IssueRef) validate() error {
	if strings.TrimSpace(i.Repository) == "" {
		return errors.New("agent issue repository is required")
	}
	if i.Number <= 0 {
		return fmt.Errorf("agent issue number must be positive: %d", i.Number)
	}
	return nil
}

func defaultVerification(status Status) string {
	if status == StatusFailed {
		return "mock verification failed"
	}
	return "mock verification completed"
}

func defaultConfidence(status Status) float64 {
	if status == StatusFailed {
		return 0.2
	}
	return 0.95
}
