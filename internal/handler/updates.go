package handler

import (
	"encoding/json"
	"net/http"

	"github.com/SilkovMax/go-musthave-metrics/internal/audit"
	"github.com/SilkovMax/go-musthave-metrics/internal/model"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

// UpdatesHandler обрабатывает POST-запросы на batch-обновление метрик через JSON.
// URL: /updates или /updates/
// Ожидает JSON-массив метрик в теле запроса.
// Все метрики сохраняются в одной транзакции (для DBStorage).
// Требует Content-Type: application/json.
type UpdatesHandler struct {
	storage      repository.Storage
	auditManager *audit.Manager
}

// NewUpdatesHandler создаёт новый хендлер для batch-обновления метрик.
// Если auditManager не nil, отправляет событие аудита с именами всех обновлённых метрик.
func NewUpdatesHandler(storage repository.Storage, auditManager *audit.Manager) *UpdatesHandler {
	return &UpdatesHandler{
		storage:      storage,
		auditManager: auditManager,
	}
}

// ServeHTTP обрабатывает POST-запрос с JSON-массивом метрик.
// Возвращает 200 OK при успехе, 400/405/500 при ошибках.
func (h *UpdatesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	var metrics []model.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		http.Error(w, "empty batch", http.StatusBadRequest)
		return
	}

	if err := h.storage.SetBatch(metrics); err != nil {
		http.Error(w, "storage error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if h.auditManager != nil {
		// Собираем имена всех метрик
		metricNames := make([]string, 0, len(metrics))
		for _, m := range metrics {
			metricNames = append(metricNames, m.ID)
		}

		ipAddress := extractIPAddress(r)
		event := audit.NewAuditEvent(metricNames, ipAddress)
		h.auditManager.Notify(event)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
