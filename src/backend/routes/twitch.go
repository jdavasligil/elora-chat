package routes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/hpwn/EloraChat/src/backend/auth" // Replace with the actual import path of your auth package
	"github.com/hpwn/EloraChat/src/backend/client"

	"github.com/gorilla/mux"
)

// SetupTwitchRoutes sets up the routes for Twitch authentication
func SetupTwitchRoutes(router *mux.Router) {
	router.HandleFunc("/auth/twitch", redirectForTwitchAuth).Methods("GET")
	router.HandleFunc("/auth/twitch/callback", twitchAuthCallback).Methods("GET")
	router.HandleFunc("/auth/twitch/test-refresh", testTwitchTokenRefresh).Methods("GET")
}

func redirectForTwitchAuth(w http.ResponseWriter, r *http.Request) {
	redirectURI := os.Getenv("TWITCH_REDIRECT_URI")
	clientID := os.Getenv("TWITCH_CLIENT_ID")
	scopes := []string{"chat:read", "chat:edit"}
	scopesString := strings.Join(scopes, " ")
	authURL := fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s", clientID, url.QueryEscape(redirectURI), url.QueryEscape(scopesString))
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func twitchAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code provided", http.StatusBadRequest)
		return
	}

	twitchTokenURL := "https://id.twitch.tv/oauth2/token"
	values := url.Values{
		"client_id":     {os.Getenv("TWITCH_CLIENT_ID")},
		"client_secret": {os.Getenv("TWITCH_CLIENT_SECRET")},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {os.Getenv("TWITCH_REDIRECT_URI")},
	}

	resp, err := http.PostForm(twitchTokenURL, values)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error during Twitch authentication: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Twitch Authentication failed", http.StatusInternalServerError)
		return
	}

	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading response body: %v", err), http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &tokens)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error unmarshaling tokens: %v", err), http.StatusInternalServerError)
		return
	}

	// Store the tokens and expiry date using the functions from the auth package
	auth.SetTwitchTokens(tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresIn)

	// Successful authentication, now start the Twitch chat client
	go startTwitchChatClient(tokens.AccessToken)

	// You would typically redirect to another page here
	fmt.Fprintln(w, "Twitch Authentication successful!")
}

func startTwitchChatClient(accessToken string) {
	twitchUsername := os.Getenv("TWITCH_USERNAME") // Retrieve the username from environment variable
	if twitchUsername == "" {
		log.Printf("Twitch username is not set in environment variables")
		return
	}

	client.StartChatClient(twitchUsername, accessToken)
	// Since StartChatClient does not return an error, no error handling is needed here
}

func testTwitchTokenRefresh(w http.ResponseWriter, r *http.Request) {
	// Manually expire the Twitch token for testing purposes
	auth.ExpireTwitchAccessToken()

	// Try to get a valid Twitch access token
	accessToken, err := auth.GetValidTwitchAccessToken()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error during Twitch token refresh test: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Twitch token refresh test completed. Access Token: %s", accessToken)
}
