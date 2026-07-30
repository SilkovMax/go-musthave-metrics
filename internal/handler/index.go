package handler

import (
	"fmt"
	"net/http"

	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

type IndexHandler struct {
	storage repository.Storage
}

func NewIndexHandler(storage repository.Storage) *IndexHandler {
	return &IndexHandler{storage: storage}
}

func (h *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	gauges := h.storage.GetAllGauges()
	counters := h.storage.GetAllCounters()

	// Формируем HTML-страницу
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprint(w, "<!DOCTYPE html><html><head><title>Metrics</title></head><body>")
	fmt.Fprint(w, "<h1>Metrics</h1>")

	// Gauge-метрики
	fmt.Fprint(w, "<h2>Gauge</h2>")
	fmt.Fprint(w, "<ul>")
	for name, value := range gauges {
		fmt.Fprintf(w, "<li>%s: %g</li>", name, value)
	}
	fmt.Fprint(w, "</ul>")

	// Counter-метрики
	fmt.Fprint(w, "<h2>Counter</h2>")
	fmt.Fprint(w, "<ul>")
	for name, value := range counters {
		fmt.Fprintf(w, "<li>%s: %d</li>", name, value)
	}
	fmt.Fprint(w, "</ul>")

	fmt.Fprint(w, "</body></html>")
}
