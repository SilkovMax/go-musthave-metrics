package main

import (
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/SilkovMax/go-musthave-metrics/internal/audit"
	"github.com/SilkovMax/go-musthave-metrics/internal/config/db"
	"github.com/SilkovMax/go-musthave-metrics/internal/handler"
	"github.com/SilkovMax/go-musthave-metrics/internal/middleware"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

// Глобальные переменные для информации о сборке.
// Их значения подменяются на этапе линковки через флаг -ldflags "-X main.buildVersion=..."
var (
	buildVersion  string
	buildDate     string
	buildCommit   string
	cryptoKeyPath string
)

var Log *zap.Logger = zap.NewNop()

// getEnv получает значение переменной окружения или возвращает fallback
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// loadPrivateKey загружает и парсит RSA приватный ключ из PEM-файла.
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	if path == "" {
		return nil, nil
	}

	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing private key")
	}

	// Пробуем распарсить как PKCS#1 (BEGIN RSA PRIVATE KEY)
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return privKey, nil
	}

	// Если не вышло, пробуем как PKCS#8 (BEGIN PRIVATE KEY)
	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	if rsaKey, ok := parsedKey.(*rsa.PrivateKey); ok {
		return rsaKey, nil
	}

	return nil, fmt.Errorf("private key is not an RSA key")
}

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

func main() {
	// Выводим информацию о сборке
	printBuildInfo()
	//add flag on address + port
	address := flag.String("a", "localhost:8080", "address and port to run server")
	logLevel := flag.String("l", "info", "log level")

	storeInterval := flag.Int("i", 300, "store interval in seconds")
	fileStoragePath := flag.String("f", "metrics.json", "file storage path")
	restore := flag.Bool("r", false, "reatore metrics from file on start app")
	auditFile := flag.String("audit-file", "", "path to audit log file")
	auditURL := flag.String("audit-url", "", "URL to send audit events")

	databaseDSN := flag.String("d", "", "connect to DB")

	hashKey := flag.String("k", "", "HMAC key for signing requests")
	flag.StringVar(&cryptoKeyPath, "crypto-key", getEnv("CRYPTO_KEY", ""), "path to private key file")

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

	if envInterval := os.Getenv("STORE_INTERVAL"); envInterval != "" {
		if val, err := strconv.Atoi(envInterval); err == nil {
			*storeInterval = val
		}
	}

	if envPath := os.Getenv("FILE_STORAGE_PATH"); envPath != "" {
		*fileStoragePath = envPath
	}

	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if val, err := strconv.ParseBool(envRestore); err == nil {
			*restore = val
		}
	}

	if envDSN := os.Getenv("DATABASE_DSN"); envDSN != "" {
		*databaseDSN = envDSN
	}

	if envKey := os.Getenv("KEY"); envKey != "" {
		*hashKey = envKey
	}

	if envAuditFile := os.Getenv("AUDIT_FILE"); envAuditFile != "" {
		*auditFile = envAuditFile
	}

	if envAuditURL := os.Getenv("AUDIT_URL"); envAuditURL != "" {
		*auditURL = envAuditURL
	}

	defer func() {
		if err := Log.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to sync logger: %v\n", err)
		}
	}()

	// Загрузка приватного ключа
	privKey, err := loadPrivateKey(cryptoKeyPath)
	if err != nil {
		Log.Fatal("Ошибка загрузки приватного ключа", zap.Error(err))
	}
	if privKey != nil {
		Log.Info("Приватный ключ успешно загружен. Шифрование включено.")
	}

	var sqlDB *sql.DB

	var storage repository.Storage

	if *databaseDSN != "" {
		var dbErr error
		sqlDB, dbErr = db.New(*databaseDSN)
		if dbErr != nil {
			Log.Fatal("Ошибка подключения к базе данных", zap.Error(dbErr))
		}
		defer func() {
			if closeErr := sqlDB.Close(); closeErr != nil {
				Log.Error("Ошибка закрытия БД", zap.Error(closeErr))
			}
		}()
		storage = repository.NewDBStorage(sqlDB)
		Log.Info("Успешно подключено к PostgreSQL")
	} else {

		intervalDuration := time.Duration(*storeInterval) * time.Second
		memStorage := repository.NewMemStorage(*fileStoragePath, intervalDuration, *restore)

		storage = memStorage
		defer memStorage.Stop()
		Log.Info("Используется файловое хранилище")
	}

	// СОЗДАНИЕ AUDIT MANAGER!
	var auditManager *audit.Manager

	if *auditFile != "" || *auditURL != "" {
		auditManager = audit.NewManager()

		// Добавляем FileObserver если указан файл
		if *auditFile != "" {
			fileObserver, fileErr := audit.NewFileObserver(*auditFile)
			if fileErr != nil {
				Log.Error("Ошибка создания FileObserver", zap.Error(fileErr))
			} else {
				auditManager.AddObserver(fileObserver)
				Log.Info("FileObserver добавлен", zap.String("file", *auditFile))
			}
		}

		// Добавляем HTTPObserver если указан URL
		if *auditURL != "" {
			httpObserver := audit.NewHTTPObserver(*auditURL)
			auditManager.AddObserver(httpObserver)
			Log.Info("HTTPObserver добавлен", zap.String("url", *auditURL))
		}

		// Откладываем остановку менеджера
		defer auditManager.Stop()
	}

	r := chi.NewRouter()

	if privKey != nil {
		r.Use(middleware.DecryptMiddleware(privKey))
	}

	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.HashRequestMiddleware(*hashKey))
	r.Use(middleware.HashResponseMiddleware(*hashKey))

	//запускаю логировангие для каждого запроса
	r.Use(middleware.LoggingMiddleware(Log))

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		if sqlDB != nil {
			if err := sqlDB.Ping(); err != nil {
				http.Error(w, "database ping failed", http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	r.Post("/update", handler.NewUpdateJSONHandler(storage, auditManager).ServeHTTP)
	r.Post("/update/", handler.NewUpdateJSONHandler(storage, auditManager).ServeHTTP) //для автотеста

	r.Post("/updates", handler.NewUpdatesHandler(storage, auditManager).ServeHTTP)
	r.Post("/updates/", handler.NewUpdatesHandler(storage, auditManager).ServeHTTP)

	r.Post("/value", handler.NewValueJSONHandler(storage).ServeHTTP)
	r.Post("/value/", handler.NewValueJSONHandler(storage).ServeHTTP) //для автотеста

	r.Post("/update/{type}/{name}/{value}", handler.NewUpdateHandler(storage, auditManager).ServeHTTP)

	r.Get("/value/{type}/{name}", handler.NewValueHandler(storage).ServeHTTP)

	r.Get("/", handler.NewIndexHandler(storage).ServeHTTP)

	Log.Info("Сервер запущен", zap.String("address", *address))

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil)) // pprof endpoint
	}()

	if listenErr := http.ListenAndServe(*address, r); listenErr != nil {
		Log.Fatal("Ошибка запуска сервера", zap.Error(listenErr))
	}
}

// printBuildInfo выводит информацию о версии сборки в stdout.
func printBuildInfo() {
	version := buildVersion
	if version == "" {
		version = "N/A"
	}

	date := buildDate
	if date == "" {
		date = "N/A"
	}

	commit := buildCommit
	if commit == "" {
		commit = "N/A"
	}

	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
}
