// broadcast/broadcast.go
package broadcast

import (
	"errors"

	"github.com/wfunc/gameserver/room"

	"github.com/wfunc/gameserver/session"
)

var (
	ErrRoomNotFound = errors.New("room not found")
)

// 广播接口
type Broadcaster interface {
	BroadcastToRoom(roomID string, msgID uint16, data []byte) error
	BroadcastToAll(msgID uint16, data []byte) error
	BroadcastToUsers(userIDs []int64, msgID uint16, data []byte) error
}

// 基于房间的广播器
type RoomBroadcaster struct {
	roomManager    *room.Manager
	sessionManager *session.Manager
}

func NewRoomBroadcaster(roomManager *room.Manager, sessionManager *session.Manager) *RoomBroadcaster {
	return &RoomBroadcaster{
		roomManager:    roomManager,
		sessionManager: sessionManager,
	}
}

func (b *RoomBroadcaster) BroadcastToRoom(roomID string, msgID uint16, data []byte) error {
	room, exists := b.roomManager.GetRoom(roomID)
	if !exists {
		return ErrRoomNotFound
	}

	// Get a thread-safe copy of the sessions
	sessions := room.GetSessions()

	for _, s := range sessions {
		if err := s.Send(msgID, data); err != nil {
			// 处理发送错误，可能需要移除玩家
			continue
		}
	}

	return nil
}

func (b *RoomBroadcaster) BroadcastToAll(msgID uint16, data []byte) error {
	// 获取所有房间并广播
	// 实现略...
	return nil
}

func (b *RoomBroadcaster) BroadcastToUsers(userIDs []int64, msgID uint16, data []byte) error {
	for _, userID := range userIDs {
		sessions := b.sessionManager.GetByUserID(userID)
		for _, s := range sessions {
			if err := s.Send(msgID, data); err != nil {
				// 处理发送错误
				continue
			}
		}
	}
	return nil
}
