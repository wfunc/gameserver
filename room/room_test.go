package room

import (
	"net"
	"testing"
	"time"

	"github.com/wfunc/gameserver/network"
	"github.com/wfunc/gameserver/session"
)

// MockBroadcaster is a test double for the Broadcaster interface.
type MockBroadcaster struct{}

func (m *MockBroadcaster) BroadcastToRoom(roomID string, msgID uint16, data []byte) error {
	return nil
}

// MockConnection is a test double for the network.Connection interface.
type MockConnection struct{}

func (m *MockConnection) Send(msgID uint16, data []byte) error { return nil }
func (m *MockConnection) Close() error                         { return nil }
func (m *MockConnection) RemoteAddr() net.Addr                 { return &net.TCPAddr{} }
func (m *MockConnection) SetHeartbeat(interval time.Duration)  {}
func (m *MockConnection) ReadPacket() (*network.Packet, error) { return nil, nil }

// newTestSession creates a dummy session for testing purposes.
func newTestSession(id string) *session.Session {
	return session.NewSession(id, &MockConnection{})
}

func TestRoomManager_CreateAndGetRoom(t *testing.T) {
	manager := NewRoomManager()
	mockBroadcaster := &MockBroadcaster{}

	roomID := "test_room_1"
	room := manager.CreateRoom(roomID, "Test Room", "test_game", 4, mockBroadcaster)

	if room == nil {
		t.Fatal("CreateRoom should not return nil")
	}

	if room.ID != roomID {
		t.Errorf("Expected room ID %s, got %s", roomID, room.ID)
	}

	retrievedRoom, exists := manager.GetRoom(roomID)
	if !exists {
		t.Fatal("GetRoom should find the created room")
	}

	if retrievedRoom != room {
		t.Error("GetRoom should return the same room instance")
	}
}

func TestRoom_AddPlayer(t *testing.T) {
	mockBroadcaster := &MockBroadcaster{}
	room := NewRoom("test_room_2", "Add Player Test", "test_game", 2, mockBroadcaster)

	player1 := newTestSession("player1")

	added := room.AddPlayer(player1)
	if !added {
		t.Fatal("Failed to add first player")
	}

	if len(room.Players) != 1 {
		t.Errorf("Expected player count to be 1, got %d", len(room.Players))
	}

	if _, exists := room.Players[player1.GetID()]; !exists {
		t.Error("Player was not correctly added to the room's player map")
	}
}

func TestRoom_AddPlayer_Full(t *testing.T) {
	mockBroadcaster := &MockBroadcaster{}
	room := NewRoom("test_room_3", "Full Room Test", "test_game", 1, mockBroadcaster)

	player1 := newTestSession("player1")
	player2 := newTestSession("player2")

	// Add first player, should succeed
	if !room.AddPlayer(player1) {
		t.Fatal("Failed to add the first player")
	}

	// Add second player, should fail
	if room.AddPlayer(player2) {
		t.Fatal("Should not be able to add a player to a full room")
	}

	if len(room.Players) != 1 {
		t.Errorf("Expected player count to be 1 after trying to add to a full room, got %d", len(room.Players))
	}
}

func TestRoom_RemovePlayer(t *testing.T) {
	mockBroadcaster := &MockBroadcaster{}
	room := NewRoom("test_room_4", "Remove Player Test", "test_game", 2, mockBroadcaster)

	player1 := newTestSession("player1")
	room.AddPlayer(player1)

	if len(room.Players) != 1 {
		t.Fatalf("Setup failed: player not added correctly. Count: %d", len(room.Players))
	}

	room.RemovePlayer(player1.GetID())

	if len(room.Players) != 0 {
		t.Errorf("Expected player count to be 0 after removing player, got %d", len(room.Players))
	}

	if _, exists := room.Players[player1.GetID()]; exists {
		t.Error("Player was not correctly removed from the room's player map")
	}
}
