package registry

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ioworker0/timage/pkg/proxy"
)

// Client represents a Docker registry client
type Client struct {
	httpClient *http.Client
	auth       *AuthHandler
	baseURL    string
}

// NewClient creates a new registry client
func NewClient(registryURL string, authConfig *AuthConfig, proxyURL string) (*Client, error) {
	// Create HTTP client with proxy support
	httpClient, err := createHTTPClient(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Normalize registry URL
	if registryURL == "" || registryURL == "docker.io" {
		registryURL = "https://registry-1.docker.io"
	}

	if !strings.HasPrefix(registryURL, "http://") && !strings.HasPrefix(registryURL, "https://") {
		registryURL = "https://" + registryURL
	}

	// Parse and validate URL
	parsedURL, err := url.Parse(registryURL)
	if err != nil {
		return nil, fmt.Errorf("invalid registry URL: %w", err)
	}

	auth := NewAuthHandler(httpClient, authConfig)

	return &Client{
		httpClient: httpClient,
		auth:       auth,
		baseURL:    parsedURL.String(),
	}, nil
}

// createHTTPClient creates an HTTP client with proxy support
func createHTTPClient(proxyURL string) (*http.Client, error) {
	if proxyURL != "" {
		return proxy.NewHTTPClient(proxyURL)
	}
	return &http.Client{}, nil
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, path string, headers map[string]string) (*http.Response, error) {
	// Build full URL
	fullURL := c.baseURL + "/v2" + path

	// Create request
	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Add authentication
	c.auth.AddAuth(req)

	// Set default headers
	req.Header.Set("Docker-Distribution-API-Version", "registry/2.0")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Handle authentication challenge
	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.auth.HandleAuthChallenge(req, resp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
		resp.Body.Close()

		// Retry with new auth
		req, err = http.NewRequest(method, fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create retry request: %w", err)
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		c.auth.AddAuth(req)
		req.Header.Set("Docker-Distribution-API-Version", "registry/2.0")

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("retry request failed: %w", err)
		}
	}

	return resp, nil
}

// Ping checks if the registry is accessible
func (c *Client) Ping() error {
	resp, err := c.doRequest("GET", "/", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping failed with status: %d", resp.StatusCode)
	}

	return nil
}

// VerifyCredentials verifies if the current credentials are valid
func (c *Client) VerifyCredentials() error {
	// Try v2 ping first
	resp, err := c.doRequest("GET", "/", nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	// 200 OK means registry is accessible and credentials are valid
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// 401 Unauthorized means credentials are wrong
	if resp.StatusCode == http.StatusUnauthorized {
		authHeader := resp.Header.Get("Www-Authenticate")
		if authHeader != "" {
			return fmt.Errorf("access denied. Server requested: %s\n\nPlease check your username and password.", authHeader)
		}
		return fmt.Errorf("access denied. Please check your username and password.")
	}

	// 401 might be expected if we don't have credentials yet, try with catalog
	return fmt.Errorf("unexpected status code: %d (registry may require authentication)", resp.StatusCode)
}
