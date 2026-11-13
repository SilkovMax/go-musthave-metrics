package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/SilkovMax/go-musthave-metrics/internal/audit"
	"github.com/SilkovMax/go-musthave-metrics/internal/config"
	"github.com/SilkovMax/go-musthave-metrics/internal/config/db"
	"github.com/SilkovMax/go-musthave-metrics/internal/handler"
	"github.com/SilkovMax/go-musthave-metrics/internal/middleware"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

// Глобальные переменные для информации о сборке.
// Их значения подменяются на этапе линковки через флаг -ldflags "-X main.buildVersion=..."
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

var Log *zap.Logger = zap.NewNop()

// ServerConfigJSON описывает структуру файла конфигурации.
type ServerConfigJSON struct {
	Address         *string `json:"address"`
	StoreInterval   *string `json:"store_interval"`
	FileStoragePath *string `json:"store_file"`
	DatabaseDSN     *string `json:"database_dsn"`
	Restore         *bool   `json:"restore"`
	LogLevel        *string `json:"log_level"`
	Key             *string `json:"key"`
	CryptoKey       *string `json:"crypto_key"`
	AuditFile       *string `json:"audit_file"`
	AuditURL        *string `json:"audit_url"`
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
	flagAddress := flag.String("a", "localhost:8080", "address and port to run server")
	flagLogLevel := flag.String("l", "info", "log level")

	flagStoreInterval := flag.String("i", "300s", "store interval (e.g., 300s, 5m)")
	flagFileStoragePath := flag.String("f", "metrics.json", "file storage path")
	flagRestore := flag.Bool("r", false, "restore metrics from file on start app")
	flagAuditFile := flag.String("audit-file", "", "path to audit log file")
	flagAuditURL := flag.String("audit-url", "", "URL to send audit events")
	flagDatabaseDSN := flag.String("d", "", "connect to DB")
	flagHashKey := flag.String("k", "", "HMAC key for signing requests")
	flagCryptoKey := flag.String("crypto-key", "", "path to private key file")

	var flagConfig string
	flag.StringVar(&flagConfig, "config", "", "path to JSON config file")
	flag.StringVar(&flagConfig, "c", "", "path to JSON config file -c")
	flag.Parse()

	// Определяем путь к конфигу (приоритет: Флаг > Env)
	configPath := ""
	if config.IsFlagSet("config") || config.IsFlagSet("c") {
		configPath = flagConfig
	} else if envConfig := os.Getenv("CONFIG"); envConfig != "" {
		configPath = envConfig
	}

	// Загружаем и парсим JSON, если путь указан
	var jsonCfg ServerConfigJSON
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			log.Fatalf("Failed to read config file %s: %v", configPath, err)
		}
		if err := json.Unmarshal(data, &jsonCfg); err != nil {
			log.Fatalf("Failed to parse config file %s: %v", configPath, err)
		}
	}

	// Применяем приоритеты для всех переменных
	address := config.ResolveString(*flagAddress, "ADDRESS", jsonCfg.Address, "localhost:8080", "a")
	logLevel := config.ResolveString(*flagLogLevel, "LOG_LEVEL", jsonCfg.LogLevel, "info", "l")
	storeIntervalStr := config.ResolveString(*flagStoreInterval, "STORE_INTERVAL", jsonCfg.StoreInterval, "300s", "i")
	fileStoragePath := config.ResolveString(*flagFileStoragePath, "FILE_STORAGE_PATH", jsonCfg.FileStoragePath, "metrics.json", "f")
	restore := config.ResolveBool(*flagRestore, "RESTORE", jsonCfg.Restore, false, "r")
	databaseDSN := config.ResolveString(*flagDatabaseDSN, "DATABASE_DSN", jsonCfg.DatabaseDSN, "", "d")
	hashKey := config.ResolveString(*flagHashKey, "KEY", jsonCfg.Key, "", "k")
	cryptoKeyPath := config.ResolveString(*flagCryptoKey, "CRYPTO_KEY", jsonCfg.CryptoKey, "", "crypto-key")
	auditFile := config.ResolveString(*flagAuditFile, "AUDIT_FILE", jsonCfg.AuditFile, "", "audit-file")
	auditURL := config.ResolveString(*flagAuditURL, "AUDIT_URL", jsonCfg.AuditURL, "", "audit-url")

	// Парсим интервал из строки в time.Duration
	storeInterval, err := time.ParseDuration(storeIntervalStr)
	if err != nil {
		log.Fatalf("Некорректный формат store_interval: %v", err)
	}

	if initErr := Initialize(logLevel); initErr != nil {
		log.Fatalf("failed to initialize logger: %v", initErr)
	}

	defer func() {
		if syncErr := Log.Sync(); syncErr != nil {
			fmt.Fprintf(os.Stderr, "failed to sync logger: %v\n", syncErr)
		}
	}()

	// Загрузка приватного ключа
	privKey, loadErr := loadPrivateKey(cryptoKeyPath)
	if loadErr != nil {
		Log.Fatal("Ошибка загрузки приватного ключа", zap.Error(loadErr))
	}
	if privKey != nil {
		Log.Info("Приватный ключ успешно загружен. Шифрование включено.")
	}

	var sqlDB *sql.DB

	var storage repository.Storage

	if databaseDSN != "" {
		var dbErr error
		sqlDB, dbErr = db.New(databaseDSN)
		if dbErr != nil {
			Log.Fatal("Ошибка подключения к базе данных", zap.Error(dbErr))
		}
		storage = repository.NewDBStorage(sqlDB)
		Log.Info("Успешно подключено к PostgreSQL")
	} else {

		memStorage := repository.NewMemStorage(fileStoragePath, storeInterval, restore)

		storage = memStorage
		defer memStorage.Stop()
		Log.Info("Используется файловое хранилище")
	}

	var auditManager *audit.Manager

	if auditFile != "" || auditURL != "" {
		auditManager = audit.NewManager()

		// Добавляем FileObserver если указан файл
		if auditFile != "" {
			fileObserver, fileErr := audit.NewFileObserver(auditFile)
			if fileErr != nil {
				Log.Error("Ошибка создания FileObserver", zap.Error(fileErr))
			} else {
				auditManager.AddObserver(fileObserver)
				Log.Info("FileObserver добавлен", zap.String("file", auditFile))
			}
		}

		// Добавляем HTTPObserver если указан URL
		if auditURL != "" {
			httpObserver := audit.NewHTTPObserver(auditURL)
			auditManager.AddObserver(httpObserver)
			Log.Info("HTTPObserver добавлен", zap.String("url", auditURL))
		}

	}

	r := chi.NewRouter()

	if privKey != nil {
		r.Use(middleware.DecryptMiddleware(privKey))
	}

	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.HashRequestMiddleware(hashKey))
	r.Use(middleware.HashResponseMiddleware(hashKey))

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

	Log.Info("Сервер запущен", zap.String("address", address))

	// Создаём контекст с сигналами завершения
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()

	//Создаем сервер для pprof для корректного завершения
	pprofSrv := &http.Server{
		Addr: "localhost:6060",
	}

	go func() {
		if err := pprofSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			Log.Error("Ошибка запуска pprof сервера", zap.Error(err))
		}
	}()

	// Создаём http.Server вместо прямого вызова ListenAndServe
	srv := &http.Server{
		Addr:    address,
		Handler: r,
	}

	// Запускаем сервер в горутине
	go func() {
		Log.Info("Сервер запущен", zap.String("address", address))
		if serveErr := srv.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			Log.Fatal("Ошибка запуска сервера", zap.Error(serveErr))
		}
	}()

	// Блокируемся, пока не придёт сигнал
	<-ctx.Done()
	Log.Info("Получен сигнал завершения. Начинаем graceful shutdown...")

	// Создаём контекст с таймаутом для завершения (даём 10 секунд)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Graceful shutdown HTTP-сервера (дождётся завершения текущих запросов)
	if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
		Log.Error("Ошибка при завершении сервера", zap.Error(shutdownErr))
	}

	// Корректная остановка pprof сервера
	if pprofErr := pprofSrv.Shutdown(shutdownCtx); pprofErr != nil {
		Log.Error("Ошибка при завершении pprof сервера", zap.Error(pprofErr))
	} else {
		Log.Info("pprof сервер корректно остановлен")
	}

	// Сохраняем несохранённые данные
	Log.Info("Сохраняем несохранённые данные...")
	if memStorage, ok := storage.(*repository.MemStorage); ok {
		if flushErr := memStorage.SaveToFile(); flushErr != nil {
			Log.Error("Ошибка сохранения данных в файл", zap.Error(flushErr))
		} else {
			Log.Info("Данные успешно сохранены")
		}
	}

	// Останавливаем Audit Manager (если он создан)
	if auditManager != nil {
		auditManager.Stop()
		Log.Info("Audit Manager остановлен")
	}

	// Закрываем соединение с БД (если используется)
	if sqlDB != nil {
		if closeErr := sqlDB.Close(); closeErr != nil {
			Log.Error("Ошибка закрытия БД", zap.Error(closeErr))
		} else {
			Log.Info("Соединение с БД закрыто")
		}
	}

	Log.Info("Сервер корректно завершил работу")
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
