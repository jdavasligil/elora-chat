// src/backend/routes/chat.go

package routes

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"

	"github.com/gorilla/mux"
)

// fetchTwitchChat fetches chat messages for the Twitch user "Dayoman"
func fetchTwitchChat(w http.ResponseWriter, r *http.Request) {
	log.Println("Fetching Twitch chat messages")

	// Command execution setup
	chatDownloaderPath := "/mnt/c/Users/hwpDesktop/Documents/Content/Repos/EloraChat/python/venv/bin/chat_downloader"
	cmd := exec.Command(chatDownloaderPath, "https://www.twitch.tv/johnstone", "--max_messages", "1")

	// Ensure the response is HTML
	w.Header().Set("Content-Type", "text/html")

	// Include HTMX attributes in the response to continue polling
	fmt.Fprint(w, `<div id="twitch-chat" hx-get="/twitch" hx-trigger="every 5s" hx-swap="outerHTML">`)

	// Stream the output of chat_downloader directly to the response
	cmd.Stdout = w
	cmd.Stderr = w // Consider how to handle STDERR in production

	if err := cmd.Run(); err != nil {
		log.Printf("Error fetching Twitch chat messages: %v", err)
		// Optionally write a more user-friendly message to the response
	}

	// Close the div tag for the HTMX container
	fmt.Fprint(w, "</div>")
}

// fetchYoutubeChat fetches chat messages for the YouTube channel "Dayoman"
func fetchYoutubeChat(w http.ResponseWriter, r *http.Request) {
	log.Println("Fetching YouTube chat messages")

	// Command execution setup
	chatDownloaderPath := "/mnt/c/Users/hwpDesktop/Documents/Content/Repos/EloraChat/python/venv/bin/chat_downloader"
	cmd := exec.Command(chatDownloaderPath, "https://www.youtube.com/@drdisrespect", "--max_messages", "1")

	// Ensure the response is HTML
	w.Header().Set("Content-Type", "text/html")

	// Include HTMX attributes in the response to continue polling
	fmt.Fprint(w, `<div id="youtube-chat" hx-get="/youtube" hx-trigger="every 5s" hx-swap="outerHTML">`)

	// Stream the output of chat_downloader directly to the response
	cmd.Stdout = w
	cmd.Stderr = w // Consider how to handle STDERR in production

	if err := cmd.Run(); err != nil {
		log.Printf("Error fetching YouTube chat messages: %v", err)
		// Optionally write a more user-friendly message to the response
	}

	// Close the div tag for the HTMX container
	fmt.Fprint(w, "</div>")
}

// SetupChatRoutes sets up routes to fetch chat messages from Twitch and YouTube
func SetupChatRoutes(router *mux.Router) {
	router.HandleFunc("/twitch", fetchTwitchChat).Methods("GET")
	router.HandleFunc("/youtube", fetchYoutubeChat).Methods("GET")
}
