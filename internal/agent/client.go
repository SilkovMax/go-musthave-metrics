package agent

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/SilkovMax/go-musthave-metrics/internal/crypto"
	"github.com/SilkovMax/go-musthave-metrics/internal/model"
)

type Client struct {
	baseURL string
	client  *resty.Client
	key     string
	pubKey  *rsa.PublicKey
}

// Создаем Http клиент с параметрами
func NewClient(baseURL string, key string, pubKey *rsa.PublicKey) *Client {
	return &Client{
		baseURL: baseURL,
		client:  resty.New(),
		key:     key,
		pubKey:  pubKey,
	}
}

// computeHMAC вычисляет SHA256 HMAC от данных и возвращает hex-строку.
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

// withRetryHTTP выполняет retry для временных ошибок.
// Попыток всего 4 (1+3) / Интервалы: 1s,3s,5s
func withRetryHTTP(operationName string, fn func() (*resty.Response, error)) error {
	delays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {
		resp, err := fn()

		if err == nil && resp.StatusCode() == http.StatusOK {
			return nil // Успех
		}

		if !isRetriableHTTP(err, resp) {
			if err != nil {
				return fmt.Errorf("%s: %w", operationName, err)
			}
			return fmt.Errorf("%s: unexpected status code: %d", operationName, resp.StatusCode())
		}

		if attempt < 3 {
			log.Printf("Client.%s: retry %d/3", operationName, attempt+1)
			time.Sleep(delays[attempt])
		} else {
			if err != nil {
				return fmt.Errorf("%s: все retry провалились: %w", operationName, err)
			}
			return fmt.Errorf("%s: unexpected status code: %d после всех retry", operationName, resp.StatusCode())
		}
	}

	return fmt.Errorf("выход из цикла retry для %s", operationName)
}

// SendGauge отправляет gauge-метрику на сервер
func (c *Client) SendGauge(name string, value float64) error {
	url := fmt.Sprintf("%s/update/gauge/%s/%s", c.baseURL, name, strconv.FormatFloat(value, 'f', -1, 64))

	return withRetryHTTP("SendGauge", func() (*resty.Response, error) {
		req := c.client.R().SetHeader("Content-Type", "text/plain")

		if c.key != "" {
			hash := c.computeHMAC([]byte{})
			req.SetHeader("HashSHA256", hash)
		}

		return req.Post(url)
	})
}

// SendCounter отправляет counter-метрику на сервер
func (c *Client) SendCounter(name string, value int64) error {
	url := fmt.Sprintf("%s/update/counter/%s/%s", c.baseURL, name, strconv.FormatInt(value, 10))

	return withRetryHTTP("SendCounter", func() (*resty.Response, error) {
		req := c.client.R().SetHeader("Content-Type", "text/plain")

		if c.key != "" {
			hash := c.computeHMAC([]byte{})
			req.SetHeader("HashSHA256", hash)
		}

		return req.Post(url)
	})
}

// SendMetric отправляет метрику в JSON формате с gzip и опциональным шифрованием
func (c *Client) SendMetric(m model.Metrics) error {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	// 1. Вычисляем хеш от НЕСЖАТЫХ данных.
	// На сервере middleware расшифрует -> распакует gzip -> проверит этот хеш.
	hash := ""
	if c.key != "" {
		hash = c.computeHMAC(jsonData)
	}

	// 2. Сжимаем JSON в буфер в памяти
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(jsonData); err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}
	compressedData := buf.Bytes()

	// 3. Шифруем сжатые данные, если есть публичный ключ
	var payloadToSend []byte
	if c.pubKey != nil {
		encryptedPayload, err := c.encryptPayload(compressedData)
		if err != nil {
			return fmt.Errorf("failed to encrypt metric: %w", err)
		}
		payloadToSend = encryptedPayload
	} else {
		payloadToSend = compressedData
	}

	// 4. Отправляем запрос
	return withRetryHTTP("SendMetric", func() (*resty.Response, error) {
		req := c.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip"). // Указываем, что после расшифровки серверу нужно будет сделать gzip-декомпрессию
			SetHeader("Accept-Encoding", "gzip").
			SetBody(bytes.NewReader(payloadToSend))

		if hash != "" {
			req.SetHeader("HashSHA256", hash)
		}

		return req.Post(c.baseURL + "/update")
	})
}

// SendBatch отправляет батч метрик с gzip и опциональным шифрованием
func (c *Client) SendBatch(metrics []model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	// 1. Хеш от несжатых данных
	hash := ""
	if c.key != "" {
		hash = c.computeHMAC(jsonData)
	}

	// 2. Сжатие
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(jsonData); err != nil {
		return fmt.Errorf("failed to compress batch: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}
	compressedData := buf.Bytes()

	// 3. Шифрование через гибридный метод
	var payloadToSend []byte
	if c.pubKey != nil {
		encryptedPayload, err := c.encryptPayload(compressedData) // 🔄 ВЫЗЫВАЕМ НОВЫЙ МЕТОД
		if err != nil {
			return fmt.Errorf("failed to encrypt batch: %w", err)
		}
		payloadToSend = encryptedPayload
	} else {
		payloadToSend = compressedData
	}

	// 4. Отправка
	return withRetryHTTP("SendBatch", func() (*resty.Response, error) {
		req := c.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(bytes.NewReader(payloadToSend))

		if hash != "" {
			req.SetHeader("HashSHA256", hash)
		}

		return req.Post(c.baseURL + "/updates/")
	})
}

// encryptPayload реализует гибридное шифрование: шифрует данные через AES-GCM, а сам AES-ключ шифрует через RSA-OAEP.
func (c *Client) encryptPayload(data []byte) ([]byte, error) {
	// 1. Генерируем случайный AES-ключ (32 байта = 256 бит)
	aesKey := make([]byte, 32)
	if _, genErr := rand.Read(aesKey); genErr != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", genErr)
	}

	// 2. Создаём AES-шифр
	block, cipherErr := aes.NewCipher(aesKey)
	if cipherErr != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", cipherErr)
	}

	// 3. Используем режим AES-GCM (Authenticated Encryption)
	aesGCM, gcmErr := cipher.NewGCM(block)
	if gcmErr != nil {
		return nil, fmt.Errorf("failed to create AES-GCM: %w", gcmErr)
	}

	// 4. Генерируем уникальный nonce (вектор инициализации)
	nonce := make([]byte, aesGCM.NonceSize())
	if _, nonceErr := rand.Read(nonce); nonceErr != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", nonceErr)
	}

	// 5. Шифруем данные (метод Seal автоматически добавляет nonce в начало результата)
	encryptedData := aesGCM.Seal(nonce, nonce, data, nil)

	// 6. Шифруем сам AES-ключ через RSA-OAEP
	encryptedKey, rsaErr := rsa.EncryptOAEP(sha256.New(), rand.Reader, c.pubKey, aesKey, nil)
	if rsaErr != nil {
		return nil, fmt.Errorf("failed to encrypt AES key with RSA: %w", rsaErr)
	}

	// 7. Упаковываем всё в JSON-конверт и кодируем в base64
	envelope := crypto.Envelope{
		EncryptedKey:  base64.StdEncoding.EncodeToString(encryptedKey),
		EncryptedData: base64.StdEncoding.EncodeToString(encryptedData),
	}

	return json.Marshal(envelope)
}
