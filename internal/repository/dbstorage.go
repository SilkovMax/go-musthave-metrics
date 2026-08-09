package repository

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/SilkovMax/go-musthave-metrics/internal/model"
)

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(db *sql.DB) *DBStorage {
	return &DBStorage{db: db}
}

func (s *DBStorage) Ping() error {
	return s.db.Ping()
}

//Запись
func (s *DBStorage) SetGauge(name string, value float64) {
	query := `INSERT INTO gauges (name, value) VALUES ($1, $2)
	          ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value`
	if _, err := s.db.Exec(query, name, value); err != nil {
		log.Printf("DBStorage.SetGauge: ошибка сохранения %s: %v", name, err)
	}
}

func (s *DBStorage) IncrementCounter(name string, delta int64) {
	query := `INSERT INTO counters (name, value) VALUES ($1, $2)
	          ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value`
	if _, err := s.db.Exec(query, name, delta); err != nil {
		log.Printf("DBStorage.IncrementCounter: ошибка увеличения %s: %v", name, err)
	}
}

//Чтение
func (s *DBStorage) GetGauge(name string) (float64, error) {
	var value float64
	err := s.db.QueryRow("SELECT value FROM gauges WHERE name = $1", name).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("metric %s not found", name)
	}
	if err != nil {
		return 0, fmt.Errorf("ошибка чтения gauge %s: %w", name, err)
	}
	return value, nil
}

func (s *DBStorage) GetCounter(name string) (int64, error) {
	var value int64
	err := s.db.QueryRow("SELECT value FROM counters WHERE name = $1", name).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("metric %s not found", name)
	}
	if err != nil {
		return 0, fmt.Errorf("ошибка чтения counter %s: %w", name, err)
	}
	return value, nil
}

func (s *DBStorage) GetAllGauges() map[string]float64 {
	result := make(map[string]float64)
	rows, err := s.db.Query("SELECT name, value FROM gauges")
	if err != nil {
		log.Printf("DBStorage.GetAllGauges: ошибка чтения: %v", err)
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			log.Printf("DBStorage.GetAllGauges: ошибка сканирования: %v", err)
			continue
		}
		result[name] = value
	}
	return result
}

func (s *DBStorage) GetAllCounters() map[string]int64 {
	result := make(map[string]int64)
	rows, err := s.db.Query("SELECT name, value FROM counters")
	if err != nil {
		log.Printf("DBStorage.GetAllCounters: ошибка чтения: %v", err)
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			log.Printf("DBStorage.GetAllCounters: ошибка сканирования: %v", err)
			continue
		}
		result[name] = value
	}
	return result
}

func (s *DBStorage) SetBatch(metrics []model.Metrics) error {

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	gaugeStmt, err := tx.Prepare(`INSERT INTO gauges (name, value) VALUES ($1, $2)
	                              ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("prepare gauge stmt: %w", err)
	}
	defer gaugeStmt.Close()

	counterStmt, err := tx.Prepare(`INSERT INTO counters (name, value) VALUES ($1, $2)
	                                ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("prepare counter stmt: %w", err)
	}
	defer counterStmt.Close()

	for _, m := range metrics {
		switch m.MType {
		case "gauge":
			if m.Value == nil {
				continue // пропускаем битые данные
			}
			if _, err := gaugeStmt.Exec(m.ID, *m.Value); err != nil {
				tx.Rollback()
				return fmt.Errorf("exec gauge %s: %w", m.ID, err)
			}
		case "counter":
			if m.Delta == nil {
				continue
			}
			if _, err := counterStmt.Exec(m.ID, *m.Delta); err != nil {
				tx.Rollback()
				return fmt.Errorf("exec counter %s: %w", m.ID, err)
			}
		}
	}

	return tx.Commit()
}