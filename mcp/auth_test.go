package mcp

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateBearerToken(t *testing.T) {
	tests := []struct {
		name          string
		expectedToken string
		authHeader    string
		queryToken    string
		want          bool
	}{
		{
			name:          "valid bearer token in header",
			expectedToken: "secret-token",
			authHeader:    "Bearer secret-token",
			want:          true,
		},
		{
			name:          "invalid bearer token",
			expectedToken: "secret-token",
			authHeader:    "Bearer wrong-token",
			want:          false,
		},
		{
			name:          "missing bearer prefix",
			expectedToken: "secret-token",
			authHeader:    "secret-token",
			want:          false,
		},
		{
			name:          "empty auth header, valid query token",
			expectedToken: "secret-token",
			authHeader:    "",
			queryToken:    "secret-token",
			want:          true,
		},
		{
			name:          "empty auth header, invalid query token",
			expectedToken: "secret-token",
			authHeader:    "",
			queryToken:    "wrong-token",
			want:          false,
		},
		{
			name:          "empty expected token - fail secure",
			expectedToken: "",
			authHeader:    "Bearer anything",
			want:          false,
		},
		{
			name:          "empty expected token with empty request - fail secure",
			expectedToken: "",
			authHeader:    "",
			want:          false,
		},
		{
			name:          "case sensitive token",
			expectedToken: "Secret-Token",
			authHeader:    "Bearer secret-token",
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.queryToken != "" {
				q := req.URL.Query()
				q.Set("token", tt.queryToken)
				req.URL.RawQuery = q.Encode()
			}

			got := validateBearerToken(req, tt.expectedToken)
			if got != tt.want {
				t.Errorf("validateBearerToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateBasicAuth(t *testing.T) {
	tests := []struct {
		name         string
		expectedUser string
		expectedPass string
		authHeader   string
		want         bool
	}{
		{
			name:         "valid credentials",
			expectedUser: "admin",
			expectedPass: "password123",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:password123")),
			want:         true,
		},
		{
			name:         "invalid password",
			expectedUser: "admin",
			expectedPass: "password123",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:wrongpass")),
			want:         false,
		},
		{
			name:         "invalid username",
			expectedUser: "admin",
			expectedPass: "password123",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte("wronguser:password123")),
			want:         false,
		},
		{
			name:         "missing auth header",
			expectedUser: "admin",
			expectedPass: "password123",
			authHeader:   "",
			want:         false,
		},
		{
			name:         "wrong auth type",
			expectedUser: "admin",
			expectedPass: "password123",
			authHeader:   "Bearer sometoken",
			want:         false,
		},
		{
			name:         "invalid base64",
			expectedUser: "admin",
			expectedPass: "password123",
			authHeader:   "Basic not-valid-base64!!!",
			want:         false,
		},
		{
			name:         "missing colon separator",
			expectedUser: "admin",
			expectedPass: "password123",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte("adminpassword123")),
			want:         false,
		},
		{
			name:         "empty credentials - fail secure",
			expectedUser: "",
			expectedPass: "",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte(":")),
			want:         false,
		},
		{
			name:         "password with colon",
			expectedUser: "admin",
			expectedPass: "pass:word:123",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:pass:word:123")),
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			got := validateBasicAuth(req, tt.expectedUser, tt.expectedPass)
			if got != tt.want {
				t.Errorf("validateBasicAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name        string
		headerName  string
		expectedKey string
		headers     map[string]string
		queryKey    string
		want        bool
	}{
		{
			name:        "valid API key in default header",
			headerName:  "X-API-Key",
			expectedKey: "my-api-key",
			headers:     map[string]string{"X-API-Key": "my-api-key"},
			want:        true,
		},
		{
			name:        "valid API key in custom header",
			headerName:  "X-Custom-Auth",
			expectedKey: "my-api-key",
			headers:     map[string]string{"X-Custom-Auth": "my-api-key"},
			want:        true,
		},
		{
			name:        "invalid API key",
			headerName:  "X-API-Key",
			expectedKey: "my-api-key",
			headers:     map[string]string{"X-API-Key": "wrong-key"},
			want:        false,
		},
		{
			name:        "missing header, valid query param",
			headerName:  "X-API-Key",
			expectedKey: "my-api-key",
			headers:     map[string]string{},
			queryKey:    "my-api-key",
			want:        true,
		},
		{
			name:        "missing header, invalid query param",
			headerName:  "X-API-Key",
			expectedKey: "my-api-key",
			headers:     map[string]string{},
			queryKey:    "wrong-key",
			want:        false,
		},
		{
			name:        "empty expected key - fail secure",
			headerName:  "X-API-Key",
			expectedKey: "",
			headers:     map[string]string{"X-API-Key": "anything"},
			want:        false,
		},
		{
			name:        "case sensitive key",
			headerName:  "X-API-Key",
			expectedKey: "My-Api-Key",
			headers:     map[string]string{"X-API-Key": "my-api-key"},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			if tt.queryKey != "" {
				q := req.URL.Query()
				q.Set("api_key", tt.queryKey)
				req.URL.RawQuery = q.Encode()
			}

			got := validateAPIKey(req, tt.headerName, tt.expectedKey)
			if got != tt.want {
				t.Errorf("validateAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthMiddlewareIntegration(t *testing.T) {
	// Test that auth functions work with real HTTP handlers
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Test bearer auth middleware
	bearerMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !validateBearerToken(r, "test-token") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// Test with valid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	bearerMiddleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Test with invalid token
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rr = httptest.NewRecorder()
	bearerMiddleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}
