package model

import (
	"testing"
)

// TestMetrics_Reset проверяет, что метод Reset() корректно обнуляет все поля.
func TestMetrics_Reset(t *testing.T) {
	// 1. Создаём структуру с ненулевыми значениями
	delta := int64(100)
	value := 42.5
	m := &Metrics{
		ID:    "test_cpu",
		MType: "gauge",
		Delta: &delta,
		Value: &value,
		Hash:  "abc123",
	}

	// 2. Вызываем сгенерированный метод Reset
	m.Reset()

	// 3. Проверяем, что все поля вернулись к нулевым значениям
	if m.ID != "" {
		t.Errorf("ожидали ID = '', получили %q", m.ID)
	}
	if m.MType != "" {
		t.Errorf("ожидали MType = '', получили %q", m.MType)
	}
	if m.Delta == nil || *m.Delta != 0 {
		t.Errorf("ожидали Delta = 0, получили %v", m.Delta)
	}
	if m.Value == nil || *m.Value != 0 {
		t.Errorf("ожидали Value = 0, получили %v", m.Value)
	}
	if m.Hash != "" {
		t.Errorf("ожидали Hash = '', получили %q", m.Hash)
	}
}

// TestMetrics_Reset_NilPointer проверяет, что вызов Reset() на nil не вызывает панику.
func TestMetrics_Reset_NilPointer(t *testing.T) {
	var m *Metrics // m равно nil

	// Если генератор не добавил проверку "if rs == nil", здесь будет паника, и тест упадёт.
	m.Reset()

	// Если мы дошли до этой строки, значит паники не было.
}