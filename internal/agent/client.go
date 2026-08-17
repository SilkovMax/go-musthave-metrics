package agent

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/SilkovMax/go-musthave-metrics/internal/model"
)

type Client struct {
	baseURL string
	client  *resty.Client
	key     string
}

// Создаем Http клиент с параметрами
func NewClient(baseURL string, key string) *Client {
	return &Client{
		baseURL: baseURL,
		client:  resty.New(),
		key:     key,
	}
}

// Вычисляем Sha и выдаем hex
func (c *Client) computeHMAC(data []byte) string {
	mac := hmac.New(sha256.New, []byte(c.key))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

func isRetriableHTTP(err error, resp *resty.Response) bool {
	// Сетевая ошибка
	if err != nil {
		return true
	}

	// HTTP 5xx
	if resp != nil && resp.StatusCode() >= 500 {
		return true
	}

	return false
}

// SendGauge отправляет gauge-метрику на сервер
func (c *Client) SendGauge(name string, value float64) error {
	url := fmt.Sprintf("%s/update/gauge/%s/%s", c.baseURL, name, strconv.FormatFloat(value, 'f', -1, 64))
	delays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {

		req := c.client.R().SetHeader("Content-Type", "text/plain")

		if c.key != "" {
			hash := c.computeHMAC([]byte{}) // пустое тело для GET запросов
			req.SetHeader("HashSHA256", hash)
		}

		resp, err := req.Post(url)

		if err == nil && resp.StatusCode() == 200 {
			return nil
		}

		if !isRetriableHTTP(err, resp) {
			if err != nil {
				return err
			}
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}

		if attempt < 3 {
			time.Sleep(delays[attempt])
		} else {
			if err != nil {
				return fmt.Errorf("все retry провалились: %w", err)
			}
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}
	}

	return fmt.Errorf("выход из цикла retry")
}

// SendCounter отправляет counter-метрику на сервер
func (c *Client) SendCounter(name string, value int64) error {
	url := fmt.Sprintf("%s/update/counter/%s/%s", c.baseURL, name, strconv.FormatInt(value, 10))
	delays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {

		req := c.client.R().SetHeader("Content-Type", "text/plain")

		if c.key != "" {
			hash := c.computeHMAC([]byte{})
			req.SetHeader("HashSHA256", hash)
		}

		// Отправляем запрос
		resp, err := req.Post(url)

		if err == nil && resp.StatusCode() == 200 {
			return nil
		}

		if !isRetriableHTTP(err, resp) {
			if err != nil {
				return err
			}
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}

		if attempt < 3 {
			time.Sleep(delays[attempt])
		} else {
			if err != nil {
				return fmt.Errorf("все retry провалились: %w", err)
			}
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}
	}

	return fmt.Errorf("выход из цикла retry")
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

	compressedData := buf.Bytes()

	delays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {
		req := c.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Accept-Encoding", "gzip").
			SetBody(bytes.NewReader(compressedData))

		if c.key != "" {
			hash := c.computeHMAC(jsonData)
			req.SetHeader("HashSHA256", hash)
		}

		// Отправляем запрос
		resp, err := req.Post(c.baseURL + "/update")

		if err == nil && resp.StatusCode() == http.StatusOK {
			return nil
		}

		if !isRetriableHTTP(err, resp) {
			if err != nil {
				return fmt.Errorf("failed to send request: %w", err)
			}
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}

		if attempt < 3 {
			time.Sleep(delays[attempt])
		} else {
			if err != nil {
				return fmt.Errorf("failed to send request: %w", err)
			}
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}
	}

	return fmt.Errorf("выход из цикла retry")
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

	compressedData := buf.Bytes()

	delays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {
		req := c.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(bytes.NewReader(compressedData))

		if c.key != "" {
			hash := c.computeHMAC(jsonData)
			req.SetHeader("HashSHA256", hash)
		}

		resp, err := req.Post(c.baseURL + "/updates/")

		if err == nil && resp.StatusCode() == http.StatusOK {
			return nil
		}

		if !isRetriableHTTP(err, resp) {
			if err != nil {
				return fmt.Errorf("failed to send batch: %w", err)
			}
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}

		if attempt < 3 {
			fmt.Printf("SendBatch: retry %d/3 (status: %d)\n", attempt+1, resp.StatusCode())
			time.Sleep(delays[attempt])
		} else {
			if err != nil {
				return fmt.Errorf("failed to send batch: %w", err)
			}
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}
	}

	return fmt.Errorf("выход из цикла retry")
}
