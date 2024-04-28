package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"time"

	"github.com/gorilla/mux"
	ytbot "github.com/ketan-10/ytLiveChatBot"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// Global variable for the chat bot, assuming you're managing one YouTube stream
var chatBot *ytbot.LiveChatBot

func init() {
	apiKey := "AIzaSyBjKvYvbpwybafW7OdvAt5-GS61kds4vBI" // Retrieve API key from environment variable

	// channelID := "UCHToAogHtFnv2uksbDzKsYA" // hp_az
	// channelID := "UCSJ4gkVC6NrvII8umztf0Ow" // lofigirl
	channelID := "UC2c4NxvHnbXs3NLpCm641ew" // dayoman

	if err := cacheLiveStreamURL(apiKey, channelID); err == nil {
		liveURL, _ := redisClient.Get(ctx, "youtube:live:url").Result()
		startChatBot(liveURL)
	}

	// Start periodic refresh of YouTube Auth Token every 30 minutes
	go refreshYouTubeAuthTokenEvery(30*time.Minute, apiKey, channelID)
}

func startChatBot(url string) {
	chatBot = ytbot.NewLiveChatBot(&ytbot.LiveChatBotInput{
		Urls: []string{url},
	})
}

func refreshYouTubeAuthTokenEvery(interval time.Duration, apiKey, channelID string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := refreshYouTubeAuthToken(apiKey, channelID); err != nil {
			log.Printf("Error refreshing YouTube auth token: %v", err)
		} else {
			log.Println("Successfully refreshed YouTube auth token")
		}
	}
}

func refreshYouTubeAuthToken(apiKey, channelID string) error {
	return cacheLiveStreamURL(apiKey, channelID) // Reusing existing function to refresh URL
}

func cacheLiveStreamURL(apiKey, channelID string) error {
	url, err := fetchLiveStreamURL(apiKey, channelID)
	if err != nil {
		log.Printf("Error fetching YouTube live stream URL: %v", err)
		return err
	}

	// Store the URL in Redis
	err = redisClient.Set(ctx, "youtube:live:url", url, 0).Err() // No expiration
	if err != nil {
		log.Printf("Error caching YouTube live stream URL in Redis: %v", err)
		return err
	}
	return nil
}

func fetchLiveStreamURL(apiKey, channelID string) (string, error) {
	ctx := context.Background()
	youtubeService, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("error creating YouTube service: %v", err)
	}

	call := youtubeService.Search.List([]string{"id"}).ChannelId(channelID).EventType("live").Type("video").MaxResults(1)
	response, err := call.Do()
	if err != nil {
		return "", fmt.Errorf("error making search API call: %v", err)
	}

	if len(response.Items) == 0 {
		return "", fmt.Errorf("no live streams currently on this channel")
	}

	liveVideoID := response.Items[0].Id.VideoId
	return fmt.Sprintf("https://www.youtube.com/watch?v=%s", liveVideoID), nil
}

// sendMessageHandler handles requests to send messages to both Twitch and YouTube chats.
func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := "AIzaSyBjKvYvbpwybafW7OdvAt5-GS61kds4vBI"
	channelID := "UC2c4NxvHnbXs3NLpCm641ew"

	err := cacheLiveStreamURL(apiKey, channelID)
	if err != nil {
		http.Error(w, "Failed to refresh YouTube live stream URL", http.StatusInternalServerError)
		return
	}

	liveURL, err := redisClient.Get(ctx, "youtube:live:url").Result()
	if err != nil {
		http.Error(w, "Failed to retrieve live stream URL from cache", http.StatusInternalServerError)
		return
	}

	// Reinitialize the bot with the new URL
	startChatBot(liveURL)

	// Parse request body to get the message content
	var requestBody struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Retrieve session token from cookies
	sessionToken, err := getSessionTokenFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized: No session token", http.StatusUnauthorized)
		return
	}

	// Retrieve the username associated with this session
	username, err := getUsernameFromSession(sessionToken)
	if err != nil {
		http.Error(w, "Failed to get username from session", http.StatusInternalServerError)
		return
	}

	// Check if the username matches "Dayoman"
	if username != "Dayoman" {
		http.Error(w, "Unauthorized: Incorrect user", http.StatusUnauthorized)
		return
	}

	// Send message to Twitch
	if err := sendMessageToTwitch(sessionToken, "Dayoman", requestBody.Message); err != nil {
		log.Printf("Error sending message to Twitch: %v", err)
		// Consider how to handle partial failure
	}

	// Send message to YouTube using ytLiveChatBot
	if len(chatBot.ChatWriters) > 0 {
		for _, chatWriter := range chatBot.ChatWriters {
			chatWriter <- requestBody.Message
			break // Send to the first stream only for simplicity
		}
		log.Println("Message sent to YouTube via ytLiveChatBot:", requestBody.Message)
	} else {
		log.Println("ytLiveChatBot is not connected to any YouTube live stream.")
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Message sent successfully to Twitch and YouTube"))
}

// Helper function to retrieve the username from session data
func getUsernameFromSession(sessionToken string) (string, error) {
	sessionDataJson, err := redisClient.Get(context.Background(), fmt.Sprintf("session:%s", sessionToken)).Result()
	if err != nil {
		return "", fmt.Errorf("error retrieving session data from Redis: %v", err)
	}

	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(sessionDataJson), &sessionData); err != nil {
		return "", fmt.Errorf("error unmarshalling session data: %v", err)
	}

	// Assuming the username is nested under "data" which is an array of user information
	userData, ok := sessionData["data"].([]interface{})
	if !ok || len(userData) == 0 {
		return "", fmt.Errorf("user data is not found or is not in the expected format")
	}

	userMap, ok := userData[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("user data format is incorrect or empty")
	}

	username, ok := userMap["login"].(string)
	if !ok {
		return "", fmt.Errorf("username not found in session data")
	}

	return username, nil
}

// getTwitchOAuthToken retrieves the OAuth token for Twitch from the session data.
func getTwitchOAuthToken(sessionToken string) (string, error) {
	// Retrieve session data from Redis (or your storage solution)
	sessionDataJson, err := redisClient.Get(context.Background(), fmt.Sprintf("session:%s", sessionToken)).Result()
	if err != nil {
		return "", fmt.Errorf("error retrieving session data from Redis: %v", err)
	}

	// Parse session data to extract the Twitch OAuth token
	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(sessionDataJson), &sessionData); err != nil {
		return "", fmt.Errorf("error unmarshalling session data: %v", err)
	}

	// Assuming the token is stored under a "twitch_token" key
	token, ok := sessionData["twitch_token"].(string)
	if !ok {
		return "", fmt.Errorf("twitch OAuth token not found in session data")
	}

	return token, nil
}

// sendMessageToTwitch sends a message to the Twitch chat channel associated with the provided sessionToken.
func sendMessageToTwitch(sessionToken string, channel string, message string) error {
	// Retrieve the OAuth token for Twitch using the session token
	oauthToken, err := getTwitchOAuthToken(sessionToken)
	if err != nil {
		return fmt.Errorf("error retrieving Twitch OAuth token: %v", err)
	}

	// Twitch IRC server details
	server := "irc.chat.twitch.tv:6667"
	nickname := "Dayoman" // The streamer's Twitch username

	// Connect to Twitch IRC server
	conn, err := net.Dial("tcp", server)
	if err != nil {
		return fmt.Errorf("error connecting to Twitch IRC: %v", err)
	}
	defer conn.Close()

	// Authenticate and join the channel
	fmt.Fprintf(conn, "PASS oauth:%s\r\n", oauthToken)
	fmt.Fprintf(conn, "NICK %s\r\n", nickname)
	fmt.Fprintf(conn, "JOIN #%s\r\n", channel)

	ircConn := textproto.NewConn(conn)

	// Send the message
	if err := ircConn.PrintfLine("PRIVMSG #%s :%s", channel, message); err != nil {
		return fmt.Errorf("error sending message to Twitch chat: %v", err)
	}

	log.Printf("Message sent to Twitch channel #%s: %s", channel, message)

	return nil
}

// getSessionTokenFromRequest extracts the session token from the request cookies.
func getSessionTokenFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		log.Printf("Session token cookie not found: %v", err)
		return "", fmt.Errorf("session token cookie not found")
	}
	return cookie.Value, nil
}

// Setup function to register the send message route
func SetupSendRoutes(router *mux.Router) {
	// Protect the send-message route with middleware that checks for authenticated session
	authRoutes := router.PathPrefix("/auth").Subrouter()
	authRoutes.HandleFunc("/send-message", sendMessageHandler).Methods("POST")
}
