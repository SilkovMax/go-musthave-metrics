package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-resty/resty/v2"

	"github.com/SilkovMax/go-musthave-metrics/internal/model"
)

type Client struct {
	baseURL string
	client  *resty.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client:  resty.New(),
	}
}

// SendGauge отправляет gauge-метрику на сервер
func (c *Client) SendGauge(name string, value float64) error {
	url := fmt.Sprintf("%s/update/gauge/%s/%s", c.baseURL, name, strconv.FormatFloat(value, 'f', -1, 64))

	resp, err := c.client.R().
		SetHeader("Content-Type", "text/plain").
		Post(url)

	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	return nil
}

// SendCounter отправляет counter-метрику на сервер
func (c *Client) SendCounter(name string, value int64) error {
	url := fmt.Sprintf("%s/update/counter/%s/%s", c.baseURL, name, strconv.FormatInt(value, 10))

	resp, err := c.client.R().
		SetHeader("Content-Type", "text/plain").
		Post(url)

	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	return nil
}

// SendMetrics отправляем в Json Формате с gzip
func (c *Client) SendMetric(m model.Metrics) error {

	jsonData, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	// Сжимаем JSON в буфер в памяти
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)

	if _, err := gw.Write(jsonData); err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	if err := gw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetBody(&buf).
		Post(c.baseURL + "/update")

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	return nil
}

func (c *Client) SendBatch(metrics []model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(jsonData); err != nil {
		return fmt.Errorf("failed to compress batch: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(&buf).
		Post(c.baseURL + "/updates/")

	if err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	return nil
}
