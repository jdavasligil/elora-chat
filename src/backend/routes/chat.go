package routes

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"github.com/jdavasligil/emodl"
	"github.com/jdavasligil/golang-dsa/bucket"
)

// Initialize a Redis client as a global variable.
var redisClient *redis.Client
var ctx = context.Background()
var chatFetchCmds sync.Map // string -> *exec.Cmd

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity; adjust as needed for security
	},
}

var emoteCache map[string]Emote

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
		PoolSize:        200,                         // Adjusted pool size
		MinIdleConns:    10,                          // Maintain a minimum of 10 idle connections
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

	// Load third party emotes
	downloader := emodl.NewDownloader(emodl.DownloaderOptions{
		// TEMP: (Dayoman ID hard coded)
		SevenTV: &emodl.SevenTVOptions{
			Platform:   "twitch",
			PlatformID: "39226538",
		},
		BTTV: &emodl.BTTVOptions{
			Platform:   "twitch",
			PlatformID: "39226538",
		},
		FFZ: &emodl.FFZOptions{
			Platform:   "twitch",
			PlatformID: "39226538",
		},
	})
	emoteCacheTmp, err := downloader.Load()
	if err != nil {
		log.Printf("Failed to load third party emotes: %v", err)
	}
	emoteCache = make(map[string]Emote, len(emoteCacheTmp))
	for name, emote := range emoteCacheTmp {
		emoteCache[name] = Emote{
			ID:        emote.ID,
			Name:      emote.Name,
			Locations: emote.Locations,
			Images:    []Image{Image(emote.Images[0])},
		}
	}
	// DEBUG
	// log.Println("3P EMOTES SUPPORTED")
	// for _, e := range emoteCache {
	// 	log.Println(e.Name)
	// }
}

func StartChatFetch(urls []string) {
	pythonExecPath := "/usr/local/bin/python3"
	fetchChatScript := "/app/python/fetch_chat.py"

	for _, url := range urls {
		go monitorAndRestartChatFetch(url, pythonExecPath, fetchChatScript)
	}
}

func monitorAndRestartChatFetch(url, pythonExecPath, fetchChatScript string) {
	for {
		cmd := startChatFetch(url, pythonExecPath, fetchChatScript)
		chatFetchCmds.Store(url, cmd)

		err := cmd.Wait() // Waits for the command to exit
		if err != nil {
			log.Printf("Chat fetch for %s stopped: %v", url, err)
		}

		// Wait for a short duration before restarting to prevent rapid restart loops
		log.Println("Restarting chat fetch...")
		time.Sleep(1 * time.Second)
	}
}

func startChatFetch(url, pythonExecPath, fetchChatScript string) *exec.Cmd {
	cmd := exec.Command(pythonExecPath, "-u", fetchChatScript, url)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal("Failed to create stdout pipe:", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal("Failed to start command:", err)
	}

	log.Println("Fetching chat from URL: ", url)

	go processChatOutput(stdout, url)
	return cmd
}

func processChatOutput(stdout io.ReadCloser, url string) {
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

		// Find third party emotes in chat message
		uniqueEmotes := map[string]struct{}{}
		tokens := strings.Fields(msg.Message)
		for _, tok := range tokens {
			if e, ok := emoteCache[tok]; ok {
				uniqueEmotes[e.Name] = struct{}{}
			}
		}
		for name := range uniqueEmotes {
			msg.Emotes = append(msg.Emotes, emoteCache[name])
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
			Values: map[string]any{"message": string(modifiedMessage)},
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
	messageChan := make(chan []byte)
	leakyBucket, err := bucket.NewBucket[[]byte](&bucket.BucketOptions{
		Capacity:        64,
		LowLatency:      false,
		DropBias:        0.90,
		DropInterval:    333 * time.Millisecond,
		MinDropInterval: 50 * time.Millisecond,
		MaxDropInterval: time.Second,
		MaxWaitTime:     5 * time.Second,
		UpdateInterval:  12 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

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
					// Before sending a message to the client
					// log.Printf("Sending message to client: %s\n", message)
					leakyBucket.AddDrop([]byte(message.Values["message"].(string)))
					lastID = message.ID // Update last ID to the newest message
				}
			}
		}
	}()

	// Bucket transfer
	go func() {
		dropCount := 0
		for {
			drop, err := leakyBucket.AwaitDrop()
			if errors.Is(err, &bucket.BucketClosedError{}) {
				log.Println("Bucket has closed. Shutting down")
				dropsRemaining := leakyBucket.Drain()
				log.Printf("Drops Remaining: %v\n", dropsRemaining)
				close(done)
				return
			} else if err != nil {
				log.Println(err)
			} else {
				messageChan<-drop
			}
			dropCount++

			if dropCount % 16 == 0 {
				log.Println(leakyBucket.Status())
			}
		}
	}()

	// Websocket writer
	go func() {
		// Keep alive ticker
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case m := <-messageChan:
				if err := conn.WriteMessage(websocket.TextMessage, m); err != nil {
					log.Println("WebSocket write error:", err)
					return
				}
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

// StopChatFetches stops all ongoing chat fetch commands
func StopChatFetches(w http.ResponseWriter, r *http.Request) {
	chatFetchCmds.Range(func(key, value any) bool {
		cmd, ok := value.(*exec.Cmd)
		if !ok {
			log.Println("Failed to coerce chatFetchCmds value to exec.Cmd")
			return true
		}
		if cmd != nil && cmd.Process != nil {
			err := cmd.Process.Kill()
			if err != nil {
				log.Printf("Failed to stop chat fetch command: %v", err)
			}
		}
		return true
	})
	fmt.Fprintln(w, "Chat fetch commands stopped. Restarting...")
}

// SetupChatRoutes sets up WebSocket routes
func SetupChatRoutes(router *mux.Router) {
	// Public routes
	router.HandleFunc("/ws/chat", StreamChat).Methods("GET")
	router.HandleFunc("/imageproxy", ImageProxy).Methods("GET")

	// Subrouter for chat routes that require authentication
	protectedRoutes := router.PathPrefix("").Subrouter()
	protectedRoutes.Use(SessionMiddleware)

	// Add protected chat routes to protectedRoutes
	protectedRoutes.HandleFunc("/restart-server", StopChatFetches).Methods("POST")
}
