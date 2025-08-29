// room/room.go
package room

import (
	"sync"
	"time"

	"github.com/wfunc/gameserver/session"
	"github.com/wfunc/gameserver/state"
)

// RoomStatus 表示房间的业务状态，例如等待、游戏中等
type RoomStatus int

const (
	StatusIdle RoomStatus = iota
	StatusWaiting
	StatusGaming
	StatusSettlement
)

// Room 是游戏房间的核心结构
type Room struct {
	ID           string
	Name         string
	GameType     string
	MaxPlayers   int
	Status       RoomStatus
	Players      map[string]*session.Session // sessionID -> session
	StateMachine state.StateMachine
	CreatedAt    time.Time
	GameData     interface{} // 游戏特定数据
	broadcaster  Broadcaster // Use the interface, not the concrete type
	statusMutex  sync.RWMutex
	playerMutex  sync.RWMutex
	ticker       *time.Ticker
	closeChan    chan bool
}

// NewRoom 创建一个新房间
func NewRoom(id, name, gameType string, maxPlayers int, broadcaster Broadcaster) *Room {
	room := &Room{
		ID:           id,
		Name:         name,
		GameType:     gameType,
		MaxPlayers:   maxPlayers,
		Status:       StatusIdle,
		Players:      make(map[string]*session.Session),
		CreatedAt:    time.Now(),
		closeChan:    make(chan bool),
		broadcaster:  broadcaster,
	}

	// 初始化状态机，将房间自身(room)作为上下文传入
	initialState := state.NewWaitingState(room)
	room.StateMachine = state.NewBaseStateMachine(initialState)
	room.SetStatus(StatusWaiting)

	// 启动房间心跳
	room.ticker = time.NewTicker(100 * time.Millisecond) // 10 FPS
	go room.loop()

	return room
}

// --- 实现 state.RoomContext 接口 ---

// GetID 返回房间ID
func (r *Room) GetID() string {
	return r.ID
}

// GetGameType 获取游戏类型
func (r *Room) GetGameType() string {
	return r.GameType
}

// GetMaxPlayers returns the maximum number of players in the room.
func (r *Room) GetMaxPlayers() int {
	return r.MaxPlayers
}

// GetPlayers 获取房间中的所有玩家，返回的map值为 state.Player 接口
func (r *Room) GetPlayers() map[string]state.Player {
	r.playerMutex.RLock()
	defer r.playerMutex.RUnlock()

	// 返回副本以避免并发修改
	players := make(map[string]state.Player)
	for k, v := range r.Players {
		players[k] = v // session.Session 实现了 state.Player 接口
	}
	return players
}

// ChangeState 改变房间的状态机状态
func (r *Room) ChangeState(newState state.State) error {
	return r.StateMachine.ChangeState(newState)
}

// Broadcast sends a message to all players in the room.
func (r *Room) Broadcast(msgID uint16, data []byte) error {
	return r.broadcaster.BroadcastToRoom(r.ID, msgID, data)
}

// --- 房间核心逻辑 ---

// AddPlayer 添加一个玩家到房间
func (r *Room) AddPlayer(s *session.Session) bool {
	r.playerMutex.Lock()
	defer r.playerMutex.Unlock()

	if len(r.Players) >= r.MaxPlayers {
		return false
	}

	r.Players[s.ID] = s
	s.RoomID = r.ID
	return true
}

// RemovePlayer 从房间移除一个玩家
func (r *Room) RemovePlayer(sessionID string) {
	r.playerMutex.Lock()
	defer r.playerMutex.Unlock()

	if player, exists := r.Players[sessionID]; exists {
		player.RoomID = ""
		delete(r.Players, sessionID)
	}
}

// GetPlayer 获取单个玩家
func (r *Room) GetPlayer(sessionID string) (*session.Session, bool) {
	r.playerMutex.RLock()
	defer r.playerMutex.RUnlock()

	player, exists := r.Players[sessionID]
	return player, exists
}

// GetSessions returns a slice of all sessions in the room (thread-safe).
func (r *Room) GetSessions() []*session.Session {
	r.playerMutex.RLock()
	defer r.playerMutex.RUnlock()

	sessions := make([]*session.Session, 0, len(r.Players))
	for _, s := range r.Players {
		sessions = append(sessions, s)
	}
	return sessions
}

// SetStatus 设置房间的业务状态
func (r *Room) SetStatus(status RoomStatus) {
	r.statusMutex.Lock()
	defer r.statusMutex.Unlock()
	r.Status = status
}

// GetStatus 获取房间的业务状态
func (r *Room) GetStatus() RoomStatus {
	r.statusMutex.RLock()
	defer r.statusMutex.RUnlock()
	return r.Status
}

// loop 是房间的主循环，定时驱动状态更新
func (r *Room) loop() {
	for {
		select {
		case <-r.ticker.C:
			r.Update()
		case <-r.closeChan:
			r.ticker.Stop()
			return
		}
	}
}

// Update 由主循环调用，驱动状态机更新
func (r *Room) Update() {
	if r.StateMachine != nil {
		currentState := r.StateMachine.GetCurrentState()
		if currentState != nil {
			currentState.OnUpdate()
		}
	}
}

// Close 关闭房间，停止主循环
func (r *Room) Close() {
	close(r.closeChan)
}

// --- 房间管理器 ---

// Manager 管理所有房间
type Manager struct {
	rooms map[string]*Room
	mutex sync.RWMutex
}

// NewRoomManager 创建一个新的房间管理器
func NewRoomManager() *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
	}
}

// CreateRoom 创建一个新房间并添加到管理器
func (m *Manager) CreateRoom(id, name, gameType string, maxPlayers int, broadcaster Broadcaster) *Room {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	room := NewRoom(id, name, gameType, maxPlayers, broadcaster)
	m.rooms[id] = room
	return room
}

// RemoveRoom 从管理器中移除并关闭一个房间
func (m *Manager) RemoveRoom(id string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if room, exists := m.rooms[id]; exists {
		room.Close()
		delete(m.rooms, id)
	}
}

// GetRoom 从管理器中获取一个房间
func (m *Manager) GetRoom(id string) (*Room, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	room, exists := m.rooms[id]
	return room, exists
}

// FindAvailableRoom 查找一个可用的房间
func (m *Manager) FindAvailableRoom() *Room {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, room := range m.rooms {
		if len(room.Players) < room.MaxPlayers && room.Status == StatusWaiting {
			return room
		}
	}
	return nil
}