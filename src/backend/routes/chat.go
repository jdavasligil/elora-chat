package routes

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity; adjust as needed for security
	},
}

type Image struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	ID     string `json:"id"`
}

type Emote struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Locations []string `json:"locations"`
	Images    []Image  `json:"images"`
}

type Badge struct {
	Name        string  `json:"name"`
	Title       string  `json:"title"`
	ClickAction string  `json:"clickAction"`
	ClickURL    string  `json:"clickURL"`
	Icons       []Image `json:"icons"`
}

type Author struct {
	DisplayName string  `json:"name"`
	Badges      []Badge `json:"badges"`
	// Other fields as needed
}

type Message struct {
	Author  string  `json:"author"` // Adjusted to directly receive the author's name as a string
	Message string  `json:"message"`
	Emotes  []Emote `json:"emotes"`
	Badges  []Badge `json:"badges"`
	Source  string  `json:"source"`
	Colour  string  `json:"colour"`
}

// messageChannel is a channel for sending chat messages to WebSocket connections
var messageChannel = make(chan Message, 10) // Buffered channel

// StartChatFetch starts fetching chat messages for each provided URL
func StartChatFetch(urls []string) {
	const pythonExecPath = "/mnt/c/Users/hwpDesktop/Documents/Content/Repos/elora-chat/python/venv/bin/python3"
	var fetchChatScript = filepath.Join("/mnt/c/Users/hwpDesktop/Documents/Content/Repos/elora-chat/python", "fetch_chat.py")

	for _, url := range urls {
		go func(url string) {
			cmd := exec.Command(pythonExecPath, "-u", fetchChatScript, url)
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
				rawMessage := scanner.Bytes()
				if err := json.Unmarshal(rawMessage, &msg); err != nil {
					log.Printf("Failed to unmarshal message: %v, Raw message: %s\n", err, string(rawMessage))
					continue
				}
				if strings.Contains(url, "twitch.tv") {
					msg.Source = "Twitch"
				} else if strings.Contains(url, "youtube.com") {
					msg.Source = "YouTube"
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
	router.HandleFunc("/ws/chat", StreamChat).Methods("GET")
}
