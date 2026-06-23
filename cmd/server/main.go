package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"merchant-transactions-api/internal/adapters/httpadapter"
	"merchant-transactions-api/internal/adapters/jsonserver"
	"merchant-transactions-api/internal/adapters/numerator"
	"merchant-transactions-api/internal/app"
)

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	jsonServerURL := envOr("JSON_SERVER_URL", "http://json-server:8080")
	numeratorURL := envOr("NUMERATOR_URL", "http://numerator-api:3000")
	listenAddr := envOr("LISTEN_ADDR", ":4000")

	client := &http.Client{Timeout: 10 * time.Second}
	txRepo := jsonserver.NewClient(jsonServerURL, client)
	receivableRepo := jsonserver.NewClient(jsonServerURL, client)
	numeratorClient, err := numerator.NewClient(numeratorURL, client, 10)
	if err != nil {
		log.Fatal(err)
	}

	orchestrator := app.NewOrchestrator(txRepo, receivableRepo, numeratorClient)
	apiHandler := httpadapter.NewHandler(orchestrator, txRepo, receivableRepo)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: apiHandler.NewMux(),
	}

	log.Printf("Starting orchestration API on %s", listenAddr)
	log.Fatal(server.ListenAndServe())
}
