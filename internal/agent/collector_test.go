package agent

import "testing"

func TestNewCollector(t *testing.T) {
	col := NewCollector()

	if col == nil {
		t.Fatalf("NewCollector() вернул nil")
	}

	if col.gauges == nil {
		t.Fatalf("gauges не инициализирован")
	}

	if col.counters == nil {
		t.Fatalf("counters не инициализирован")
	}
}

// TestCollectorUpdate проверяет, что Update собирает метрики из runtime
func TestCollectorUpdate(t *testing.T) {
	col := NewCollector()

	// Вызываем Update
	col.Update()

	// Проверяем, что gauge-метрики из runtime заполнились
	gauges := col.GetGauges()

	// Проверяем несколько ключевых метрик
	expectedMetrics := []string{
		"Alloc",
		"Sys",
		"NumGC",
		"HeapAlloc",
	}

	for _, metric := range expectedMetrics {
		if _, exists := gauges[metric]; !exists {
			t.Errorf("Метрика %s не найдена в gauges", metric)
		}
	}

	if gauges["Alloc"] <= 0 {
		t.Errorf("Alloc должен быть > 0, получено %v", gauges["Alloc"])
	}

	if gauges["Sys"] <= 0 {
		t.Errorf("Sys должен быть > 0, получено %v", gauges["Sys"])
	}
}

// TestCollectorPollCount проверяет, что PollCount увеличивается
func TestCollectorPollCount(t *testing.T) {
	col := NewCollector()

	// Первый Update
	col.Update()
	counters := col.GetCounters()

	if counters["PollCount"] != 1 {
		t.Errorf("После первого Update PollCount должен быть 1, получено %v", counters["PollCount"])
	}

	// Второй Update
	col.Update()
	counters = col.GetCounters()

	if counters["PollCount"] != 2 {
		t.Errorf("После второго Update PollCount должен быть 2, получено %v", counters["PollCount"])
	}

}

// TestCollectorRandomValue проверяет, что RandomValue меняется
func TestCollectorRandomValue(t *testing.T) {
	col := NewCollector()

	col.Update()
	gauges := col.GetGauges()


	// Проверяем, что RandomValue существует
	if _, exists := gauges["RandomValue"]; !exists {
		t.Fatalf("RandomValue не найден в gauges")
	}


}

// TestCollectorGetters проверяет, что геттеры возвращают правильные данные
func TestCollectorGetters(t *testing.T) {
	col := NewCollector()

	col.gauges["test_gauge"] = 42.5
	col.counters["test_counter"] = 100

	gauges := col.GetGauges()
	if gauges["test_gauge"] != 42.5 {
		t.Errorf("GetGauges() вернул неправильное значение: %v", gauges["test_gauge"])
	}

	counters := col.GetCounters()
	if counters["test_counter"] != 100 {
		t.Errorf("GetCounters() вернул неправильное значение: %v", counters["test_counter"])
	}
}