package handlers

import (
	"client-dashboard/internal/database"
	"client-dashboard/internal/models"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Session management (simple in-memory for now)
var sessions = make(map[string]int) // sessionToken -> userID

// LoginHandler handles user login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user from database
	user, err := models.GetUserByUsername(database.DB, credentials.Username)
	if err != nil {
		log.Printf("Login failed: User not found - %s (error: %v)", credentials.Username, err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Validate password
	if !user.ValidatePassword(credentials.Password) {
		log.Printf("Login failed: Invalid password for user %s", credentials.Username)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	log.Printf("Login successful for user: %s", credentials.Username)

	// Create session
	sessionToken := generateSessionToken()
	sessions[sessionToken] = user.ID

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400, // 24 hours
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
}

// LogoutHandler handles user logout
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		delete(sessions, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// AuthMiddleware checks if user is authenticated
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		_, exists := sessions[cookie.Value]
		if !exists {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// generateSessionToken generates a simple session token
func generateSessionToken() string {
	return time.Now().Format("20060102150405") + randomString(16)
}

// randomString generates a random string (simplified version)
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// IsSessionValid checks if a session token is valid
func IsSessionValid(token string) bool {
	_, exists := sessions[token]
	return exists
}
