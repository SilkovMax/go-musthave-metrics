package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSendGauge проверяет, что SendGauge отправляет POST на правильный путь
func TestSendGauge(t *testing.T) {
	receivedPath := ""

	// Фейковый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "123")
	err := client.SendGauge("cpu", 45.5)
	if err != nil {
		t.Fatalf("SendGauge вернул ошибку: %v", err)
	}

	if receivedPath != "/update/gauge/cpu/45.5" {
		t.Errorf("Ожидался путь /update/gauge/cpu/45.5, получен %s", receivedPath)
	}
}

// TestSendCounter проверяет, что SendCounter отправляет POST на правильный путь
func TestSendCounter(t *testing.T) {
	receivedPath := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "123")
	err := client.SendCounter("requests", 5)
	if err != nil {
		t.Fatalf("SendCounter вернул ошибку: %v", err)
	}

	if receivedPath != "/update/counter/requests/5" {
		t.Errorf("Ожидался путь /update/counter/requests/5, получен %s", receivedPath)
	}
}
