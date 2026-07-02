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


	col := agent.NewCollector()


	serverURL := fmt.Sprintf("http://%s", *address)
	client := agent.NewClient(serverURL)


	pollDuration := time.Duration(*pollInterval) * time.Second
	reportDuration := time.Duration(*reportInterval) * time.Second


	// Сколько обновлений должно пройти между отправками
	reportEvery := int(reportDuration / pollDuration)
	counter := 0


	fmt.Printf("Агент запущен. Сервер: %s, poll: %ds, report: %ds\n", *address, *pollInterval, *reportInterval)


	for {
		col.Update()
		fmt.Println("Метрики обновлены")

		counter++

		if counter >= reportEvery {
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

			counter = 0
		}

		time.Sleep(pollDuration)
	}
}