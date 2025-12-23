package registry

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DownloadBlob downloads a blob (layer or config) from the registry
func (c *Client) DownloadBlob(name, digest string, destPath string) error {
	path := fmt.Sprintf("/%s/blobs/%s", name, digest)

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return fmt.Errorf("failed to get blob: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create destination file
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy blob data to file
	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("failed to write blob: %w", err)
	}

	return nil
}

// DownloadBlobWithProgress downloads a blob with progress reporting
func (c *Client) DownloadBlobWithProgress(name, digest, destPath string, progress func(int64, int64)) error {
	path := fmt.Sprintf("/%s/blobs/%s", name, digest)

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return fmt.Errorf("failed to get blob: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	// Get content length
	size := resp.ContentLength

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create destination file
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy with progress
	writer := &progressWriter{
		writer:   file,
		total:    size,
		progress: progress,
	}

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("failed to write blob: %w", err)
	}

	return nil
}

// progressWriter wraps an io.Writer to report progress
type progressWriter struct {
	writer   io.Writer
	total    int64
	written  int64
	progress func(int64, int64)
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	pw.written += int64(n)

	if pw.progress != nil {
		pw.progress(pw.written, pw.total)
	}

	return n, err
}

// CheckBlob checks if a blob exists in the registry
func (c *Client) CheckBlob(name, digest string) (bool, error) {
	path := fmt.Sprintf("/%s/blobs/%s", name, digest)

	resp, err := c.doRequest("HEAD", path, nil)
	if err != nil {
		return false, fmt.Errorf("failed to check blob: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status code %d", resp.StatusCode)
}

// GetBlobSize returns the size of a blob
func (c *Client) GetBlobSize(name, digest string) (int64, error) {
	path := fmt.Sprintf("/%s/blobs/%s", name, digest)

	resp, err := c.doRequest("HEAD", path, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to head blob: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return resp.ContentLength, nil
}

// UploadBlob uploads a single blob (monolithic upload)
func (c *Client) UploadBlob(name, digest, srcPath string) error {
	// Open the source file
	file, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Start upload session
	uploadURL, err := c.StartBlobUpload(name)
	if err != nil {
		return fmt.Errorf("failed to start upload: %w", err)
	}

	// Upload the blob
	if err := c.UploadBlobMonolithic(uploadURL, file, fileInfo.Size(), digest); err != nil {
		return fmt.Errorf("failed to upload blob: %w", err)
	}

	return nil
}

// StartBlobUpload initiates a blob upload session
func (c *Client) StartBlobUpload(name string) (string, error) {
	path := fmt.Sprintf("/%s/blobs/uploads/", name)

	resp, err := c.doRequest("POST", path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to start upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	// Get upload URL from Location header
	uploadURL := resp.Header.Get("Location")
	if uploadURL == "" {
		return "", fmt.Errorf("no upload URL in response")
	}

	// Handle relative URLs
	if !strings.HasPrefix(uploadURL, "http://") && !strings.HasPrefix(uploadURL, "https://") {
		uploadURL = c.baseURL + "/v2" + uploadURL
	}

	return uploadURL, nil
}

// UploadBlobMonolithic uploads a blob in a single request
func (c *Client) UploadBlobMonolithic(uploadURL string, reader io.Reader, size int64, digest string) error {
	// Create request
	req, err := http.NewRequest("PUT", uploadURL, reader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", size))
	req.ContentLength = size

	// Add digest query parameter (preserving existing query params)
	if req.URL.RawQuery != "" {
		req.URL.RawQuery += "&digest=" + digest
	} else {
		req.URL.RawQuery = "digest=" + digest
	}

	// Add auth
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
