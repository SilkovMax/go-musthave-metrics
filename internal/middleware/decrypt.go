package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"io"
	"net/http"
)

// DecryptMiddleware расшифровывает тело запроса с помощью RSA-OAEP.
func DecryptMiddleware(privKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Читаем всё зашифрованное тело
			encryptedBody, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}
			_ = r.Body.Close()

			// Расшифровываем с использованием OAEP и SHA-256
			decryptedBody, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privKey, encryptedBody, nil)
			if err != nil {
				http.Error(w, "failed to decrypt request body", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(decryptedBody))

			next.ServeHTTP(w, r)
		})
	}
}
