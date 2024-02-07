package auth

import (
	"context"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// NewYoutubeOAuthConfig dynamically generates the OAuth2 config for YouTube
func NewYoutubeOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
		ClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("YOUTUBE_REDIRECT_URI"),
		Scopes:       []string{"https://www.googleapis.com/auth/youtube.force-ssl"},
		Endpoint:     google.Endpoint,
	}
}

// Placeholder for in-memory token storage (should be replaced with a proper storage solution)
var youtubeTokens *oauth2.Token

// GetYoutubeAuthURL returns the URL to the OAuth2 authorization page for YouTube
func GetYoutubeAuthURL() string {
	config := NewYoutubeOAuthConfig() // Use the new function to get the config
	return config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

// GetYoutubeToken exchanges the OAuth2 authorization code for a token
func GetYoutubeToken(code string) (*oauth2.Token, error) {
	config := NewYoutubeOAuthConfig() // Use the new function to get the config
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, err
	}
	youtubeTokens = token
	return token, nil
}

// RefreshYoutubeToken refreshes the YouTube access token using the refresh token
func RefreshYoutubeToken() error {
	if youtubeTokens == nil {
		return nil // Early return if there's no token to refresh
	}
	config := NewYoutubeOAuthConfig() // Use the new function to get the config
	tokenSource := config.TokenSource(context.Background(), youtubeTokens)
	newToken, err := tokenSource.Token() // This automatically refreshes the token if needed
	if err != nil {
		return err
	}
	youtubeTokens = newToken
	return nil
}

// GetValidYoutubeAccessToken returns a valid YouTube access token, refreshing if necessary
func GetValidYoutubeAccessToken() (string, error) {
	if youtubeTokens == nil || time.Now().After(youtubeTokens.Expiry) {
		err := RefreshYoutubeToken()
		if err != nil {
			return "", err
		}
	}
	return youtubeTokens.AccessToken, nil
}

// ExpireYoutubeAccessToken is a utility function to manually expire the YouTube access token for testing
func ExpireYoutubeAccessToken() {
	if youtubeTokens != nil {
		// Set the expiry time to the past to expire the token
		youtubeTokens.Expiry = time.Now().Add(-time.Second)
	}
}
