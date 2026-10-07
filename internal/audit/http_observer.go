package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// наблюдатель, который отправляет события по HTTP POST
type HTTPObserver struct {
	url    string
	client *http.Client
}

func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send отправляет событие по HTTP POST.
func (o *HTTPObserver) Send(event AuditEvent) error {
	// Сериализуем событие в JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("ошибка сериализации события аудита: %w", err)
	}

	// Отправляем POST-запрос
	resp, err := o.client.Post(o.url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("ошибка отправки события аудита: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус код
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("аудит сервер вернул статус: %d", resp.StatusCode)
	}

	return nil
}
