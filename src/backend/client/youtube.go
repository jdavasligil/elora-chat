package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/hpwn/EloraChat/src/backend/auth"
)

// LiveChatMessagesListResponse represents the response from the YouTube Data API's liveChatMessages.list endpoint
type LiveChatMessagesListResponse struct {
	Items []struct {
		Snippet struct {
			DisplayMessage string `json:"displayMessage"`
		} `json:"snippet"`
		AuthorDetails struct {
			DisplayName string `json:"displayName"`
		} `json:"authorDetails"`
	} `json:"items"`
	PollingIntervalMillis int64  `json:"pollingIntervalMillis"`
	NextPageToken         string `json:"nextPageToken"`
}

// getLiveChatMessages fetches messages for a given liveChatId
func getLiveChatMessages(liveChatId string, pageToken string) (*LiveChatMessagesListResponse, error) {
	accessToken, err := auth.GetValidYoutubeAccessToken()
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	apiUrl := "https://www.googleapis.com/youtube/v3/liveChat/messages"
	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("liveChatId", liveChatId)
	q.Add("part", "id,snippet,authorDetails")
	q.Add("pageToken", pageToken)
	q.Add("maxResults", "200")
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("YouTube API request failed: %s", string(bodyBytes))
	}

	var chatResponse LiveChatMessagesListResponse
	err = json.NewDecoder(resp.Body).Decode(&chatResponse)
	if err != nil {
		return nil, err
	}

	return &chatResponse, nil
}

// StartYoutubeChatClient starts a long-polling client to retrieve chat messages
func StartYoutubeChatClient(liveChatId string) {
	var nextPageToken string
	var pollMessages func() // Declare the variable as a function type

	// Define the function
	pollMessages = func() {
		chatData, err := getLiveChatMessages(liveChatId, nextPageToken)
		if err != nil {
			log.Printf("Error fetching YouTube live chat messages: %v", err)
			time.Sleep(10 * time.Second) // Retry after 10 seconds on error
			go pollMessages()            // Recursive call using the function variable
			return
		}

		nextPageToken = chatData.NextPageToken
		pollingInterval := time.Duration(chatData.PollingIntervalMillis) * time.Millisecond

		for _, message := range chatData.Items {
			displayName := message.AuthorDetails.DisplayName
			messageText := message.Snippet.DisplayMessage
			log.Printf("%s: %s", displayName, messageText)
		}

		time.Sleep(pollingInterval)
		go pollMessages() // Recursive call using the function variable
	}

	go pollMessages() // Initial call to start the polling process
}
