package config

import (
	"os"
	"path/filepath"
)

// AuthEntry represents authentication credentials for a registry
type AuthEntry struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
}

// GetAuth returns authentication credentials for a registry
func (m *Manager) GetAuth(registry string) (username, password string, ok bool) {
	entry, exists := m.config.Auth[registry]
	if !exists {
		return "", "", false
	}
	return entry.Username, entry.Password, true
}

// SetAuth saves authentication credentials for a registry
func (m *Manager) SetAuth(registry, username, password string) {
	if m.config.Auth == nil {
		m.config.Auth = make(map[string]AuthEntry)
	}
	m.config.Auth[registry] = AuthEntry{
		Username: username,
		Password: password,
	}
}

// RemoveAuth removes authentication credentials for a registry
func (m *Manager) RemoveAuth(registry string) {
	if m.config.Auth != nil {
		delete(m.config.Auth, registry)
	}
}

// GetConfigDir returns the default configuration directory
func GetConfigDir() (string, error) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".timage")
	return configDir, nil
}

// GetStorageDir returns the default storage directory (root of timage data)
func GetStorageDir() (string, error) {
	// Return the config directory (~/.timage), which contains images subdir
	return GetConfigDir()
}
