package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

// helper создаёт chi-роутер с зарегистрированным ValueHandler
func setupValueRouter(storage repository.Storage) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", NewValueHandler(storage).ServeHTTP)
	return r
}

// TestValueHandlerGaugeFound проверяет получение существующей gauge-метрики
func TestValueHandlerGaugeFound(t *testing.T) {
	storage := repository.NewMemStorage("", 0, false)
	storage.SetGauge("cpu", 45.5)

	router := setupValueRouter(storage)

	req := httptest.NewRequest("GET", "/value/gauge/cpu", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус 200, получен %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "45.5") {
		t.Errorf("Ожидалось значение 45.5 в теле ответа, получено: %s", body)
	}
}

// TestValueHandlerCounterFound проверяет получение существующей counter-метрики
func TestValueHandlerCounterFound(t *testing.T) {
	storage := repository.NewMemStorage("", 0, false)
	storage.IncrementCounter("requests", 10)

	router := setupValueRouter(storage)

	req := httptest.NewRequest("GET", "/value/counter/requests", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус 200, получен %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "10") {
		t.Errorf("Ожидалось значение 10 в теле ответа, получено: %s", body)
	}
}

// TestValueHandlerNotFound проверяет получение несуществующей метрики
func TestValueHandlerNotFound(t *testing.T) {
	storage := repository.NewMemStorage("", 0, false)
	router := setupValueRouter(storage)

	req := httptest.NewRequest("GET", "/value/gauge/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Ожидался статус 404, получен %d", w.Code)
	}
}

// TestValueHandlerWrongType проверяет неправильный тип метрики
func TestValueHandlerWrongType(t *testing.T) {
	storage := repository.NewMemStorage("", 0, false)
	router := setupValueRouter(storage)

	req := httptest.NewRequest("GET", "/value/invalid/cpu", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Ожидался статус 404, получен %d", w.Code)
	}
}