package routes

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
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
		Addr:            os.Getenv("REDIS_ADDR"),
		Password:        os.Getenv("REDIS_PASSWORD"), // The password for the Redis server (if required)
		DB:              0,                           // Default DB
		ConnMaxIdleTime: 5 * time.Minute,             // Maximum amount of time a connection may be idle.
		ConnMaxLifetime: 30 * time.Minute,            // Maximum amount of time a connection may be reused.
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

				// Add the modified message to Redis Stream.
				_, err = redisClient.XAdd(ctx, &redis.XAddArgs{
					Stream: "chatMessages",
					Values: map[string]interface{}{"message": string(modifiedMessage)},
					MaxLen: 100,
					Approx: true,
				}).Result()
				if err != nil {
					log.Printf("Failed to add message to stream: %v, Modified message: %s\n", err, string(modifiedMessage))
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

	// Channel to signal closure of WebSocket connection
	done := make(chan struct{})

	lastID := "0" // Start from the beginning of the stream

	// Read the last 100 messages from the stream to send to the client immediately.
	streams, err := redisClient.XRevRangeN(ctx, "chatMessages", "+", "-", 100).Result()
	if err != nil {
		log.Printf("Failed to read messages from stream: %v\n", err)
		return
	}

	// Send the messages in reverse order so the newest will be at the bottom
	for i := len(streams) - 1; i >= 0; i-- {
		message := streams[i]
		if err := conn.WriteMessage(websocket.TextMessage, []byte(message.Values["message"].(string))); err != nil {
			log.Println("WebSocket write error:", err)
			return
		}
		if message.ID > lastID {
			lastID = message.ID // Update last ID to the newest message
		}
	}

	// Go routine to receive new messages from Redis Stream and forward them to WebSocket
	go func() {
		for {
			streams, err := redisClient.XRead(ctx, &redis.XReadArgs{
				Streams: []string{"chatMessages", lastID},
				Block:   0,
			}).Result()

			if err != nil {
				log.Println("Error reading from stream:", err)
				return
			}

			for _, stream := range streams {
				for _, message := range stream.Messages {
					if err := conn.WriteMessage(websocket.TextMessage, []byte(message.Values["message"].(string))); err != nil {
						if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
							log.Println("WebSocket write error:", err)
						}
						return
					}
					lastID = message.ID // Update last ID to the newest message
				}
			}
		}
	}()

	// Keep-alive go routine
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.TextMessage, []byte("__keepalive__")); err != nil {
					log.Println("Failed to send keep-alive message:", err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Read loop to keep connection alive and detect close
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
				log.Println("WebSocket read error, closing connection:", err)
			}
			close(done)
			break
		}
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
