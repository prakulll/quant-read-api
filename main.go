package main

import (
	"log"
	"net/http"

	"quant-read-api/routes"
	"quant-read-api/services"

	"github.com/gorilla/mux"
)

func main() {
	if err := services.InitClickHouse("clickhouse://localhost:9000/second_data"); err != nil {
		log.Fatal("ClickHouse connection failed:", err)
	}

	r := mux.NewRouter()
	routes.ServeRoutes(r)

	log.Println("Read API running on :8081")

	handler := services.ZstdMiddleware(r)
	log.Fatal(http.ListenAndServe(":8081", handler))
}
