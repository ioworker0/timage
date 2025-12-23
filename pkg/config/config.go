package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the application configuration
type Config struct {
	DefaultProxy string                   `json:"default_proxy,omitempty"`
	Auth         map[string]AuthEntry     `json:"auth,omitempty"`
	Registries   map[string]RegistryEntry `json:"registries,omitempty"`
}

// RegistryEntry represents registry-specific configuration
type RegistryEntry struct {
	Proxy string `json:"proxy,omitempty"`
}

// Manager manages configuration
type Manager struct {
	configPath string
	config     *Config
}

// NewManager creates a new configuration manager
func NewManager(configDir string) (*Manager, error) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.json")

	mgr := &Manager{
		configPath: configPath,
		config:     &Config{},
	}

	// Load existing configuration
	if err := mgr.Load(); err != nil {
		// If file doesn't exist, create default config
		if os.IsNotExist(err) {
			mgr.config = &Config{
				Auth:       make(map[string]AuthEntry),
				Registries: make(map[string]RegistryEntry),
			}
		} else {
			return nil, err
		}
	}

	return mgr, nil
}

// Load loads configuration from disk
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	m.config = &config

	// Initialize maps if nil
	if m.config.Auth == nil {
		m.config.Auth = make(map[string]AuthEntry)
	}
	if m.config.Registries == nil {
		m.config.Registries = make(map[string]RegistryEntry)
	}

	return nil
}

// Save saves configuration to disk
func (m *Manager) Save() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// GetDefaultProxy returns the default proxy URL
func (m *Manager) GetDefaultProxy() string {
	return m.config.DefaultProxy
}

// SetDefaultProxy sets the default proxy URL
func (m *Manager) SetDefaultProxy(proxy string) {
	m.config.DefaultProxy = proxy
}

// GetRegistryProxy returns the proxy URL for a specific registry
func (m *Manager) GetRegistryProxy(registry string) string {
	if entry, ok := m.config.Registries[registry]; ok {
		return entry.Proxy
	}
	return m.config.DefaultProxy
}

// SetRegistryProxy sets the proxy URL for a specific registry
func (m *Manager) SetRegistryProxy(registry, proxy string) {
	if m.config.Registries == nil {
		m.config.Registries = make(map[string]RegistryEntry)
	}
	m.config.Registries[registry] = RegistryEntry{Proxy: proxy}
}
