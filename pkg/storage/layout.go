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
	// Replace : with _COLON_ and / with _SLASH_ to preserve the original format
	safeName := strings.ReplaceAll(imageName, ":", "_COLON_")
	safeName = strings.ReplaceAll(safeName, "/", "_SLASH_")

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

// decodeOldEncoding decodes old encoding format (where both / and : were replaced with _)
// Uses heuristic: last _ is likely the tag separator (:), others are path separators (/)
func decodeOldEncoding(dirName string) string {
	// Find the last underscore
	lastUnderscore := strings.LastIndex(dirName, "_")
	if lastUnderscore == -1 {
		return dirName
	}

	// Replace last _ with : (tag separator)
	// Replace all other _ with / (path separators)
	beforeTag := strings.ReplaceAll(dirName[:lastUnderscore], "_", "/")
	tag := dirName[lastUnderscore+1:]
	return beforeTag + ":" + tag
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
			dirName := entry.Name()
			var imageName string

			// Detect encoding format
			if strings.Contains(dirName, "_SLASH_") || strings.Contains(dirName, "_COLON_") {
				// New encoding format
				imageName = strings.ReplaceAll(dirName, "_SLASH_", "/")
				imageName = strings.ReplaceAll(imageName, "_COLON_", ":")
			} else {
				// Old encoding format (both / and : were replaced with _)
				imageName = decodeOldEncoding(dirName)
			}

			images = append(images, imageName)
		}
	}

	return images, nil
}

// encodeOldEncoding encodes image name using old format (both / and : replaced with _)
func encodeOldEncoding(imageName string) string {
	safeName := strings.ReplaceAll(imageName, ":", "_")
	safeName = strings.ReplaceAll(safeName, "/", "_")
	return safeName
}

// findImageDir finds the actual directory for an image, checking both new and old encoding formats
func (l *Layout) findImageDir(imageName string) string {
	imagesDir := filepath.Join(l.rootDir, "images")

	// Try new encoding format first
	newDirName := strings.ReplaceAll(imageName, ":", "_COLON_")
	newDirName = strings.ReplaceAll(newDirName, "/", "_SLASH_")
	newPath := filepath.Join(imagesDir, newDirName)
	if _, err := os.Stat(newPath); err == nil {
		return newPath
	}

	// Try old encoding format
	oldDirName := encodeOldEncoding(imageName)
	oldPath := filepath.Join(imagesDir, oldDirName)
	if _, err := os.Stat(oldPath); err == nil {
		return oldPath
	}

	return ""
}

// ImageExists checks if an image exists in storage
func (l *Layout) ImageExists(imageName string) bool {
	return l.findImageDir(imageName) != ""
}

// RemoveImage removes an image from storage
func (l *Layout) RemoveImage(imageName string) error {
	imageDir := l.findImageDir(imageName)
	if imageDir == "" {
		return fmt.Errorf("image not found")
	}

	if err := os.RemoveAll(imageDir); err != nil {
		return fmt.Errorf("failed to remove image directory: %w", err)
	}

	return nil
}
