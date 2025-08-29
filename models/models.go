// models/models.go
package models

import (
	"time"
)

// PlayerData 玩家数据模型
type PlayerData struct {
	UserID     int64                  `json:"user_id"`
	Name       string                 `json:"name"`
	Level      int                    `json:"level"`
	Experience int                    `json:"experience"`
	Coins      int64                  `json:"coins"`
	Items      map[string]interface{} `json:"items"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// GameRecord 游戏记录模型
type GameRecord struct {
	RoomID    string                 `json:"room_id"`
	GameType  string                 `json:"game_type"`
	Players   []PlayerInfo           `json:"players"`
	Result    map[string]interface{} `json:"result"`
	CreatedAt time.Time              `json:"created_at"`
}

// PlayerInfo 玩家信息（用于游戏记录）
type PlayerInfo struct {
	UserID  int64  `json:"user_id"`
	Name    string `json:"name"`
	Outcome string `json:"outcome"` // win/lose/draw
	Points  int    `json:"points"`
}

// RoomState 房间状态模型
type RoomState struct {
	RoomID    string                 `json:"room_id"`
	GameType  string                 `json:"game_type"`
	State     string                 `json:"state"`
	Players   map[string]interface{} `json:"players"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}
