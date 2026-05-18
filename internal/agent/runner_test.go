package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/PolarKits/PolarSwarm/internal/workflow"
)

func TestMockRunnerCompleted(t *testing.T) {
	result, err := (MockRunner{}).Run(context.Background(), testRequest())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if result.Status != StatusCompleted {
		t.Fatalf("unexpected status: got %q want %q", result.Status, StatusCompleted)
	}
	if !result.Success() {
		t.Fatal("completed result should be successful")
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("completed result should validate: %v", err)
	}
}

func TestMockRunnerFailedByConfiguration(t *testing.T) {
	result, err := (MockRunner{
		Status:       StatusFailed,
		Verification: "unit verification failed",
		Confidence:   0.4,
		Error:        "configured failure",
	}).Run(context.Background(), testRequest())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if result.Status != StatusFailed {
		t.Fatalf("unexpected status: got %q want %q", result.Status, StatusFailed)
	}
	if result.Success() {
		t.Fatal("failed result must not be successful")
	}
	if result.Error == "" {
		t.Fatal("failed result should include an error message")
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("failed result should validate: %v", err)
	}
}

func TestMockRunnerFailedByRequestFlag(t *testing.T) {
	req := testRequest()
	req.ForceFailure = true

	result, err := (MockRunner{}).Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if result.Status != StatusFailed {
		t.Fatalf("unexpected status: got %q want %q", result.Status, StatusFailed)
	}
	if result.Success() {
		t.Fatal("forced failed result must not be successful")
	}
}

func TestResultRequiredFields(t *testing.T) {
	result, err := (MockRunner{}).Run(context.Background(), testRequest())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	for _, want := range []string{result.Role, result.Issue.Repository, result.Branch, string(result.Status), result.Verification} {
		if strings.TrimSpace(want) == "" {
			t.Fatalf("required string field is empty in result: %#v", result)
		}
	}
	if result.Issue.Number <= 0 {
		t.Fatalf("issue number should be positive: %#v", result.Issue)
	}
	if result.Confidence <= 0 || result.Confidence > 1 {
		t.Fatalf("confidence should be in (0,1]: %v", result.Confidence)
	}

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

func TestResultValidateRejectsMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		result  Result
		wantErr string
	}{
		{name: "role", result: validResult(func(r *Result) { r.Role = "" }), wantErr: "role is required"},
		{name: "issue repository", result: validResult(func(r *Result) { r.Issue.Repository = "" }), wantErr: "repository is required"},
		{name: "issue number", result: validResult(func(r *Result) { r.Issue.Number = 0 }), wantErr: "number must be positive"},
		{name: "branch", result: validResult(func(r *Result) { r.Branch = "" }), wantErr: "branch is required"},
		{name: "status", result: validResult(func(r *Result) { r.Status = Status("done") }), wantErr: "invalid agent result status"},
		{name: "verification", result: validResult(func(r *Result) { r.Verification = "" }), wantErr: "verification is required"},
		{name: "confidence", result: validResult(func(r *Result) { r.Confidence = 1.5 }), wantErr: "confidence must be between 0 and 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestFailedResultDoesNotAdvanceWorkflowToDone(t *testing.T) {
	result, err := (MockRunner{Status: StatusFailed}).Run(context.Background(), testRequest())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	_, err = workflow.TransitionAfterResult(workflow.StateReview, workflow.StateDone, result.Success())
	if err == nil {
		t.Fatal("expected workflow to reject done transition after failed result")
	}
	if !strings.Contains(err.Error(), "unsuccessful result") {
		t.Fatalf("unexpected transition error: %v", err)
	}
}

func testRequest() Request {
	return Request{
		Role:   "developer",
		Issue:  IssueRef{Repository: "PolarKits/PolarSwarm", Number: 5, Title: "Mock Agent Runner"},
		Branch: "agent/mock-runner",
	}
}

func validResult(mutators ...func(*Result)) Result {
	result := Result{
		Role:         "developer",
		Issue:        IssueRef{Repository: "PolarKits/PolarSwarm", Number: 5},
		Branch:       "agent/mock-runner",
		Status:       StatusCompleted,
		Verification: "unit verification completed",
		Confidence:   0.95,
	}
	for _, mutate := range mutators {
		mutate(&result)
	}
	return result
}
