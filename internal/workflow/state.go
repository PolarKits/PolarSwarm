package workflow

import (
	"fmt"
	"strings"
)

const statusLabelPrefix = "status:"

// State is the M1 issue workflow state.
type State string

const (
	StateNew        State = "new"
	StateAssigned   State = "assigned"
	StateInProgress State = "in-progress"
	StateReview     State = "review"
	StateDone       State = "done"
)

var nextStates = map[State]State{
	StateNew:        StateAssigned,
	StateAssigned:   StateInProgress,
	StateInProgress: StateReview,
	StateReview:     StateDone,
}

// IsValid reports whether state is part of the M1 workflow.
func (s State) IsValid() bool {
	switch s {
	case StateNew, StateAssigned, StateInProgress, StateReview, StateDone:
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
func Transition(current, target State) (State, error) {
	if err := validateState("current", current); err != nil {
		return "", err
	}
	if err := validateState("target", target); err != nil {
		return "", err
	}
	if current == target {
		return target, nil
	}
	if nextStates[current] != target {
		return "", fmt.Errorf("illegal workflow transition: %q -> %q", current, target)
	}
	return target, nil
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

func validateState(name string, state State) error {
	if !state.IsValid() {
		return fmt.Errorf("invalid %s workflow state %q", name, state)
	}
	return nil
}
