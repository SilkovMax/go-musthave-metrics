package main

import (
	"flag"
	"fmt"

	"strconv"
	"sync"
	"time"

	"github.com/SilkovMax/go-musthave-metrics/internal/agent"
	"github.com/SilkovMax/go-musthave-metrics/internal/model"

	"os"
)

func main() {

	address := flag.String("a", "localhost:8080", "address and port to run server")
	reportInterval := flag.Int("r", 10, "report interval in seconds")
	pollInterval := flag.Int("p", 2, "poll interval in seconds")
	key := flag.String("k", "", "HMAC key for requests")
	rateLimit := flag.Int("l", 1, "rate limit (number of workers)")

	flag.Parse()

	// add Env and check if ""
	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		*address = envAddress
	}

	if envRepInterval := os.Getenv("REPORT_INTERVAL"); envRepInterval != "" {

		if val, err := strconv.Atoi(envRepInterval); err == nil {
			*reportInterval = val

		} else {

			fmt.Println("Некорректное значение переменной , ввели не число")
		}

	}

	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {

		if val, err := strconv.Atoi(envPollInterval); err == nil {
			*pollInterval = val

		} else {

			fmt.Println("Некорректное значение переменной , ввели не число")
		}

	}

	if envKey := os.Getenv("KEY"); envKey != "" {
		*key = envKey
	}

	if envRateLimit := os.Getenv("RATE_LIMIT"); envRateLimit != "" {
		if val, err := strconv.Atoi(envRateLimit); err == nil {
			*rateLimit = val
		} else {
			fmt.Println("Некорректное значение переменной RATE_LIMIT")
		}
	}

	// Валидация интервалов
	if *reportInterval <= 0 || *pollInterval <= 0 {
		fmt.Println("Интервалы должны быть больше нуля")
		return
	}

	col := agent.NewCollector()

	serverURL := fmt.Sprintf("http://%s", *address)
	client := agent.NewClient(serverURL, *key)

	pollDuration := time.Duration(*pollInterval) * time.Second
	reportDuration := time.Duration(*reportInterval) * time.Second

	// буферизованный в 100 штук, не нашел в задании сколько нужно
	jobs := make(chan []model.Metrics, 100)

	var wg sync.WaitGroup

	for i := 1; i <= *rateLimit; i++ {
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

	fmt.Printf("Агент запущен. Сервер: %s, poll: %ds, report: %ds, workers: %d\n",
		*address, *pollInterval, *reportInterval, *rateLimit)

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
