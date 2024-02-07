package routes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/hpwn/EloraChat/src/backend/auth"
	"github.com/hpwn/EloraChat/src/backend/client" // Replace with the actual import path of your YouTube client package
)

// LiveBroadcastsListResponse represents the response from the YouTube Data API's liveBroadcasts.list endpoint
type LiveBroadcastsListResponse struct {
	Items []struct {
		Snippet struct {
			LiveChatID string `json:"liveChatId"`
		} `json:"snippet"`
	} `json:"items"`
}

// SetupYoutubeRoutes sets up the routes for YouTube authentication
func SetupYoutubeRoutes(router *mux.Router) {
	router.HandleFunc("/auth/youtube", redirectForYoutubeAuth).Methods("GET")
	router.HandleFunc("/auth/youtube/callback", youtubeAuthCallback).Methods("GET")
	router.HandleFunc("/auth/youtube/test-refresh", testYoutubeTokenRefresh).Methods("GET")
}

func redirectForYoutubeAuth(w http.ResponseWriter, r *http.Request) {
	authURL := auth.GetYoutubeAuthURL()
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func youtubeAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code provided", http.StatusBadRequest)
		return
	}

	token, err := auth.GetYoutubeToken(code)
	if err != nil {
		http.Error(w, "YouTube Authentication failed", http.StatusInternalServerError)
		return
	}

	liveChatId, err := getLiveChatId(token.AccessToken)
	if err != nil {
		http.Error(w, "YouTube Authentication failed", http.StatusInternalServerError)
		return
	}

	go client.StartYoutubeChatClient(liveChatId) // Replace with your actual function to start the YouTube chat client

	fmt.Fprintln(w, "YouTube Authentication successful!")
}

func getLiveChatId(accessToken string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	apiUrl := "https://www.googleapis.com/youtube/v3/liveBroadcasts"
	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return "", err
	}

	// Define query parameters
	q := req.URL.Query()
	q.Add("part", "snippet")
	q.Add("broadcastStatus", "active")
	q.Add("broadcastType", "all")
	req.URL.RawQuery = q.Encode()

	// Set authorization header
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("YouTube API request failed: %s", string(bodyBytes))
	}

	var response LiveBroadcastsListResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	if len(response.Items) == 0 {
		return "", fmt.Errorf("no active live broadcasts found")
	}

	liveChatId := response.Items[0].Snippet.LiveChatID
	if liveChatId == "" {
		return "", fmt.Errorf("the live broadcast does not have an associated live chat")
	}

	return liveChatId, nil
}

func testYoutubeTokenRefresh(w http.ResponseWriter, r *http.Request) {
	auth.ExpireYoutubeAccessToken()

	accessToken, err := auth.GetValidYoutubeAccessToken()
	if err != nil {
		http.Error(w, "Token refresh test failed", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Token refresh test completed. Access Token: %s", accessToken)
}
