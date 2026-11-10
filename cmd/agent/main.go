package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"

	"strconv"
	"sync"
	"time"
	"log"

	"github.com/SilkovMax/go-musthave-metrics/internal/agent"
	"github.com/SilkovMax/go-musthave-metrics/internal/model"

	"os"
)

// Глобальные переменные для информации о сборке.
var (
	buildVersion  string
	buildDate     string
	buildCommit   string
)

// Структура для парсинга JSON-конфига агента.
type AgentConfigJSON struct {
	Address        *string `json:"address"`
	ReportInterval *string `json:"report_interval"`
	PollInterval   *string `json:"poll_interval"`
	Key            *string `json:"key"`
	RateLimit      *int    `json:"rate_limit"`
	CryptoKey      *string `json:"crypto_key"`
}

// Проверяет, был ли флаг передан явно
func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// Приоритет для строк: Флаг > Env > JSON > Default
func resolveString(flagVal, envName string, jsonVal *string, defaultVal string, flagName string) string {
	if isFlagSet(flagName) {
		return flagVal
	}
	if envVal := os.Getenv(envName); envVal != "" {
		return envVal
	}
	if jsonVal != nil {
		return *jsonVal
	}
	return defaultVal
}

// Приоритет для int: Флаг > Env > JSON > Default
func resolveInt(flagVal int, envName string, jsonVal *int, defaultVal int, flagName string) int {
	if isFlagSet(flagName) {
		return flagVal
	}
	if envVal := os.Getenv(envName); envVal != "" {
		if v, err := strconv.Atoi(envVal); err == nil {
			return v
		}
	}
	if jsonVal != nil {
		return *jsonVal
	}
	return defaultVal
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
	flagConfig := flag.String("config", "", "path to JSON config file")

	flag.Parse()

	// Определяем путь к конфигу (приоритет: Флаг > Env)
	configPath := ""
	if isFlagSet("config") {
		configPath = *flagConfig
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
	address := resolveString(*flagAddress, "ADDRESS", jsonCfg.Address, "localhost:8080", "a")
	reportIntervalStr := resolveString(*flagReportInterval, "REPORT_INTERVAL", jsonCfg.ReportInterval, "10s", "r")
	pollIntervalStr := resolveString(*flagPollInterval, "POLL_INTERVAL", jsonCfg.PollInterval, "2s", "p")
	key := resolveString(*flagKey, "KEY", jsonCfg.Key, "", "k")
	rateLimit := resolveInt(*flagRateLimit, "RATE_LIMIT", jsonCfg.RateLimit, 1, "l")
	cryptoKeyPath := resolveString(*flagCryptoKey, "CRYPTO_KEY", jsonCfg.CryptoKey, "", "crypto-key")

	// Парсим интервалы из строки в time.Duration
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
		fmt.Printf("Ошибка загрузки публичного ключа: %v\n", err)
		// Не делаем Fatal, чтобы агент мог работать без шифрования, если ключ не указан
	} else if pubKey != nil {
		fmt.Println("Публичный ключ успешно загружен. Шифрование включено.")
	}

	col := agent.NewCollector()

	serverURL := fmt.Sprintf("http://%s", address)
	client := agent.NewClient(serverURL, key, pubKey)

	pollDuration := pollInterval
	reportDuration := reportInterval

	// буферизованный в 100 штук, не нашел в задании сколько нужно
	jobs := make(chan []model.Metrics, 100)

	var wg sync.WaitGroup

	for i := 1; i <= rateLimit; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			// Воркер читает батчи из канала и отправляет их
			for batch := range jobs {
				if err := client.SendBatch(batch); err != nil {
					fmt.Printf("[Worker %d] Ошибка batch-отправки: %v\n", workerID, err)
				} else {
					fmt.Printf("[Worker %d] Отправлено метрик: %d\n", workerID, len(batch))
				}
			}
			fmt.Printf("[Worker %d] Завершён\n", workerID)
		}(i)
	}

	go func() {
		ticker := time.NewTicker(pollDuration)
		defer ticker.Stop()
		for {
			<-ticker.C
			col.Update()
			fmt.Println("Метрики обновлены (runtime)")
		}
	}()

	go func() {
		ticker := time.NewTicker(pollDuration)
		defer ticker.Stop()
		for {
			<-ticker.C
			col.UpdateGopsutil()
			fmt.Println("Метрики обновлены (gopsutil)")
		}
	}()

	reportTicker := time.NewTicker(reportDuration)
	defer reportTicker.Stop()

	fmt.Printf("Агент запущен. Сервер: %s, poll: %v, report: %v, workers: %d\n",
		address, pollInterval, reportInterval, rateLimit)

	for {
		<-reportTicker.C
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

		jobs <- batch
		fmt.Println("Батч добавлен в очередь")
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
