package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Layout manages the storage directory layout
type Layout struct {
	rootDir string
}

// NewLayout creates a new storage layout manager
func NewLayout(rootDir string) (*Layout, error) {
	// Create root directory if it doesn't exist
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create root directory: %w", err)
	}

	return &Layout{
		rootDir: rootDir,
	}, nil
}

// GetImageDir returns the directory for a specific image
// Image name format: registry/namespace/name:tag
// e.g., docker.io/library/nginx:latest
func (l *Layout) GetImageDir(imageName string) string {
	// Replace special characters with underscores
	safeName := strings.ReplaceAll(imageName, ":", "_")
	safeName = strings.ReplaceAll(safeName, "/", "_")

	return filepath.Join(l.rootDir, "images", safeName)
}

// GetManifestPath returns the path to the manifest file
func (l *Layout) GetManifestPath(imageName string) string {
	return filepath.Join(l.GetImageDir(imageName), "manifest.json")
}

// GetConfigPath returns the path to the config file
func (l *Layout) GetConfigPath(imageName string) string {
	return filepath.Join(l.GetImageDir(imageName), "config.json")
}

// GetLayersDir returns the directory for layers
func (l *Layout) GetLayersDir(imageName string) string {
	return filepath.Join(l.GetImageDir(imageName), "layers")
}

// GetLayerPath returns the path to a specific layer
func (l *Layout) GetLayerPath(imageName, digest string) string {
	// Use digest as filename (with : replaced by _)
	safeDigest := strings.ReplaceAll(digest, ":", "_")
	return filepath.Join(l.GetLayersDir(imageName), safeDigest+".tar.gz")
}

// CreateImageDir creates the directory structure for an image
func (l *Layout) CreateImageDir(imageName string) error {
	layersDir := l.GetLayersDir(imageName)

	if err := os.MkdirAll(layersDir, 0755); err != nil {
		return fmt.Errorf("failed to create image directory: %w", err)
	}

	return nil
}

// ListImages returns a list of all stored images
func (l *Layout) ListImages() ([]string, error) {
	imagesDir := filepath.Join(l.rootDir, "images")

	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read images directory: %w", err)
	}

	var images []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Convert directory name back to image name
			imageName := strings.ReplaceAll(entry.Name(), "_", ":")
			imageName = strings.ReplaceAll(imageName, "___", "/") // Temporary fix
			images = append(images, imageName)
		}
	}

	return images, nil
}

// ImageExists checks if an image exists in storage
func (l *Layout) ImageExists(imageName string) bool {
	manifestPath := l.GetManifestPath(imageName)
	_, err := os.Stat(manifestPath)
	return err == nil
}

// RemoveImage removes an image from storage
func (l *Layout) RemoveImage(imageName string) error {
	imageDir := l.GetImageDir(imageName)

	if err := os.RemoveAll(imageDir); err != nil {
		return fmt.Errorf("failed to remove image directory: %w", err)
	}

	return nil
}
