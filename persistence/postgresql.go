// persistence/postgresql.go
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	// PostgreSQL 驱动
	_ "github.com/lib/pq" // PostgreSQL 驱动
)

// PostgreSQL 数据库实现
type PostgreSQL struct {
	db *sql.DB
}

// NewPostgreSQL 创建 PostgreSQL 数据库连接
func NewPostgreSQL(host string, port int, user, password, dbname string) (*PostgreSQL, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 初始化表结构
	if err := initTables(db); err != nil {
		return nil, err
	}

	return &PostgreSQL{db: db}, nil
}

// initTables 初始化数据库表结构
func initTables(db *sql.DB) error {
	// 创建玩家表
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS players (
            id SERIAL PRIMARY KEY,
            user_id BIGINT UNIQUE NOT NULL,
            data JSONB NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
	if err != nil {
		return err
	}

	// 创建游戏记录表
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS game_records (
            id SERIAL PRIMARY KEY,
            room_id VARCHAR(255) NOT NULL,
            game_type VARCHAR(100) NOT NULL,
            players JSONB NOT NULL,
            result JSONB NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
	if err != nil {
		return err
	}

	// 创建房间表
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS rooms (
            id SERIAL PRIMARY KEY,
            room_id VARCHAR(255) UNIQUE NOT NULL,
            game_type VARCHAR(100) NOT NULL,
            state VARCHAR(50) NOT NULL,
            players JSONB NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
	if err != nil {
		return err
	}

	// 创建索引以提高查询性能
	_, err = db.Exec(`
        CREATE INDEX IF NOT EXISTS idx_players_user_id ON players(user_id);
        CREATE INDEX IF NOT EXISTS idx_game_records_room_id ON game_records(room_id);
        CREATE INDEX IF NOT EXISTS idx_game_records_created_at ON game_records(created_at);
        CREATE INDEX IF NOT EXISTS idx_rooms_room_id ON rooms(room_id);
    `)

	return err
}

// SavePlayerData 保存玩家数据
func (p *PostgreSQL) SavePlayerData(playerID int64, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 使用 UPSERT 操作 (PostgreSQL 9.5+)
	query := `
        INSERT INTO players (user_id, data) 
        VALUES ($1, $2)
        ON CONFLICT (user_id) 
        DO UPDATE SET data = $2, updated_at = CURRENT_TIMESTAMP
    `

	_, err = p.db.ExecContext(ctx, query, playerID, jsonData)
	return err
}

// LoadPlayerData 加载玩家数据
func (p *PostgreSQL) LoadPlayerData(playerID int64, result interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var data []byte
	query := `SELECT data FROM players WHERE user_id = $1`
	err := p.db.QueryRowContext(ctx, query, playerID).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrRecordNotFound
		}
		return err
	}

	return json.Unmarshal(data, result)
}

// SaveGameRecord 保存游戏记录
func (p *PostgreSQL) SaveGameRecord(record interface{}) error {
	// 根据实际记录结构实现
	// 这里假设 record 是一个包含房间ID、游戏类型、玩家信息和结果的结构体
	jsonData, err := json.Marshal(record)
	if err != nil {
		return err
	}

	// 将记录解析为map以提取字段
	var recordMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &recordMap); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
        INSERT INTO game_records (room_id, game_type, players, result) 
        VALUES ($1, $2, $3, $4)
    `

	_, err = p.db.ExecContext(ctx, query,
		recordMap["room_id"],
		recordMap["game_type"],
		recordMap["players"],
		recordMap["result"])

	return err
}

// SaveRoomState 保存房间状态
func (p *PostgreSQL) SaveRoomState(roomID, gameType, state string, players interface{}) error {
	playersJSON, err := json.Marshal(players)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
        INSERT INTO rooms (room_id, game_type, state, players) 
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (room_id) 
        DO UPDATE SET state = $3, players = $4, updated_at = CURRENT_TIMESTAMP
    `

	_, err = p.db.ExecContext(ctx, query, roomID, gameType, state, playersJSON)
	return err
}

// LoadRoomState 加载房间状态
func (p *PostgreSQL) LoadRoomState(roomID string, result interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var data []byte
	query := `SELECT players FROM rooms WHERE room_id = $1`
	err := p.db.QueryRowContext(ctx, query, roomID).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrRecordNotFound
		}
		return err
	}

	return json.Unmarshal(data, result)
}

// Close 关闭数据库连接
func (p *PostgreSQL) Close() error {
	return p.db.Close()
}
