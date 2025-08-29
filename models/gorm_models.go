// models/gorm_models.go
package models

import (
	"gorm.io/gorm"
)

// GormPlayer 玩家模型
type GormPlayer struct {
	gorm.Model
	UserID     int64                  `gorm:"uniqueIndex;not null"`
	Name       string                 `gorm:"not null"`
	Level      int                    `gorm:"default:1"`
	Experience int                    `gorm:"default:0"`
	Coins      int64                  `gorm:"default:1000"`
	Items      map[string]interface{} `gorm:"type:jsonb"`
	Stats      map[string]interface{} `gorm:"type:jsonb"`
}

// GormGameRecord 游戏记录模型
type GormGameRecord struct {
	gorm.Model
	RoomID   string                 `gorm:"index;not null"`
	GameType string                 `gorm:"not null"`
	Players  map[string]interface{} `gorm:"type:jsonb;not null"`
	Result   map[string]interface{} `gorm:"type:jsonb;not null"`
	Duration int                    `gorm:"default:0"` // 游戏时长(秒)
}

// GormRoom 房间模型
type GormRoom struct {
	gorm.Model
	RoomID   string                 `gorm:"uniqueIndex;not null"`
	GameType string                 `gorm:"not null"`
	State    string                 `gorm:"not null"`
	Players  map[string]interface{} `gorm:"type:jsonb"`
	Settings map[string]interface{} `gorm:"type:jsonb"`
}

// PlayerStats 玩家统计信息
type PlayerStats struct {
	TotalGames int   `json:"total_games"`
	Wins       int   `json:"wins"`
	Losses     int   `json:"losses"`
	TotalCoins int64 `json:"total_coins"`
	PlayTime   int   `json:"play_time"` // 总游戏时长(分钟)
}

// GormGameConfig 游戏配置
type GormGameConfig struct {
	gorm.Model
	GameType string                 `gorm:"uniqueIndex;not null"`
	Config   map[string]interface{} `gorm:"type:jsonb;not null"`
	Enabled  bool                   `gorm:"default:true"`
}