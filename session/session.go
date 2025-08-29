// session/session.go
package session

import (
	"sync"
	"time"

	"github.com/wfunc/gameserver/network"
)

type Session struct {
	ID         string
	Conn       network.Connection
	UserID     int64
	RoomID     string
	Data       map[string]interface{} // 自定义数据
	CreatedAt  time.Time
	LastActive time.Time
	mutex      sync.RWMutex
}

func NewSession(id string, conn network.Connection) *Session {
	now := time.Now()
	return &Session{
		ID:         id,
		Conn:       conn,
		CreatedAt:  now,
		LastActive: now,
		Data:       make(map[string]interface{}),
	}
}

func (s *Session) Set(key string, value interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Data[key] = value
}

func (s *Session) Get(key string) interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.Data[key]
}

func (s *Session) Send(msgID uint16, data []byte) error {
	s.LastActive = time.Now()
	return s.Conn.Send(msgID, data)
}

func (s *Session) GetID() string {
	return s.ID
}

func (s *Session) Close() error {
	return s.Conn.Close()
}

// Session管理器
type Manager struct {
	sessions map[string]*Session
	mutex    sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

func (m *Manager) Add(session *Session) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.sessions[session.ID] = session
}

func (m *Manager) Remove(sessionID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.sessions, sessionID)
}

func (m *Manager) Get(sessionID string) (*Session, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	session, exists := m.sessions[sessionID]
	return session, exists
}

func (m *Manager) GetByUserID(userID int64) []*Session {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*Session
	for _, session := range m.sessions {
		if session.UserID == userID {
			result = append(result, session)
		}
	}
	return result
}
