package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/SilkovMax/go-musthave-metrics/internal/agent"
)

func main() {

	address := flag.String("a", "localhost:8080", "address and port to run server")
	reportInterval := flag.Int("r", 10, "report interval in seconds")
	pollInterval := flag.Int("p", 2, "poll interval in seconds")


	flag.Parse()

	// Валидация интервалов
	if *reportInterval <= 0 || *pollInterval <= 0 {
		fmt.Println("Интервалы должны быть больше нуля")
		return
	}


	col := agent.NewCollector()


	serverURL := fmt.Sprintf("http://%s", *address)
	client := agent.NewClient(serverURL)


	pollDuration := time.Duration(*pollInterval) * time.Second
	reportDuration := time.Duration(*reportInterval) * time.Second


	// Создаем два тикера
	pollTicker := time.NewTicker(pollDuration)
	reportTicker := time.NewTicker(reportDuration)

	defer pollTicker.Stop()
	defer reportTicker.Stop()


	fmt.Printf("Агент запущен. Сервер: %s, poll: %ds, report: %ds\n", *address, *pollInterval, *reportInterval)

	for {
		select {
		case <-pollTicker.C:
			col.Update()
			fmt.Println("Метрики обновлены")

		case <-reportTicker.C:
			fmt.Println("Отправка метрик на сервер")

			for name, value := range col.GetGauges() {
				err := client.SendGauge(name, value)
				if err != nil {
					fmt.Printf("Ошибка отправки gauge %s: %v\n", name, err)
				}
			}

			for name, value := range col.GetCounters() {
				err := client.SendCounter(name, value)
				if err != nil {
					fmt.Printf("Ошибка отправки counter %s: %v\n", name, err)
				}
			}

			fmt.Println("Метрики отправлены")
		}
	}


}