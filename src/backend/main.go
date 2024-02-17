package main

import (
	"log"
	"net/http"

	"github.com/hpwn/EloraChat/src/backend/routes" // Ensure this is the correct path to your routes package

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	// Set up WebSocket chat routes
	routes.SetupChatRoutes(r)

	// Serve static files from the "public" directory
	fs := http.FileServer(http.Dir("public"))
	r.PathPrefix("/").Handler(http.StripPrefix("/", fs))

	// Start fetching chat messages
	chatURLs := []string{
		//"https://www.twitch.tv/hp_az",
		//"https://www.youtube.com/@hp_az/live",
		"https://www.youtube.com/watch?v=jfKfPfyJRdk",
		"https://www.twitch.tv/Johnstone",
	}
	routes.StartChatFetch(chatURLs)

	// Start the server
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal("Server start error:", err)
	}
}
