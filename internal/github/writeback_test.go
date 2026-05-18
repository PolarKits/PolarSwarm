package github

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/PolarKits/PolarSwarm/internal/agent"
	"github.com/PolarKits/PolarSwarm/internal/workflow"
)

func TestPlanAgentResultWriteDryRun(t *testing.T) {
	result := testAgentResult()
	plan, err := PlanAgentResultWrite(result, []string{"area:github", "status:in-progress"}, workflow.StateReview, WriteOptions{
		DryRun: true,
		Now:    time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("PlanAgentResultWrite returned error: %v", err)
	}

	if !plan.DryRun {
		t.Fatal("expected dry-run plan")
	}
	gotKinds := operationKinds(plan.Operations)
	wantKinds := []WriteOperationKind{WriteCreateComment, WriteRemoveLabel, WriteAddLabel}
	if !reflect.DeepEqual(gotKinds, wantKinds) {
		t.Fatalf("unexpected operation order: got %#v want %#v", gotKinds, wantKinds)
	}
	if plan.Operations[1].Label != "status:in-progress" || plan.Operations[2].Label != "status:review" {
		t.Fatalf("unexpected label projection operations: %#v", plan.Operations)
	}
	if !strings.Contains(plan.DryRunText(), "```polarswarm-agent-result") {
		t.Fatalf("dry-run output missing agent result block: %s", plan.DryRunText())
	}
}

func TestPlanAgentResultWriteCanCleanAmbiguousStatusLabelsWithExplicitCurrentState(t *testing.T) {
	plan, err := PlanAgentResultWrite(testAgentResult(), []string{"status:assigned", "status:in-progress", "area:github"}, workflow.StateReview, WriteOptions{
		DryRun:       true,
		CurrentState: workflow.StateInProgress,
	})
	if err != nil {
		t.Fatalf("PlanAgentResultWrite returned error: %v", err)
	}

	got := []string{plan.Operations[1].Label, plan.Operations[2].Label, plan.Operations[3].Label}
	want := []string{"status:assigned", "status:in-progress", "status:review"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected label cleanup: got %#v want %#v", got, want)
	}
}

func TestPlanAgentResultWriteRejectsUnconfirmedNonDryRun(t *testing.T) {
	_, err := PlanAgentResultWrite(testAgentResult(), []string{"status:in-progress"}, workflow.StateReview, WriteOptions{})
	if err == nil {
		t.Fatal("expected unconfirmed non-dry-run error")
	}
	if !strings.Contains(err.Error(), "ConfirmWrites") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanAgentResultWriteAllowsConfirmedNonDryRun(t *testing.T) {
	plan, err := PlanAgentResultWrite(testAgentResult(), []string{"status:in-progress"}, workflow.StateReview, WriteOptions{
		ConfirmWrites: true,
	})
	if err != nil {
		t.Fatalf("PlanAgentResultWrite returned error: %v", err)
	}
	if plan.DryRun {
		t.Fatal("confirmed non-dry-run plan should not be marked dry-run")
	}
}

func TestPlanAgentResultWriteRejectsIllegalTransition(t *testing.T) {
	_, err := PlanAgentResultWrite(testAgentResult(), []string{"status:in-progress"}, workflow.StateDone, WriteOptions{
		DryRun: true,
	})
	if err == nil {
		t.Fatal("expected illegal transition error")
	}
	if !strings.Contains(err.Error(), "illegal workflow transition") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenderAgentResultCommentStableJSON(t *testing.T) {
	body, err := RenderAgentResultComment(testAgentResult(), time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("RenderAgentResultComment returned error: %v", err)
	}

	payload := extractAgentResultPayload(t, body)
	for _, key := range []string{"version", "type", "role", "issue", "branch", "status", "verification", "confidence", "ts"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("payload missing key %q: %#v", key, payload)
		}
	}
	if payload["role"] != "developer" || payload["status"] != "completed" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	issue, ok := payload["issue"].(map[string]any)
	if !ok || issue["repository"] != "PolarKits/PolarSwarm" || issue["number"].(float64) != 6 {
		t.Fatalf("unexpected issue payload: %#v", payload["issue"])
	}
}

func TestWritePlanExecuteRequiresConfirmWrites(t *testing.T) {
	plan, err := PlanAgentResultWrite(testAgentResult(), []string{"status:in-progress"}, workflow.StateReview, WriteOptions{
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("PlanAgentResultWrite returned error: %v", err)
	}
	plan.DryRun = false

	err = plan.Execute(context.Background(), &recordingWriter{}, WriteOptions{})
	if err == nil {
		t.Fatal("expected unconfirmed execute error")
	}
}

func TestWritePlanExecuteOrder(t *testing.T) {
	plan, err := PlanAgentResultWrite(testAgentResult(), []string{"status:assigned"}, workflow.StateInProgress, WriteOptions{
		ConfirmWrites: true,
	})
	if err != nil {
		t.Fatalf("PlanAgentResultWrite returned error: %v", err)
	}
	writer := &recordingWriter{}

	if err := plan.Execute(context.Background(), writer, WriteOptions{ConfirmWrites: true}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	want := []string{"comment", "remove:status:assigned", "add:status:in-progress"}
	if !reflect.DeepEqual(writer.calls, want) {
		t.Fatalf("unexpected write order: got %#v want %#v", writer.calls, want)
	}
}

func testAgentResult() agent.Result {
	return agent.Result{
		Role: "developer",
		Issue: agent.IssueRef{
			Repository: "PolarKits/PolarSwarm",
			Number:     6,
			Title:      "T006",
		},
		Branch:       "task/6",
		Status:       agent.StatusCompleted,
		Verification: "go test ./...",
		Confidence:   0.95,
		Message:      "mock agent completed",
	}
}

func operationKinds(ops []WriteOperation) []WriteOperationKind {
	kinds := make([]WriteOperationKind, 0, len(ops))
	for _, op := range ops {
		kinds = append(kinds, op.Kind)
	}
	return kinds
}

func extractAgentResultPayload(t *testing.T, body string) map[string]any {
	t.Helper()
	start := strings.Index(body, "```polarswarm-agent-result\n")
	if start < 0 {
		t.Fatalf("missing agent result block: %s", body)
	}
	start += len("```polarswarm-agent-result\n")
	end := strings.Index(body[start:], "\n```")
	if end < 0 {
		t.Fatalf("missing agent result block terminator: %s", body)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(body[start:start+end]), &payload); err != nil {
		t.Fatalf("payload is not stable JSON: %v\n%s", err, body[start:start+end])
	}
	return payload
}

type recordingWriter struct {
	calls []string
}

func (w *recordingWriter) CreateIssueComment(ctx context.Context, repository string, issue int, body string) error {
	w.calls = append(w.calls, "comment")
	return nil
}

func (w *recordingWriter) RemoveIssueLabel(ctx context.Context, repository string, issue int, label string) error {
	w.calls = append(w.calls, "remove:"+label)
	return nil
}

func (w *recordingWriter) AddIssueLabel(ctx context.Context, repository string, issue int, label string) error {
	w.calls = append(w.calls, "add:"+label)
	return nil
}
