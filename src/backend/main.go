package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hpwn/EloraChat/src/backend/routes"
	"github.com/joho/godotenv"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, world!")
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Create a new router
	r := mux.NewRouter()

	// Set up routes
	routes.SetupTwitchRoutes(r)

	// Register the helloHandler as a fallback or test route
	r.HandleFunc("/", helloHandler)

	log.Println("Starting server on :8080")
	err = http.ListenAndServe(":8080", r) // Pass the router r here
	if err != nil {
		log.Fatal(err)
	}
}
