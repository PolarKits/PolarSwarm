// Copyright 2025 PolarKit. All rights reserved.
// Use of this source code is governed by a MIT license

package github

import (
	"encoding/json"
	"strings"
	"time"
)

// CommentParser extracts structured information from GitHub comments.
type CommentParser struct{}

// NewCommentParser creates a new CommentParser.
func NewCommentParser() *CommentParser {
	return &CommentParser{}
}

// CheckpointType represents the type of checkpoint message.
type CheckpointType string

const (
	CheckpointTypeRequest  CheckpointType = "polarswarm-checkpoint"
	CheckpointTypeResponse CheckpointType = "polarswarm-checkpoint-response"
)

// Checkpoint represents a parsed polarswarm-checkpoint comment.
type Checkpoint struct {
	CheckpointID string    `json:"checkpoint"`
	IssueRef     int       `json:"issue_ref,omitempty"`
	Question     string    `json:"question,omitempty"`
	Options      []string  `json:"options,omitempty"`
	Recommended  string    `json:"recommended,omitempty"`
	ContextRefs  []int     `json:"context_refs,omitempty"`
	Timestamp    time.Time `json:"ts"`
}

// CheckpointResponse represents a parsed polarswarm-checkpoint-response comment.
type CheckpointResponse struct {
	CheckpointID string    `json:"checkpoint"`
	Response     string    `json:"response"`
	Mode         string    `json:"mode,omitempty"`
	From         string    `json:"from,omitempty"`
	Confidence   float64   `json:"confidence,omitempty"`
	Note         string    `json:"note,omitempty"`
	Timestamp    time.Time `json:"ts"`
}

// ParsedComment holds all extracted structured information from a comment.
type ParsedComment struct {
	Checkpoint    *Checkpoint
	CheckpointRsp *CheckpointResponse
	RawJSON       string
}

// Parse extracts structured information from a GitHub comment body.
// It recognizes polarswarm-checkpoint and polarswarm-checkpoint-response JSON blocks.
func (p *CommentParser) Parse(comment Comment) ParsedComment {
	result := ParsedComment{}
	body := comment.Body

	// Try to extract polarswarm-checkpoint block
	if idx := strings.Index(body, "```polarswarm-checkpoint"); idx >= 0 {
		result.Checkpoint = p.parseCheckpoint(comment.Body, idx)
	}

	// Try to extract polarswarm-checkpoint-response block
	if idx := strings.Index(body, "```polarswarm-checkpoint-response"); idx >= 0 {
		result.CheckpointRsp = p.parseCheckpointResponse(comment.Body, idx)
	}

	return result
}

// parseCheckpoint extracts and parses a polarswarm-checkpoint JSON block.
func (p *CommentParser) parseCheckpoint(body string, startIdx int) *Checkpoint {
	// Find the opening brace after the code fence
	codeStart := strings.Index(body[startIdx:], "{")
	if codeStart < 0 {
		return nil
	}
	blockquoteStart := startIdx + codeStart

	// Find the closing code fence
	codeEnd := strings.Index(body[blockquoteStart:], "```")
	if codeEnd < 0 {
		return nil
	}
	jsonEnd := blockquoteStart + codeEnd

	jsonStr := strings.TrimSpace(body[blockquoteStart:jsonEnd])
	if strings.HasPrefix(jsonStr, "```") {
		// Strip the ```polarswarm-checkpoint line if present
		lines := strings.SplitN(jsonStr, "\n", 2)
		if len(lines) > 1 {
			jsonStr = strings.TrimSpace(lines[1])
		}
	}

	var cp Checkpoint
	if err := json.Unmarshal([]byte(jsonStr), &cp); err != nil {
		return nil
	}
	return &cp
}

// parseCheckpointResponse extracts and parses a polarswarm-checkpoint-response JSON block.
func (p *CommentParser) parseCheckpointResponse(body string, startIdx int) *CheckpointResponse {
	// Find the opening brace after the code fence
	codeStart := strings.Index(body[startIdx:], "{")
	if codeStart < 0 {
		return nil
	}
	blockquoteStart := startIdx + codeStart

	// Find the closing code fence
	codeEnd := strings.Index(body[blockquoteStart:], "```")
	if codeEnd < 0 {
		return nil
	}
	jsonEnd := blockquoteStart + codeEnd

	jsonStr := strings.TrimSpace(body[blockquoteStart:jsonEnd])
	if strings.HasPrefix(jsonStr, "```") {
		// Strip the ```polarswarm-checkpoint-response line if present
		lines := strings.SplitN(jsonStr, "\n", 2)
		if len(lines) > 1 {
			jsonStr = strings.TrimSpace(lines[1])
		}
	}

	var rsp CheckpointResponse
	if err := json.Unmarshal([]byte(jsonStr), &rsp); err != nil {
		return nil
	}
	return &rsp
}

// IsCheckpoint returns true if the comment contains a checkpoint request.
func (p *CommentParser) IsCheckpoint(comment Comment) bool {
	return strings.Contains(comment.Body, "```polarswarm-checkpoint")
}

// IsCheckpointResponse returns true if the comment contains a checkpoint response.
func (p *CommentParser) IsCheckpointResponse(comment Comment) bool {
	return strings.Contains(comment.Body, "```polarswarm-checkpoint-response")
}