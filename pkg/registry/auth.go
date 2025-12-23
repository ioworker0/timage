package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// TokenResponse represents the token response from auth server
type TokenResponse struct {
	Token        string `json:"token"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	IssuedAt     string `json:"issued_at"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Username string
	Password string
	Token    string
}

// AuthHandler handles authentication with Docker registry
type AuthHandler struct {
	client *http.Client
	auth   *AuthConfig
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(client *http.Client, auth *AuthConfig) *AuthHandler {
	return &AuthHandler{
		client: client,
		auth:   auth,
	}
}

// AddAuth adds authentication headers to the request
func (a *AuthHandler) AddAuth(req *http.Request) {
	// If we have a Bearer token, use it
	if a.auth.Token != "" {
		req.Header.Set("Authorization", "Bearer "+a.auth.Token)
		return
	}

	// Otherwise, use basic auth
	if a.auth.Username != "" && a.auth.Password != "" {
		auth := basicAuth(a.auth.Username, a.auth.Password)
		req.Header.Set("Authorization", "Basic "+auth)
	}
}

// basicAuth creates a basic authentication string
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// HandleAuthChallenge handles 401 Unauthorized responses
func (a *AuthHandler) HandleAuthChallenge(req *http.Request, resp *http.Response) error {
	authHeader := resp.Header.Get("Www-Authenticate")
	if authHeader == "" {
		return fmt.Errorf("no Www-Authenticate header found")
	}

	// Parse the auth header to get the realm
	if strings.HasPrefix(authHeader, "Bearer") {
		// Bearer token authentication
		return a.handleBearerAuth(req, authHeader)
	}

	if strings.HasPrefix(authHeader, "Basic") {
		// Basic authentication - already added by AddAuth, just retry
		// This might happen if the credentials are wrong, but we'll retry anyway
		return nil
	}

	return fmt.Errorf("unsupported authentication method: %s", authHeader)
}

// handleBearerAuth handles Bearer token authentication
func (a *AuthHandler) handleBearerAuth(req *http.Request, authHeader string) error {
	// Extract the realm from Www-Authenticate header
	// Format: Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/nginx:pull"
	params := parseAuthHeader(authHeader)
	realm, ok := params["realm"]
	if !ok {
		return fmt.Errorf("no realm found in Www-Authenticate header")
	}

	service := params["service"]
	scope := params["scope"]

	// Fetch token from auth server
	token, err := a.fetchToken(realm, service, scope)
	if err != nil {
		return fmt.Errorf("failed to fetch token: %w", err)
	}

	// Save the token for future requests
	a.auth.Token = token

	return nil
}

// fetchToken fetches a Bearer token from the auth server
func (a *AuthHandler) fetchToken(realm, service, scope string) (string, error) {
	// Build token request URL
	tokenURL, err := buildTokenURL(realm, service, scope)
	if err != nil {
		return "", err
	}

	// Create request
	tokenReq, err := http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	// Add basic auth if we have credentials
	if a.auth.Username != "" && a.auth.Password != "" {
		auth := basicAuth(a.auth.Username, a.auth.Password)
		tokenReq.Header.Set("Authorization", "Basic "+auth)
	}

	// Send request
	resp, err := a.client.Do(tokenReq)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	// Use token or access_token (they should be the same)
	if tokenResp.Token != "" {
		return tokenResp.Token, nil
	}
	if tokenResp.AccessToken != "" {
		return tokenResp.AccessToken, nil
	}

	return "", fmt.Errorf("no token in response")
}

// buildTokenURL builds the token request URL
func buildTokenURL(realm, service, scope string) (string, error) {
	u, err := url.Parse(realm)
	if err != nil {
		return "", err
	}

	query := u.Query()
	if service != "" {
		query.Set("service", service)
	}
	if scope != "" {
		query.Set("scope", scope)
	}

	// For anonymous access to public images, we don't need to specify scope
	// The registry will tell us what scope we need

	u.RawQuery = query.Encode()
	return u.String(), nil
}

// parseAuthHeader parses the Www-Authenticate header
func parseAuthHeader(header string) map[string]string {
	params := make(map[string]string)

	// Remove "Bearer " prefix
	header = strings.TrimPrefix(header, "Bearer ")

	// Parse key="value" pairs
	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.Trim(kv[1], `"`)
			params[key] = value
		}
	}

	return params
}
