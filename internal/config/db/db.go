package db

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)


func New(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("не удалось подключиться к БД: %w", err)
	}



	return db, nil
}