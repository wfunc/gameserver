// state/interfaces.go
package state

// Player defines the minimal interface for a player entity that a state needs to interact with.
type Player interface {
	GetID() string
}

// RoomContext defines the interface that a Room must implement to be managed by the state machine.
// This breaks the import cycle between room and state.
type RoomContext interface {
	GetID() string
	GetGameType() string
	GetPlayers() map[string]Player
	GetMaxPlayers() int
	ChangeState(newState State) error
	Broadcast(msgID uint16, data []byte) error
}
