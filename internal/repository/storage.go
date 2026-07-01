package repository

// Storage описывает интерфейс для работы с хранилищем метрик
type Storage interface {
	SetGauge(name string, value float64)

	IncrementCounter(name string, delta int64)
}