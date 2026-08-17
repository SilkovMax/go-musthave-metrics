package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SilkovMax/go-musthave-metrics/internal/model"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

type UpdateHandler struct {
	storage repository.Storage
}

func NewUpdateHandler(storage repository.Storage) *UpdateHandler {
	return &UpdateHandler{storage: storage}
}

func (h *UpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	contentType := r.Header.Get("Content-Type")
	if contentType != "" && contentType != "text/plain" {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	// Извлекаем части URL
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	valueStr := chi.URLParam(r, "value")

	// Проверяем есть ли метрика
	if metricName == "" {
		http.Error(w, "Metric name is not found", http.StatusNotFound)
		return
	}

	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			http.Error(w, "Invalid gauge value", http.StatusBadRequest)
			return
		}
		h.storage.SetGauge(metricName, value)

	case "counter":
		value, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid counter value", http.StatusBadRequest)
			return
		}
		h.storage.IncrementCounter(metricName, value)

	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}")) //Для автотестов

}

type UpdateJSONHandler struct {
	storage repository.Storage
}

func NewUpdateJSONHandler(storage repository.Storage) *UpdateJSONHandler {
	return &UpdateJSONHandler{storage: storage}
}

func (h *UpdateJSONHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}

	var m model.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch m.MType {
	case model.Gauge:
		if m.Value != nil {
			h.storage.SetGauge(m.ID, *m.Value)
		}
	case model.Counter:
		if m.Delta != nil {
			h.storage.IncrementCounter(m.ID, *m.Delta)
		}
	default:
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
