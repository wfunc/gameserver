// persistence/interface.go
package persistence

import (
	"fmt"

	"gorm.io/gorm"
)

// Database 数据库接口
type Database interface {
	SavePlayerData(playerID int64, data interface{}) error
	LoadPlayerData(playerID int64, result interface{}) error
	SaveGameRecord(record interface{}) error
	SaveRoomState(roomID, gameType, state string, players interface{}) error
	LoadRoomState(roomID string, result interface{}) error
	Transaction(fn func(tx *gorm.DB) error) error
	GetPlayerStats(userID int64) (map[string]interface{}, error)
	Close() error
}

// 错误定义
var (
	ErrRecordNotFound = fmt.Errorf("record not found")
)
