package repository

import "github.com/SilkovMax/go-musthave-metrics/internal/model"

// Storage описывает интерфейс для работы с хранилищем метрик
type Storage interface {
	SetGauge(name string, value float64)

	IncrementCounter(name string, delta int64)
	GetGauge(name string) (float64, error)
	GetCounter(name string) (int64, error)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64

	SetBatch(metrics []model.Metrics) error
}
