package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

// helper создаёт chi-роутер с зарегистрированным IndexHandler
func setupIndexRouter(storage repository.Storage) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", NewIndexHandler(storage).ServeHTTP)
	return r
}

// TestIndexHandlerEmpty проверяет главную страницу без метрик
func TestIndexHandlerEmpty(t *testing.T) {
	storage := repository.NewMemStorage("", 0, false)
	router := setupIndexRouter(storage)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус 200, получен %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Ожидался Content-Type text/html, получен %s", contentType)
	}


}

// TestIndexHandlerWithMetrics проверяет главную страницу с метриками
func TestIndexHandlerWithMetrics(t *testing.T) {
	storage := repository.NewMemStorage("", 0, false)
	storage.SetGauge("cpu", 45.5)
	storage.IncrementCounter("requests", 10)

	router := setupIndexRouter(storage)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус 200, получен %d", w.Code)
	}

	body := w.Body.String()

	if !strings.Contains(body, "cpu") {
		t.Errorf("Ожидалось имя метрики 'cpu' в HTML, получено: %s", body)
	}
	if !strings.Contains(body, "requests") {
		t.Errorf("Ожидалось имя метрики 'requests' в HTML, получено: %s", body)
	}
}