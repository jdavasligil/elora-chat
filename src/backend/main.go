package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hpwn/EloraChat/src/backend/routes" // Ensure this is the correct path to your routes package
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Create a new router
	r := mux.NewRouter()

	// Register the chat fetching routes
	routes.SetupChatRoutes(r)

	// Serve static files from the "public" directory
	fs := http.FileServer(http.Dir("public"))
	// Handle all other requests with the file server, which will serve up index.html by default
	r.PathPrefix("/").Handler(http.StripPrefix("/", fs))

	// Start the server
	log.Println("Starting server on :8080")
	err = http.ListenAndServe(":8080", r) // Pass the router r here
	if err != nil {
		log.Fatal(err)
	}
}
