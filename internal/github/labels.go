// Copyright 2025 PolarKit. All rights reserved.
// Use of this source code is governed by a MIT license

package github

import "strings"

// Label categories as defined in PolarSwarm.md section 3.2

const (
	// Status labels (mutually exclusive per Issue)
	StatusPendingTriage = "status:pending-triage"
	StatusNew           = "status:new"
	StatusTriaged       = "status:triaged"
	StatusAssigned      = "status:assigned"
	StatusInProgress    = "status:in-progress"
	StatusBlocked       = "status:blocked"
	StatusReview        = "status:review"
	StatusRework        = "status:rework"
	StatusDone          = "status:done"
	StatusAbandoned     = "status:abandoned"

	// Agent labels
	AgentOrchestrator = "agent:orchestrator"
	AgentArchitect    = "agent:architect"
	AgentDeveloper    = "agent:developer"
	AgentReviewer     = "agent:reviewer"
	AgentSecurity     = "agent:security"
	AgentDocumenter   = "agent:documenter"
	AgentDebugger     = "agent:debugger"
	AgentMerger       = "agent:merger"
	AgentTester       = "agent:tester"
	AgentDevops       = "agent:devops"
	AgentCompliance   = "agent:compliance"
	AgentHuman        = "agent:human"

	// Decision labels
	DecisionApproved  = "decision:approved"
	DecisionRejected  = "decision:rejected"
	DecisionNeedsInfo = "decision:needs-info"
)

// LabelParser extracts structured information from GitHub labels.
type LabelParser struct{}

// NewLabelParser creates a new LabelParser.
func NewLabelParser() *LabelParser {
	return &LabelParser{}
}

// ParsedLabels holds the structured label information extracted from an Issue.
type ParsedLabels struct {
	Status   string   // status:* label value (without prefix), empty if none
	Agents   []string // agent:* labels
	Decision string   // decision:* label value, empty if none
	Priority string   // priority:* label value, empty if none
	Effort   string   // effort:* label value, empty if none
	Type     string   // type:* label value, empty if none
}

// Parse extracts structured information from a list of GitHub labels.
// It follows the constraints defined in PolarSwarm.md section 3.2:
// - Only one status:* label should be present at a time
// - Multiple agent:* labels may coexist
// - Only one decision:* label should be present at a time
func (p *LabelParser) Parse(labels []Label) ParsedLabels {
	result := ParsedLabels{}

	for _, label := range labels {
		name := label.Name
		switch {
		case strings.HasPrefix(name, "status:"):
			result.Status = strings.TrimPrefix(name, "status:")
		case strings.HasPrefix(name, "agent:"):
			result.Agents = append(result.Agents, strings.TrimPrefix(name, "agent:"))
		case strings.HasPrefix(name, "decision:"):
			result.Decision = strings.TrimPrefix(name, "decision:")
		case strings.HasPrefix(name, "priority:"):
			result.Priority = strings.TrimPrefix(name, "priority:")
		case strings.HasPrefix(name, "effort:"):
			result.Effort = strings.TrimPrefix(name, "effort:")
		case strings.HasPrefix(name, "type:"):
			result.Type = strings.TrimPrefix(name, "type:")
		}
	}

	return result
}

// IsTerminalStatus returns true if the status indicates the issue is in a terminal state.
func (p *LabelParser) IsTerminalStatus(status string) bool {
	return status == "done" || status == "abandoned"
}

// IsBlockingStatus returns true if the status blocks further progress.
func (p *LabelParser) IsBlockingStatus(status string) bool {
	return status == "blocked" || status == "pending-triage"
}

// ValidStatusValues returns all valid status values per PolarSwarm.md.
func (p *LabelParser) ValidStatusValues() []string {
	return []string{
		"pending-triage", "new", "triaged", "assigned",
		"in-progress", "blocked", "review", "rework", "done", "abandoned",
	}
}

// ValidAgentValues returns all valid agent values per PolarSwarm.md.
func (p *LabelParser) ValidAgentValues() []string {
	return []string{
		"orchestrator", "architect", "developer", "reviewer",
		"security", "documenter", "debugger", "merger",
		"tester", "devops", "compliance", "human",
	}
}