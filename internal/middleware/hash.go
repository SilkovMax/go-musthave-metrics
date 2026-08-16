package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
)

// Проверяем HMAC-SHA256 подпись входящего запроса POST
func HashRequestMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))

			requestHash := r.Header.Get("HashSHA256")
			if requestHash == "" {
				http.Error(w, "missing HashSHA256 header", http.StatusBadRequest)
				return
			}

			expectedHash := computeHMAC(body, key)

			if !hmac.Equal([]byte(requestHash), []byte(expectedHash)) {
				http.Error(w, "invalid hash", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Подписываем HMAC-SHA256 тело ответа сервера
func HashResponseMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			crw := &capturingResponseWriter{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
			}

			next.ServeHTTP(crw, r)

			hash := computeHMAC(crw.body.Bytes(), key)

			w.Header().Set("HashSHA256", hash)

			w.Write(crw.body.Bytes())
		})
	}
}

// Вычисляем HMAC-SHA256 от данных с ключом и возвращаем hex
func computeHMAC(data []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

type capturingResponseWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (crw *capturingResponseWriter) Write(b []byte) (int, error) {
	return crw.body.Write(b)
}

func (crw *capturingResponseWriter) WriteHeader(statusCode int) {
	crw.statusCode = statusCode
	crw.ResponseWriter.WriteHeader(statusCode)
}
