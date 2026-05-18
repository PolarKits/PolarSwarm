package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Runner executes an Agent request and returns a stable result payload.
type Runner interface {
	Run(ctx context.Context, req Request) (Result, error)
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

// Request describes the work assigned to an Agent.
type Request struct {
	Role         string   `json:"role"`
	Issue        IssueRef `json:"issue"`
	Branch       string   `json:"branch"`
	ForceFailure bool     `json:"force_failure,omitempty"`
}

// Result is the payload T006 can render as polarswarm-agent-result.
type Result struct {
	Role         string   `json:"role"`
	Issue        IssueRef `json:"issue"`
	Branch       string   `json:"branch"`
	Status       Status   `json:"status"`
	Verification string   `json:"verification"`
	Confidence   float64  `json:"confidence"`
	Message      string   `json:"message,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// Success reports whether the result may be treated as a successful Agent run.
func (r Result) Success() bool {
	return r.Status == StatusCompleted
}

// Validate checks that a Result contains the required stable fields.
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
type MockRunner struct {
	Status       Status
	Verification string
	Confidence   float64
	Message      string
	Error        string
}

// Run returns a completed or failed mock result for req.
func (r MockRunner) Run(ctx context.Context, req Request) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := req.validate(); err != nil {
		return Result{}, err
	}

	status := r.Status
	if status == "" {
		status = StatusCompleted
	}
	if req.ForceFailure {
		status = StatusFailed
	}
	if !status.IsValid() {
		return Result{}, fmt.Errorf("invalid mock agent status %q", status)
	}

	result := Result{
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
