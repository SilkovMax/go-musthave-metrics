package repository

import "sync"


type MemStorage struct {
	mu sync.RWMutex  // RWMutex лучше, обчного Mu, т.к. читать могут многие с помощью  RW
	gauges map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges: make(map[string]float64),
		counters: make(map[string]int64),
	}
}


func (s *MemStorage) SetGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
}

func (s *MemStorage) IncrementCounter(name string, delta int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += delta

}

func (s *MemStorage) GetAllGauges() map[string]float64  {
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