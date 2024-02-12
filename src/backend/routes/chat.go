// src/backend/routes/chat.go

package routes

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"

	"github.com/gorilla/mux"
)

// Adjust these paths as necessary
const pythonExecPath = "/mnt/c/Users/hwpDesktop/Documents/Content/Repos/EloraChat/python/venv/bin/python3"

var fetchChatScript = filepath.Join("/mnt/c/Users/hwpDesktop/Documents/Content/Repos/EloraChat/python", "fetch_chat.py")

// fetchTwitchChat fetches chat messages for the Twitch user "Dayoman"
func fetchTwitchChat(w http.ResponseWriter, r *http.Request) {
	log.Println("Fetching Twitch chat messages")

	// Command execution setup
	cmd := exec.Command(pythonExecPath, fetchChatScript, "https://www.twitch.tv/johnstone", "twitch")

	var output bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("Error fetching Twitch chat messages: %v, STDERR: %s", err, stderr.String())
		// Handle error by ensuring HTMX attributes are still set for polling
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div id="twitch-chat" hx-get="/twitch" hx-trigger="every 5s" hx-swap="outerHTML">Error fetching messages. Retrying...</div>`)
		return
	}

	// If the command succeeds, return the chat messages with HTMX attributes for the next cycle
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<div id="twitch-chat" hx-get="/twitch" hx-trigger="every 5s" hx-swap="outerHTML">`)
	fmt.Fprint(w, output.String())
	fmt.Fprint(w, "</div>")
}

// fetchYoutubeChat fetches chat messages for the YouTube channel "Dayoman"
func fetchYoutubeChat(w http.ResponseWriter, r *http.Request) {
	log.Println("Fetching YouTube chat messages")

	// Command execution setup
	cmd := exec.Command(pythonExecPath, fetchChatScript, "https://www.youtube.com/watch?v=XyIun_e19qU", "youtube")

	var output bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("Error fetching YouTube chat messages: %v, STDERR: %s", err, stderr.String())
		// Handle error by ensuring HTMX attributes are still set for polling
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div id="youtube-chat" hx-get="/youtube" hx-trigger="every 5s" hx-swap="outerHTML">Error fetching messages. Retrying...</div>`)
		return
	}

	// If the command succeeds, return the chat messages with HTMX attributes for the next cycle
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<div id="youtube-chat" hx-get="/youtube" hx-trigger="every 5s" hx-swap="outerHTML">`)
	fmt.Fprint(w, output.String())
	fmt.Fprint(w, "</div>")
}

// SetupChatRoutes sets up routes to fetch chat messages from Twitch and YouTube
func SetupChatRoutes(router *mux.Router) {
	router.HandleFunc("/twitch", fetchTwitchChat).Methods("GET")
	router.HandleFunc("/youtube", fetchYoutubeChat).Methods("GET")
}
