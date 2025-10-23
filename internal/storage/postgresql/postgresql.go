package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Kulibyka/effective-mobile/internal/config"
	_ "github.com/lib/pq"
	"time"
)

type Storage struct {
	db *sql.DB
}

func New(cfg config.PostgreConfig) (*Storage, error) {
	const op = "storage.postgresql.New"

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) GetDB() *sql.DB {
	return s.db
}

func (s *Storage) Close() error {
	return s.db.Close()
}
