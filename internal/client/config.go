package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/niekvdm/digit-link/internal/tunnel"
)

// SavedConfig represents the saved client configuration
type SavedConfig struct {
	Server   string                 `json:"server"`
	Token    string                 `json:"token"`
	Forwards []tunnel.ForwardConfig `json:"forwards"`
	Insecure bool                   `json:"insecure,omitempty"`
}

// getConfigDir returns the configuration directory path
func getConfigDir() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		// Use %APPDATA%\digit-link
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		configDir = filepath.Join(appData, "digit-link")
	case "darwin":
		// Use ~/Library/Application Support/digit-link
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, "Library", "Application Support", "digit-link")
	default:
		// Use $XDG_CONFIG_HOME/digit-link or ~/.config/digit-link
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			xdgConfig = filepath.Join(home, ".config")
		}
		configDir = filepath.Join(xdgConfig, "digit-link")
	}

	return configDir, nil
}

// getConfigPath returns the full path to the config file
func getConfigPath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

// SaveConfig saves the client configuration to disk
func SaveConfig(cfg SavedConfig) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Write file with restricted permissions (user read/write only)
	return os.WriteFile(configPath, data, 0600)
}

// LoadConfig loads the client configuration from disk
func LoadConfig() (*SavedConfig, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil // No config file, return nil without error
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Unmarshal config
	var cfg SavedConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// DeleteConfig removes the saved configuration file
func DeleteConfig() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Remove file if it exists
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// ConfigExists checks if a config file exists
func ConfigExists() bool {
	configPath, err := getConfigPath()
	if err != nil {
		return false
	}

	_, err = os.Stat(configPath)
	return err == nil
}
