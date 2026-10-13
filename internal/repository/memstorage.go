package repository

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/SilkovMax/go-musthave-metrics/internal/model"
)

// MemStorage реализует Storage в памяти с опциональным сохранением в файл.
// Потокобезопасен через sync.RWMutex.
//
// Параметры:
//   - filepath: путь к файлу для персистентности (пустая строка отключает сохранение)
//   - interval: интервал периодического сохранения (0 = синхронное сохранение при каждом update)
//   - restore: загружать ли метрики из файла при старте
type MemStorage struct {
	mu         sync.RWMutex
	gauges     map[string]float64
	counters   map[string]int64
	gaugeBuf   []float64
	counterBuf []int64

	filepath string
	interval time.Duration
	stopCh   chan struct{}
}

// NewMemStorage создаёт новое in-memory хранилище.
//
// Если restore=true и filepath не пустой, загружает метрики из файла при старте.
// Если interval>0 и filepath не пустой, запускает фоновую горутину для периодического сохранения.
// Если interval=0, сохраняет метрики в файл синхронно при каждом update (для тестирования).
func NewMemStorage(filepath string, interval time.Duration, restore bool) *MemStorage {
	s := &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
		filepath: filepath,
		interval: interval,
		stopCh:   make(chan struct{}),
	}

	if restore && filepath != "" {
		s.LoadFromFile()
	}

	if interval > 0 && filepath != "" {
		go s.backgroundSave()
	}

	return s
}

// Stop останавливает фоновую горутину сохранения (если она запущена).
// Должен вызываться при завершении работы с хранилищем.
func (s *MemStorage) Stop() {
	close(s.stopCh)
}

// backgroundSave периодически сохраняет метрики в файл.
func (s *MemStorage) backgroundSave() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.SaveToFile()
		case <-s.stopCh:
			return
		}
	}
}

// SetGauge сохраняет значение gauge метрики.
// Если interval=0 и filepath задан, синхронно сохраняет все метрики в файл.
func (s *MemStorage) SetGauge(name string, value float64) {
	metrics := func() []model.Metrics {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.gauges[name] = value

		if s.interval == 0 && s.filepath != "" {
			return s.snapshotLocked()
		}
		return nil
	}()

	if metrics != nil {
		if err := s.saveSnapshotToFile(metrics); err != nil {
			log.Printf("MemStorage: ошибка сохранения: %v", err)
		}
	}
}

// IncrementCounter увеличивает значение counter метрики на delta.
// Если interval=0 и filepath задан, синхронно сохраняет все метрики в файл.
func (s *MemStorage) IncrementCounter(name string, delta int64) {
	metrics := func() []model.Metrics {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.counters[name] += delta

		if s.interval == 0 && s.filepath != "" {
			return s.snapshotLocked()
		}
		return nil
	}()

	if metrics != nil {
		if err := s.saveSnapshotToFile(metrics); err != nil {
			log.Printf("MemStorage: ошибка сохранения: %v", err)
		}
	}
}

// GetAllGauges возвращает копию всех gauge метрик.
// Потокобезопасен, использует RLock для чтения.
func (s *MemStorage) GetAllGauges() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]float64, len(s.gauges))
	for k, v := range s.gauges {
		result[k] = v
	}

	return result
}

// GetAllCounters возвращает копию всех counter метрик.
// Потокобезопасен, использует RLock для чтения.
func (s *MemStorage) GetAllCounters() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int64, len(s.counters))
	for k, v := range s.counters {
		result[k] = v
	}

	return result
}

// GetGauge возвращает значение gauge метрики по имени.
// Возвращает ошибку, если метрика не найдена.
func (s *MemStorage) GetGauge(name string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.gauges[name]
	if !ok {
		return 0, errors.New("metric not found")
	}
	return val, nil
}

// GetCounter возвращает значение counter метрики по имени.
// Возвращает ошибку, если метрика не найдена.
func (s *MemStorage) GetCounter(name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.counters[name]
	if !ok {
		return 0, errors.New("metric not found")
	}
	return val, nil
}

// GetAllMetrics возвращает слайс всех метрик в формате model.Metrics.
// Используется для сериализации в JSON при сохранении в файл.
func (s *MemStorage) GetAllMetrics() []model.Metrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var metrics []model.Metrics
	for k, v := range s.gauges {
		val := v
		metrics = append(metrics, model.Metrics{ID: k, MType: model.Gauge, Value: &val})
	}
	for k, v := range s.counters {
		val := v
		metrics = append(metrics, model.Metrics{ID: k, MType: model.Counter, Delta: &val})
	}
	return metrics
}

// RestoreMetrics восстанавливает метрики из слайса model.Metrics.
// Используется при загрузке из файла.
func (s *MemStorage) RestoreMetrics(metrics []model.Metrics) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range metrics {
		if m.MType == model.Gauge && m.Value != nil {
			s.gauges[m.ID] = *m.Value
		} else if m.MType == model.Counter && m.Delta != nil {
			s.counters[m.ID] = *m.Delta
		}
	}
}

// SaveToFile сохраняет все метрики в файл (если filepath задан).
// Возвращает nil, если filepath пустой.
func (s *MemStorage) SaveToFile() error {
	if s.filepath == "" {
		return nil
	}

	metrics := func() []model.Metrics {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.snapshotLocked()
	}()

	return s.saveSnapshotToFile(metrics)
}

// LoadFromFile загружает метрики из файла (если filepath задан и файл существует).
// Возвращает nil, если файл не существует или filepath пустой.
func (s *MemStorage) LoadFromFile() error {
	if s.filepath == "" {
		return nil
	}
	data, err := os.ReadFile(s.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var metrics []model.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}
	s.RestoreMetrics(metrics)
	return nil
}

// SetBatch сохраняет множество метрик.
// Если interval=0 и filepath задан, синхронно сохраняет все метрики в файл.
func (s *MemStorage) SetBatch(metrics []model.Metrics) error {
	snapshot := func() []model.Metrics {
		s.mu.Lock()
		defer s.mu.Unlock()

		for _, m := range metrics {
			switch m.MType {
			case "gauge":
				if m.Value != nil {
					s.gauges[m.ID] = *m.Value
				}
			case "counter":
				if m.Delta != nil {
					s.counters[m.ID] += *m.Delta
				}
			}
		}

		if s.interval == 0 && s.filepath != "" {
			return s.snapshotLocked()
		}
		return nil
	}()

	if snapshot != nil {
		return s.saveSnapshotToFile(snapshot)
	}
	return nil
}

// snapshotLocked создаёт снапшот всех метрик (вызывается под Lock).
func (s *MemStorage) snapshotLocked() []model.Metrics {
	// ОПТИМИЗАЦИЯ 1: preallocate slice с нужной ёмкостью
	totalLen := len(s.gauges) + len(s.counters)
	metrics := make([]model.Metrics, 0, totalLen)

	// ОПТИМИЗАЦИЯ 2: переиспользуем буфер значений
	if cap(s.gaugeBuf) < len(s.gauges) {
		s.gaugeBuf = make([]float64, len(s.gauges))
	}
	if cap(s.counterBuf) < len(s.counters) {
		s.counterBuf = make([]int64, len(s.counters))
	}

	i := 0
	for k, v := range s.gauges {
		s.gaugeBuf[i] = v
		metrics = append(metrics, model.Metrics{
			ID:    k,
			MType: model.Gauge,
			Value: &s.gaugeBuf[i],
		})
		i++
	}

	j := 0
	for k, v := range s.counters {
		s.counterBuf[j] = v
		metrics = append(metrics, model.Metrics{
			ID:    k,
			MType: model.Counter,
			Delta: &s.counterBuf[j],
		})
		j++
	}
	return metrics
}

// saveSnapshotToFile сохраняет снапшот метрик в файл.
func (s *MemStorage) saveSnapshotToFile(metrics []model.Metrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	file, err := os.Create(s.filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}
