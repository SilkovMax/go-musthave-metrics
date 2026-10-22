package handler

import (
	"fmt"
	"net/http"

	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

// IndexHandler обрабатывает GET-запросы на главную страницу.
// Возвращает HTML-страницу со списком всех метрик (gauge и counter).
type IndexHandler struct {
	storage repository.Storage
}

// NewIndexHandler создаёт новый хендлер для отображения метрик.
func NewIndexHandler(storage repository.Storage) *IndexHandler {
	return &IndexHandler{storage: storage}
}

// ServeHTTP обрабатывает HTTP-запрос и возвращает HTML со всеми метриками.
// Устанавливает Content-Type: text/html; charset=utf-8.
func (h *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	gauges := h.storage.GetAllGauges()
	counters := h.storage.GetAllCounters()

	// Формируем HTML-страницу
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_,_ = fmt.Fprint(w, "<!DOCTYPE html><html><head><title>Metrics</title></head><body>")
	_,_ = fmt.Fprint(w, "<h1>Metrics</h1>")

	// Gauge-метрики
	_,_ = fmt.Fprint(w, "<h2>Gauge</h2>")
	_,_ = fmt.Fprint(w, "<ul>")
	for name, value := range gauges {
		_, _ = fmt.Fprintf(w, "<li>%s: %g</li>", name, value)
	}
	_,_ = fmt.Fprint(w, "</ul>")

	// Counter-метрики
	_,_ = fmt.Fprint(w, "<h2>Counter</h2>")
	_,_ = fmt.Fprint(w, "<ul>")
	for name, value := range counters {
		_,_ = fmt.Fprintf(w, "<li>%s: %d</li>", name, value)
	}
	_,_ = fmt.Fprint(w, "</ul>")

	_,_ = fmt.Fprint(w, "</body></html>")
}
