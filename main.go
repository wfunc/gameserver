package main

import (
	"github.com/wfunc/gameserver/config"
	"github.com/wfunc/gameserver/logger"
	"github.com/wfunc/gameserver/persistence"
	"github.com/wfunc/gameserver/server"
)

func main() {
	// Initialize logger
	logger.Init()

	// Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		logger.Log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Database
	db, err := persistence.NewGormPostgreSQL(
		cfg.Database.Postgres.Host,
		cfg.Database.Postgres.Port,
		cfg.Database.Postgres.User,
		cfg.Database.Postgres.Password,
		cfg.Database.Postgres.DBName,
	)
	if err != nil {
		logger.Log.Fatalf("Failed to connect to database: %v", err)
	}
	logger.Log.Info("Database connection successful.")

	// Initialize Game Server
	gameServer := server.NewGameServer(cfg.Server.HTTPAddress, cfg.Server.RPCAddress, db)

	// Start Server
	logger.Log.Infof("Starting game server on %s", cfg.Server.HTTPAddress)
	if err := gameServer.Start(); err != nil {
		logger.Log.Fatalf("Failed to start server: %v", err)
	}
}
