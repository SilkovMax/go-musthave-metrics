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

type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64

	filepath string
	interval time.Duration
	stopCh   chan struct{}
}

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

func (s *MemStorage) Stop() {
	close(s.stopCh)

}

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

func (s *MemStorage) GetAllGauges() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]float64, len(s.gauges))
	for k, v := range s.gauges {
		result[k] = v
	}

	return result
}

func (s *MemStorage) GetAllCounters() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int64, len(s.counters))
	for k, v := range s.counters {
		result[k] = v
	}

	return result
}

func (s *MemStorage) GetGauge(name string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.gauges[name]
	if !ok {
		return 0, errors.New("metric not found")
	}
	return val, nil
}

func (s *MemStorage) GetCounter(name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.counters[name]
	if !ok {
		return 0, errors.New("metric not found")
	}
	return val, nil
}

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

func (s *MemStorage) snapshotLocked() []model.Metrics {
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
