package repository

import "github.com/SilkovMax/go-musthave-metrics/internal/model"

// Storage описывает интерфейс для работы с хранилищем метрик.
// Реализации включают MemStorage (in-memory с файловой персистентностью)
// и DBStorage (PostgreSQL).
type Storage interface {
	// SetGauge сохраняет значение gauge метрики по имени.
	// Перезаписывает предыдущее значение, если метрика уже существует.
	SetGauge(name string, value float64)

	// IncrementCounter увеличивает значение counter метрики на delta.
	// Если метрика не существует, создаёт её со значением delta.
	IncrementCounter(name string, delta int64)

	// GetGauge возвращает значение gauge метрики по имени.
	// Возвращает ошибку, если метрика не найдена.
	GetGauge(name string) (float64, error)

	// GetCounter возвращает значение counter метрики по имени.
	// Возвращает ошибку, если метрика не найдена.
	GetCounter(name string) (int64, error)

	// GetAllGauges возвращает копию всех gauge метрик в виде map[name]value.
	GetAllGauges() map[string]float64

	// GetAllCounters возвращает копию всех counter метрик в виде map[name]value.
	GetAllCounters() map[string]int64

	// SetBatch сохраняет множество метрик в одной транзакции.
	// Возвращает ошибку, если сохранение не удалось.
	SetBatch(metrics []model.Metrics) error
}
