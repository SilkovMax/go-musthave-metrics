package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size //  размер ответа
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func LoggingMiddleware(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// По умолчанию 200 OK
			resData := &responseData{
				status: http.StatusOK,
				size:   0,
			}

			lw := &loggingResponseWriter{
				ResponseWriter: w,
				responseData:   resData,
			}

			next.ServeHTTP(lw, r)

			duration := time.Since(start)

			logger.Info("HTTP request data",
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
				zap.Duration("duration", duration),
				zap.Int("status", resData.status),
				zap.Int("size", resData.size),
			)
		})
	}
}
