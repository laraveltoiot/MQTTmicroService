package auth

import (
	"crypto/subtle"
	"net/http"

	"MQTTmicroService/internal/logger"
)

// Config holds the authentication configuration
type Config struct {
	// API key authentication
	EnableAPIKey bool
	APIKeys      []string
}

// Auth handles authentication for the API
type Auth struct {
	config *Config
	logger *logger.Logger
}

// GetEnableAPIKey returns the value of the EnableAPIKey flag
func (a *Auth) GetEnableAPIKey() bool {
	return a.config.EnableAPIKey
}

// New creates a new Auth instance
func New(config *Config, log *logger.Logger) *Auth {
	return &Auth{
		config: config,
		logger: log,
	}
}

// DefaultConfig returns the default authentication configuration
func DefaultConfig() *Config {
	return &Config{
		EnableAPIKey: false,
		APIKeys:      []string{},
	}
}

// ValidateAPIKey validates an API key
func (a *Auth) ValidateAPIKey(apiKey string) bool {
	// Log that we're validating an API key
	a.logger.WithFields(map[string]interface{}{
		"enableAPIKey": a.config.EnableAPIKey,
	}).Info("Validating API key")

	if !a.config.EnableAPIKey {
		a.logger.Info("API key validation skipped: API key authentication is disabled")
		return false
	}

	for _, key := range a.config.APIKeys {
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(key)) == 1 {
			a.logger.Info("API key validation successful")
			return true
		}
	}

	a.logger.Info("API key validation failed: invalid API key")
	return false
}

// AuthMiddleware is a middleware that authenticates requests using API keys
func (a *Auth) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for health check endpoint
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip authentication if API key authentication is not enabled
		if !a.config.EnableAPIKey {
			// Log that we're skipping authentication because it's disabled
			a.logger.WithFields(map[string]interface{}{
				"path":          r.URL.Path,
				"enableAPIKey":  a.config.EnableAPIKey,
			}).Info("Skipping authentication: API key authentication is disabled")
			next.ServeHTTP(w, r)
			return
		}

		// Check for API key in header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// Check for API key in query parameter
			apiKey = r.URL.Query().Get("api_key")
		}

		// Check for Bearer token in Authorization header
		if apiKey == "" {
			authHeader := r.Header.Get("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				apiKey = authHeader[7:]
			}
		}

		// Validate API key
		if apiKey != "" && a.ValidateAPIKey(apiKey) {
			next.ServeHTTP(w, r)
			return
		}

		// Authentication failed
		a.logger.WithField("path", r.URL.Path).Info("Authentication failed: invalid or missing API key")
		http.Error(w, "Unauthorized: invalid or missing API key", http.StatusUnauthorized)
	})
}
