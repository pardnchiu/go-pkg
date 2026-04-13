package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"
)

type PostgresqlConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

func NewPostgresql(ctx context.Context, cfg *PostgresqlConfig) (*sql.DB, error) {
	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		c := PostgresqlConfig{
			Host:     pgEnv("PG_HOST", "localhost"),
			Port:     pgEnvInt("PG_PORT", 5432),
			User:     pgEnv("PG_USER", "postgres"),
			Password: pgEnv("PG_PASSWORD", ""),
			Database: pgEnv("PG_DATABASE", "postgres"),
			SSLMode:  pgEnv("PG_SSLMODE", "disable"),
		}
		if cfg != nil {
			if cfg.Host != "" {
				c.Host = cfg.Host
			}
			if cfg.Port != 0 {
				c.Port = cfg.Port
			}
			if cfg.User != "" {
				c.User = cfg.User
			}
			if cfg.Password != "" {
				c.Password = cfg.Password
			}
			if cfg.Database != "" {
				c.Database = cfg.Database
			}
			if cfg.SSLMode != "" {
				c.SSLMode = cfg.SSLMode
			}
		}
		dsn = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
		)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return db, nil
}

func pgEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func pgEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
