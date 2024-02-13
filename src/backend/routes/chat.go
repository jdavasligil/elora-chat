// src/backend/routes/chat.go

package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// Adjust these paths as necessary
const pythonExecPath = "/mnt/c/Users/hwpDesktop/Documents/Content/Repos/EloraChat/python/venv/bin/python3"

var fetchChatScript = filepath.Join("/mnt/c/Users/hwpDesktop/Documents/Content/Repos/EloraChat/python", "fetch_chat.py")

// fetchTwitchChat fetches chat messages for the Twitch user "Dayoman"
func fetchTwitchChat(w http.ResponseWriter, r *http.Request) {
	log.Println("Fetching Twitch chat messages")

	// Command execution setup
	cmd := exec.Command(pythonExecPath, fetchChatScript, "https://www.twitch.tv/hasanabi", "messages")

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

	// Parse the JSON output from the Python script
	var messages []map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &messages); err != nil {
		log.Printf("Error parsing Twitch chat messages: %v", err)
		// Handle error appropriately
		return
	}

	// Convert messages to HTML
	var htmlMessages strings.Builder
	for _, msg := range messages {
		if msg["message_type"] == "text_message" {
			username := msg["author"].(map[string]interface{})["display_name"].(string)
			message := msg["message"].(string)
			htmlMessages.WriteString(fmt.Sprintf("<div class='chat-message'><b>Twitch %s:</b> %s</div>",
				username, message))
		}
	}

	// Return the HTML messages with HTMX attributes for the next cycle
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<div id="twitch-chat" hx-get="/twitch" hx-trigger="every 5s" hx-swap="outerHTML">`)
	fmt.Fprint(w, htmlMessages.String())
	fmt.Fprint(w, "</div>")
}

// fetchYoutubeChat fetches chat messages for the YouTube channel
func fetchYoutubeChat(w http.ResponseWriter, r *http.Request) {
	log.Println("Fetching YouTube chat messages")

	// Command execution setup
	cmd := exec.Command(pythonExecPath, fetchChatScript, "https://www.youtube.com/watch?v=jfKfPfyJRdk", "messages")

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

	// Parse the JSON output from the Python script
	var messages []map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &messages); err != nil {
		log.Printf("Error parsing YouTube chat messages: %v", err)
		// Handle error appropriately
		return
	}

	// Convert messages to HTML
	var htmlMessages strings.Builder
	for _, msg := range messages {
		if msg["message_type"] == "text_message" {
			username := msg["author"].(map[string]interface{})["name"].(string)
			message := msg["message"].(string)
			htmlMessages.WriteString(fmt.Sprintf("<div class='chat-message'><b>Youtube %s:</b> %s</div>",
				username, message))
		}
	}

	// Return the HTML messages with HTMX attributes for the next cycle
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<div id="youtube-chat" hx-get="/youtube" hx-trigger="every 5s" hx-swap="outerHTML">`)
	fmt.Fprint(w, htmlMessages.String())
	fmt.Fprint(w, "</div>")
}

// SetupChatRoutes sets up routes to fetch chat messages from Twitch and YouTube
func SetupChatRoutes(router *mux.Router) {
	router.HandleFunc("/twitch", fetchTwitchChat).Methods("GET")
	router.HandleFunc("/youtube", fetchYoutubeChat).Methods("GET")
}
