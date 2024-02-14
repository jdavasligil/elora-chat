package routes

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity; adjust as needed for security
	},
}

type Author struct {
	DisplayName string `json:"name"`
	// Add other fields from the author dictionary as needed
}

type Message struct {
	Author  Author `json:"author"`
	Message string `json:"message"`
	// Include other fields as necessary, based on the chat item fields documentation
}

// messageChannel is a channel for sending chat messages to WebSocket connections
var messageChannel = make(chan Message, 10) // Buffered channel

// StartChatFetch starts fetching chat messages for each provided URL
func StartChatFetch(urls []string) {

	const pythonExecPath = "/mnt/c/Users/hwpDesktop/Documents/Content/Repos/elora-chat/python/venv/bin/python3"
	var fetchChatScript = filepath.Join("/mnt/c/Users/hwpDesktop/Documents/Content/Repos/elora-chat/python", "fetch_chat.py")

	for _, url := range urls {
		go func(url string) {
			cmd := exec.Command(pythonExecPath, fetchChatScript, url)

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				log.Fatal("Failed to create stdout pipe:", err)
			}

			if err := cmd.Start(); err != nil {
				log.Fatal("Failed to start command:", err)
			}

			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				var msg Message
				if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
					log.Println("Failed to unmarshal message:", err)
					continue
				}
				messageChannel <- msg
			}

			if err := scanner.Err(); err != nil {
				log.Println("Error reading standard output:", err)
			}
		}(url)
	}
}

// StreamChat initializes a WebSocket connection and streams chat messages
func StreamChat(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	for msg := range messageChannel {
		// Create a simplified message for the frontend, if needed
		simplifiedMsg := map[string]string{
			"Author":  msg.Author.DisplayName, // Use DisplayName for the author
			"Message": msg.Message,
		}

		if err := conn.WriteJSON(simplifiedMsg); err != nil {
			log.Println("WebSocket write error:", err)
			break
		}
	}
}

// SetupChatRoutes sets up WebSocket routes
func SetupChatRoutes(router *mux.Router) {
	router.HandleFunc("/ws/chat", StreamChat)
}
