package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/proxy"
)

// NewSOCKS5Transport creates an HTTP transport with SOCKS5 proxy
func NewSOCKS5Transport(proxyURL string) (*http.Transport, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	// Extract host and port
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "1080" // default SOCKS5 port
	}

	// Create SOCKS5 proxy
	auth := &proxy.Auth{}
	if u.User != nil {
		auth.User = u.User.Username()
		auth.Password, _ = u.User.Password()
	}

	dialer, err := proxy.SOCKS5("tcp", net.JoinHostPort(host, port), auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
	}

	transport := &http.Transport{
		DialContext: dialer.(proxy.ContextDialer).DialContext,
	}

	return transport, nil
}

// GetProxyFromEnv retrieves proxy URL from environment variables
func GetProxyFromEnv() string {
	// Check common environment variables
	for _, env := range []string{"TIMAGE_PROXY", "HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		if proxyURL := getEnv(env); proxyURL != "" {
			return proxyURL
		}
	}
	return ""
}

func getEnv(key string) string {
	return os.Getenv(key)
}

// GetProxyURL returns the proxy URL based on priority
func GetProxyURL(cmdProxy string) string {
	// Priority: command line > environment variable
	if cmdProxy != "" {
		return cmdProxy
	}
	return GetProxyFromEnv()
}

// IsProxyEnabled checks if proxy is configured
func IsProxyEnabled(proxyURL string) bool {
	return strings.TrimSpace(proxyURL) != ""
}
