package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SilkovMax/go-musthave-metrics/internal/audit"
	"github.com/SilkovMax/go-musthave-metrics/internal/model"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

type UpdateHandler struct {
	storage      repository.Storage
	auditManager *audit.Manager
}

func NewUpdateHandler(storage repository.Storage, auditManager *audit.Manager) *UpdateHandler {
	return &UpdateHandler{
		storage:      storage,
		auditManager: auditManager,
	}
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

	if h.auditManager != nil {
		metrics := []string{metricName}
		ipAddress := extractIPAddress(r)
		event := audit.NewAuditEvent(metrics, ipAddress)
		h.auditManager.Notify(event)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}")) //Для автотестов

}

type UpdateJSONHandler struct {
	storage      repository.Storage
	auditManager *audit.Manager
}

func NewUpdateJSONHandler(storage repository.Storage, auditManager *audit.Manager) *UpdateJSONHandler {
	return &UpdateJSONHandler{
		storage:      storage,
		auditManager: auditManager,
	}
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

	if h.auditManager != nil {
		metrics := []string{m.ID}
		ipAddress := extractIPAddress(r)
		event := audit.NewAuditEvent(metrics, ipAddress)
		h.auditManager.Notify(event)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// extractIPAddress извлекает IP-адрес из запроса.
func extractIPAddress(r *http.Request) string {
	// Пробуем получить из заголовка X-Real-IP или X-Forwarded-For (если есть прокси)
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}

	// Иначе берём из RemoteAddr (IP:Port)
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
