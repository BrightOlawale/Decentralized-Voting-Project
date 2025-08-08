package database

import (
	"database/sql"
	"fmt"

	"voting-system/pkg/config"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// NewConnection creates a new database connection based on configuration
func NewConnection(cfg *config.DatabaseConfig) (*sql.DB, error) {
	var dsn string
	var driverName string

	switch cfg.Type {
	case "postgres":
		driverName = "postgres"
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)
	case "sqlite":
		driverName = "sqlite3"
		dsn = cfg.Path
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.MaxLifetime)

	return db, nil
}
