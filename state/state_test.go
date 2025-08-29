package state

import (
	"testing"
)

// MockState is a test double for the State interface.
// It helps us track which methods have been called.
type MockState struct {
	ID            string
	OnEnterCalled bool
	OnExitCalled  bool
	OnUpdateCalled bool
}

func (m *MockState) OnEnter() {
	m.OnEnterCalled = true
}

func (m *MockState) OnExit() {
	m.OnExitCalled = true
}

func (m *MockState) OnUpdate() {
	m.OnUpdateCalled = true
}

func (m *MockState) GetID() string {
	return m.ID
}

func (m *MockState) HandleAction(player Player, actionData []byte) error {
	return nil
}

// reset clears the call tracking flags.
func (m *MockState) reset() {
	m.OnEnterCalled = false
	m.OnExitCalled = false
	m.OnUpdateCalled = false
}

func TestStateMachine_InitialState(t *testing.T) {
	initialState := &MockState{ID: "initial"}
	sm := NewBaseStateMachine(initialState)

	if !initialState.OnEnterCalled {
		t.Error("Expected OnEnter to be called on the initial state")
	}

	if sm.GetCurrentState() != initialState {
		t.Error("GetCurrentState should return the initial state")
	}
}

func TestStateMachine_ChangeState(t *testing.T) {
	initialState := &MockState{ID: "initial"}
	nextState := &MockState{ID: "next"}

	sm := NewBaseStateMachine(initialState)
	initialState.reset() // Reset after initialization

	err := sm.ChangeState(nextState)
	if err != nil {
		t.Fatalf("ChangeState should not return an error, but got: %v", err)
	}

	if !initialState.OnExitCalled {
		t.Error("Expected OnExit to be called on the old state")
	}

	if !nextState.OnEnterCalled {
		t.Error("Expected OnEnter to be called on the new state")
	}

	if sm.GetCurrentState() != nextState {
		t.Error("GetCurrentState should return the new state")
	}
}

func TestStateMachine_AddAndUseTransition(t *testing.T) {
	stateA := &MockState{ID: "A"}
	stateB := &MockState{ID: "B"}
	stateC := &MockState{ID: "C"}

	sm := NewBaseStateMachine(stateA)

	// Add a valid transition from A to B
	err := sm.AddTransition(stateA, stateB, func() bool { return true })
	if err != nil {
		t.Fatalf("AddTransition failed: %v", err)
	}

	// Add a blocked transition from B to C
	err = sm.AddTransition(stateB, stateC, func() bool { return false })
	if err != nil {
		t.Fatalf("AddTransition failed: %v", err)
	}

	// --- Test valid transition ---
	stateA.reset()
	err = sm.ChangeState(stateB)
	if err != nil {
		t.Errorf("Expected transition from A to B to be allowed, but got error: %v", err)
	}
	if sm.GetCurrentState().GetID() != "B" {
		t.Errorf("Expected current state to be B, but got %s", sm.GetCurrentState().GetID())
	}

	// --- Test blocked transition ---
	stateB.reset()
	err = sm.ChangeState(stateC)
	if err != ErrTransitionNotAllowed {
		t.Errorf("Expected ErrTransitionNotAllowed, but got: %v", err)
	}
	if sm.GetCurrentState().GetID() != "B" {
		t.Errorf("Expected current state to remain B after a blocked transition, but got %s", sm.GetCurrentState().GetID())
	}
	if stateB.OnExitCalled {
		t.Error("OnExit should not be called on the current state if transition is blocked")
	}
	if stateC.OnEnterCalled {
		t.Error("OnEnter should not be called on the new state if transition is blocked")
	}
}
