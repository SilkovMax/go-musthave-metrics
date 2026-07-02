package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"


	"github.com/SilkovMax/go-musthave-metrics/internal/handler"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

func main() {
	storage := repository.NewMemStorage()

	r :=chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", handler.NewUpdateHandler(storage).ServeHTTP)

	r.Get("/value/{type}/{name}", handler.NewValueHandler(storage).ServeHTTP)

	r.Get("/", handler.NewIndexHandler(storage).ServeHTTP)




	fmt.Println("Сервер запущен на http://localhost:8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		fmt.Printf("Ошибка запуска сервера: %v\n", err)
	}
}