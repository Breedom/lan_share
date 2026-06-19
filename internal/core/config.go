package core

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Security SecurityConfig `json:"security"`
	Shares   []ShareConfig  `json:"shares"`
	Clipboard ClipboardConfig `json:"clipboard"`
	General  GeneralConfig  `json:"general"`
}

type ServerConfig struct {
	HTTPPort int `json:"http_port"`
	TCPPort  int `json:"tcp_port"`
	UDPPort  int `json:"udp_port"`
}

type SecurityConfig struct {
	EncryptionEnabled bool   `json:"encryption_enabled"`
	AutoPair          bool   `json:"auto_pair"`
	Password          string `json:"password,omitempty"`
}

type ShareConfig struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Readonly bool   `json:"readonly"`
}

type ClipboardConfig struct {
	SyncEnabled   bool `json:"sync_enabled"`
	SyncInterval  int  `json:"sync_interval"`
}

type GeneralConfig struct {
	AutoStart      bool   `json:"auto_start"`
	MinimizeToTray bool   `json:"minimize_to_tray"`
	DeviceName     string `json:"device_name"`
}

func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	downloadDir := filepath.Join(homeDir, "Downloads")

	return &Config{
		Server: ServerConfig{
			HTTPPort: 8080,
			TCPPort:  53317,
			UDPPort:  53317,
		},
		Security: SecurityConfig{
			EncryptionEnabled: true,
			AutoPair:          false,
		},
		Shares: []ShareConfig{
			{
				Name:     "Downloads",
				Path:     downloadDir,
				Readonly: false,
			},
		},
		Clipboard: ClipboardConfig{
			SyncEnabled:  true,
			SyncInterval: 1000,
		},
		General: GeneralConfig{
			AutoStart:      true,
			MinimizeToTray: true,
			DeviceName:     "My PC",
		},
	}
}

func LoadConfig() *Config {
	configPath := getConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		SaveConfig(config)
		return config
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultConfig()
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return DefaultConfig()
	}

	return &config
}

func SaveConfig(config *Config) error {
	configPath := getConfigPath()
	dir := filepath.Dir(configPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func getConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return filepath.Join(configDir, "lanshare", "config.json")
}
