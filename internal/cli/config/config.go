package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	appDirectory = "opentunnel"
	configFile   = "config.json"
)

// Config contains non-secret CLI preferences.
type Config struct {
	APIURL       string `json:"api_url,omitempty"`
	LastDomainID string `json:"last_domain_id,omitempty"`
	LastPort     int    `json:"last_port,omitempty"`
}

// Store persists CLI preferences in the user's configuration directory.
type Store struct {
	path string
}

func NewStore() (*Store, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("find user config directory: %w", err)
	}
	return NewStoreAt(filepath.Join(root, appDirectory, configFile)), nil
}

func NewStoreAt(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() (Config, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read CLI config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode CLI config: %w", err)
	}
	return cfg, nil
}

func (s *Store) Save(cfg Config) error {
	if cfg.LastPort < 0 || cfg.LastPort > 65535 {
		return fmt.Errorf("invalid saved port %d", cfg.LastPort)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode CLI config: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create CLI config directory: %w", err)
	}
	if err := writeProtected(s.path, data); err != nil {
		return fmt.Errorf("write CLI config: %w", err)
	}
	return nil
}

func writeProtected(path string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".config-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()

	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}
