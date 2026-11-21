package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"net"
	"os/signal"
	"syscall"

	"log"
	"time"

	"github.com/SilkovMax/go-musthave-metrics/internal/agent"
	"github.com/SilkovMax/go-musthave-metrics/internal/config"
	"github.com/SilkovMax/go-musthave-metrics/internal/model"
	"os"

	"golang.org/x/sync/errgroup"
)

// Глобальные переменные для информации о сборке.
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// BatchSender определяет интерфейс для отправки батчей метрик.
type BatchSender interface {
	SendBatch(metrics []model.Metrics) error
}

// Структура для парсинга JSON-конфига агента.
type AgentConfigJSON struct {
	Address        *string `json:"address"`
	ReportInterval *string `json:"report_interval"`
	PollInterval   *string `json:"poll_interval"`
	Key            *string `json:"key"`
	RateLimit      *int    `json:"rate_limit"`
	CryptoKey      *string `json:"crypto_key"`
	GRPCAddress    *string `json:"grpc_address"`
}

// loadPublicKey загружает и парсит RSA публичный ключ из PEM-файла.
func loadPublicKey(path string) (*rsa.PublicKey, error) {
	if path == "" {
		return nil, nil
	}

	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}

	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	pubKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not an RSA key")
	}

	return pubKey, nil
}

func main() {
	// Выводим информацию о сборке
	printBuildInfo()

	flagAddress := flag.String("a", "localhost:8080", "address and port to run server")
	flagReportInterval := flag.String("r", "10s", "report interval (e.g., 10s)")
	flagPollInterval := flag.String("p", "2s", "poll interval (e.g., 2s)")
	flagKey := flag.String("k", "", "HMAC key for requests")
	flagRateLimit := flag.Int("l", 1, "rate limit (number of workers)")
	flagCryptoKey := flag.String("crypto-key", "", "path to public key file")
	flagGRPCAddress := flag.String("grpc-address", "", "gRPC server address")

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
	var jsonCfg AgentConfigJSON
	if configPath != "" {
		data, readErr := os.ReadFile(configPath)
		if readErr != nil {
			log.Fatalf("Failed to read config file %s: %v", configPath, readErr)
		}
		if unmarshalErr := json.Unmarshal(data, &jsonCfg); unmarshalErr != nil {
			log.Fatalf("Failed to parse config file %s: %v", configPath, unmarshalErr)

		}
	}

	// Применяем приоритеты для всех переменных
	address := config.ResolveString(*flagAddress, "ADDRESS", jsonCfg.Address, "localhost:8080", "a")
	reportIntervalStr := config.ResolveString(*flagReportInterval, "REPORT_INTERVAL", jsonCfg.ReportInterval, "10s", "r")
	pollIntervalStr := config.ResolveString(*flagPollInterval, "POLL_INTERVAL", jsonCfg.PollInterval, "2s", "p")
	key := config.ResolveString(*flagKey, "KEY", jsonCfg.Key, "", "k")
	rateLimit := config.ResolveInt(*flagRateLimit, "RATE_LIMIT", jsonCfg.RateLimit, 1, "l")
	cryptoKeyPath := config.ResolveString(*flagCryptoKey, "CRYPTO_KEY", jsonCfg.CryptoKey, "", "crypto-key")
	grpcAddress := config.ResolveString(*flagGRPCAddress, "GRPC_ADDRESS", jsonCfg.GRPCAddress, "", "grpc-address")

	reportInterval, err := time.ParseDuration(reportIntervalStr)
	if err != nil {
		fmt.Printf("Некорректный формат report_interval: %v\n", err)
		return
	}
	pollInterval, err := time.ParseDuration(pollIntervalStr)
	if err != nil {
		fmt.Printf("Некорректный формат poll_interval: %v\n", err)
		return
	}

	// Валидация интервалов
	if reportInterval <= 0 || pollInterval <= 0 {
		fmt.Println("Интервалы должны быть больше нуля")
		return
	}

	// Загрузка публичного ключа
	pubKey, err := loadPublicKey(cryptoKeyPath)
	if err != nil {
		//Если путь к ключу задан, но загрузка не удалась, должны завершить работу
		log.Fatalf("Ошибка загрузки публичного ключа: %v\n", err)
	} else if pubKey != nil {
		fmt.Println("Публичный ключ успешно загружен. Шифрование включено.")
	}

	// Создаём контекст, который отменится при получении сигнала
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,    // SIGINT (Ctrl+C)
		syscall.SIGTERM, // SIGTERM
		syscall.SIGQUIT, // SIGQUIT
	)
	defer stop() // Освобождаем ресурсы при выходе

	col := agent.NewCollector()

	serverURL := fmt.Sprintf("http://%s", address)

	localIP := getLocalIP()
	fmt.Printf("Локальный IP агента: %s\n", localIP)

	var sender BatchSender

	if grpcAddress != "" {
		// Если задан grpc-адрес, используем grpc клиент
		grpcClient, err := agent.NewGRPCClient(grpcAddress, localIP)
		if err != nil {
			log.Fatalf("Ошибка инициализации grpc клиента: %v", err)
		}
		sender = grpcClient
		fmt.Println("Используется grpc транспорт для отправки метрик")
	} else {
		// Иначе используем HTTP клиент
		sender = agent.NewClient(serverURL, key, pubKey, localIP)
		fmt.Println("Используется HTTP транспорт для отправки метрик")
	}

	pollDuration := pollInterval
	reportDuration := reportInterval

	// буферизованный в 100 штук, не нашел в задании сколько нужно
	jobs := make(chan []model.Metrics, 100)

	// Создаем errgroup, привязанный к контексту сигналов
	g, gCtx := errgroup.WithContext(ctx)

	for i := 1; i <= rateLimit; i++ {
		workerID := i
		g.Go(func() error {
			for {
				select {
				case <-gCtx.Done():
					// Контекст группы отменен
					fmt.Printf("[Worker %d] Завершение из-за отмены контекста\n", workerID)
					return gCtx.Err()

				case batch, ok := <-jobs:
					if !ok {
						// Канал закрыт — штатное завершение
						fmt.Printf("[Worker %d] Завершён\n", workerID)
						return nil
					}

					if err := sender.SendBatch(batch); err != nil {
						fmt.Printf("[Worker %d] Критическая ошибка: %v\n", workerID, err)
						// Возвращаем ошибку — это отменит контекст для всех воркеров
						return fmt.Errorf("worker %d failed: %w", workerID, err)
					}

					fmt.Printf("[Worker %d] Отправлено метрик: %d\n", workerID, len(batch))
				}
			}
		})
	}

	go func() {
		ticker := time.NewTicker(pollDuration)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Poll-горутина (runtime) остановлена")
				return
			case <-ticker.C:
				col.Update()
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(pollDuration)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done(): // Слушаем сигнал завершения
				fmt.Println("Poll-горутина (gopsutil) остановлена")
				return
			case <-ticker.C:
				col.UpdateGopsutil()
			}
		}
	}()

	reportTicker := time.NewTicker(reportDuration)
	defer reportTicker.Stop()

	fmt.Printf("Агент запущен. Сервер: %s, poll: %v, report: %v, workers: %d\n",
		address, pollInterval, reportInterval, rateLimit)

	for {
		select {
		case <-ctx.Done(): // 1. Если пришёл сигнал завершения (Ctrl+C, kill)
			fmt.Println("\nПолучен сигнал завершения. Начинаем graceful shutdown")

			// Закрываем канал. Воркеры, читающие из него, поймут, что новых задач не будет
			close(jobs)

			// Ждём, пока все воркеры закончат отправку текущих данных
			fmt.Println("Ожидаем завершения отправки оставшихся метрик")
			if err := g.Wait(); err != nil {
				fmt.Printf("Ошибка при завершении воркеров: %v\n", err)
			}

			fmt.Println("Агент корректно завершил работу. Все данные отправлены.")
			return // Выходим из main(), программа завершается

		case <-reportTicker.C: // 2. Если пришло время плановой отправки
			fmt.Println("Отправка метрик на сервер (batch)")

			var batch []model.Metrics

			for name, value := range col.GetGauges() {
				v := value
				batch = append(batch, model.Metrics{
					ID:    name,
					MType: model.Gauge,
					Value: &v,
				})
			}

			for name, value := range col.GetCounters() {
				v := value
				batch = append(batch, model.Metrics{
					ID:    name,
					MType: model.Counter,
					Delta: &v,
				})
			}

			// Используем select, чтобы не заблокироваться, если сигнал придёт прямо сейчас
			select {
			case jobs <- batch:
				fmt.Println("Батч добавлен в очередь")
			case <-ctx.Done():
				// Если сигнал пришёл в этот момент, отправляем последний батч синхронно
				fmt.Println("Сигнал получен во время формирования батча. Отправляем синхронно...")
				if err := sender.SendBatch(batch); err != nil {
					fmt.Printf("Ошибка отправки последнего батча: %v\n", err)
				}
				close(jobs)
				if err := g.Wait(); err != nil {
					fmt.Printf("Ошибка при завершении воркеров: %v\n", err)
				}
				fmt.Println("Агент корректно завершил работу.")
				return
			}
		}
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

// getLocalIP пытается найти IP-адрес машины.
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}
