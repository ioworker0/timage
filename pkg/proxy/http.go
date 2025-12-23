package proxy

import (
	"net/http"
	"net/url"
)

// NewHTTPClient creates an HTTP client with proxy support
func NewHTTPClient(proxyURL string) (*http.Client, error) {
	client := &http.Client{}

	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}

		// Check if it's SOCKS5 proxy
		if proxy.Scheme == "socks5" {
			transport, err := NewSOCKS5Transport(proxyURL)
			if err != nil {
				return nil, err
			}
			client.Transport = transport
		} else {
			// HTTP/HTTPS proxy
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxy),
			}
		}
	}

	return client, nil
}
