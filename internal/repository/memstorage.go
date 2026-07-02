package repository


type MemStorage struct {
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
	s.gauges[name] = value
}

func (s *MemStorage) IncrementCounter(name string, delta int64) {
	s.counters[name] += delta

}

func (s *MemStorage) GetAllGauges() map[string]float64  {
	return s.gauges
}

func (s *MemStorage) GetAllCounters() map[string]int64 {
	return s.counters
}