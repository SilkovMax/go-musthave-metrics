package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

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

//Ошибки
func isRetriable(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Class 08 — Connection Exception
		if strings.HasPrefix(pgErr.Code, "08") {
			return true
		}
	}

	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}

//Запись
func (s *DBStorage) SetGauge(name string, value float64) {
	query := `INSERT INTO gauges (name, value) VALUES ($1, $2)
	          ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value`
	delays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {
		_, err := s.db.Exec(query, name, value)
		if err == nil {
			return 
		}

		if !isRetriable(err) {
			log.Printf("DBStorage.SetGauge: ошибка %s: %v", name, err)
			return
		}

		if attempt < 3 {
			log.Printf("DBStorage.SetGauge: retry %d/3 для %s (ошибка: %v)", attempt+1, name, err)
			time.Sleep(delays[attempt])
		} else {
			log.Printf("DBStorage.SetGauge: все retry провалились для %s: %v", name, err)
		}
	}
}

func (s *DBStorage) IncrementCounter(name string, delta int64) {
	query := `INSERT INTO counters (name, value) VALUES ($1, $2)
	          ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value`
	delays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {
		_, err := s.db.Exec(query, name, delta)
		if err == nil {
			return
		}

		if !isRetriable(err) {
			log.Printf("DBStorage.IncrementCounter: ошибка %s: %v", name, err)
			return
		}

		if attempt < 3 {
			log.Printf("DBStorage.IncrementCounter: retry %d/3 для %s", attempt+1, name)
			time.Sleep(delays[attempt])
		} else {
			log.Printf("DBStorage.IncrementCounter: все retry провалились для %s: %v", name, err)
		}
	}
}

//Чтение
func (s *DBStorage) GetGauge(name string) (float64, error) {
	delays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {
		var value float64
		err := s.db.QueryRow("SELECT value FROM gauges WHERE name = $1", name).Scan(&value)

		if err == nil {
			return value, nil // Успех
		}

		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("metric %s not found", name)
		}

		if !isRetriable(err) {
			return 0, fmt.Errorf("ошибка чтения gauge %s: %w", name, err)
		}

		if attempt < 3 {
			log.Printf("DBStorage.GetGauge: retry %d/3 для %s", attempt+1, name)
			time.Sleep(delays[attempt])
		} else {
			return 0, fmt.Errorf("ошибка чтения gauge %s после всех retry: %w", name, err)
		}
	}

	return 0, fmt.Errorf("выход из цикла retry")
}

func (s *DBStorage) GetCounter(name string) (int64, error) {
	delays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {
		var value int64
		err := s.db.QueryRow("SELECT value FROM counters WHERE name = $1", name).Scan(&value)

		if err == nil {
			return value, nil
		}

		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("metric %s not found", name)
		}

		if !isRetriable(err) {
			return 0, fmt.Errorf("ошибка чтения counter %s: %w", name, err)
		}

		if attempt < 3 {
			log.Printf("DBStorage.GetCounter: retry %d/3 для %s", attempt+1, name)
			time.Sleep(delays[attempt])
		} else {
			return 0, fmt.Errorf("ошибка чтения counter %s после всех retry: %w", name, err)
		}
	}

	return 0, fmt.Errorf("выход из цикла retry")
}

func (s *DBStorage) GetAllGauges() map[string]float64 {
	result := make(map[string]float64)
	delays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {
		rows, err := s.db.Query("SELECT name, value FROM gauges")
		if err == nil {
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
			return result // Успех
		}

		if !isRetriable(err) {
			log.Printf("DBStorage.GetAllGauges: ошибка: %v", err)
			return result
		}

		if attempt < 3 {
			log.Printf("DBStorage.GetAllGauges: retry %d/3", attempt+1)
			time.Sleep(delays[attempt])
		} else {
			log.Printf("DBStorage.GetAllGauges: все retry провалились: %v", err)
		}
	}

	return result
}

func (s *DBStorage) GetAllCounters() map[string]int64 {
	result := make(map[string]int64)
	delays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {
		rows, err := s.db.Query("SELECT name, value FROM counters")
		if err == nil {
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

		if !isRetriable(err) {
			log.Printf("DBStorage.GetAllCounters: ошибка: %v", err)
			return result
		}

		if attempt < 3 {
			log.Printf("DBStorage.GetAllCounters: retry %d/3", attempt+1)
			time.Sleep(delays[attempt])
		} else {
			log.Printf("DBStorage.GetAllCounters: все retry провалились: %v", err)
		}
	}

	return result
}

// обертка с ретраями
func (s *DBStorage) SetBatch(metrics []model.Metrics) error {
	delays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {
		err := s.executeBatchTransaction(metrics)
		if err == nil {
			return nil
		}

		if !isRetriable(err) {
			return fmt.Errorf("ошибка batch: %w", err)
		}

		if attempt < 3 {
			log.Printf("DBStorage.SetBatch: retry %d/3 (ошибка: %v)", attempt+1, err)
			time.Sleep(delays[attempt])
		} else {
			return fmt.Errorf("ошибка batch после всех retry: %w", err)
		}
	}

	return fmt.Errorf("выход из цикла retry")
}

// для проверки на ретраи , логика вынесена за скобки
func (s *DBStorage) executeBatchTransaction(metrics []model.Metrics) error {
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
				continue
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
