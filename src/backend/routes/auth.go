package routes

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ravener/discord-oauth2"
	"golang.org/x/oauth2"
)

// Setup the OAuth2 config with Discord's endpoints and your credentials.
var (
	clientID     = "1215459352987832350"
	clientSecret = "FDRj6lnFl7LoCQsVjIl84zA-rVtabiBN"
	redirectURL  = "http://localhost:8080/callback"
	oauth2Config = &oauth2.Config{
		RedirectURL:  redirectURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"identify", "email"},
		Endpoint:     discord.Endpoint,
	}
)

// loginHandler to initiate OAuth with Discord
func loginHandler(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// Store "valid" as the state value for later validation
	err = redisClient.Set(ctx, "oauth-state:"+state, "valid", 10*time.Minute).Err()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	url := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// callbackHandler for Discord's OAuth
func callbackHandler(w http.ResponseWriter, r *http.Request) {
	// Check if an error query parameter is present
	if errorReason := r.FormValue("error"); errorReason != "" {
		// Optional: log the error reason or display it to the user
		fmt.Printf("OAuth error: %s, Description: %s\n", errorReason, r.FormValue("error_description"))

		// Redirect back to the main page or an appropriate error page
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
	token, err := oauth2Config.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Use the access token to access the Discord API
	client := oauth2Config.Client(context.Background(), token)
	res, err := client.Get(discord.Endpoint.TokenURL + "/users/@me")
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		http.Error(w, "Failed to read response: "+err.Error(), http.StatusInternalServerError)
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

	if err != nil || storedState != "valid" {
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
			// Handle error (optional)
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
	router.HandleFunc("/login", loginHandler)
	router.HandleFunc("/callback", callbackHandler)

	// Subrouter for routes that require authentication
	authRoutes := router.PathPrefix("/auth").Subrouter()
	authRoutes.Use(SessionMiddleware)

	// Register the session check endpoint
	authRoutes.HandleFunc("/check-session", sessionCheckHandler).Methods("GET")

	// Register logout endpoint
	authRoutes.HandleFunc("/logout", logoutHandler).Methods("POST")
	// Other protected routes...
}
