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

// Message represents a generic chat message
type Message struct {
	Author  string `json:"author"`
	Message string `json:"message"`
}

// messageChannel is a channel for sending chat messages to WebSocket connections
var messageChannel = make(chan Message, 10) // Buffered channel

// StartChatFetch starts fetching chat messages for each provided URL
func StartChatFetch(urls []string) {

	const pythonExecPath = "/mnt/c/Users/hwpDesktop/Documents/Content/Repos/EloraChat/python/venv/bin/python3"
	var fetchChatScript = filepath.Join("/mnt/c/Users/hwpDesktop/Documents/Content/Repos/EloraChat/python", "fetch_chat.py")

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
		if err := conn.WriteJSON(msg); err != nil {
			log.Println("WebSocket write error:", err)
			break
		}
	}
}

// SetupChatRoutes sets up WebSocket routes
func SetupChatRoutes(router *mux.Router) {
	router.HandleFunc("/ws/chat", StreamChat)
}
