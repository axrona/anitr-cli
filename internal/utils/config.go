package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config struct
type Config struct {
	DefaultSource string `json:"default_source"`
	HistoryLimit  int    `json:"history_limit"`
	DisableRPC    *bool  `json:"disable_rpc"`
	DownloadDir string `json:"download_dir"`
}

// LoadConfig config'i yükler
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}
	
	if cfg.DownloadDir == "" {
    	cfg.DownloadDir = filepath.Join(os.Getenv("HOME"), "Downloads", "anitr-cli")
	}

	return &cfg, nil
}
