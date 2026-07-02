package repository

import "testing"

func TestNewMemStorage(t *testing.T) {
	storage := NewMemStorage()

	if storage == nil {
		t.Fatalf("NewMemStorage() вернул nil")
	}

	if storage.gauges == nil {
		t.Fatalf("gauges не инициализирован")
	}
	if storage.counters == nil {
		t.Fatalf("counters не инициализирован")
	}
}

func TestSetGauge(t *testing.T) {
	storage := NewMemStorage()

	storage.SetGauge("cpu", 45.5)

	if storage.gauges["cpu"] != 45.5 {
		t.Errorf("Ожидалось 45.5, получено %v", storage.gauges["cpu"])
	}
}

func TestSetGaugeOverwrite(t *testing.T) {
	storage := NewMemStorage()

	storage.SetGauge("cpu", 45.5)

	storage.SetGauge("cpu", 50.0)

	if storage.gauges["cpu"] != 50.0 {
		t.Errorf("Ожидалось 50.0, получено %v", storage.gauges["cpu"])
	}
}

func TestIncrementCounter(t *testing.T) {
	storage := NewMemStorage()

	storage.IncrementCounter("requests", 1)

	if storage.counters["requests"] != 1 {
		t.Errorf("Ожидалось 1, получено %v", storage.counters["requests"])
	}

	storage.IncrementCounter("requests", 5)

	if storage.counters["requests"] != 6 {
		t.Errorf("Ожидалось 6, получено %v", storage.counters["requests"])
	}
}

func TestMultipleMetrics(t *testing.T) {
	storage := NewMemStorage()

	storage.SetGauge("cpu", 45.5)
	storage.SetGauge("memory", 80.2)
	storage.SetGauge("temperature", 36.6)

	storage.IncrementCounter("requests", 10)
	storage.IncrementCounter("errors", 2)

	if storage.gauges["cpu"] != 45.5 {
		t.Errorf("cpu: ожидалось 45.5, получено %v", storage.gauges["cpu"])
	}
	if storage.gauges["memory"] != 80.2 {
		t.Errorf("memory: ожидалось 80.2, получено %v", storage.gauges["memory"])
	}
	if storage.gauges["temperature"] != 36.6 {
		t.Errorf("temperature: ожидалось 36.6, получено %v", storage.gauges["temperature"])
	}

	if storage.counters["requests"] != 10 {
		t.Errorf("requests: ожидалось 10, получено %v", storage.counters["requests"])
	}
	if storage.counters["errors"] != 2 {
		t.Errorf("errors: ожидалось 2, получено %v", storage.counters["errors"])
	}
}

func TestGetAllGauges(t *testing.T) {
	storage := NewMemStorage()

	storage.SetGauge("cpu", 45.5)
	storage.SetGauge("memory", 80.2)

	gauges := storage.GetAllGauges()

	if len(gauges) != 2 {
		t.Errorf("Ожидалось 2 gauge-метрики, получено %d", len(gauges))
	}

	if gauges["cpu"] != 45.5 {
		t.Errorf("cpu: ожидалось 45.5, получено %v", gauges["cpu"])
	}
	if gauges["memory"] != 80.2 {
		t.Errorf("memory: ожидалось 80.2, получено %v", gauges["memory"])
	}
}

func TestGetAllCounters(t *testing.T) {
	storage := NewMemStorage()

	storage.IncrementCounter("requests", 10)
	storage.IncrementCounter("errors", 2)

	counters := storage.GetAllCounters()

	if len(counters) != 2 {
		t.Errorf("Ожидалось 2 counter-метрики, получено %d", len(counters))
	}

	if counters["requests"] != 10 {
		t.Errorf("requests: ожидалось 10, получено %v", counters["requests"])
	}
	if counters["errors"] != 2 {
		t.Errorf("errors: ожидалось 2, получено %v", counters["errors"])
	}
}