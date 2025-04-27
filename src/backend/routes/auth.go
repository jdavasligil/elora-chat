package routes

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

// Twitch OAuth configuration
var twitchOAuthConfig = &oauth2.Config{
	ClientID:     os.Getenv("TWITCH_CLIENT_ID"),
	ClientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
	RedirectURL:  os.Getenv("TWITCH_REDIRECT_URL"),
	Scopes:       []string{"chat:edit", "chat:read"}, // Updated scopes
	Endpoint:     twitch.Endpoint,                    // Make sure to import "golang.org/x/oauth2/twitch"
}

// loginHandler to initiate OAuth with Twitch
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.URL.Path, "/twitch") {
		http.Error(w, "Unsupported platform", http.StatusBadRequest)
		return
	}

	// Generate a new random state for the OAuth flow
	state, err := generateState()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Store the state in Redis with an expiration time to validate it later
	err = redisClient.Set(ctx, "oauth-state:"+state, "valid", 10*time.Minute).Err()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Construct the OAuth URL and redirect the user to the Twitch authentication page
	url := twitchOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.URL.Path, "/twitch") {
		http.Error(w, "Unsupported platform", http.StatusBadRequest)
		return
	}

	var oauthConfig *oauth2.Config = twitchOAuthConfig
	var userInfoURL string = "https://api.twitch.tv/helix/users"
	var service string = "twitch"

	// Check if an error query parameter is present
	if errorReason := r.FormValue("error"); errorReason != "" {
		fmt.Printf("OAuth error: %s, Description: %s\n", errorReason, r.FormValue("error_description"))
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Verify the state parameter matches
	receivedState := r.FormValue("state")
	if receivedState == "" || !validateState(receivedState) {
		http.Error(w, "State mismatch or missing state", http.StatusBadRequest)
		return
	}

	// Exchange the auth code for an access token
	token, err := oauthConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var req *http.Request
	var res *http.Response

	// For Twitch, manually set the headers and create the request
	userInfoURL = "https://api.twitch.tv/helix/users"
	req, err = http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Set necessary headers for Twitch API
	req.Header.Set("Client-ID", oauthConfig.ClientID)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	res, err = http.DefaultClient.Do(req)

	if err != nil || res.StatusCode != 200 {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	// Unmarshal the user data
	var userData map[string]any
	if err = json.Unmarshal(body, &userData); err != nil {
		http.Error(w, "Failed to parse user data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Before generating a new session token, check if an existing session token is present
	var sessionToken string
	if cookie, err := r.Cookie("session_token"); err == nil {
		// Use the existing session token if present
		sessionToken = cookie.Value
	} else {
		// Generate a new session token if it does not exist
		sessionToken = generateSessionToken()
	}

	// Include the service in the userData map
	userData["service"] = service

	// Assuming user data fetching was successful, include OAuth token in userData.
	// The key for storing the token should match what you'll use in sendMessage functions.
	if service == "twitch" {
		userData["twitch_token"] = token.AccessToken
	} else if service == "youtube" {
		userData["youtube_token"] = token.AccessToken
	}

	// Store refresh token and expiry time if available
	if token.RefreshToken != "" {
		userData["refresh_token"] = token.RefreshToken
	}
	userData["token_expiry"] = token.Expiry.Unix() // Store as Unix timestamp for simplicity

	// Now, use a function to update the session data with this service login
	// This should include setting the session token in a cookie
	updateSessionDataForService(w, userData, service, sessionToken)

	// Redirect the user to the main page or dashboard
	http.Redirect(w, r, "/", http.StatusFound)
}

func updateSessionDataForService(w http.ResponseWriter, userData map[string]any, service string, sessionToken string) {
	// Initialize existing session data map
	existingSessionData := make(map[string]any)

	// Retrieve existing session data from Redis, if available
	existingSessionDataJson, err := redisClient.Get(ctx, fmt.Sprintf("session:%s", sessionToken)).Result()
	if err == nil && existingSessionDataJson != "" {
		// Existing session data found, unmarshal into the map
		err = json.Unmarshal([]byte(existingSessionDataJson), &existingSessionData)
		if err != nil {
			log.Printf("Error unmarshalling existing session data: %v", err)
			// Handle error, for example, by initializing existingSessionData with default values
		}
	}

	// Check if services key exists and is a slice, then update or add the current service
	services, ok := existingSessionData["services"].([]any)
	if !ok {
		// Services key does not exist or is not a slice, initialize it
		services = []any{}
	}
	if !contains(services, service) {
		services = append(services, service)
	}
	existingSessionData["services"] = services

	// Merge userData into existingSessionData, excluding the "services" slice to avoid duplication
	for key, value := range userData {
		if key != "services" {
			existingSessionData[key] = value
		}
	}

	// Marshal the updated session data back into JSON for storage in Redis
	updatedSessionDataJson, err := json.Marshal(existingSessionData)
	if err != nil {
		log.Printf("Error marshalling updated session data: %v", err)
		// Handle error appropriately
	}

	// Store the updated session data in Redis
	err = redisClient.Set(ctx, fmt.Sprintf("session:%s", sessionToken), updatedSessionDataJson, 24*time.Hour).Err()
	if err != nil {
		log.Printf("Failed to store updated session data in Redis: %v", err)
		// Handle error appropriately
	} else {
		log.Printf("Updated session data stored successfully for sessionToken: %s, Data: %s", sessionToken, string(updatedSessionDataJson))
	}

	// Update the client's session cookie
	setSessionCookie(w, sessionToken)
}

// Helper function to check if a service is already in the services slice
func contains(slice []any, str string) bool {
	for _, v := range slice {
		if s, ok := v.(string); ok && s == str {
			return true
		}
	}
	return false
}

// Function to set session cookie
func setSessionCookie(w http.ResponseWriter, sessionToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,                 // Adjust based on your deployment (HTTPS)
		SameSite: http.SameSiteLaxMode, // Or adjust based on your cross-origin policy
	})
}

// generateState creates a new random state for OAuth flow.
func generateState() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	state := hex.EncodeToString(b)
	return state, nil
}

// validateState checks the provided state against the value stored in Redis.
func validateState(state string) bool {
	key := "oauth-state:" + state
	storedState, err := redisClient.Get(ctx, key).Result()

	// Delete the one-time-use state from Redis after validation
	redisClient.Del(ctx, key)

	// Check if the stored state value starts with "valid"
	if err != nil || !strings.HasPrefix(storedState, "valid") {
		return false
	}
	return true
}

// generateSessionToken creates a new secure, random session token.
func generateSessionToken() string {
	b := make([]byte, 32) // 32 bytes results in a 44-character base64 encoded string
	_, err := rand.Read(b)
	if err != nil {
		// Handle error; it's crucial to securely generate a random token.
		return "" // Return empty string or handle it appropriately.
	}
	return base64.URLEncoding.EncodeToString(b)
}

// SessionMiddleware checks for a valid session token in the request cookies.
func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Attempt to retrieve the session token cookie
		cookie, err := r.Cookie("session_token")
		if err != nil {
			log.Printf("No session token cookie found: %v\n", err)
			http.Error(w, "Unauthorized: No session token", http.StatusUnauthorized)
			return
		}

		sessionToken := cookie.Value

		// Retrieve session data from Redis
		sessionDataJson, err := redisClient.Get(ctx, fmt.Sprintf("session:%s", sessionToken)).Result()
		if err != nil {
			log.Printf("Session token not found in Redis or is invalid: %v\n", err)
			http.Error(w, "Unauthorized: Invalid session token", http.StatusUnauthorized)
			return
		}

		var sessionData map[string]any
		err = json.Unmarshal([]byte(sessionDataJson), &sessionData)
		if err != nil {
			log.Printf("Error unmarshalling session data: %v\n", err)
			http.Error(w, "Error processing session data", http.StatusInternalServerError)
			return
		}

		services, ok := sessionData["services"].([]any)
		if !ok {
			log.Println("Services array missing or incorrect format in session data")
			http.Error(w, "Unauthorized: Required services not found", http.StatusUnauthorized)
			return
		}

		var hasTwitch bool
		for _, service := range services {
			if serviceName, ok := service.(string); ok && serviceName == "twitch" {
				hasTwitch = true
				// Refresh Twitch token if necessary
				if err := refreshToken("twitch", sessionToken); err != nil {
					log.Printf("Error refreshing Twitch token: %v", err)
				}
				break // Since we're only interested in Twitch, we can break early
			}
		}

		if !hasTwitch {
			log.Println("User has not logged in with Twitch")
			http.Error(w, "Unauthorized: Twitch service not logged in", http.StatusUnauthorized)
			return
		}

		// Proceed with the request
		next.ServeHTTP(w, r)
	})
}

func sessionCheckHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		// If the session token is not found, it means the user is not logged in.
		// Instead of returning an error, return a response indicating no session is active.
		// log.Println("Session token not found:", err)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"services": []}`)) // Indicate no services are logged in.
		return
	}

	sessionToken := cookie.Value
	sessionDataJson, err := redisClient.Get(ctx, fmt.Sprintf("session:%s", sessionToken)).Result()
	if err != nil {
		// If session data is not found in Redis, it's likely the session has expired or is invalid.
		// log.Println("Session data not found or expired:", err)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"services": []}`)) // Similarly, indicate no services are logged in.
		return
	}

	// If we reach this point, we have valid session data.
	// Send the session data back to the client.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(sessionDataJson))
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil && cookie != nil {
		// Delete the session token from Redis
		sessionToken := cookie.Value
		_, err := redisClient.Del(ctx, fmt.Sprintf("session:%s", sessionToken)).Result()
		if err != nil {
			// Update this error handling with best practices
			http.Error(w, "Error logging out", http.StatusBadRequest)
			return
		}
	}

	// Invalidate the cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true, // Set to true if you're serving your site over HTTPS
		SameSite: http.SameSiteLaxMode,
	})
	// Redirect to home or login page
	http.Redirect(w, r, "/", http.StatusFound)
}

func refreshToken(service string, sessionToken string) error {
	// Retrieve the session data from Redis
	sessionDataJson, err := redisClient.Get(ctx, fmt.Sprintf("session:%s", sessionToken)).Result()
	if err != nil {
		return fmt.Errorf("failed to retrieve session data: %v", err)
	}

	var sessionData map[string]any
	err = json.Unmarshal([]byte(sessionDataJson), &sessionData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal session data: %v", err)
	}

	// Extract the necessary data for token refresh
	expiry, expiryOk := sessionData["token_expiry"].(int64)
	refreshToken, refreshTokenOk := sessionData["refresh_token"].(string)
	if !expiryOk || !refreshTokenOk || time.Now().Unix() < expiry {
		// Token hasn't expired or necessary data is not available, no refresh needed
		return nil
	}

	var oauthConfig *oauth2.Config = twitchOAuthConfig

	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}
	ts := oauthConfig.TokenSource(context.Background(), token)
	newToken, err := ts.Token() // This refreshes the token
	if err != nil {
		return fmt.Errorf("failed to refresh token: %v", err)
	}

	// Prepare userData with the new token information
	userData := map[string]any{
		fmt.Sprintf("%s_token", service): newToken.AccessToken,
		"refresh_token":                  newToken.RefreshToken,
		"token_expiry":                   newToken.Expiry.Unix(),
	}

	// Use the existing function to update the session data
	// Note: This function already handles Redis update and session cookie reset
	updateSessionDataForService(nil, userData, service, sessionToken) // Assuming w http.ResponseWriter is not required for Redis update

	return nil
}

func SetupAuthRoutes(router *mux.Router) {
	// Existing setup...
	router.HandleFunc("/login/twitch", loginHandler).Methods("GET")
	router.HandleFunc("/callback/twitch", callbackHandler)
	router.HandleFunc("/logout", logoutHandler).Methods("POST")

	// This route is now outside of the authRoutes subrouter to be accessible without both services logged in
	router.HandleFunc("/check-session", sessionCheckHandler).Methods("GET")

	// Subrouter for routes that require authentication
	authRoutes := router.PathPrefix("/auth").Subrouter()
	authRoutes.Use(SessionMiddleware)
}
