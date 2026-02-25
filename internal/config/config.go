// Package config handles configuration file parsing and path defaults.
// Config stored at ~/.config/pocketmux/config.toml.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
)

const (
	appDir          = "pocketmux"
	keysDir         = "keys"
	pairedDevicesFile = "paired_devices.json"
	configFile      = "config.toml"

	// DefaultServerURL is the production signaling server.
	DefaultServerURL = "https://signal.pocketmux.dev"

	// EnvServerURL is the environment variable to override the signaling server URL.
	EnvServerURL = "PMUX_AGENT_SIGNAL_URL"
)

// ServerURL returns the signaling server URL. It checks PMUX_AGENT_SIGNAL_URL
// first, falling back to DefaultServerURL.
func ServerURL() string {
	if v := os.Getenv(EnvServerURL); v != "" {
		return v
	}
	return DefaultServerURL
}

// Paths holds resolved filesystem paths for PocketMux configuration and keys.
type Paths struct {
	ConfigDir      string // ~/.config/pocketmux
	KeysDir        string // ~/.config/pocketmux/keys
	PairedDevices  string // ~/.config/pocketmux/paired_devices.json
	ConfigFile     string // ~/.config/pocketmux/config.toml
}

// DefaultPaths returns the standard PocketMux directory paths based on $HOME.
func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("determine home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", appDir)
	return Paths{
		ConfigDir:     configDir,
		KeysDir:       filepath.Join(configDir, keysDir),
		PairedDevices: filepath.Join(configDir, pairedDevicesFile),
		ConfigFile:    filepath.Join(configDir, configFile),
	}, nil
}

// EnsureDirs creates the config and keys directories if they don't exist.
func (p Paths) EnsureDirs() error {
	if err := os.MkdirAll(p.KeysDir, 0700); err != nil {
		return fmt.Errorf("create keys directory: %w", err)
	}
	return nil
}

// Config holds user-editable PocketMux configuration from config.toml.
type Config struct {
	Name string `toml:"name"`
}

// LoadConfig reads the TOML config file. Returns a zero Config (not an error)
// if the file doesn't exist yet.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// SaveConfig writes the config to a TOML file with 0600 permissions.
func SaveConfig(path string, cfg Config) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// DefaultHostName returns the OS hostname as a default host name.
func DefaultHostName() string {
	name, err := os.Hostname()
	if err != nil {
		return "my-host"
	}
	return name
}
