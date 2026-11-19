package repository

import (
	"testing"
	"time"

	"github.com/SilkovMax/go-musthave-metrics/internal/model"
)

// BenchmarkSetGauge измеряет скорость записи одной gauge метрики
// Интервал = 0 означает синхронное сохранение в файл на каждом update
func BenchmarkSetGauge(b *testing.B) {
	// Создаём хранилище с interval=0 (самый тяжёлый режим)
	storage := NewMemStorage("bench_metrics.json", 0, false)
	defer storage.Stop()

	// Предзаполняем метриками для реалистичности
	for i := 0; i < 100; i++ {
		storage.SetGauge("pre_fill_"+string(rune('A'+i%26)), float64(i))
	}

	b.ResetTimer()
	b.ReportAllocs() //  показывать количество аллокаций

	for i := 0; i < b.N; i++ {
		storage.SetGauge("CPU", 36.6)
	}
}

// BenchmarkIncrementCounter измеряет скорость инкремента counter метрики
func BenchmarkIncrementCounter(b *testing.B) {
	storage := NewMemStorage("bench_metrics.json", 0, false)
	defer storage.Stop()

	for i := 0; i < 100; i++ {
		storage.IncrementCounter("pre_fill_"+string(rune('A'+i%26)), 1)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		storage.IncrementCounter("PollCount", 1)
	}
}

// BenchmarkSetBatch измеряет скорость батчевой записи
func BenchmarkSetBatch(b *testing.B) {
	storage := NewMemStorage("bench_metrics.json", 0, false)
	defer storage.Stop()

	// Подготавливаем батч из 100 метрик
	metrics := make([]model.Metrics, 100)
	for i := 0; i < 100; i++ {
		val := float64(i)
		metrics[i] = model.Metrics{
			ID:    "metric_" + string(rune('A'+i%26)),
			MType: model.Gauge,
			Value: &val,
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = storage.SetBatch(metrics)
	}
}

// BenchmarkGetAllMetrics измеряет скорость получения всех метрик (snapshot)
func BenchmarkGetAllMetrics(b *testing.B) {
	storage := NewMemStorage("bench_metrics.json", time.Second*300, false) // не пишем в файл
	defer storage.Stop()

	// Предзаполняем 1000 метрик
	for i := 0; i < 500; i++ {
		storage.SetGauge("gauge_"+string(rune('A'+i%26))+string(rune(i)), float64(i))
		storage.IncrementCounter("counter_"+string(rune('A'+i%26))+string(rune(i)), int64(i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = storage.GetAllMetrics()
	}
}
