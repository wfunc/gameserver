package room

// Broadcaster defines the interface for broadcasting messages to a room.
// This is defined here to break the import cycle between room and broadcast.
type Broadcaster interface {
	BroadcastToRoom(roomID string, msgID uint16, data []byte) error
}
