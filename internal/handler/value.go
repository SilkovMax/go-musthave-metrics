package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

type ValueHandler struct {
	storage repository.Storage
}

func NewValueHandler(storage repository.Storage) *ValueHandler {
	return &ValueHandler{storage: storage}
}

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

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%g", value)

	case "counter":
		counters := h.storage.GetAllCounters()

		value, exists := counters[metricName]
		if !exists {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%d", value)

	default:
		http.Error(w, "Invalid metric type", http.StatusNotFound)
		return
	}
}