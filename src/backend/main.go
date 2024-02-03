package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/hpwn/EloraChat/src/backend/auth"
	"github.com/hpwn/EloraChat/src/backend/client"
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

	// Start the Twitch chat client in a separate goroutine
	go func() {
		accessToken, err := auth.GetValidTwitchAccessToken()
		if err != nil {
			log.Fatal("Failed to get valid Twitch access token:", err)
		}
		twitchUsername := os.Getenv("TWITCH_USERNAME") // Retrieve the username from environment variable
		client.StartChatClient(twitchUsername, accessToken)
	}()

	// Register the helloHandler as a fallback or test route
	r.HandleFunc("/", helloHandler)

	log.Println("Starting server on :8080")
	err = http.ListenAndServe(":8080", r) // Pass the router r here
	if err != nil {
		log.Fatal(err)
	}
}
