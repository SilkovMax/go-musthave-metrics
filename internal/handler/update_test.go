package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

// helper создаёт chi-роутер с зарегистрированным UpdateHandler
func setupUpdateRouter(storage repository.Storage) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", NewUpdateHandler(storage).ServeHTTP)
	return r
}

// TestUpdateHandlerSuccessGauge проверяет успешный запрос gauge
func TestUpdateHandlerSuccessGauge(t *testing.T) {
	storage := repository.NewMemStorage()
	router := setupUpdateRouter(storage)

	req := httptest.NewRequest("POST", "/update/gauge/cpu/45.5", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус 200, получен %d", w.Code)
	}
}

// TestUpdateHandlerSuccessCounter проверяет успешный запрос counter
func TestUpdateHandlerSuccessCounter(t *testing.T) {
	storage := repository.NewMemStorage()
	router := setupUpdateRouter(storage)

	req := httptest.NewRequest("POST", "/update/counter/requests/5", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус 200, получен %d", w.Code)
	}
}

// TestUpdateHandlerWrongMethod проверяет неправильный метод
func TestUpdateHandlerWrongMethod(t *testing.T) {
	storage := repository.NewMemStorage()
	router := setupUpdateRouter(storage)

	req := httptest.NewRequest("GET", "/update/gauge/cpu/45.5", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Ожидался статус 405, получен %d", w.Code)
	}
}

// TestUpdateHandlerWrongType проверяет неправильный тип метрики
func TestUpdateHandlerWrongType(t *testing.T) {
	storage := repository.NewMemStorage()
	router := setupUpdateRouter(storage)

	req := httptest.NewRequest("POST", "/update/invalid/cpu/45.5", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Ожидался статус 400, получен %d", w.Code)
	}
}

// TestUpdateHandlerInvalidValue проверяет некорректное значение
func TestUpdateHandlerInvalidValue(t *testing.T) {
	storage := repository.NewMemStorage()
	router := setupUpdateRouter(storage)

	req := httptest.NewRequest("POST", "/update/gauge/cpu/abc", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Ожидался статус 400, получен %d", w.Code)
	}
}

// TestUpdateHandlerNotFound проверяет неправильный URL
func TestUpdateHandlerNotFound(t *testing.T) {
	storage := repository.NewMemStorage()
	router := setupUpdateRouter(storage)

	req := httptest.NewRequest("POST", "/update/gauge", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Ожидался статус 404, получен %d", w.Code)
	}
}