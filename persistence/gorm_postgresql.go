// persistence/gorm_postgresql.go
package persistence

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormPostgreSQL 使用GORM的PostgreSQL实现
type GormPostgreSQL struct {
	db *gorm.DB
}

// NewGormPostgreSQL 创建GORM PostgreSQL数据库连接
func NewGormPostgreSQL(host string, port int, user, password, dbname string) (*GormPostgreSQL, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// 配置GORM日志
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second,   // 慢SQL阈值
			LogLevel:      logger.Silent, // 日志级别
			Colorful:      false,         // 禁用彩色打印
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	// 获取通用数据库对象 sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移表结构
	if err := autoMigrate(db); err != nil {
		return nil, err
	}

	return &GormPostgreSQL{db: db}, nil
}

// 定义GORM模型
type PlayerModel struct {
	ID        uint                   `gorm:"primaryKey"`
	UserID    int64                  `gorm:"uniqueIndex;not null"`
	Data      map[string]interface{} `gorm:"type:jsonb"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type GameRecordModel struct {
	ID        uint                   `gorm:"primaryKey"`
	RoomID    string                 `gorm:"index;not null"`
	GameType  string                 `gorm:"not null"`
	Players   map[string]interface{} `gorm:"type:jsonb"`
	Result    map[string]interface{} `gorm:"type:jsonb"`
	CreatedAt time.Time
}

type RoomModel struct {
	ID        uint                   `gorm:"primaryKey"`
	RoomID    string                 `gorm:"uniqueIndex;not null"`
	GameType  string                 `gorm:"not null"`
	State     string                 `gorm:"not null"`
	Players   map[string]interface{} `gorm:"type:jsonb"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// autoMigrate 自动迁移表结构
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&PlayerModel{},
		&GameRecordModel{},
		&RoomModel{},
	)
}

// SavePlayerData 保存玩家数据
func (p *GormPostgreSQL) SavePlayerData(playerID int64, data interface{}) error {
	playerData, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid player data type")
	}

	// 使用UPSERT操作
	var player PlayerModel
	result := p.db.Where("user_id = ?", playerID).First(&player)

	if result.Error == gorm.ErrRecordNotFound {
		// 创建新记录
		player = PlayerModel{
			UserID: playerID,
			Data:   playerData,
		}
		return p.db.Create(&player).Error
	} else if result.Error != nil {
		return result.Error
	}

	// 更新现有记录
	player.Data = playerData
	player.UpdatedAt = time.Now()
	return p.db.Save(&player).Error
}

// LoadPlayerData 加载玩家数据
func (p *GormPostgreSQL) LoadPlayerData(playerID int64, result interface{}) error {
	var player PlayerModel
	if err := p.db.Where("user_id = ?", playerID).First(&player).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrRecordNotFound
		}
		return err
	}

	// 将数据转换为目标类型
	data, ok := result.(*map[string]interface{})
	if ok {
		*data = player.Data
		return nil
	}

	return fmt.Errorf("invalid result type")
}

// SaveGameRecord 保存游戏记录
func (p *GormPostgreSQL) SaveGameRecord(record interface{}) error {
	recordData, ok := record.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid game record type")
	}

	gameRecord := GameRecordModel{
		RoomID:   recordData["room_id"].(string),
		GameType: recordData["game_type"].(string),
		Players:  recordData["players"].(map[string]interface{}),
		Result:   recordData["result"].(map[string]interface{}),
	}

	return p.db.Create(&gameRecord).Error
}

// SaveRoomState 保存房间状态
func (p *GormPostgreSQL) SaveRoomState(roomID, gameType, state string, players interface{}) error {
	playersData, ok := players.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid players data type")
	}

	var room RoomModel
	result := p.db.Where("room_id = ?", roomID).First(&room)

	if result.Error == gorm.ErrRecordNotFound {
		// 创建新记录
		room = RoomModel{
			RoomID:   roomID,
			GameType: gameType,
			State:    state,
			Players:  playersData,
		}
		return p.db.Create(&room).Error
	} else if result.Error != nil {
		return result.Error
	}

	// 更新现有记录
	room.State = state
	room.Players = playersData
	room.UpdatedAt = time.Now()
	return p.db.Save(&room).Error
}

// LoadRoomState 加载房间状态
func (p *GormPostgreSQL) LoadRoomState(roomID string, result interface{}) error {
	var room RoomModel
	if err := p.db.Where("room_id = ?", roomID).First(&room).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrRecordNotFound
		}
		return err
	}

	// 将数据转换为目标类型
	data, ok := result.(*map[string]interface{})
	if ok {
		*data = room.Players
		return nil
	}

	return fmt.Errorf("invalid result type")
}

// Close 关闭数据库连接
func (p *GormPostgreSQL) Close() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// 添加事务支持
func (p *GormPostgreSQL) Transaction(fn func(tx *gorm.DB) error) error {
	return p.db.Transaction(fn)
}

// 添加高级查询方法
func (p *GormPostgreSQL) GetPlayerStats(userID int64) (map[string]interface{}, error) {
	var stats map[string]interface{}

	// 示例：使用原生SQL查询
	err := p.db.Raw(
		`
        SELECT 
            COUNT(*) as total_games,
            SUM(CASE WHEN result->>'outcome' = 'win' THEN 1 ELSE 0 END) as wins,
            SUM(CASE WHEN result->>'outcome' = 'lose' THEN 1 ELSE 0 END) as losses
        FROM game_records 
        WHERE players @> ?`,
		fmt.Sprintf(`{"%d": {}}`, userID),
	).Scan(&stats).Error

	return stats, err
}