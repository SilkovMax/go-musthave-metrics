package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/SilkovMax/go-musthave-metrics/internal/model"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

// ValueHandler обрабатывает GET-запросы на получение значения метрики через URL.
// URL формат: /value/{type}/{name}
// Возвращает значение метрики в теле ответа как plain-text.
type ValueHandler struct {
	storage repository.Storage
}

// NewValueHandler создаёт новый хендлер для получения значения метрики через URL-параметры.
func NewValueHandler(storage repository.Storage) *ValueHandler {
	return &ValueHandler{storage: storage}
}

// ServeHTTP обрабатывает GET-запрос и возвращает значение метрики.
// Возвращает 200 OK с значением при успехе, 404 если метрика не найдена.
func (h *ValueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	switch metricType {
	case "gauge":
		gauges := h.storage.GetAllGauges()

		// Проверяем, есть ли нужная метрика
		value, exists := gauges[metricName]
		if !exists {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%g", value)

	case "counter":
		counters := h.storage.GetAllCounters()

		value, exists := counters[metricName]
		if !exists {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%d", value)

	default:
		http.Error(w, "Invalid metric type", http.StatusNotFound)
		return
	}
}

// ValueJSONHandler обрабатывает POST-запросы на получение значения метрики через JSON.
// URL: /value или /value/
// Ожидает JSON с полями: id и type. Возвращает JSON с id, type и value/delta.
// Требует Content-Type: application/json.
type ValueJSONHandler struct {
	storage repository.Storage
}

// NewValueJSONHandler создаёт новый хендлер для получения значения метрики через JSON.
func NewValueJSONHandler(storage repository.Storage) *ValueJSONHandler {
	return &ValueJSONHandler{storage: storage}
}

// ServeHTTP обрабатывает POST-запрос с JSON-телом для получения значения метрики.
// Возвращает 200 OK с JSON-ответом при успехе, 400/404/415 при ошибках.
func (h *ValueJSONHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	var req model.Metrics
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp := model.Metrics{
		ID:    req.ID,
		MType: req.MType,
	}

	switch req.MType {
	case model.Gauge:
		val, err := h.storage.GetGauge(req.ID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp.Value = &val
	case model.Counter:
		val, err := h.storage.GetCounter(req.ID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp.Delta = &val
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
