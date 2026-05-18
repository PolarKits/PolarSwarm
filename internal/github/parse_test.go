package github

import (
	"strings"
	"testing"
	"time"
)

func TestLabelParserParsesStatusLabels(t *testing.T) {
	parser := NewLabelParser()

	tests := []struct {
		name      string
		labels    []Label
		wantStatus string
	}{
		{
			name:      "status:new",
			labels:    []Label{{Name: "status:new"}},
			wantStatus: "new",
		},
		{
			name:      "status:in-progress",
			labels:    []Label{{Name: "status:in-progress"}},
			wantStatus: "in-progress",
		},
		{
			name:      "status:done",
			labels:    []Label{{Name: "status:done"}},
			wantStatus: "done",
		},
		{
			name:      "status:pending-triage",
			labels:    []Label{{Name: "status:pending-triage"}},
			wantStatus: "pending-triage",
		},
		{
			name:      "no status label",
			labels:    []Label{{Name: "area:github"}},
			wantStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.labels)
			if result.Status != tt.wantStatus {
				t.Errorf("Parse() status = %q, want %q", result.Status, tt.wantStatus)
			}
		})
	}
}

func TestLabelParserParsesAgentLabels(t *testing.T) {
	parser := NewLabelParser()

	labels := []Label{
		{Name: "status:new"},
		{Name: "agent:developer"},
		{Name: "agent:reviewer"},
		{Name: "priority:high"},
	}

	result := parser.Parse(labels)

	if len(result.Agents) != 2 {
		t.Fatalf("Parse() agents = %v, want 2 agents", result.Agents)
	}
	foundDeveloper := false
	foundReviewer := false
	for _, a := range result.Agents {
		if a == "developer" {
			foundDeveloper = true
		}
		if a == "reviewer" {
			foundReviewer = true
		}
	}
	if !foundDeveloper || !foundReviewer {
		t.Errorf("Parse() agents = %v, want developer and reviewer", result.Agents)
	}
	if result.Status != "new" {
		t.Errorf("Parse() status = %q, want new", result.Status)
	}
	if result.Priority != "high" {
		t.Errorf("Parse() priority = %q, want high", result.Priority)
	}
}

func TestLabelParserParsesDecisionLabels(t *testing.T) {
	parser := NewLabelParser()

	labels := []Label{
		{Name: "status:review"},
		{Name: "decision:approved"},
	}

	result := parser.Parse(labels)

	if result.Decision != "approved" {
		t.Errorf("Parse() decision = %q, want approved", result.Decision)
	}
}

func TestLabelParserParsesEffortAndType(t *testing.T) {
	parser := NewLabelParser()

	labels := []Label{
		{Name: "status:triaged"},
		{Name: "effort:5"},
		{Name: "type:feature"},
	}

	result := parser.Parse(labels)

	if result.Effort != "5" {
		t.Errorf("Parse() effort = %q, want 5", result.Effort)
	}
	if result.Type != "feature" {
		t.Errorf("Parse() type = %q, want feature", result.Type)
	}
}

func TestLabelParserIsTerminalStatus(t *testing.T) {
	parser := NewLabelParser()

	tests := []struct {
		status string
		want   bool
	}{
		{"done", true},
		{"abandoned", true},
		{"new", false},
		{"in-progress", false},
		{"review", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := parser.IsTerminalStatus(tt.status); got != tt.want {
				t.Errorf("IsTerminalStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestLabelParserIsBlockingStatus(t *testing.T) {
	parser := NewLabelParser()

	tests := []struct {
		status string
		want   bool
	}{
		{"blocked", true},
		{"pending-triage", true},
		{"new", false},
		{"in-progress", false},
		{"done", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := parser.IsBlockingStatus(tt.status); got != tt.want {
				t.Errorf("IsBlockingStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestCommentParserParsesCheckpoint(t *testing.T) {
	parser := NewCommentParser()

	// Build body with proper triple backticks
	open := "```polarswarm-checkpoint"
	close := "```"
	body := "[POLARSWARM:CHECKPOINT] cp:arch_review — WAITING\n\n等待架构评审人工确认。\n\n" +
		open + "\n{\n  \"checkpoint\": \"cp:arch_review\",\n  \"issue_ref\": 42,\n  \"question\": \"架构方案是否可接受？\",\n  \"options\": [\"approve\", \"approve_with_changes\", \"reject\", \"defer\"],\n  \"recommended\": \"approve\",\n  \"context_refs\": [43],\n  \"ts\": \"2026-05-17T14:00:00Z\"\n}\n" + close + "\n"

	comment := Comment{
		ID:        1,
		Author:    "polar-swarm[bot]",
		Body:      body,
		CreatedAt: time.Date(2026, 5, 17, 14, 0, 0, 0, time.UTC),
	}

	result := parser.Parse(comment)

	if result.Checkpoint == nil {
		t.Fatal("Parse() checkpoint is nil, want non-nil")
	}
	if result.Checkpoint.CheckpointID != "cp:arch_review" {
		t.Errorf("Checkpoint.CheckpointID = %q, want cp:arch_review", result.Checkpoint.CheckpointID)
	}
	if result.Checkpoint.IssueRef != 42 {
		t.Errorf("Checkpoint.IssueRef = %d, want 42", result.Checkpoint.IssueRef)
	}
	if len(result.Checkpoint.Options) != 4 {
		t.Errorf("Checkpoint.Options len = %d, want 4", len(result.Checkpoint.Options))
	}
	if result.Checkpoint.Recommended != "approve" {
		t.Errorf("Checkpoint.Recommended = %q, want approve", result.Checkpoint.Recommended)
	}
}

func TestCommentParserParsesCheckpointResponse(t *testing.T) {
	parser := NewCommentParser()

	open := "```polarswarm-checkpoint-response"
	close := "```"
	body := "[POLARSWARM:HUMAN] cp:arch_review — approve_with_changes\n\n批准，注意补充 Token 过期的边界处理文档。\n\n" +
		open + "\n{\n  \"checkpoint\": \"cp:arch_review\",\n  \"response\": \"approve_with_changes\",\n  \"note\": \"在 ADR-005 中补充 Token 过期边界场景的处理说明\",\n  \"ts\": \"2026-05-17T14:38:00Z\"\n}\n" + close + "\n"

	comment := Comment{
		ID:        2,
		Author:    "human-user",
		Body:      body,
		CreatedAt: time.Date(2026, 5, 17, 14, 38, 0, 0, time.UTC),
	}

	result := parser.Parse(comment)

	if result.CheckpointRsp == nil {
		t.Fatal("Parse() checkpointRsp is nil, want non-nil")
	}
	if result.CheckpointRsp.CheckpointID != "cp:arch_review" {
		t.Errorf("CheckpointResponse.CheckpointID = %q, want cp:arch_review", result.CheckpointRsp.CheckpointID)
	}
	if result.CheckpointRsp.Response != "approve_with_changes" {
		t.Errorf("CheckpointResponse.Response = %q, want approve_with_changes", result.CheckpointRsp.Response)
	}
}

func TestCommentParserParsesAutoDecision(t *testing.T) {
	parser := NewCommentParser()

	open := "```polarswarm-checkpoint-response"
	close := "```"
	body := "[POLARSWARM:AUTO-DECISION] cp:arch_review — approve\n\n架构方案评估通过，自动批准（全自动模式）。\n\n" +
		open + "\n{\n  \"checkpoint\": \"cp:arch_review\",\n  \"response\": \"approve\",\n  \"mode\": \"autonomous\",\n  \"from\": \"agent:orchestrator\",\n  \"confidence\": 0.91,\n  \"note\": \"方案符合项目技术规范，无安全红线，复杂度在可接受范围内\",\n  \"ts\": \"2026-05-17T14:00:05Z\"\n}\n" + close + "\n"

	comment := Comment{
		ID:        3,
		Author:    "polar-swarm[bot]",
		Body:      body,
		CreatedAt: time.Date(2026, 5, 17, 14, 0, 5, 0, time.UTC),
	}

	result := parser.Parse(comment)

	if result.CheckpointRsp == nil {
		t.Fatal("Parse() checkpointRsp is nil, want non-nil")
	}
	if result.CheckpointRsp.Mode != "autonomous" {
		t.Errorf("CheckpointResponse.Mode = %q, want autonomous", result.CheckpointRsp.Mode)
	}
	if result.CheckpointRsp.From != "agent:orchestrator" {
		t.Errorf("CheckpointResponse.From = %q, want agent:orchestrator", result.CheckpointRsp.From)
	}
	if result.CheckpointRsp.Confidence != 0.91 {
		t.Errorf("CheckpointResponse.Confidence = %f, want 0.91", result.CheckpointRsp.Confidence)
	}
}

func TestCommentParserIsCheckpoint(t *testing.T) {
	parser := NewCommentParser()

	// Note: polarswarm-checkpoint-response contains polarswarm-checkpoint as prefix
	// so IsCheckpoint returns true for both types. Use body that doesn't contain "checkpoint" at all.
	tests := []struct {
		body string
		want bool
	}{
		{"some text ```polarswarm-checkpoint\n{}", true},
		{"some text ```polarswarm-checkpoint-response\n{}", true}, // contains the prefix
		{"no checkpoint here", false},
		{"some text ```polarswarm-no-such-type\n{}", false},
	}

	for _, tt := range tests {
		t.Run(tt.body[:min(30, len(tt.body))], func(t *testing.T) {
			comment := Comment{Body: tt.body}
			if got := parser.IsCheckpoint(comment); got != tt.want {
				t.Errorf("IsCheckpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommentParserIsCheckpointResponse(t *testing.T) {
	parser := NewCommentParser()

	tests := []struct {
		body string
		want bool
	}{
		{strings.Repeat("a", 20) + "```polarswarm-checkpoint\n{}", false},
		{strings.Repeat("a", 20) + "```polarswarm-checkpoint-response\n{}", true},
		{"no checkpoint here", false},
	}

	for _, tt := range tests {
		t.Run(tt.body[:min(30, len(tt.body))], func(t *testing.T) {
			comment := Comment{Body: tt.body}
			if got := parser.IsCheckpointResponse(comment); got != tt.want {
				t.Errorf("IsCheckpointResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommentParserHandlesNonJSON(t *testing.T) {
	parser := NewCommentParser()

	body := "This is just a regular comment without any structured data."

	comment := Comment{
		ID:     99,
		Author: "random-user",
		Body:   body,
	}

	result := parser.Parse(comment)

	if result.Checkpoint != nil {
		t.Errorf("Parse() checkpoint = %v, want nil", result.Checkpoint)
	}
	if result.CheckpointRsp != nil {
		t.Errorf("Parse() checkpointRsp = %v, want nil", result.CheckpointRsp)
	}
}