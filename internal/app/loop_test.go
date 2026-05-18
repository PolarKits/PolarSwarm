package app

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/PolarKits/PolarSwarm/internal/agent"
	"github.com/PolarKits/PolarSwarm/internal/config"
	gh "github.com/PolarKits/PolarSwarm/internal/github"
	"github.com/PolarKits/PolarSwarm/internal/workflow"
)

func TestAcceptanceLoopDryRunEndToEnd(t *testing.T) {
	loop := AcceptanceLoop{
		Reader: gh.IssueReader{Client: acceptanceFakeClient()},
		Runner: agent.MockRunner{},
		Now:    time.Date(2026, 5, 19, 8, 0, 0, 0, time.UTC),
	}

	got, err := loop.Run(context.Background(), acceptanceOptions(false))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if got.Skipped {
		t.Fatalf("acceptance loop unexpectedly skipped: %s", got.SkipReason)
	}
	if got.Current != workflow.StateInProgress || got.Target != workflow.StateReview {
		t.Fatalf("unexpected workflow transition: %s -> %s", got.Current, got.Target)
	}
	if got.AgentResult.Status != agent.StatusCompleted {
		t.Fatalf("unexpected agent status: %s", got.AgentResult.Status)
	}
	kinds := make([]gh.WriteOperationKind, 0, len(got.WritePlan.Operations))
	for _, op := range got.WritePlan.Operations {
		kinds = append(kinds, op.Kind)
	}
	want := []gh.WriteOperationKind{gh.WriteCreateComment, gh.WriteRemoveLabel, gh.WriteAddLabel}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("unexpected write operation kinds: got %#v want %#v", kinds, want)
	}
	if !strings.Contains(got.WritePlan.DryRunText(), "```polarswarm-agent-result") {
		t.Fatalf("dry-run plan missing agent result block: %s", got.WritePlan.DryRunText())
	}
}

func TestAcceptanceLoopIdempotencySetSkipsRepeatedResult(t *testing.T) {
	loop := AcceptanceLoop{
		Reader: gh.IssueReader{Client: acceptanceFakeClient()},
		Runner: agent.MockRunner{},
	}

	first, err := loop.Run(context.Background(), acceptanceOptions(false))
	if err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}
	second, err := loop.Run(context.Background(), acceptanceOptions(false))
	if err != nil {
		t.Fatalf("second Run returned error: %v", err)
	}

	if first.OperationID == "" || first.OperationID != second.OperationID {
		t.Fatalf("operation id should be stable: first %q second %q", first.OperationID, second.OperationID)
	}
	if !second.Skipped || second.SkipReason != "agent result already processed" {
		t.Fatalf("second run should skip duplicate result: %#v", second)
	}
	if len(second.WritePlan.Operations) != 0 {
		t.Fatalf("duplicate run should not produce write operations: %#v", second.WritePlan.Operations)
	}
}

func TestAcceptanceLoopFailedResultDoesNotAdvanceLabel(t *testing.T) {
	loop := AcceptanceLoop{
		Reader: gh.IssueReader{Client: acceptanceFakeClient()},
		Runner: agent.MockRunner{},
	}

	got, err := loop.Run(context.Background(), acceptanceOptions(true))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if got.AgentResult.Success() {
		t.Fatal("forced failure should produce an unsuccessful agent result")
	}
	if got.Current != workflow.StateInProgress || got.Target != workflow.StateInProgress {
		t.Fatalf("failed result should stay in current state: %s -> %s", got.Current, got.Target)
	}
	if len(got.WritePlan.Operations) != 1 || got.WritePlan.Operations[0].Kind != gh.WriteCreateComment {
		t.Fatalf("failed result should only create a comment: %#v", got.WritePlan.Operations)
	}
}

func TestAcceptanceLoopSkipsIssueWithoutTargetLabel(t *testing.T) {
	client := gh.NewFakeClient()
	client.AddIssue(gh.Issue{
		Repository: gh.Repository{Owner: "PolarKits", Name: "PolarSwarm"},
		Number:     7,
		Title:      "not ready",
		State:      "open",
		Labels:     []gh.Label{{Name: "status:new"}},
	}, nil)
	loop := AcceptanceLoop{Reader: gh.IssueReader{Client: client}}

	got, err := loop.Run(context.Background(), acceptanceOptions(false))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !got.Skipped || !strings.Contains(got.SkipReason, "missing status:in-progress") {
		t.Fatalf("issue without target label should be skipped: %#v", got)
	}
}

func acceptanceOptions(forceFailure bool) AcceptanceOptions {
	return AcceptanceOptions{
		Config: config.Config{
			GitHub: config.GitHubConfig{
				Owner: "PolarKits",
				Repo:  "PolarSwarm",
			},
			Workflow: config.WorkflowConfig{
				TargetLabel: "status:in-progress",
				DryRun:      true,
			},
		},
		Repository:   gh.Repository{Owner: "PolarKits", Name: "PolarSwarm"},
		IssueNumber:  7,
		Role:         "developer",
		Branch:       "task/7",
		ForceFailure: forceFailure,
	}
}

func acceptanceFakeClient() *gh.FakeClient {
	client := gh.NewFakeClient()
	client.AddIssue(gh.Issue{
		Repository: gh.Repository{Owner: "PolarKits", Name: "PolarSwarm"},
		Number:     7,
		Title:      "M1 acceptance",
		State:      "open",
		HTMLURL:    "https://github.com/PolarKits/PolarSwarm/issues/7",
		Labels: []gh.Label{
			{Name: "status:in-progress"},
			{Name: "area:workflow"},
		},
	}, nil)
	return client
}
