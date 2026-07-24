package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"


	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"


	"github.com/SilkovMax/go-musthave-metrics/internal/handler"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"


)


var Log *zap.Logger = zap.NewNop()

// Initialize инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level string) error {
    // преобразуем текстовый уровень логирования в zap.AtomicLevel
    lvl, err := zap.ParseAtomicLevel(level)
    if err != nil {
        return err
    }
    // создаём новую конфигурацию логера
    cfg := zap.NewProductionConfig()
    // устанавливаем уровень
    cfg.Level = lvl
    // создаём логер на основе конфигурации
    zl, err := cfg.Build()
    if err != nil {
        return err
    }
    // устанавливаем синглтон
    Log = zl
    return nil
}


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


func LoggingMiddleware(next http.Handler) http.Handler {
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

		Log.Info("HTTP request data",
			zap.String("uri", r.RequestURI),
			zap.String("method", r.Method),
			zap.Duration("duration", duration),
			zap.Int("status", resData.status),
			zap.Int("size", resData.size),
		)
	})
}



func main() {
	//add flag on address + port
	address := flag.String("a", "localhost:8080", "address and port to run server")
	logLevel := flag.String("l", "info", "log level")

	flag.Parse()

	// add Env and check if ""
	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		*address = envAddress
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		*logLevel = envLogLevel
	}

	if err := Initialize(*logLevel); err != nil {
		panic(fmt.Errorf("failed to initialize logger: %w", err))
	}

	defer Log.Sync()

	storage := repository.NewMemStorage()

	r :=chi.NewRouter()

	//запускаю логировангие для каждого запроса
	r.Use(LoggingMiddleware)

	r.Post("/update/", handler.NewUpdateJSONHandler(storage).ServeHTTP)
	r.Post("/value/", handler.NewValueJSONHandler(storage).ServeHTTP)

	r.Post("/update/{type}/{name}/{value}", handler.NewUpdateHandler(storage).ServeHTTP)

	r.Get("/value/{type}/{name}", handler.NewValueHandler(storage).ServeHTTP)

	r.Get("/", handler.NewIndexHandler(storage).ServeHTTP)




	Log.Info("Сервер запущен", zap.String("address", *address))
	err := http.ListenAndServe(*address, r)
	if err != nil {
		Log.Fatal("Ошибка запуска сервера", zap.Error(err))
	}
}