package main

import (
	"fmt"
	"net/http"

	"github.com/SilkovMax/go-musthave-metrics/internal/handler"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

func main() {
	storage := repository.NewMemStorage()

	updateHandler := handler.NewUpdateHandler(storage)

	// сразу mux, мне понравилось
	mux := http.NewServeMux()


	mux.Handle("/update/", updateHandler)


	fmt.Println("Сервер запущен на http://localhost:8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Printf("Ошибка запуска сервера: %v\n", err)
	}
}