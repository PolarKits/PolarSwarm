package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PolarKits/PolarSwarm/internal/agent"
	"github.com/PolarKits/PolarSwarm/internal/workflow"
)

const (
	AgentResultBlockType = "polarswarm-agent-result"
	agentResultVersion   = "1"
	agentResultType      = "agent_result"
)

// maskedReplacement is the replacement string for matched secrets.
const maskedReplacement = "***"

// MaskSecrets replaces known secret patterns in the input string with "***".
// This prevents accidental secret leakage in logs, comments, and audit records.
// The function scans for common secret formats including Bearer tokens,
// API keys (including sk-ant-* Anthropic keys), GitHub tokens, and generic
// credentials, replacing them with "***". Multiple occurrences are all masked.
func MaskSecrets(input string) string {
	if input == "" {
		return input
	}
	result := input
	// Prefixes for common secret patterns. GitHub tokens use specific prefixes
	// (ghp_, ghs_, gho_, ghu_, gh_). Bearer tokens use "bearer " with JWT-like content.
	// API keys may have various formats including sk-ant-* for Anthropic.
	prefixes := []string{
		"bearer ", "api_key=", "api-key=", "token=", "secret=", "password=",
		"ghp_", "ghs_", "gho_", "ghu_", "gh_",
		"sk-ant-",
	}
	// Characters that indicate end of a secret value.
	// Includes standard terminators plus dots and dashes common in JWTs/base64 tokens.
	secretTerminators := " \n\"',"
	// Loop until no more secrets found (handles multiple tokens in one string)
	changed := true
	for changed {
		changed = false
		for _, prefix := range prefixes {
			lower := strings.ToLower(result)
			idx := strings.Index(lower, prefix)
			if idx >= 0 {
				end := idx + len(prefix)
				// Find end of the secret
				for end < len(result) && !strings.Contains(secretTerminators, string(result[end])) {
					end++
				}
				result = result[:idx] + maskedReplacement + result[end:]
				changed = true
				break // restart scanning from beginning after each replacement
			}
		}
	}
	return result
}

type WriteOptions struct {
	DryRun        bool
	ConfirmWrites bool
	CurrentState  workflow.State
	Now           time.Time
}

type WriteOperationKind string

const (
	WriteCreateComment WriteOperationKind = "comment.create"
	WriteRemoveLabel   WriteOperationKind = "labels.remove"
	WriteAddLabel      WriteOperationKind = "labels.add"
)

type WriteOperation struct {
	Kind  WriteOperationKind `json:"kind"`
	Label string             `json:"label,omitempty"`
	Body  string             `json:"body,omitempty"`
}

type WritePlan struct {
	Repository string           `json:"repository"`
	Issue      int              `json:"issue"`
	DryRun     bool             `json:"dry_run"`
	OpID       string           `json:"op_id,omitempty"` // Idempotency key: op:{repo}:{kind}:{target}:{attempt}
	Operations []WriteOperation `json:"operations"`
}

type IssueWriter interface {
	CreateIssueComment(ctx context.Context, repository string, issue int, body string) error
	RemoveIssueLabel(ctx context.Context, repository string, issue int, label string) error
	AddIssueLabel(ctx context.Context, repository string, issue int, label string) error
}

type AgentResultComment struct {
	Version      string         `json:"version"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Issue        agent.IssueRef `json:"issue"`
	Branch       string         `json:"branch"`
	Status       agent.Status   `json:"status"`
	Verification string         `json:"verification"`
	Confidence   float64        `json:"confidence"`
	Message      string         `json:"message,omitempty"`
	Error        string         `json:"error,omitempty"`
	BackendID    string         `json:"backend_id,omitempty"`
	ModelID      string         `json:"model_id,omitempty"`
	DurationMs   int64          `json:"duration_ms,omitempty"`
	TS           string         `json:"ts"`
}

func PlanAgentResultWrite(result agent.Result, existingLabels []string, target workflow.State, opts WriteOptions) (WritePlan, error) {
	if err := result.Validate(); err != nil {
		return WritePlan{}, err
	}
	if !opts.DryRun && !opts.ConfirmWrites {
		return WritePlan{}, errors.New("refuse non-dry-run write plan without ConfirmWrites")
	}
	current := opts.CurrentState
	if current == "" {
		var err error
		current, err = workflow.StateFromLabels(existingLabels)
		if err != nil {
			return WritePlan{}, err
		}
	}
	if _, err := workflow.TransitionAfterResult(current, target, result.Success()); err != nil {
		return WritePlan{}, err
	}

	projection, err := workflow.ProjectStatusLabels(existingLabels, target)
	if err != nil {
		return WritePlan{}, err
	}

	body, err := RenderAgentResultComment(result, opts.Now)
	if err != nil {
		return WritePlan{}, err
	}

	ops := []WriteOperation{{Kind: WriteCreateComment, Body: body}}
	for _, label := range projection.Remove {
		ops = append(ops, WriteOperation{Kind: WriteRemoveLabel, Label: label})
	}
	for _, label := range projection.Add {
		ops = append(ops, WriteOperation{Kind: WriteAddLabel, Label: label})
	}

	return WritePlan{
		Repository: result.Issue.Repository,
		Issue:      result.Issue.Number,
		DryRun:     opts.DryRun,
		Operations: ops,
	}, nil
}

func RenderAgentResultComment(result agent.Result, now time.Time) (string, error) {
	if err := result.Validate(); err != nil {
		return "", err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	payload := AgentResultComment{
		Version:      agentResultVersion,
		Type:         agentResultType,
		Role:         result.Role,
		Issue:        result.Issue,
		Branch:       result.Branch,
		Status:       result.Status,
		Verification: result.Verification,
		Confidence:   result.Confidence,
		Message:      result.Message,
		Error:        MaskSecrets(result.Error),
		BackendID:    result.BackendID,
		ModelID:      result.ModelID,
		DurationMs:   result.DurationMs,
		TS:           now.UTC().Format(time.RFC3339),
	}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal agent result comment: %w", err)
	}

	return fmt.Sprintf("[POLARSWARM:AGENT-RESULT] %s - %s\n\n```%s\n%s\n```\n",
		result.Role, result.Status, AgentResultBlockType, content), nil
}

// BuildOpID creates an idempotency key for a write operation.
// Format: op:{issue_ref}:{operation}:{target}:{attempt}
// Corresponds to PolarSwarm.md idempotency design in section on "幂等重试".
func BuildOpID(repo string, issue int, kind WriteOperationKind, target string, attempt int) string {
	return fmt.Sprintf("op:%s/%d:%s:%s:%d", repo, issue, kind, target, attempt)
}

func (p WritePlan) Execute(ctx context.Context, writer IssueWriter, opts WriteOptions) error {
	if writer == nil {
		return errors.New("write plan requires a writer")
	}
	if p.Repository == "" {
		return errors.New("write plan repository is required")
	}
	if p.Issue <= 0 {
		return fmt.Errorf("write plan issue must be positive: %d", p.Issue)
	}
	if opts.DryRun || p.DryRun {
		return nil
	}
	if !opts.ConfirmWrites {
		return errors.New("refuse executing write plan without ConfirmWrites")
	}

	for _, op := range p.Operations {
		switch op.Kind {
		case WriteCreateComment:
			if err := writer.CreateIssueComment(ctx, p.Repository, p.Issue, op.Body); err != nil {
				return err
			}
		case WriteRemoveLabel:
			if err := writer.RemoveIssueLabel(ctx, p.Repository, p.Issue, op.Label); err != nil {
				return err
			}
		case WriteAddLabel:
			if err := writer.AddIssueLabel(ctx, p.Repository, p.Issue, op.Label); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown write operation kind %q", op.Kind)
		}
	}
	return nil
}

func (p WritePlan) DryRunText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "dry-run write plan for %s#%d\n", p.Repository, p.Issue)
	for _, op := range p.Operations {
		switch op.Kind {
		case WriteCreateComment:
			fmt.Fprintf(&b, "- create comment:\n%s", op.Body)
		case WriteRemoveLabel:
			fmt.Fprintf(&b, "- remove label: %s\n", op.Label)
		case WriteAddLabel:
			fmt.Fprintf(&b, "- add label: %s\n", op.Label)
		default:
			fmt.Fprintf(&b, "- %s\n", op.Kind)
		}
	}
	return b.String()
}
