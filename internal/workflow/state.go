package workflow

import (
	"errors"
	"fmt"
	"strings"
)

// statusLabelPrefix is the label prefix for all workflow status labels.
const statusLabelPrefix = "status:"

// State is the M1 issue workflow state.
// Corresponds to PolarSwarm.md Section 3.2 Issue Label State Machine.
type State string

// Workflow states as defined in PolarSwarm.md 3.2.
const (
	StatePendingTriage State = "pending-triage" // External submission, awaiting authorization
	StateNew           State = "new"             // Passed gate, awaiting Orchestrator review
	StateTriaged       State = "triaged"        // Reviewed, entering Backlog
	StateAssigned      State = "assigned"       // Assigned to Agent, awaiting start
	StateInProgress    State = "in-progress"     // Agent executing
	StateBlocked       State = "blocked"         // Waiting for dependency resolution
	StateReview        State = "review"         // Execution complete, awaiting review
	StateRework        State = "rework"         // Review rejected, rework assigned
	StateDone          State = "done"           // Completed and accepted
	StateAbandoned     State = "abandoned"      // Cancelled / will not fix
)

// nextStates defines valid forward transitions in the workflow.
// Corresponds to the state machine diagram in PolarSwarm.md 3.2:
//
//	[pending-triage] → [new] → [triaged] → [assigned] → [in-progress] → [review] → [done]
//	（外部提交）                                               ↓          ↓      ↓
//	                                                      [blocked]  [rework] [abandoned]
//	                                                          ↓          ↓
//	                                                      [assigned] [in-progress]（返工执行）
// then back to review for re-review)
var nextStates = map[State]State{
	StatePendingTriage: StateNew,
	StateNew:           StateTriaged,
	StateTriaged:       StateAssigned,
	StateAssigned:      StateInProgress,
	StateInProgress:    StateReview,
	StateReview:        StateDone,
	// Blocked path: blocked → assigned (rework execution)
	StateBlocked: StateAssigned,
	// Rework path: rework → in-progress → review (re-review)
	StateRework: StateInProgress,
}

// backwardStates defines valid backward transitions (rework/re-execution paths).
var backwardStates = map[State]State{
	StateInProgress: StateAssigned, // Can go back to assigned when blocked
	// Note: review → in-progress is NOT a direct path; it must go through rework:
	// review → rework → in-progress → review
}

// IsValid reports whether state is part of the M1 workflow.
func (s State) IsValid() bool {
	switch s {
	case StatePendingTriage, StateNew, StateTriaged, StateAssigned,
		StateInProgress, StateBlocked, StateReview, StateRework,
		StateDone, StateAbandoned:
		return true
	default:
		return false
	}
}

// Label returns the canonical status label for state.
func (s State) Label() string {
	return statusLabelPrefix + string(s)
}

// Transition validates and applies a state transition.
// Returns the new state if valid, or an error if the transition is not allowed.
// Uses transition_id as idempotency key - same state transition is idempotent.
//
// State machine paths (from PolarSwarm.md 3.2):
//   - Forward path: pending-triage → new → triaged → assigned → in-progress → review → done
//   - Blocked path: blocked → assigned (rework execution)
//   - Rework path: rework → in-progress → review (re-review)
func Transition(current, target State) (State, error) {
	if err := validateState("current", current); err != nil {
		return "", err
	}
	if err := validateState("target", target); err != nil {
		return "", err
	}
	// Idempotent: same state transition succeeds without error
	if current == target {
		return target, nil
	}
	// Check forward transition
	if nextStates[current] == target {
		return target, nil
	}
	// Check backward transition (rework paths)
	if backwardStates[current] == target {
		return target, nil
	}
	return "", fmt.Errorf("illegal workflow transition: %q -> %q", current, target)
}

// TransitionAfterResult validates a transition while honoring agent success.
func TransitionAfterResult(current, target State, success bool) (State, error) {
	if target == StateDone && !success {
		return "", fmt.Errorf("refuse workflow transition to %q after unsuccessful result", target)
	}
	return Transition(current, target)
}

// LabelProjection describes the label updates and resulting label set for a target state.
type LabelProjection struct {
	Remove []string
	Add    []string
	Labels []string
}

// ProjectStatusLabels removes stale status labels and ensures target is the only status label.
func ProjectStatusLabels(labels []string, target State) (LabelProjection, error) {
	if err := validateState("target", target); err != nil {
		return LabelProjection{}, err
	}

	targetLabel := target.Label()
	result := make([]string, 0, len(labels)+1)
	remove := make([]string, 0)
	seenRemove := make(map[string]bool)
	hasTarget := false

	for _, label := range labels {
		if !strings.HasPrefix(label, statusLabelPrefix) {
			result = append(result, label)
			continue
		}
		if label == targetLabel {
			if !hasTarget {
				result = append(result, label)
				hasTarget = true
			}
			continue
		}
		if !seenRemove[label] {
			remove = append(remove, label)
			seenRemove[label] = true
		}
	}

	add := []string(nil)
	if !hasTarget {
		result = append(result, targetLabel)
		add = []string{targetLabel}
	}
	if len(remove) == 0 {
		remove = nil
	}

	return LabelProjection{
		Remove: remove,
		Add:    add,
		Labels: result,
	}, nil
}

// StateFromLabels returns the single current workflow state projected by status labels.
func StateFromLabels(labels []string) (State, error) {
	var current State
	for _, label := range labels {
		if !strings.HasPrefix(label, statusLabelPrefix) {
			continue
		}
		state := State(strings.TrimPrefix(label, statusLabelPrefix))
		if err := validateState("label", state); err != nil {
			return "", err
		}
		if current != "" && current != state {
			return "", fmt.Errorf("multiple workflow status labels: %q and %q", current.Label(), state.Label())
		}
		current = state
	}
	if current == "" {
		return "", errors.New("missing workflow status label")
	}
	return current, nil
}

func validateState(name string, state State) error {
	if !state.IsValid() {
		return fmt.Errorf("invalid %s workflow state %q", name, state)
	}
	return nil
}
