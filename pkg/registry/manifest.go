package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Manifest represents a Docker image manifest (Schema 2)
type Manifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	MediaType     string          `json:"mediaType"`
	Config        Layer           `json:"config"`
	Layers        []Layer         `json:"layers"`
	Manifests     []ManifestEntry `json:"manifests"` // For manifest list
}

// ManifestEntry represents an entry in a manifest list
type ManifestEntry struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Platform  Platform `json:"platform"`
}

// Platform represents the platform in a manifest list
type Platform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Variant      string `json:"variant,omitempty"`
}

// Layer represents a config or layer in a manifest
type Layer struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

// GetManifest fetches the manifest for an image
func (c *Client) GetManifest(name, reference string) (*Manifest, error) {
	path := fmt.Sprintf("/%s/manifests/%s", name, reference)
	headers := map[string]string{
		"Accept": "application/vnd.docker.distribution.manifest.v2+json,application/vnd.docker.distribution.manifest.list.v2+json",
	}

	resp, err := c.doRequest("GET", path, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Get content type
	contentType := resp.Header.Get("Content-Type")

	// Parse manifest
	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	// If it's a manifest list or OCI image index, resolve to the specific manifest for our platform
	isManifestList := manifest.MediaType == "application/vnd.docker.distribution.manifest.list.v2+json" ||
	                  contentType == "application/vnd.docker.distribution.manifest.list.v2+json" ||
	                  manifest.MediaType == "application/vnd.oci.image.index.v1+json" ||
	                  contentType == "application/vnd.oci.image.index.v1+json"

	if isManifestList {
		// Find amd64 manifest
		for _, m := range manifest.Manifests {
			if m.Platform.Architecture == "amd64" && m.Platform.OS == "linux" {
				// Fetch the actual manifest
				return c.GetManifest(name, m.Digest)
			}
		}
		return nil, fmt.Errorf("no amd64 manifest found in manifest list")
	}

	return &manifest, nil
}

// GetManifestRaw fetches the raw manifest bytes
func (c *Client) GetManifestRaw(name, reference string) ([]byte, string, error) {
	path := fmt.Sprintf("/%s/manifests/%s", name, reference)
	headers := map[string]string{
		"Accept": "application/vnd.docker.distribution.manifest.v2+json,application/vnd.docker.distribution.manifest.list.v2+json",
	}

	resp, err := c.doRequest("GET", path, headers)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Get content type
	contentType := resp.Header.Get("Content-Type")

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	return body, contentType, nil
}

// GetManifestDigest gets the digest of a manifest
func (c *Client) GetManifestDigest(name, reference string) (string, error) {
	path := fmt.Sprintf("/%s/manifests/%s", name, reference)
	headers := map[string]string{
		"Accept": "application/vnd.docker.distribution.manifest.v2+json",
	}

	resp, err := c.doRequest("HEAD", path, headers)
	if err != nil {
		return "", fmt.Errorf("failed to head manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	// Docker-Content-Digest header contains the digest
	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", fmt.Errorf("no digest found in response")
	}

	return digest, nil
}

// PutManifest uploads a manifest to the registry
func (c *Client) PutManifest(name, reference string, manifest []byte, contentType string) error {
	path := fmt.Sprintf("/%s/manifests/%s", name, reference)

	// Create request with manifest body
	fullURL := c.baseURL + "/v2" + path
	req, err := http.NewRequest("PUT", fullURL, bytes.NewReader(manifest))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(manifest)))
	c.auth.AddAuth(req)
	req.Header.Set("Docker-Distribution-API-Version", "registry/2.0")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
