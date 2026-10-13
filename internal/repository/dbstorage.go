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

// DBStorage реализует Storage на основе PostgreSQL.
// Все операции выполняются с retry-логикой для временных ошибок подключения.
type DBStorage struct {
	db *sql.DB
}

// NewDBStorage создаёт новое хранилище на основе PostgreSQL.
// Параметр db должен быть инициализированным *sql.DB с подключением к базе данных.
func NewDBStorage(db *sql.DB) *DBStorage {
	return &DBStorage{db: db}
}

// Ping проверяет подключение к базе данных.
func (s *DBStorage) Ping() error {
	return s.db.Ping()
}

// withRetry выполняет retry для временных ошибок.
// Попыток всего 4 (1+3) / Интервалы: 1s,3s,5s
func withRetry(operationName string, fn func() error) error {
	delays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {
		err := fn()

		if err == nil {
			return nil // Успех
		}

		if !isRetriable(err) {
			log.Printf("DBStorage.%s: ошибка: %v", operationName, err)
			return err
		}

		if attempt < 3 {
			log.Printf("DBStorage.%s: retry %d/3 (ошибка: %v)", operationName, attempt+1, err)
			time.Sleep(delays[attempt])
		} else {
			log.Printf("DBStorage.%s: все retry провалились: %v", operationName, err)
		}
	}

	return fmt.Errorf("все retry провалились для %s", operationName)
}

// Ошибки
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

// SetGauge сохраняет значение gauge метрики в PostgreSQL.
// Использует UPSERT (INSERT ... ON CONFLICT DO UPDATE) для перезаписи существующих значений.
func (s *DBStorage) SetGauge(name string, value float64) {
	query := `INSERT INTO gauges (name, value) VALUES ($1, $2)
	          ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value`

	withRetry("SetGauge", func() error {
		_, err := s.db.Exec(query, name, value)
		return err
	})
}

// IncrementCounter увеличивает значение counter метрики в PostgreSQL.
// Использует UPSERT с инкрементом для атомарного обновления.
func (s *DBStorage) IncrementCounter(name string, delta int64) {
	query := `INSERT INTO counters (name, value) VALUES ($1, $2)
	          ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value`

	withRetry("IncrementCounter", func() error {
		_, err := s.db.Exec(query, name, delta)
		return err
	})
}

// GetGauge возвращает значение gauge метрики из PostgreSQL.
// Возвращает ошибку, если метрика не найдена.
func (s *DBStorage) GetGauge(name string) (float64, error) {
	var value float64

	err := withRetry("GetGauge", func() error {
		err := s.db.QueryRow("SELECT value FROM gauges WHERE name = $1", name).Scan(&value)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("metric %s not found", name)
		}
		return err
	})

	if err != nil {
		return 0, err
	}
	return value, nil
}

// GetCounter возвращает значение counter метрики из PostgreSQL.
// Возвращает ошибку, если метрика не найдена.
func (s *DBStorage) GetCounter(name string) (int64, error) {
	var value int64

	err := withRetry("GetCounter", func() error {
		err := s.db.QueryRow("SELECT value FROM counters WHERE name = $1", name).Scan(&value)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("metric %s not found", name)
		}
		return err
	})

	if err != nil {
		return 0, err
	}
	return value, nil
}

// GetAllGauges возвращает все gauge метрики из PostgreSQL.
// Возвращает пустую map в случае ошибки (ошибка логируется).
func (s *DBStorage) GetAllGauges() map[string]float64 {
	result := make(map[string]float64)

	err := withRetry("GetAllGauges", func() error {
		rows, err := s.db.Query("SELECT name, value FROM gauges")
		if err != nil {
			return err
		}
		defer rows.Close() // defer снаружи цикла

		result = make(map[string]float64)

		for rows.Next() {
			var name string
			var value float64
			if err := rows.Scan(&name, &value); err != nil {
				return err
			}
			result[name] = value
		}

		// проверяем ошибку итерации
		if err := rows.Err(); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Printf("DBStorage.GetAllGauges: %v", err)
	}

	return result
}

// GetAllCounters возвращает все counter метрики из PostgreSQL.
// Возвращает пустую map в случае ошибки (ошибка логируется).
func (s *DBStorage) GetAllCounters() map[string]int64 {
	result := make(map[string]int64)

	err := withRetry("GetAllCounters", func() error {
		rows, err := s.db.Query("SELECT name, value FROM counters")
		if err != nil {
			return err
		}
		defer rows.Close() // defer снаружи цикла

		result = make(map[string]int64)

		for rows.Next() {
			var name string
			var value int64
			if err := rows.Scan(&name, &value); err != nil {
				return err
			}
			result[name] = value
		}

		if err := rows.Err(); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Printf("DBStorage.GetAllCounters: %v", err)
	}

	return result
}

// SetBatch сохраняет множество метрик в PostgreSQL в рамках одной транзакции.
// Все операции выполняются атомарно: либо все метрики сохраняются, либо ни одна.
func (s *DBStorage) SetBatch(metrics []model.Metrics) error {
	return withRetry("SetBatch", func() error {
		return s.executeBatchTransaction(metrics)
	})
}

// executeBatchTransaction выполняет batch-операцию в одной транзакции.
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
