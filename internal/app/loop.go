package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PolarKits/PolarSwarm/internal/agent"
	"github.com/PolarKits/PolarSwarm/internal/config"
	gh "github.com/PolarKits/PolarSwarm/internal/github"
	"github.com/PolarKits/PolarSwarm/internal/workflow"
)

type AcceptanceLoop struct {
	Reader    gh.Reader
	Runner    agent.BackendRunner
	Processed map[string]bool
	Now       time.Time
}

type AcceptanceOptions struct {
	Config       config.Config
	Repository   gh.Repository
	IssueNumber  int
	Role         string
	Branch       string
	ForceFailure bool
}

type AcceptanceResult struct {
	Issue       gh.Issue
	Current     workflow.State
	Target      workflow.State
	AgentResult agent.RunResult
	WritePlan   gh.WritePlan
	OperationID string
	Skipped     bool
	SkipReason  string
}

func (l *AcceptanceLoop) Run(ctx context.Context, opts AcceptanceOptions) (AcceptanceResult, error) {
	if l.Reader == nil {
		return AcceptanceResult{}, errors.New("acceptance loop requires an issue reader")
	}
	if l.Runner == nil {
		l.Runner = agent.MockRunner{}
	}
	if l.Processed == nil {
		l.Processed = make(map[string]bool)
	}
	if err := opts.Config.Validate("acceptance options"); err != nil {
		return AcceptanceResult{}, err
	}
	if err := opts.Repository.Validate(); err != nil {
		return AcceptanceResult{}, err
	}
	if opts.IssueNumber <= 0 {
		return AcceptanceResult{}, fmt.Errorf("issue number must be positive: %d", opts.IssueNumber)
	}
	if opts.Role == "" {
		opts.Role = "developer"
	}
	if opts.Branch == "" {
		opts.Branch = fmt.Sprintf("task/%d", opts.IssueNumber)
	}

	issue, err := l.Reader.ReadIssue(ctx, opts.Repository, opts.IssueNumber)
	if err != nil {
		return AcceptanceResult{}, err
	}

	labels := labelNames(issue.Labels)
	current, err := workflow.StateFromLabels(labels)
	if err != nil {
		return AcceptanceResult{}, err
	}
	dispatchState, err := stateFromStatusLabel(opts.Config.Workflow.TargetLabel)
	if err != nil {
		return AcceptanceResult{}, err
	}
	if !hasLabel(labels, opts.Config.Workflow.TargetLabel) {
		return AcceptanceResult{
			Issue:      issue,
			Current:    current,
			Skipped:    true,
			SkipReason: fmt.Sprintf("issue is not dispatchable: missing %s", opts.Config.Workflow.TargetLabel),
		}, nil
	}
	if current != dispatchState {
		return AcceptanceResult{}, fmt.Errorf("target_label %q does not match current workflow state %q", opts.Config.Workflow.TargetLabel, current)
	}

	result, err := l.Runner.Run(ctx, agent.RunRequest{
		Role: opts.Role,
		Issue: agent.IssueRef{
			Repository: opts.Repository.String(),
			Number:     issue.Number,
			Title:      issue.Title,
			URL:        issue.HTMLURL,
		},
		Branch:       opts.Branch,
		ForceFailure: opts.ForceFailure,
	})
	if err != nil {
		return AcceptanceResult{}, err
	}

	target := current
	if result.Success() {
		target, err = nextWorkflowState(current)
		if err != nil {
			return AcceptanceResult{}, err
		}
	}
	opID := acceptanceOperationID(result, target)
	if l.Processed[opID] {
		return AcceptanceResult{
			Issue:       issue,
			Current:     current,
			Target:      target,
			AgentResult: result,
			OperationID: opID,
			Skipped:     true,
			SkipReason:  "agent result already processed",
		}, nil
	}

	plan, err := gh.PlanAgentResultWrite(result, labels, target, gh.WriteOptions{
		DryRun:       opts.Config.Workflow.DryRun,
		CurrentState: current,
		Now:          l.Now,
	})
	if err != nil {
		return AcceptanceResult{}, err
	}
	l.Processed[opID] = true

	return AcceptanceResult{
		Issue:       issue,
		Current:     current,
		Target:      target,
		AgentResult: result,
		WritePlan:   plan,
		OperationID: opID,
	}, nil
}

func acceptanceOperationID(result agent.RunResult, target workflow.State) string {
	return fmt.Sprintf("%s#%d:%s:%s:%s:%s", result.Issue.Repository, result.Issue.Number, result.Role, result.Branch, result.Status, target)
}

func stateFromStatusLabel(label string) (workflow.State, error) {
	stateName, ok := strings.CutPrefix(label, "status:")
	if !ok || stateName == "" {
		return "", fmt.Errorf("target_label must be a status label: %q", label)
	}
	state := workflow.State(stateName)
	if !state.IsValid() {
		return "", fmt.Errorf("target_label has invalid workflow state: %q", label)
	}
	return state, nil
}

func nextWorkflowState(current workflow.State) (workflow.State, error) {
	switch current {
	case workflow.StateNew:
		return workflow.StateAssigned, nil
	case workflow.StateAssigned:
		return workflow.StateInProgress, nil
	case workflow.StateInProgress:
		return workflow.StateReview, nil
	case workflow.StateReview:
		return workflow.StateDone, nil
	default:
		return "", fmt.Errorf("workflow state %q cannot be advanced by the M1 loop", current)
	}
}

func hasLabel(labels []string, want string) bool {
	for _, label := range labels {
		if label == want {
			return true
		}
	}
	return false
}
