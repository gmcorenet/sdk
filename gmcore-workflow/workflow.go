package gmcore_workflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Package gmcore_workflow provides a state machine implementation for managing
// transitions between states with support for guards, actions, and markings.
//
// Basic usage:
//
//	cfg := NewConfig("order").
//		WithInitialMarking("cart").
//		WithState("cart").WithTransition("checkout", "processing", nil, nil).
//		WithState("processing").WithTransition("pay", "paid", nil, nil).
//		WithState("paid")
//
//	workflow := NewWorkflow(cfg)
//	marking, _ := workflow.Apply(ctx, workflow.InitialMarking(), "checkout")
//	marking, _ := workflow.Apply(ctx, marking, "pay")
//
// The marking represents the current state(s) of a subject within the workflow.

// Transition represents a transition between workflow states.
type Transition struct {
	Name string
	From string
	To   string
	Guard func(ctx context.Context, marking map[string]bool) bool
}

// Workflow represents a state machine workflow.
type Workflow struct {
	name          string
	places        []string
	transitions   []Transition
	markingStore  MarkingStore
	mu            sync.RWMutex
}

// MarkingStore interface for persisting workflow state.
type MarkingStore interface {
	GetMarking(subject interface{}) (map[string]bool, error)
	SetMarking(subject interface{}, marking map[string]bool) error
}

// MemoryMarkingStore is an in-memory marking store.
type MemoryMarkingStore struct {
	markings map[interface{}]map[string]bool
	mu       sync.RWMutex
}

// NewMemoryMarkingStore creates a new in-memory marking store.
func NewMemoryMarkingStore() *MemoryMarkingStore {
	return &MemoryMarkingStore{
		markings: make(map[interface{}]map[string]bool),
	}
}

// GetMarking returns the current marking for a subject.
func (s *MemoryMarkingStore) GetMarking(subject interface{}) (map[string]bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if marking, ok := s.markings[subject]; ok {
		result := make(map[string]bool, len(marking))
		for k, v := range marking {
			result[k] = v
		}
		return result, nil
	}
	return make(map[string]bool), nil
}

// SetMarking sets the marking for a subject.
func (s *MemoryMarkingStore) SetMarking(subject interface{}, marking map[string]bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.markings[subject] = marking
	return nil
}

// WorkflowRegistry manages multiple workflows.
type WorkflowRegistry struct {
	workflows map[string]*Workflow
	mu        sync.RWMutex
}

// NewWorkflowRegistry creates a new workflow registry.
func NewWorkflowRegistry() *WorkflowRegistry {
	return &WorkflowRegistry{
		workflows: make(map[string]*Workflow),
	}
}

// Register registers a workflow with the given name.
func (r *WorkflowRegistry) Register(name string, w *Workflow) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflows[name] = w
}

// Get returns a workflow by name.
func (r *WorkflowRegistry) Get(name string) (*Workflow, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w, ok := r.workflows[name]
	return w, ok
}

// New creates a new workflow.
func New(name string, places []string, transitions []Transition) *Workflow {
	return &Workflow{
		name:          name,
		places:        places,
		transitions:   transitions,
		markingStore:  NewMemoryMarkingStore(),
	}
}

// SetMarkingStore sets the marking store for this workflow.
func (w *Workflow) SetMarkingStore(store MarkingStore) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.markingStore = store
}

// Can returns true if the transition can be applied from the current marking.
func (w *Workflow) Can(ctx context.Context, marking map[string]bool, transitionName string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, t := range w.transitions {
		if t.Name != transitionName {
			continue
		}
		if !marking[t.From] {
			return false
		}
		if t.Guard != nil && !t.Guard(ctx, marking) {
			return false
		}
		return true
	}
	return false
}

// Apply applies a transition to the marking.
func (w *Workflow) Apply(ctx context.Context, marking map[string]bool, transitionName string) (map[string]bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, t := range w.transitions {
		if t.Name != transitionName {
			continue
		}
		if !marking[t.From] {
			return nil, fmt.Errorf("workflow %s: cannot apply transition %s: not in state %s", w.name, transitionName, t.From)
		}
		if t.Guard != nil && !t.Guard(ctx, marking) {
			return nil, fmt.Errorf("workflow %s: transition %s blocked by guard", w.name, transitionName)
		}

		newMarking := make(map[string]bool, len(marking))
		for k, v := range marking {
			newMarking[k] = v
		}
		newMarking[t.From] = false
		newMarking[t.To] = true
		return newMarking, nil
	}

	return nil, fmt.Errorf("workflow %s: unknown transition %s", w.name, transitionName)
}

// GetEnabledTransitions returns all transitions that can be applied from the current marking.
func (w *Workflow) GetEnabledTransitions(ctx context.Context, marking map[string]bool) []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var enabled []string
	for _, t := range w.transitions {
		if !marking[t.From] {
			continue
		}
		if t.Guard != nil && !t.Guard(ctx, marking) {
			continue
		}
		enabled = append(enabled, t.Name)
	}
	return enabled
}

// GetTransitionsFrom returns all transitions that can be applied from a given place.
func (w *Workflow) GetTransitionsFrom(place string) []Transition {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var result []Transition
	for _, t := range w.transitions {
		if t.From == place {
			result = append(result, t)
		}
	}
	return result
}

// GetPlaces returns the workflow places.
func (w *Workflow) GetPlaces() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.places
}

// GetTransitions returns all transitions.
func (w *Workflow) GetTransitions() []Transition {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.transitions
}

// GetName returns the workflow name.
func (w *Workflow) GetName() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.name
}

// ApplyToSubject applies a transition to a subject and updates its marking.
func (w *Workflow) ApplyToSubject(ctx context.Context, subject interface{}, transitionName string) error {
	marking, err := w.markingStore.GetMarking(subject)
	if err != nil {
		return fmt.Errorf("workflow %s: failed to get marking: %w", w.name, err)
	}
	if len(marking) == 0 {
		marking = w.InitialMarking()
	}

	newMarking, err := w.Apply(ctx, marking, transitionName)
	if err != nil {
		return err
	}

	if err := w.markingStore.SetMarking(subject, newMarking); err != nil {
		return fmt.Errorf("workflow %s: failed to set marking: %w", w.name, err)
	}

	return nil
}

// CanApplyToSubject checks if a transition can be applied to a subject.
func (w *Workflow) CanApplyToSubject(ctx context.Context, subject interface{}, transitionName string) bool {
	marking, err := w.markingStore.GetMarking(subject)
	if err != nil {
		return false
	}
	if len(marking) == 0 {
		marking = w.InitialMarking()
	}
	return w.Can(ctx, marking, transitionName)
}

// GetEnabledTransitionsForSubject returns enabled transitions for a subject.
func (w *Workflow) GetEnabledTransitionsForSubject(ctx context.Context, subject interface{}) []string {
	marking, err := w.markingStore.GetMarking(subject)
	if err != nil {
		return nil
	}
	if len(marking) == 0 {
		marking = w.InitialMarking()
	}
	return w.GetEnabledTransitions(ctx, marking)
}

// InitialMarking returns the initial marking (first place marked).
func (w *Workflow) InitialMarking() map[string]bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if len(w.places) == 0 {
		return make(map[string]bool)
	}
	return map[string]bool{w.places[0]: true}
}

// BlockedError represents an error when a transition is blocked by a guard.
type BlockedError struct {
	Workflow    string
	Transition string
	Reason     string
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf("workflow %s: transition %s blocked: %s", e.Workflow, e.Transition, e.Reason)
}

// Marking represents a workflow marking (current state).
type Marking map[string]bool

// Has returns true if the marking contains the given place.
func (m Marking) Has(place string) bool {
	return m[place]
}

// Count returns the number of active places.
func (m Marking) Count() int {
	count := 0
	for _, v := range m {
		if v {
			count++
		}
	}
	return count
}

// String returns a string representation of the marking.
func (m Marking) String() string {
	var active []string
	for p, activePlace := range m {
		if activePlace {
			active = append(active, p)
		}
	}
	return fmt.Sprintf("{%s}", strings.Join(active, ", "))
}
