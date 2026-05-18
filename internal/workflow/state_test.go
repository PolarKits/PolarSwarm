package workflow

import (
	"reflect"
	"strings"
	"testing"
)

func TestTransition(t *testing.T) {
	tests := []struct {
		name    string
		current State
		target  State
		want    State
		wantErr string
	}{
		// Forward path: pending-triage → new → triaged → assigned → in-progress → review → done
		{name: "pending-triage to new", current: StatePendingTriage, target: StateNew, want: StateNew},
		{name: "new to triaged", current: StateNew, target: StateTriaged, want: StateTriaged},
		{name: "triaged to assigned", current: StateTriaged, target: StateAssigned, want: StateAssigned},
		{name: "assigned to in-progress", current: StateAssigned, target: StateInProgress, want: StateInProgress},
		{name: "in-progress to review", current: StateInProgress, target: StateReview, want: StateReview},
		{name: "review to done", current: StateReview, target: StateDone, want: StateDone},

		// Idempotent transitions
		{name: "same state is idempotent", current: StateReview, target: StateReview, want: StateReview},
		{name: "same state in-progress is idempotent", current: StateInProgress, target: StateInProgress, want: StateInProgress},

		// Reject illegal forward transitions
		{name: "skip forward rejected", current: StateNew, target: StateInProgress, wantErr: "illegal workflow transition"},
		{name: "skip multiple steps rejected", current: StateNew, target: StateReview, wantErr: "illegal workflow transition"},

		// Reject backward transitions (except rework paths)
		{name: "move backward rejected", current: StateReview, target: StateInProgress, wantErr: "illegal workflow transition"},

		// Done terminal state
		{name: "done cannot advance", current: StateDone, target: StateReview, wantErr: "illegal workflow transition"},
		{name: "done is terminal", current: StateDone, target: StateDone, want: StateDone},

		// Invalid states
		{name: "invalid current", current: State("invalid"), target: StateReview, wantErr: "invalid current workflow state"},
		{name: "invalid target", current: StateReview, target: State("invalid"), wantErr: "invalid target workflow state"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Transition(tt.current, tt.target)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Transition returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected state: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestTransitionBlockedPath(t *testing.T) {
	// Blocked path: blocked → assigned (rework execution)
	tests := []struct {
		name    string
		current State
		target  State
		want    State
		wantErr string
	}{
		{name: "blocked to assigned (rework execution)", current: StateBlocked, target: StateAssigned, want: StateAssigned},
		{name: "blocked is idempotent", current: StateBlocked, target: StateBlocked, want: StateBlocked},
		{name: "blocked cannot go to in-progress directly", current: StateBlocked, target: StateInProgress, wantErr: "illegal workflow transition"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Transition(tt.current, tt.target)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Transition returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected state: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestTransitionReworkPath(t *testing.T) {
	// Rework path: rework → in-progress → review (re-review)
	tests := []struct {
		name    string
		current State
		target  State
		want    State
		wantErr string
	}{
		{name: "rework to in-progress", current: StateRework, target: StateInProgress, want: StateInProgress},
		{name: "rework is idempotent", current: StateRework, target: StateRework, want: StateRework},
		{name: "rework cannot skip to review", current: StateRework, target: StateReview, wantErr: "illegal workflow transition"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Transition(tt.current, tt.target)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Transition returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected state: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestTransitionAfterResultPreventsDoneOnFailure(t *testing.T) {
	tests := []struct {
		name    string
		current State
		target  State
		success bool
		want    State
		wantErr string
	}{
		{name: "successful result may finish", current: StateReview, target: StateDone, success: true, want: StateDone},
		{name: "failed result cannot finish", current: StateReview, target: StateDone, success: false, wantErr: "unsuccessful result"},
		{name: "failed result may stay in review", current: StateReview, target: StateReview, success: false, want: StateReview},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransitionAfterResult(tt.current, tt.target, tt.success)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("TransitionAfterResult returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected state: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestProjectStatusLabels(t *testing.T) {
	tests := []struct {
		name       string
		labels     []string
		target     State
		wantRemove []string
		wantAdd    []string
		wantLabels []string
		wantErr    string
	}{
		{
			name:       "replaces old status",
			labels:     []string{"area:workflow", "status:new", "priority:p1"},
			target:     StateAssigned,
			wantRemove: []string{"status:new"},
			wantAdd:    []string{"status:assigned"},
			wantLabels: []string{"area:workflow", "priority:p1", "status:assigned"},
		},
		{
			name:       "pending-triage to new",
			labels:     []string{"status:pending-triage"},
			target:     StateNew,
			wantRemove: []string{"status:pending-triage"},
			wantAdd:    []string{"status:new"},
			wantLabels: []string{"status:new"},
		},
		{
			name:       "new to triaged",
			labels:     []string{"status:new"},
			target:     StateTriaged,
			wantRemove: []string{"status:new"},
			wantAdd:    []string{"status:triaged"},
			wantLabels: []string{"status:triaged"},
		},
		{
			name:       "already target is idempotent",
			labels:     []string{"area:workflow", "status:review"},
			target:     StateReview,
			wantLabels: []string{"area:workflow", "status:review"},
		},
		{
			name:       "keeps one target status",
			labels:     []string{"status:new", "status:review", "status:review", "kind:task"},
			target:     StateReview,
			wantRemove: []string{"status:new"},
			wantLabels: []string{"status:review", "kind:task"},
		},
		{
			name:       "adds status when absent",
			labels:     []string{"area:workflow"},
			target:     StateNew,
			wantAdd:    []string{"status:new"},
			wantLabels: []string{"area:workflow", "status:new"},
		},
		{
			name:       "blocked to assigned (rework execution)",
			labels:     []string{"status:blocked"},
			target:     StateAssigned,
			wantRemove: []string{"status:blocked"},
			wantAdd:    []string{"status:assigned"},
			wantLabels: []string{"status:assigned"},
		},
		{
			name:       "rework to in-progress",
			labels:     []string{"status:rework"},
			target:     StateInProgress,
			wantRemove: []string{"status:rework"},
			wantAdd:    []string{"status:in-progress"},
			wantLabels: []string{"status:in-progress"},
		},
		{
			name:    "invalid target",
			labels:  []string{"status:new"},
			target:  State("invalid-state"),
			wantErr: "invalid target workflow state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProjectStatusLabels(tt.labels, tt.target)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ProjectStatusLabels returned error: %v", err)
			}
			if !reflect.DeepEqual(got.Remove, tt.wantRemove) {
				t.Fatalf("unexpected remove labels: got %#v want %#v", got.Remove, tt.wantRemove)
			}
			if !reflect.DeepEqual(got.Add, tt.wantAdd) {
				t.Fatalf("unexpected add labels: got %#v want %#v", got.Add, tt.wantAdd)
			}
			if !reflect.DeepEqual(got.Labels, tt.wantLabels) {
				t.Fatalf("unexpected projected labels: got %#v want %#v", got.Labels, tt.wantLabels)
			}
			if countStatusLabels(got.Labels) != 1 {
				t.Fatalf("projected labels must contain exactly one status label: %#v", got.Labels)
			}
		})
	}
}

func TestStateFromLabels(t *testing.T) {
	tests := []struct {
		name    string
		labels  []string
		want    State
		wantErr string
	}{
		{name: "single status", labels: []string{"area:workflow", "status:review"}, want: StateReview},
		{name: "duplicate same status", labels: []string{"status:review", "status:review"}, want: StateReview},
		{name: "pending-triage status", labels: []string{"status:pending-triage"}, want: StatePendingTriage},
		{name: "blocked status", labels: []string{"status:blocked"}, want: StateBlocked},
		{name: "rework status", labels: []string{"status:rework"}, want: StateRework},
		{name: "missing status", labels: []string{"area:workflow"}, wantErr: "missing workflow status label"},
		{name: "multiple statuses", labels: []string{"status:new", "status:review"}, wantErr: "multiple workflow status labels"},
		{name: "invalid status", labels: []string{"status:invalid"}, wantErr: "invalid label workflow state"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StateFromLabels(tt.labels)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("StateFromLabels returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected state: got %q want %q", got, tt.want)
			}
		})
	}
}

func countStatusLabels(labels []string) int {
	var count int
	for _, label := range labels {
		if strings.HasPrefix(label, statusLabelPrefix) {
			count++
		}
	}
	return count
}
