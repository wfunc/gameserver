// services/player_service.go
package services

import (
	"fmt"

	"github.com/wfunc/gameserver/models"
	"github.com/wfunc/gameserver/persistence"
	"gorm.io/gorm"
)

type PlayerService struct {
	db persistence.Database
}

func NewPlayerService(db persistence.Database) *PlayerService {
	return &PlayerService{db: db}
}

// GetPlayerWithStats 获取玩家信息和统计
func (s *PlayerService) GetPlayerWithStats(userID int64) (map[string]interface{}, error) {
	var result map[string]interface{}

	// 使用事务确保数据一致性
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 获取玩家基本信息
		var player models.GormPlayer
		if err := tx.Where("user_id = ?", userID).First(&player).Error; err != nil {
			return err
		}

		// 获取玩家统计信息
		stats, err := s.db.GetPlayerStats(userID)
		if err != nil {
			return err
		}

		result = map[string]interface{}{
			"player": player,
			"stats":  stats,
		}

		return nil
	})

	return result, err
}

// UpdatePlayerCoins 更新玩家金币数量（原子操作）
func (s *PlayerService) UpdatePlayerCoins(userID int64, delta int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var player models.GormPlayer
		if err := tx.Where("user_id = ?", userID).First(&player).Error; err != nil {
			return err
		}

		// 检查金币是否足够（如果是减少）
		if delta < 0 && player.Coins+delta < 0 {
			return fmt.Errorf("insufficient coins")
		}

		// 更新金币数量
		if err := tx.Model(&player).Update("coins", gorm.Expr("coins + ?", delta)).Error; err != nil {
			return err
		}

		// 更新统计信息
		if err := tx.Model(&player).Update("stats", gorm.Expr(`
            jsonb_set(
                COALESCE(stats, '{}'::jsonb), 
                '{total_coins}', 
                to_jsonb(COALESCE((stats->>'total_coins')::int, 0) + ?)
            )
        `, delta)).Error; err != nil {
			return err
		}

		return nil
	})
}
