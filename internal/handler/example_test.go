package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/SilkovMax/go-musthave-metrics/internal/handler"
	"github.com/SilkovMax/go-musthave-metrics/internal/model"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

// ExampleIndexHandler демонстрирует получение HTML-страницы со всеми метриками.
func ExampleIndexHandler() {
	storage := repository.NewMemStorage("", 0, false)
	defer storage.Stop()

	storage.SetGauge("CPU", 45.5)
	pollCount := int64(10)
	storage.IncrementCounter("PollCount", pollCount)

	h := handler.NewIndexHandler(storage)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	fmt.Println(w.Code)
	fmt.Println(w.Header().Get("Content-Type"))
	// Output:
	// 200
	// text/html; charset=utf-8
}

// ExampleUpdateHandler демонстрирует обновление gauge метрики через plain-text URL.
func ExampleUpdateHandler() {
	storage := repository.NewMemStorage("", 0, false)
	defer storage.Stop()

	h := handler.NewUpdateHandler(storage, nil)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/CPU/36.6", nil)

	// ИСПРАВЛЕНО: вручную добавляем chi URL-параметры в контекст запроса
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("type", "gauge")
	rctx.URLParams.Add("name", "CPU")
	rctx.URLParams.Add("value", "36.6")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	val, _ := storage.GetGauge("CPU")
	fmt.Println(w.Code)
	fmt.Println(val)
	// Output:
	// 200
	// 36.6
}

// ExampleUpdateJSONHandler демонстрирует обновление метрики через JSON.
func ExampleUpdateJSONHandler() {
	storage := repository.NewMemStorage("", 0, false)
	defer storage.Stop()

	h := handler.NewUpdateJSONHandler(storage, nil)

	body := `{"id":"Memory","type":"gauge","value":1024.5}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	val, _ := storage.GetGauge("Memory")
	fmt.Println(w.Code)
	fmt.Println(val)
	// Output:
	// 200
	// 1024.5
}

// ExampleUpdatesHandler демонстрирует batch-обновление метрик через JSON.
func ExampleUpdatesHandler() {
	storage := repository.NewMemStorage("", 0, false)
	defer storage.Stop()

	h := handler.NewUpdatesHandler(storage, nil)

	cpuValue := 45.5
	memoryValue := 1024.0
	pollCountDelta := int64(10)

	metrics := []model.Metrics{
		{ID: "CPU", MType: model.Gauge, Value: &cpuValue},
		{ID: "Memory", MType: model.Gauge, Value: &memoryValue},
		{ID: "PollCount", MType: model.Counter, Delta: &pollCountDelta},
	}

	body, _ := json.Marshal(metrics)
	req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	fmt.Println(w.Code)
	fmt.Println(len(storage.GetAllGauges()))
	fmt.Println(len(storage.GetAllCounters()))
	// Output:
	// 200
	// 2
	// 1
}

// ExampleValueHandler демонстрирует получение значения gauge метрики через URL.
func ExampleValueHandler() {
	storage := repository.NewMemStorage("", 0, false)
	defer storage.Stop()

	storage.SetGauge("CPU", 36.6)

	h := handler.NewValueHandler(storage)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/CPU", nil)

	// ИСПРАВЛЕНО: вручную добавляем chi URL-параметры в контекст запроса
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("type", "gauge")
	rctx.URLParams.Add("name", "CPU")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
	// Output:
	// 200
	// 36.6
}

// ExampleValueJSONHandler демонстрирует получение значения метрики через JSON.
func ExampleValueJSONHandler() {
	storage := repository.NewMemStorage("", 0, false)
	defer storage.Stop()

	storage.SetGauge("CPU", 45.5)

	h := handler.NewValueJSONHandler(storage)

	body := `{"id":"CPU","type":"gauge"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	fmt.Println(w.Code)
	fmt.Println(w.Header().Get("Content-Type"))
	// Output:
	// 200
	// application/json
}
