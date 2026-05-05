package gmcore_workflow

import (
	"context"
	"testing"
)

func TestWorkflowBasic(t *testing.T) {
	places := []string{"draft", "review", "published"}
	transitions := []Transition{
		{Name: "submit", From: "draft", To: "review"},
		{Name: "publish", From: "review", To: "published"},
		{Name: "reject", From: "review", To: "draft"},
	}

	w := New("article", places, transitions)

	initial := w.InitialMarking()
	m := Marking(initial)
	if !m.Has("draft") {
		t.Error("expected initial marking to have 'draft'")
	}

	ctx := context.Background()

	if !w.Can(ctx, initial, "submit") {
		t.Error("should be able to submit from draft")
	}

	newMarking, err := w.Apply(ctx, initial, "submit")
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	nm := Marking(newMarking)
	if nm.Has("draft") {
		t.Error("draft should no longer be marked after submit")
	}
	if !nm.Has("review") {
		t.Error("review should be marked after submit")
	}

	enabled := w.GetEnabledTransitions(ctx, newMarking)
	if len(enabled) != 2 {
		t.Errorf("expected 2 enabled transitions, got %d", len(enabled))
	}
}

func TestWorkflowWithGuard(t *testing.T) {
	places := []string{"draft", "approved"}
	transitions := []Transition{
		{
			Name: "approve",
			From: "draft",
			To:   "approved",
			Guard: func(ctx context.Context, marking map[string]bool) bool {
				return true
			},
		},
	}

	w := New("task", places, transitions)
	marking := w.InitialMarking()

	ctx := context.Background()
	if !w.Can(ctx, marking, "approve") {
		t.Error("guard should allow transition")
	}

	transitions[0].Guard = func(ctx context.Context, marking map[string]bool) bool {
		return false
	}
	w = New("task", places, transitions)

	if w.Can(ctx, marking, "approve") {
		t.Error("guard should block transition")
	}
}

func TestWorkflowRegistry(t *testing.T) {
	registry := NewWorkflowRegistry()

	w1 := New("w1", []string{"a", "b"}, []Transition{{Name: "x", From: "a", To: "b"}})
	w2 := New("w2", []string{"x", "y"}, []Transition{{Name: "y", From: "x", To: "y"}})

	registry.Register("workflow1", w1)
	registry.Register("workflow2", w2)

	w, ok := registry.Get("workflow1")
	if !ok {
		t.Error("expected to get workflow1")
	}
	if w.GetName() != "w1" {
		t.Errorf("expected name w1, got %s", w.GetName())
	}

	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("should not find nonexistent workflow")
	}
}

func TestWorkflowApplyToSubject(t *testing.T) {
	type Order struct {
		ID   string
		Name string
	}

	places := []string{"cart", "paid", "shipped"}
	transitions := []Transition{
		{Name: "pay", From: "cart", To: "paid"},
		{Name: "ship", From: "paid", To: "shipped"},
	}

	w := New("order", places, transitions)

	order := &Order{ID: "123", Name: "Test Order"}

	ctx := context.Background()

	if !w.CanApplyToSubject(ctx, order, "pay") {
		t.Error("should be able to pay")
	}

	err := w.ApplyToSubject(ctx, order, "pay")
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	if !w.CanApplyToSubject(ctx, order, "ship") {
		t.Error("should be able to ship after pay")
	}
}

func TestMarking(t *testing.T) {
	m := Marking{"a": true, "b": false, "c": true}

	if !m.Has("a") {
		t.Error("should have a")
	}
	if m.Has("b") {
		t.Error("should not have b")
	}
	if m.Count() != 2 {
		t.Errorf("expected count 2, got %d", m.Count())
	}
}

func TestWorkflowUnknownTransition(t *testing.T) {
	w := New("test", []string{"a", "b"}, []Transition{})

	ctx := context.Background()
	_, err := w.Apply(ctx, w.InitialMarking(), "unknown")
	if err == nil {
		t.Error("expected error for unknown transition")
	}
}

func TestWorkflowBlockedByGuard(t *testing.T) {
	transitions := []Transition{
		{
			Name: "advance",
			From: "start",
			To:    "end",
			Guard: func(ctx context.Context, marking map[string]bool) bool {
				return false
			},
		},
	}

	w := New("blocked", []string{"start", "end"}, transitions)

	ctx := context.Background()
	_, err := w.Apply(ctx, w.InitialMarking(), "advance")
	if err == nil {
		t.Error("expected error when guard blocks transition")
	}
}
