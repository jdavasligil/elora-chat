package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
	"google.golang.org/api/youtube/v3"
)

// sendMessageHandler handles requests to send messages to both Twitch and YouTube chats.
func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
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

	// Send message to Twitch
	if err := sendMessageToTwitch(sessionToken, "hp_az", requestBody.Message); err != nil {
		log.Printf("Error sending message to Twitch: %v", err)
		// Consider how to handle partial failure
	}

	// Send message to YouTube
	if err := sendMessageToYouTube(requestBody.Message, sessionToken); err != nil {
		log.Printf("Error sending message to YouTube: %v", err)
		// Consider how to handle partial failure
	}

	// Respond to the client
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Message sent successfully to Twitch and YouTube"))
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

	log.Println(sessionData)
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

// sendMessageToYouTube sends a message to a YouTube live chat.
func sendMessageToYouTube(message string, sessionToken string) error {
	accessToken, err := getStoredAccessToken("youtube", sessionToken)
	if err != nil {
		log.Printf("Error retrieving YouTube access token: %v", err)
		return err
	}

	liveChatID, err := getYouTubeLiveChatID(accessToken)
	if err != nil {
		log.Printf("Error retrieving YouTube Live Chat ID: %v", err)
		return err
	}

	// Construct the message request for YouTube's LiveChatMessages API
	messageRequestBody := map[string]interface{}{
		"snippet": map[string]interface{}{
			"liveChatId": liveChatID,
			"type":       "textMessageEvent",
			"textMessageDetails": map[string]interface{}{
				"messageText": message,
			},
		},
	}
	requestBody, _ := json.Marshal(messageRequestBody)

	request, _ := http.NewRequest("POST", "https://www.googleapis.com/youtube/v3/liveChat/messages?part=snippet", strings.NewReader(string(requestBody)))
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Printf("Failed to send message to YouTube Live Chat: %v", err)
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		log.Printf("YouTube API responded with non-200 status: %d", response.StatusCode)
		return fmt.Errorf("YouTube API error status %d", response.StatusCode)
	}

	log.Println("Message successfully sent to YouTube Live Chat")
	return nil
}

// upgrade later to use the URL with the channel ID
// Helper function to create an OAuth2 token source from an access token.
func tokenSourceFromAccessToken(tokenStr string) oauth2.TokenSource {
	token := &oauth2.Token{AccessToken: tokenStr}
	return oauth2.StaticTokenSource(token)
}

// getYouTubeLiveChatID retrieves the live chat ID using the YouTube Data API v3.
func getYouTubeLiveChatID(accessToken string) (string, error) {
	ctx := context.Background()

	// Create an HTTP client using the access token.
	tokenSource := tokenSourceFromAccessToken(accessToken)
	httpClient := oauth2.NewClient(ctx, tokenSource)

	// Create a YouTube service.
	service, err := youtube.New(httpClient)
	if err != nil {
		log.Printf("Error creating YouTube service: %v", err)
		return "", err
	}

	// Fetch live broadcasts.
	response, err := service.LiveBroadcasts.List([]string{"snippet"}).
		BroadcastStatus("active").
		BroadcastType("all").
		Do()
	if err != nil {
		log.Printf("Error fetching live broadcasts: %v", err)
		return "", err
	}

	if len(response.Items) == 0 {
		log.Println("No active live broadcasts found.")
		return "", fmt.Errorf("no active live broadcasts found")
	}

	liveChatID := response.Items[0].Snippet.LiveChatId
	if liveChatID == "" {
		log.Println("Live chat ID not found.")
		return "", fmt.Errorf("live chat ID not found")
	}

	return liveChatID, nil
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

func getStoredAccessToken(service string, sessionToken string) (string, error) {
	// Retrieve the session data JSON from Redis using the session token
	sessionDataJson, err := redisClient.Get(ctx, fmt.Sprintf("session:%s", sessionToken)).Result()
	if err != nil {
		log.Printf("Failed to retrieve session data for token %s: %v", sessionToken, err)
		return "", fmt.Errorf("failed to retrieve session data: %w", err)
	}

	// Unmarshal the session data JSON into a map
	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(sessionDataJson), &sessionData); err != nil {
		log.Printf("Failed to unmarshal session data: %v", err)
		return "", fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	// Extract the access token for the specified service using the correct key
	// Adjust the key to match how it's stored - "youtube_token" or "twitch_token"
	var tokenKey string
	if service == "youtube" {
		tokenKey = "youtube_token"
	} else if service == "twitch" {
		tokenKey = "twitch_token"
	}

	accessToken, ok := sessionData[tokenKey].(string)
	if !ok || accessToken == "" {
		return "", fmt.Errorf("access token for service %s not found in session data", service)
	}

	return accessToken, nil
}

// Setup function to register the send message route
func SetupSendRoutes(router *mux.Router) {
	// Protect the send-message route with middleware that checks for authenticated session
	authRoutes := router.PathPrefix("/auth").Subrouter()
	authRoutes.HandleFunc("/send-message", sendMessageHandler).Methods("POST")
}
