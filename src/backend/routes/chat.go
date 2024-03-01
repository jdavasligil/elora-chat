package routes

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// Initialize a Redis client as a global variable.
var redisClient *redis.Client
var ctx = context.Background()

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

type Message struct {
	Author  string  `json:"author"` // Adjusted to directly receive the author's name as a string
	Message string  `json:"message"`
	Emotes  []Emote `json:"emotes"`
	Badges  []Badge `json:"badges"`
	Source  string  `json:"source"`
	Colour  string  `json:"colour"`
}

func init() {
	// Initialize the Redis client without TLS.
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "redis-16438.c325.us-east-1-4.ec2.cloud.redislabs.com:16438",
		Password: "oJc6z3chOy7wbMSUKuWj2feIrTF9pci3", // The password for the Redis server (if required)
		DB:       0,                                  // Default DB
	})

	// Context for Redis operations
	ctx := context.Background()

	// Check the Redis connection
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
}

// StartChatFetch starts fetching chat messages for each provided URL
func StartChatFetch(urls []string) {
	const pythonExecPath = "/usr/local/bin/python3"
	const fetchChatScript = "/app/python/fetch_chat.py"

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

				// Re-marshal the message with the Source set.
				modifiedMessage, err := json.Marshal(msg)
				if err != nil {
					log.Printf("Failed to marshal message: %v, Message: %#v\n", err, msg)
					continue
				}

				// Publish the modified message to Redis.
				err = redisClient.Publish(ctx, "chatMessages", modifiedMessage).Err()
				if err != nil {
					log.Printf("Failed to publish message: %v, Modified message: %s\n", err, string(modifiedMessage))
				}
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

	// Subscribe to the Redis channel.
	pubsub := redisClient.Subscribe(ctx, "chatMessages")
	defer pubsub.Close()

	// Go routine to receive messages.
	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			// Use the WebSocket connection to send the message to the client.
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				log.Println("WebSocket write error:", err)
				return
			}
		}
	}()

	// Start a goroutine to send keep-alive messages
	go func() {
		for {
			time.Sleep(20 * time.Second) // Send keep-alive message every 20 seconds
			if err := conn.WriteMessage(websocket.TextMessage, []byte("__keepalive__")); err != nil {
				log.Println("Failed to send keep-alive message:", err)
				break // Exit the loop if there's an error (e.g., connection closed)
			}
		}
	}()

	// Your existing code for keeping the connection open or handling other aspects of the WebSocket
	for {
		// Example blocking pattern to keep the connection open
		time.Sleep(10 * time.Second)
	}
}

func ImageProxy(w http.ResponseWriter, r *http.Request) {
	imageURL := r.URL.Query().Get("url")
	if imageURL == "" {
		http.Error(w, "Missing URL parameter", http.StatusBadRequest)
		return
	}

	resp, err := http.Get(imageURL)
	if err != nil || resp.StatusCode == 404 {
		// Log error and serve a default placeholder image
		log.Printf("Error fetching image or not found: %s", imageURL)
		http.ServeFile(w, r, "../public/refresh.png")
		return
	}
	defer resp.Body.Close()

	// Copy headers
	for key, value := range resp.Header {
		w.Header().Set(key, value[0])
	}

	// Stream the image content
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// SetupChatRoutes sets up WebSocket routes
func SetupChatRoutes(router *mux.Router) {
	router.HandleFunc("/ws/chat", StreamChat).Methods("GET")
	router.HandleFunc("/imageproxy", ImageProxy).Methods("GET") // New proxy route
}
