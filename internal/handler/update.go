package handler

import (
	"net/http"
	"strconv"
	"strings"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)


type UpdateHandler struct {
	storage *repository.MemStorage
}


func NewUpdateHandler(storage *repository.MemStorage) *UpdateHandler {
	return &UpdateHandler{storage: storage}
}


func (h *UpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "text/plain" {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	// разбиваем на части
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")


	if len(parts) != 4 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Извлекаем части URL
	metricType := parts[1]
	metricName := parts[2]
	valueStr := parts[3]

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

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))


}