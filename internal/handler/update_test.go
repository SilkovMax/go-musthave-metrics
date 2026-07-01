package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

// 200
func TestUpdateHandlerSuccessGauge(t *testing.T) {
	storage := repository.NewMemStorage()
	handler := NewUpdateHandler(storage)

	req := httptest.NewRequest("POST", "/update/gauge/cpu/45.5", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус 200, получен %d", w.Code)
	}
}

func TestUpdateHandlerSuccessCounter(t *testing.T) {
	storage := repository.NewMemStorage()
	handler := NewUpdateHandler(storage)

	req := httptest.NewRequest("POST", "/update/counter/requests/5", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус 200, получен %d", w.Code)
	}
}

func TestUpdateHandlerWrongMethod(t *testing.T) {
	storage := repository.NewMemStorage()
	handler := NewUpdateHandler(storage)

	req := httptest.NewRequest("GET", "/update/gauge/cpu/45.5", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Ожидался статус 405, получен %d", w.Code)
	}
}

func TestUpdateHandlerWrongType(t *testing.T) {
	storage := repository.NewMemStorage()
	handler := NewUpdateHandler(storage)

	req := httptest.NewRequest("POST", "/update/invalid/cpu/45.5", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Ожидался статус 400, получен %d", w.Code)
	}
}

func TestUpdateHandlerInvalidValue(t *testing.T) {
	storage := repository.NewMemStorage()
	handler := NewUpdateHandler(storage)

	req := httptest.NewRequest("POST", "/update/gauge/cpu/abc", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Ожидался статус 400, получен %d", w.Code)
	}
}

func TestUpdateHandlerNotFound(t *testing.T) {
	storage := repository.NewMemStorage()
	handler := NewUpdateHandler(storage)

	req := httptest.NewRequest("POST", "/update/gauge", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Ожидался статус 404, получен %d", w.Code)
	}
}