package main

import (
	"fmt"
	"time"

	"github.com/SilkovMax/go-musthave-metrics/internal/agent"
)

func main() {

	col := agent.NewCollector()


	client := agent.NewClient("http://localhost:8080")


	pollInterval := 2 * time.Second
	reportInterval := 10 * time.Second

	// Сколько обновлений должно пройти между отправками
	reportEvery := int(reportInterval / pollInterval)
	counter := 0

	fmt.Println("Агент запущен")

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

		time.Sleep(pollInterval)
	}
}