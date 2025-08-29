package network

const (
	MsgTypeHeartbeat   = 1
	MsgTypeJoinRoom    = 101
	MsgTypeLeaveRoom   = 102
	MsgTypeCreateRoom  = 103
	MsgTypeGameAction  = 201
	MsgTypePlayerAction = 202
	MsgTypeRoomState   = 301
	MsgTypePlayerState = 302
	MsgTypeGameStart   = 303
	MsgTypeGameSync    = 304
	MsgTypeGameEnd     = 305
)
