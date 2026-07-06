package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"


	"github.com/SilkovMax/go-musthave-metrics/internal/handler"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

func main() {
	//add flag on adress + port
	address := flag.String("a", "localhost:8080", "address and port to run server")

	flag.Parse()

	storage := repository.NewMemStorage()

	r :=chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", handler.NewUpdateHandler(storage).ServeHTTP)

	r.Get("/value/{type}/{name}", handler.NewValueHandler(storage).ServeHTTP)

	r.Get("/", handler.NewIndexHandler(storage).ServeHTTP)




	fmt.Printf("Сервер запущен на http://%s\n", *address)
	err := http.ListenAndServe(*address, r)
	if err != nil {
		fmt.Printf("Ошибка запуска сервера: %v\n", err)
	}
}