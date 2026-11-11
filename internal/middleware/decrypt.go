package middleware

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
)

// Envelope должен совпадать со структурой на стороне агента
type Envelope struct {
	EncryptedKey  string `json:"encrypted_key"`
	EncryptedData string `json:"encrypted_data"`
}

// DecryptMiddleware расшифровывает тело запроса, используя гибридное шифрование (RSA + AES-GCM).
func DecryptMiddleware(privKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Читаем всё тело запроса
			encryptedBody, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}
			_ = r.Body.Close()

			// Пробуем распарсить как Envelope
			var envelope Envelope
			if unmarshalErr := json.Unmarshal(encryptedBody, &envelope); unmarshalErr != nil {
				// Если это не JSON-конверт, значит, данные пришли без шифрования
				r.Body = io.NopCloser(bytes.NewReader(encryptedBody))
				next.ServeHTTP(w, r)
				return
			}

			// 1. Декодируем base64
			encKeyBytes, decodeKeyErr := base64.StdEncoding.DecodeString(envelope.EncryptedKey)
			if decodeKeyErr != nil {
				http.Error(w, "invalid encrypted key format", http.StatusBadRequest)
				return
			}
			encDataBytes, decodeDataErr := base64.StdEncoding.DecodeString(envelope.EncryptedData)
			if decodeDataErr != nil {
				http.Error(w, "invalid encrypted data format", http.StatusBadRequest)
				return
			}

			// 2. Расшифровываем AES-ключ через RSA-OAEP
			aesKey, decryptKeyErr := rsa.DecryptOAEP(sha256.New(), rand.Reader, privKey, encKeyBytes, nil)
			if decryptKeyErr != nil {
				http.Error(w, "failed to decrypt AES key", http.StatusForbidden)
				return
			}

			// 3. Расшифровываем данные через AES-GCM
			block, cipherErr := aes.NewCipher(aesKey)
			if cipherErr != nil {
				http.Error(w, "failed to create AES cipher", http.StatusInternalServerError)
				return
			}
			aesGCM, gcmErr := cipher.NewGCM(block)
			if gcmErr != nil {
				http.Error(w, "failed to create AES-GCM", http.StatusInternalServerError)
				return
			}

			nonceSize := aesGCM.NonceSize()
			if len(encDataBytes) < nonceSize {
				http.Error(w, "ciphertext too short", http.StatusBadRequest)
				return
			}

			nonce, ciphertext := encDataBytes[:nonceSize], encDataBytes[nonceSize:]
			decryptedData, openErr := aesGCM.Open(nil, nonce, ciphertext, nil)
			if openErr != nil {
				http.Error(w, "failed to decrypt data", http.StatusForbidden)
				return
			}

			// 4. Подменяем тело запроса на расшифрованные данные
			r.Body = io.NopCloser(bytes.NewReader(decryptedData))

			next.ServeHTTP(w, r)
		})
	}
}
