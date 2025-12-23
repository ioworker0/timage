package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ioworker0/timage/pkg/registry"
)

// Store manages local image storage
type Store struct {
	layout *Layout
}

// NewStore creates a new storage store
func NewStore(rootDir string) (*Store, error) {
	layout, err := NewLayout(rootDir)
	if err != nil {
		return nil, err
	}

	return &Store{
		layout: layout,
	}, nil
}

// SaveManifest saves the manifest to disk
func (s *Store) SaveManifest(imageName string, manifest *registry.Manifest) error {
	// Create image directory
	if err := s.layout.CreateImageDir(imageName); err != nil {
		return err
	}

	// Marshal manifest
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Write to file
	manifestPath := s.layout.GetManifestPath(imageName)
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// SaveManifestRaw saves the raw manifest bytes to disk
func (s *Store) SaveManifestRaw(imageName string, data []byte) error {
	// Create image directory
	if err := s.layout.CreateImageDir(imageName); err != nil {
		return err
	}

	// Write to file
	manifestPath := s.layout.GetManifestPath(imageName)
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// LoadManifest loads the manifest from disk
func (s *Store) LoadManifest(imageName string) (*registry.Manifest, error) {
	manifestPath := s.layout.GetManifestPath(imageName)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest registry.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	return &manifest, nil
}

// LoadManifestRaw loads the raw manifest bytes from disk
func (s *Store) LoadManifestRaw(imageName string) ([]byte, error) {
	manifestPath := s.layout.GetManifestPath(imageName)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	return data, nil
}

// SaveConfig saves the config blob to disk
func (s *Store) SaveConfig(imageName string, data []byte) error {
	// Create image directory if it doesn't exist
	if err := s.layout.CreateImageDir(imageName); err != nil {
		return err
	}

	configPath := s.layout.GetConfigPath(imageName)

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// LoadConfig loads the config blob from disk
func (s *Store) LoadConfig(imageName string) ([]byte, error) {
	configPath := s.layout.GetConfigPath(imageName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return data, nil
}

// SaveLayer saves a layer blob to disk
func (s *Store) SaveLayer(imageName, digest string, srcPath string) error {
	// Create layers directory
	layersDir := s.layout.GetLayersDir(imageName)
	if err := os.MkdirAll(layersDir, 0755); err != nil {
		return err
	}

	// Destination path
	destPath := s.layout.GetLayerPath(imageName, digest)

	// Copy file
	if err := copyFile(srcPath, destPath); err != nil {
		return fmt.Errorf("failed to copy layer: %w", err)
	}

	return nil
}

// GetLayerPath returns the path to a layer
func (s *Store) GetLayerPath(imageName, digest string) string {
	return s.layout.GetLayerPath(imageName, digest)
}

// GetConfigPath returns the path to the config file
func (s *Store) GetConfigPath(imageName string) string {
	return s.layout.GetConfigPath(imageName)
}

// ListImages returns a list of all stored images
func (s *Store) ListImages() ([]string, error) {
	return s.layout.ListImages()
}

// ImageExists checks if an image exists
func (s *Store) ImageExists(imageName string) bool {
	return s.layout.ImageExists(imageName)
}

// RemoveImage removes an image from storage
func (s *Store) RemoveImage(imageName string) error {
	return s.layout.RemoveImage(imageName)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
