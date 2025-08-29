package session

import (
	"net"
	"testing"
	"time"

	"github.com/wfunc/gameserver/network"
)

// MockConnection is a test double for the network.Connection interface.
type MockConnection struct{}

func (m *MockConnection) Send(msgID uint16, data []byte) error { return nil }
func (m *MockConnection) Close() error                         { return nil }
func (m *MockConnection) RemoteAddr() net.Addr                 { return &net.TCPAddr{} }
func (m *MockConnection) SetHeartbeat(interval time.Duration)  {}
func (m *MockConnection) ReadPacket() (*network.Packet, error) { return nil, nil }

func TestNewManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("NewManager should not return nil")
	}
	if manager.sessions == nil {
		t.Fatal("NewManager should initialize the sessions map")
	}
}

func TestManager_Add_Get_Remove(t *testing.T) {
	manager := NewManager()
	sessionID := "test_session_1"
	sess := NewSession(sessionID, &MockConnection{})

	// Test Add
	manager.Add(sess)
	if len(manager.sessions) != 1 {
		t.Fatalf("Expected session count to be 1, got %d", len(manager.sessions))
	}

	// Test Get
	retrievedSess, exists := manager.Get(sessionID)
	if !exists {
		t.Fatal("Get should find the added session")
	}
	if retrievedSess != sess {
		t.Fatal("Get should return the same session instance")
	}

	// Test Remove
	manager.Remove(sessionID)
	if len(manager.sessions) != 0 {
		t.Fatalf("Expected session count to be 0 after removal, got %d", len(manager.sessions))
	}

	_, exists = manager.Get(sessionID)
	if exists {
		t.Fatal("Get should not find the removed session")
	}
}

func TestManager_GetByUserID(t *testing.T) {
	manager := NewManager()

	sess1 := NewSession("session1", &MockConnection{})
	sess1.UserID = 100

	sess2 := NewSession("session2", &MockConnection{})
	sess2.UserID = 200

	sess3 := NewSession("session3", &MockConnection{})
	sess3.UserID = 100

	manager.Add(sess1)
	manager.Add(sess2)
	manager.Add(sess3)

	user100Sessions := manager.GetByUserID(100)
	if len(user100Sessions) != 2 {
		t.Errorf("Expected 2 sessions for UserID 100, got %d", len(user100Sessions))
	}

	user200Sessions := manager.GetByUserID(200)
	if len(user200Sessions) != 1 {
		t.Errorf("Expected 1 session for UserID 200, got %d", len(user200Sessions))
	}

	user300Sessions := manager.GetByUserID(300)
	if len(user300Sessions) != 0 {
		t.Errorf("Expected 0 sessions for UserID 300, got %d", len(user300Sessions))
	}
}

func TestSession_Set_Get(t *testing.T) {
	sess := NewSession("test_session", &MockConnection{})
	key := "test_key"
	value := "test_value"

	sess.Set(key, value)

	retrievedValue := sess.Get(key)
	if retrievedValue != value {
		t.Errorf("Expected value %v, got %v", value, retrievedValue)
	}

	nilValue := sess.Get("non_existent_key")
	if nilValue != nil {
		t.Errorf("Expected nil for non-existent key, got %v", nilValue)
	}
}
