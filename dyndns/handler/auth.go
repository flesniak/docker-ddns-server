package handler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

// Session key constants
const (
	SessionName      = "ddns_session"
	SessionUserKey   = "user"
	SessionAuthKey   = "authenticated"
	SessionCreatedAt = "created_at"
	SessionExpiresAt = "expires_at"
)

// ShowLoginPage renders the login page
func (h *Handler) ShowLoginPage(c echo.Context) error {
	// Check if already authenticated
	if h.IsAuthenticated(c) {
		return c.Redirect(http.StatusFound, "/@/hosts")
	}

	// Check if there's an error message from failed login
	errorMsg := c.QueryParam("error")

	return c.Render(http.StatusOK, "login", echo.Map{
		"title":        h.Title,
		"logoPath":     h.LogoPath,
		"poweredBy":    h.PoweredBy,
		"poweredByUrl": h.PoweredByUrl,
		"error":        errorMsg,
	})
}

// HandleLogin processes login form submission
func (h *Handler) HandleLogin(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")
	rememberMe := c.FormValue("remember_me") == "on"

	// Get client IP for logging
	clientIP := ExtractIPFromRequest(
		c.Request().RemoteAddr,
		c.Request().Header.Get("X-Forwarded-For"),
		c.Request().Header.Get("X-Real-IP"),
	)

	// Validate credentials
	authenticated, err := h.authByEnv(username, password)
	if err != nil {
		log.Error("Authentication error:", err)
		h.LogFailedAuth(clientIP, c.Request().UserAgent(), c.Path(), username, password)
		return c.Redirect(http.StatusFound, "/@/login?error=authentication_error")
	}

	if !authenticated {
		log.Warnf("Failed login attempt from IP %s, username: %s", clientIP, username)
		h.LogFailedAuth(clientIP, c.Request().UserAgent(), c.Path(), username, password)
		return c.Redirect(http.StatusFound, "/@/login?error=invalid_credentials")
	}

	// Authentication successful - create session
	sess, err := h.GetSession(c)
	if err != nil {
		log.Error("Session creation error:", err)
		return c.Redirect(http.StatusFound, "/@/login?error=session_error")
	}

	// Set session values
	sess.Values[SessionUserKey] = username
	sess.Values[SessionAuthKey] = true
	sess.Values[SessionCreatedAt] = time.Now().Unix()

	// Set expiration based on remember me
	if rememberMe {
		sess.Options.MaxAge = 30 * 24 * 60 * 60 // 30 days
		sess.Values[SessionExpiresAt] = time.Now().Add(30 * 24 * time.Hour).Unix()
	} else {
		sess.Options.MaxAge = 24 * 60 * 60 // 24 hours
		sess.Values[SessionExpiresAt] = time.Now().Add(24 * time.Hour).Unix()
	}

	// Set secure flag if using HTTPS
	if h.IsHTTPS(c) {
		sess.Options.Secure = true
	}

	// Save session
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Error("Session save error:", err)
		return c.Redirect(http.StatusFound, "/@/login?error=session_error")
	}

	log.Infof("Successful login from IP %s, username: %s", clientIP, username)

	// Redirect to originally requested page or default to hosts
	redirect := c.QueryParam("redirect")
	if redirect == "" || redirect == "/@/login" {
		redirect = "/@/hosts"
	}
	return c.Redirect(http.StatusFound, redirect)
}

// HandleLogout destroys the session and logs out the user
func (h *Handler) HandleLogout(c echo.Context) error {
	sess, err := h.GetSession(c)
	if err == nil {
		// Get username before destroying session
		username := "unknown"
		if user, ok := sess.Values[SessionUserKey].(string); ok {
			username = user
		}

		// Destroy session
		sess.Options.MaxAge = -1
		sess.Values = make(map[interface{}]interface{})
		sess.Save(c.Request(), c.Response())

		if username != "" {
			log.Infof("User %s logged out", username)
		}
	}

	// Clear session cookie
	c.SetCookie(&http.Cookie{
		Name:     SessionName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.IsHTTPS(c),
		SameSite: http.SameSiteStrictMode,
	})

	// ALWAYS render logout page (not redirect)
	// Pass LogoutUrl so JavaScript can handle delayed redirect
	return c.Render(http.StatusOK, "logout", echo.Map{
		"title":        h.Title,
		"logoPath":     h.LogoPath,
		"logoutUrl":    h.LogoutUrl, // Pass the logout URL to template
		"poweredBy":    h.PoweredBy,
		"poweredByUrl": h.PoweredByUrl,
	})
}

// IsAuthenticated checks if the current session is authenticated
func (h *Handler) IsAuthenticated(c echo.Context) bool {
	if h.DisableAdminAuth {
		return true
	}

	sess, err := h.GetSession(c)
	if err != nil {
		return false
	}

	// Check if authenticated
	authenticated, ok := sess.Values[SessionAuthKey].(bool)
	if !ok || !authenticated {
		return false
	}

	// Check if session has expired
	if expiresAt, ok := sess.Values[SessionExpiresAt].(int64); ok {
		if time.Now().Unix() > expiresAt {
			log.Info("Session expired")
			return false
		}
	}

	return true
}

// GetSession retrieves or creates a session for the request
func (h *Handler) GetSession(c echo.Context) (*Session, error) {
	return h.SessionStore.Get(c.Request(), SessionName)
}

// GenerateCSRFToken generates a random CSRF token
func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// IsHTTPS checks if the request came via HTTPS
// Checks both direct HTTPS and reverse proxy headers
func (h *Handler) IsHTTPS(c echo.Context) bool {
	// Check if direct HTTPS
	if c.Request().TLS != nil {
		return true
	}

	// Check reverse proxy headers
	proto := c.Request().Header.Get("X-Forwarded-Proto")
	if proto == "https" {
		return true
	}

	// Check other common headers
	if c.Request().Header.Get("X-Forwarded-Ssl") == "on" {
		return true
	}

	if c.Request().Header.Get("X-Url-Scheme") == "https" {
		return true
	}

	return false
}

// GetHTTPSRedirectURL constructs the HTTPS version of the current URL
func (h *Handler) GetHTTPSRedirectURL(c echo.Context) string {
	host := c.Request().Host
	uri := c.Request().RequestURI
	return "https://" + host + uri
}
