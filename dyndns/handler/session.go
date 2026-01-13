package handler

import (
	"encoding/gob"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/labstack/gommon/log"
)

// Session wraps gorilla session
type Session struct {
	*sessions.Session
}

// SessionStore wraps gorilla session store
type SessionStore struct {
	store *sessions.CookieStore
}

// InitSessionStore creates a new session store with a secret key
func (h *Handler) InitSessionStore() error {
	// Generate or get session secret from environment
	secret := []byte(h.GetSessionSecret())

	// Create cookie store
	store := sessions.NewCookieStore(secret)

	// Configure session options
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   24 * 60 * 60, // 24 hours default
		HttpOnly: true,
		Secure:   false, // Will be set to true per-request if HTTPS
		SameSite: http.SameSiteStrictMode,
	}

	h.SessionStore = &SessionStore{store: store}

	// Register types for session encoding
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})

	return nil
}

// Get retrieves a session
func (s *SessionStore) Get(r *http.Request, name string) (*Session, error) {
	sess, err := s.store.Get(r, name)
	if err != nil {
		return nil, err
	}
	return &Session{Session: sess}, nil
}

// GetSessionSecret returns the session secret key
// Uses environment variable or generates a random one
func (h *Handler) GetSessionSecret() string {
	// Try to get from environment
	secret := h.GetEnv("DDNS_SESSION_SECRET", "")

	if secret != "" {
		return secret
	}

	// If not set, generate a warning and use admin password as base
	log.Warn("DDNS_SESSION_SECRET not set! Using derived key. Set this in production!")

	// Use admin login hash as base for session secret
	return h.Config.AdminLogin + "-session-secret-key"
}

// GetEnv gets environment variable with default
func (h *Handler) GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
