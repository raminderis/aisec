package main

import (
	"log"
	"net/http"
	"os"

	"aisec/controller"

	"github.com/go-chi/chi"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: could not load .env file: %v", err)
	}

	port := getEnv("SECURITY_LISTENING_PORT", "8080")
	addr := ":" + port

	router := chi.NewRouter()
	router.Post("/add-user", controller.AddHandler)
	router.Put("/update-user", controller.UpdateHandler)
	router.Delete("/delete-user", controller.DeleteHandler)
	router.Post("/authenticate-user", controller.AuthenticateHandler)

	log.Printf("security service listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
