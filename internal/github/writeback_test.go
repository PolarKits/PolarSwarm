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

func TestBuildOpIDFormat(t *testing.T) {
	id := BuildOpID("PolarKits/PolarSwarm", 6, WriteCreateComment, "body", 1)
	if !strings.HasPrefix(id, "op:") {
		t.Fatalf("expected op: prefix, got: %s", id)
	}
	if !strings.Contains(id, "PolarKits/PolarSwarm") {
		t.Fatalf("expected repo in op_id, got: %s", id)
	}
	if !strings.Contains(id, "comment.create") {
		t.Fatalf("expected operation kind in op_id, got: %s", id)
	}
	if !strings.HasSuffix(id, ":1") {
		t.Fatalf("expected attempt suffix :1, got: %s", id)
	}
}

func TestBuildOpIDUniqueness(t *testing.T) {
	id1 := BuildOpID("repo", 1, WriteCreateComment, "target", 1)
	id2 := BuildOpID("repo", 1, WriteRemoveLabel, "target", 1)
	id3 := BuildOpID("repo", 1, WriteCreateComment, "target", 2)
	if id1 == id2 || id1 == id3 || id2 == id3 {
		t.Fatalf("expected unique op_ids, got: %s, %s, %s", id1, id2, id3)
	}
}

func TestWritePlanOpIDField(t *testing.T) {
	result := testAgentResult()
	plan, err := PlanAgentResultWrite(result, []string{"status:in-progress"}, workflow.StateReview, WriteOptions{
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("PlanAgentResultWrite returned error: %v", err)
	}

	// Plan should not auto-populate OpID; caller sets it based on attempt count
	if plan.OpID != "" {
		t.Fatalf("expected empty OpID on fresh plan, got: %s", plan.OpID)
	}
}

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

func testAgentResult() agent.RunResult {
	return agent.RunResult{
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

func TestMaskSecretsBearerToken(t *testing.T) {
	input := `Error: failed to call API: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c`
	result := MaskSecrets(input)
	if strings.Contains(result, "eyJ") {
		t.Fatalf("Bearer token was not masked: %s", result)
	}
	if !strings.Contains(result, "***") {
		t.Fatalf("masked replacement not found: %s", result)
	}
}

func TestMaskSecretsGitHubToken(t *testing.T) {
	input := `GitHub API call failed: ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`
	result := MaskSecrets(input)
	if strings.Contains(result, "ghp_") {
		t.Fatalf("GitHub token was not masked: %s", result)
	}
}

func TestMaskSecretsAPIKey(t *testing.T) {
	input := `API key is: sk-ant-api03-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`
	result := MaskSecrets(input)
	if strings.Contains(result, "sk-ant-api03-") {
		t.Fatalf("API key was not masked: %s", result)
	}
}

func TestMaskSecretsPassword(t *testing.T) {
	// password= is a prefix we detect
	input := `connection string: password=secretpassword123 is wrong`
	result := MaskSecrets(input)
	if strings.Contains(result, "secretpassword123") {
		t.Fatalf("password was not masked: %s", result)
	}
	if !strings.Contains(result, "***") {
		t.Fatalf("masked replacement not found: %s", result)
	}
}

func TestMaskSecretsEmptyInput(t *testing.T) {
	result := MaskSecrets("")
	if result != "" {
		t.Fatalf("expected empty string, got: %s", result)
	}
}

func TestMaskSecretsNoSecrets(t *testing.T) {
	input := `This is a normal log message without any secrets here`
	result := MaskSecrets(input)
	if result != input {
		t.Fatalf("expected unchanged input, got: %s", result)
	}
}

func TestMaskSecretsMultipleOccurrences(t *testing.T) {
	input := `First token: Bearer abc123.def456.ghi789 Second token: Bearer xyz123.uvw456.rst789`
	result := MaskSecrets(input)
	if strings.Contains(result, "abc123") || strings.Contains(result, "xyz123") {
		t.Fatalf("tokens were not fully masked: %s", result)
	}
}

func TestAgentResultCommentIncludesBackendID(t *testing.T) {
	result := testAgentResult()
	result.BackendID = "opencode"
	result.ModelID = "claude-sonnet-4-6"
	result.DurationMs = 45000

	body, err := RenderAgentResultComment(result, time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("RenderAgentResultComment returned error: %v", err)
	}

	payload := extractAgentResultPayload(t, body)
	if payload["backend_id"] != "opencode" {
		t.Fatalf("expected backend_id 'opencode', got: %#v", payload["backend_id"])
	}
	if payload["model_id"] != "claude-sonnet-4-6" {
		t.Fatalf("expected model_id 'claude-sonnet-4-6', got: %#v", payload["model_id"])
	}
	if int64(payload["duration_ms"].(float64)) != 45000 {
		t.Fatalf("expected duration_ms 45000, got: %#v", payload["duration_ms"])
	}
}

func TestAgentResultCommentIncludesDurationMs(t *testing.T) {
	result := testAgentResult()
	result.DurationMs = 12345

	body, err := RenderAgentResultComment(result, time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("RenderAgentResultComment returned error: %v", err)
	}

	payload := extractAgentResultPayload(t, body)
	if int64(payload["duration_ms"].(float64)) != 12345 {
		t.Fatalf("expected duration_ms 12345, got: %#v", payload["duration_ms"])
	}
}

func TestAgentResultCommentErrorMasking(t *testing.T) {
	result := testAgentResult()
	result.Error = `API call failed: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.sig`

	body, err := RenderAgentResultComment(result, time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("RenderAgentResultComment returned error: %v", err)
	}

	payload := extractAgentResultPayload(t, body)
	errorVal, ok := payload["error"].(string)
	if !ok {
		t.Fatalf("error field is not a string: %#v", payload["error"])
	}
	if strings.Contains(errorVal, "eyJ") {
		t.Fatalf("secret was not masked in error field: %s", errorVal)
	}
}
