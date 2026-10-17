package repository_test

import (
	"fmt"
	"time"

	"github.com/SilkovMax/go-musthave-metrics/internal/model"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

// ExampleMemStorage_SetGauge демонстрирует сохранение gauge метрики.
func ExampleMemStorage_SetGauge() {
	storage := repository.NewMemStorage("", 0, false)
	defer storage.Stop()

	storage.SetGauge("CPU", 36.6)

	val, err := storage.GetGauge("CPU")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(val)
	// Output:
	// 36.6
}

// ExampleMemStorage_IncrementCounter демонстрирует инкремент counter метрики.
func ExampleMemStorage_IncrementCounter() {
	storage := repository.NewMemStorage("", 0, false)
	defer storage.Stop()

	storage.IncrementCounter("PollCount", 1)
	storage.IncrementCounter("PollCount", 1)
	storage.IncrementCounter("PollCount", 1)

	val, err := storage.GetCounter("PollCount")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(val)
	// Output:
	// 3
}

// ExampleMemStorage_GetAllGauges демонстрирует получение всех gauge метрик.
func ExampleMemStorage_GetAllGauges() {
	storage := repository.NewMemStorage("", 0, false)
	defer storage.Stop()

	storage.SetGauge("CPU", 36.6)
	storage.SetGauge("Memory", 1024.0)

	gauges := storage.GetAllGauges()

	// Проверяем количество метрик
	fmt.Println(len(gauges))
	// Output:
	// 2
}

// ExampleMemStorage_SetBatch демонстрирует batch-сохранение метрик.
func ExampleMemStorage_SetBatch() {
	storage := repository.NewMemStorage("", 0, false)
	defer storage.Stop()

	cpuValue := 36.6
	memoryValue := 1024.0
	pollCountDelta := int64(10)

	metrics := []model.Metrics{
		{ID: "CPU", MType: model.Gauge, Value: &cpuValue},
		{ID: "Memory", MType: model.Gauge, Value: &memoryValue},
		{ID: "PollCount", MType: model.Counter, Delta: &pollCountDelta},
	}

	err := storage.SetBatch(metrics)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	gauges := storage.GetAllGauges()
	counters := storage.GetAllCounters()

	fmt.Println(len(gauges), len(counters))
	// Output:
	// 2 1
}

// ExampleMemStorage_SaveToFile демонстрирует сохранение метрик в файл.
func ExampleMemStorage_SaveToFile() {
	// Создаём хранилище с автосохранением каждые 100мс
	storage := repository.NewMemStorage("test_metrics.json", 100*time.Millisecond, false)
	defer storage.Stop()

	storage.SetGauge("CPU", 36.6)

	// Ждём автосохранения
	time.Sleep(150 * time.Millisecond)

	// Создаём новое хранилище и загружаем из файла
	storage2 := repository.NewMemStorage("test_metrics.json", 0, true)
	defer storage2.Stop()

	val, err := storage2.GetGauge("CPU")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(val)
	// Output:
	// 36.6
}
