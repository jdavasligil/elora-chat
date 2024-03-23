package routes

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/twitch"
)

// Twitch OAuth configuration
var twitchOAuthConfig = &oauth2.Config{
	ClientID:     "yzn1qir54by5q528tinhfranwu6o8c",
	ClientSecret: "4rvvit22eqqia76dwekw4s2godt5hy",
	RedirectURL:  "http://localhost:8080/callback/twitch",
	Scopes:       []string{"user:read:email"}, // Example scope, adjust as needed
	Endpoint:     twitch.Endpoint,             // Make sure to import "golang.org/x/oauth2/twitch"
}

// YouTube OAuth configuration
var youtubeOAuthConfig = &oauth2.Config{
	ClientID:     "456484052696-173incl6ktid55uff5f9jboucvu742l7.apps.googleusercontent.com",
	ClientSecret: "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",
	RedirectURL:  "http://localhost:8080/callback/youtube",
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"}, // Example scope, adjust as needed
	Endpoint:     google.Endpoint,                                            // Make sure to import "golang.org/x/oauth2/google"
}

// loginHandler to initiate OAuth with Twitch/YouTube
func loginHandler(w http.ResponseWriter, r *http.Request) {
	var oauthConfig *oauth2.Config

	// Determine which platform is being requested
	switch {
	case strings.Contains(r.URL.Path, "/twitch"):
		oauthConfig = twitchOAuthConfig
	case strings.Contains(r.URL.Path, "/youtube"):
		oauthConfig = youtubeOAuthConfig
	default:
		http.Error(w, "Unsupported platform", http.StatusBadRequest)
		return
	}

	state, err := generateState()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Store "valid" as the state value for later validation, including which platform it's for
	err = redisClient.Set(ctx, "oauth-state:"+state, "valid:"+oauthConfig.ClientID, 10*time.Minute).Err()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Redirect to the OAuth provider's authorization page
	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// callbackHandler for Twitch/YT OAuth
func callbackHandler(w http.ResponseWriter, r *http.Request) {
	var oauthConfig *oauth2.Config
	var userInfoURL string

	// Determine which platform the callback is for
	switch {
	case strings.Contains(r.URL.Path, "/twitch"):
		oauthConfig = twitchOAuthConfig
		userInfoURL = "https://api.twitch.tv/helix/users"
	case strings.Contains(r.URL.Path, "/youtube"):
		oauthConfig = youtubeOAuthConfig
		userInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
	default:
		http.Error(w, "Unsupported platform", http.StatusBadRequest)
		return
	}

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

	// Use the access token to fetch user information from the appropriate API
	client := oauthConfig.Client(context.Background(), token)

	var req *http.Request
	var res *http.Response

	switch {
	case strings.Contains(r.URL.Path, "/twitch"):
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

	case strings.Contains(r.URL.Path, "/youtube"):
		// YouTube login works fine, keep this block unchanged
		userInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
		res, err = client.Get(userInfoURL)
	}

	if err != nil || res.StatusCode != 200 {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	// Unmarshal the user data
	var userData map[string]interface{}
	if err = json.Unmarshal(body, &userData); err != nil {
		http.Error(w, "Failed to parse user data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a new session token
	sessionToken := generateSessionToken()
	userData["session_token"] = sessionToken

	// Store the session data in Redis with a TTL
	ctx := context.Background()
	sessionData, err := json.Marshal(userData)
	if err != nil {
		http.Error(w, "Failed to marshal session data: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = redisClient.Set(ctx, fmt.Sprintf("session:%s", sessionToken), sessionData, 24*time.Hour).Err()
	if err != nil {
		http.Error(w, "Failed to store session data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,                 // Set to true if you're serving your site over HTTPS
		SameSite: http.SameSiteLaxMode, // Adjust this based on your cross-origin policy needs
	})

	http.Redirect(w, r, "/", http.StatusFound)
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

// sessionMiddleware checks for a valid session token in the request cookies.
func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the session token from the request cookies
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized: No session token", http.StatusUnauthorized)
			return
		}

		sessionToken := cookie.Value

		// Validate the session token by checking if it exists in Redis
		ctx := context.Background()
		_, err = redisClient.Get(ctx, fmt.Sprintf("session:%s", sessionToken)).Result()
		if err != nil {
			http.Error(w, "Unauthorized: Invalid session token", http.StatusUnauthorized)
			return
		}

		// Token is valid, proceed with the request
		next.ServeHTTP(w, r)
	})
}

func sessionCheckHandler(w http.ResponseWriter, r *http.Request) {
	// If this handler is reached, sessionMiddleware has already verified the session token.
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Session is valid"))
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

func SetupAuthRoutes(router *mux.Router) {
	// Existing setup...
	router.HandleFunc("/login/twitch", loginHandler).Methods("GET")
	router.HandleFunc("/login/youtube", loginHandler).Methods("GET")
	router.HandleFunc("/callback/twitch", callbackHandler)
	router.HandleFunc("/callback/youtube", callbackHandler)

	// Subrouter for routes that require authentication
	authRoutes := router.PathPrefix("/auth").Subrouter()
	authRoutes.Use(SessionMiddleware)

	// Register the session check endpoint
	authRoutes.HandleFunc("/check-session", sessionCheckHandler).Methods("GET")

	// Register logout endpoint
	authRoutes.HandleFunc("/logout", logoutHandler).Methods("POST")
	// Other protected routes...
}
