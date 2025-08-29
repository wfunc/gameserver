package state

import (
	"errors"
	"sync"
	"time"
)

// 状态机接口
type StateMachine interface {
	ChangeState(state State) error
	GetCurrentState() State
	AddTransition(from State, to State, condition func() bool) error
}

// 状态接口
type State interface {
	OnEnter()
	OnExit()
	OnUpdate()
	GetID() string
	HandleAction(player Player, actionData []byte) error // <--- 添加此方法
}

// ErrTransitionNotAllowed is returned when a state transition is not allowed.
var ErrTransitionNotAllowed = errors.New("state transition not allowed")

// 基础状态机实现
type BaseStateMachine struct {
	currentState State
	transitions  map[string]map[string]func() bool // fromState -> toState -> condition
	mutex        sync.RWMutex
}

func NewBaseStateMachine(initialState State) *BaseStateMachine {
	machine := &BaseStateMachine{
		currentState: initialState,
		transitions:  make(map[string]map[string]func() bool),
	}
	initialState.OnEnter()
	return machine
}

func (sm *BaseStateMachine) ChangeState(newState State) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	currentID := sm.currentState.GetID()
	newID := newState.GetID()

	// 检查是否有转换条件
	if conditions, exists := sm.transitions[currentID]; exists {
		if condition, exists := conditions[newID]; exists {
			if condition != nil && !condition() {
				return ErrTransitionNotAllowed
			}
		}
	}

	sm.currentState.OnExit()
	sm.currentState = newState
	sm.currentState.OnEnter()

	return nil
}

func (sm *BaseStateMachine) GetCurrentState() State {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.currentState
}

func (sm *BaseStateMachine) AddTransition(from State, to State, condition func() bool) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	fromID := from.GetID()
	toID := to.GetID()

	if _, exists := sm.transitions[fromID]; !exists {
		sm.transitions[fromID] = make(map[string]func() bool)
	}

	sm.transitions[fromID][toID] = condition
	return nil
}

// 房间状态基础结构
type RoomStateBase struct {
	ID   string
	Room RoomContext
}

func (s *RoomStateBase) GetID() string {
	return s.ID
}

func (s *RoomStateBase) OnEnter() {
	// 默认实现
}

func (s *RoomStateBase) OnExit() {
	// 默认实现
}

func (s *RoomStateBase) OnUpdate() {
	// 默认实现
}

func (s *RoomStateBase) HandleAction(player Player, actionData []byte) error {
	// 默认实现，具体状态可以覆盖此方法
	return nil
}

// NewWaitingState creates a new waiting state.
func NewWaitingState(room RoomContext) *WaitingState {
	return &WaitingState{
		RoomStateBase: RoomStateBase{
			ID:   "waiting",
			Room: room,
		},
	}
}

// 等待状态
type WaitingState struct {
	RoomStateBase
	timer int
}

func (s *WaitingState) OnEnter() {
	s.timer = 100 // 10 seconds at 10fps
}

func (s *WaitingState) OnUpdate() {
	s.timer--
	if s.timer <= 0 {
		// 切换到游戏状态
		gamingState := NewGamingState(s.Room, 30*time.Second) // 假设游戏时长30秒
		s.Room.ChangeState(gamingState)
	}

	// 如果房间已满，立即开始游戏
	if len(s.Room.GetPlayers()) >= s.Room.GetMaxPlayers() {
		gamingState := NewGamingState(s.Room, 30*time.Second) // 假设游戏时长30秒
		s.Room.ChangeState(gamingState)
	}
}
