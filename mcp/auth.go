package mcp

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"
)

// validateBearerToken validates a Bearer token from the Authorization header
// Returns false if expectedToken is empty (auth misconfigured - fail secure)
func validateBearerToken(r *http.Request, expectedToken string) bool {
	if expectedToken == "" {
		log.Println("Warning: Bearer auth configured but token is empty - denying request")
		return false // Fail secure - don't allow empty token bypass
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		// Also check query parameter for SSE compatibility
		token := r.URL.Query().Get("token")
		return token == expectedToken
	}

	// Check Bearer token
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	return token == expectedToken
}

// validateBasicAuth validates Basic authentication
// Returns false if both username and password are empty (auth misconfigured - fail secure)
func validateBasicAuth(r *http.Request, expectedUser, expectedPass string) bool {
	if expectedUser == "" && expectedPass == "" {
		log.Println("Warning: Basic auth configured but credentials are empty - denying request")
		return false // Fail secure - don't allow empty credentials bypass
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	if !strings.HasPrefix(authHeader, "Basic ") {
		return false
	}

	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return false
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return false
	}

	return credentials[0] == expectedUser && credentials[1] == expectedPass
}

// validateAPIKey validates an API key from a custom header
// Returns false if expectedKey is empty (auth misconfigured - fail secure)
func validateAPIKey(r *http.Request, headerName, expectedKey string) bool {
	if expectedKey == "" {
		log.Println("Warning: API key auth configured but key is empty - denying request")
		return false // Fail secure - don't allow empty key bypass
	}

	apiKey := r.Header.Get(headerName)
	if apiKey == "" {
		// Also check query parameter
		apiKey = r.URL.Query().Get("api_key")
	}

	return apiKey == expectedKey
}

// ContextKey is a type for context keys
type ContextKey string

const (
	// ContextKeyUser is the key for the authenticated user in context
	ContextKeyUser ContextKey = "user"
	// ContextKeyAuthType is the key for the authentication type in context
	ContextKeyAuthType ContextKey = "auth_type"
)
