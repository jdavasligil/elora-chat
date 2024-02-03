package auth

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
)

var (
	twitchTokens struct {
		AccessToken  string    `json:"access_token"`
		RefreshToken string    `json:"refresh_token"`
		ExpiryDate   time.Time `json:"expiry_date"`
	}
)

const (
	twitchTokenURL = "https://id.twitch.tv/oauth2/token"
)

// RefreshTwitchToken refreshes the Twitch access token using the refresh token
func RefreshTwitchToken() error {
	values := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {twitchTokens.RefreshToken},
		"client_id":     {os.Getenv("TWITCH_CLIENT_ID")},
		"client_secret": {os.Getenv("TWITCH_CLIENT_SECRET")},
	}

	resp, err := http.PostForm(twitchTokenURL, values)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to refresh Twitch token")
	}

	var newTokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &newTokens)
	if err != nil {
		return err
	}

	twitchTokens.AccessToken = newTokens.AccessToken
	twitchTokens.RefreshToken = newTokens.RefreshToken
	twitchTokens.ExpiryDate = time.Now().Add(time.Second * time.Duration(newTokens.ExpiresIn))

	return nil
}

// GetValidTwitchAccessToken returns a valid Twitch access token, refreshing if necessary
func GetValidTwitchAccessToken() (string, error) {
	if twitchTokens.AccessToken == "" || time.Now().After(twitchTokens.ExpiryDate) {
		err := RefreshTwitchToken()
		if err != nil {
			return "", err
		}
	}
	return twitchTokens.AccessToken, nil
}

// ExpireTwitchAccessToken is a utility function to manually expire the Twitch access token for testing
func ExpireTwitchAccessToken() {
	twitchTokens.ExpiryDate = time.Now().Add(-time.Second) // Set to 1 second in the past
}

// SetTwitchTokens sets the current Twitch tokens
func SetTwitchTokens(accessToken, refreshToken string, expiryDuration int64) {
	twitchTokens.AccessToken = accessToken
	twitchTokens.RefreshToken = refreshToken
	twitchTokens.ExpiryDate = time.Now().Add(time.Second * time.Duration(expiryDuration))
}
