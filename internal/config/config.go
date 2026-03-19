package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	vectorpadHomeEnv     = "VECTORPAD_HOME"
	defaultVectorpadHome = ".vectorpad"
	configFileName       = "config.json"
)

// Config holds persistent VectorPad settings.
type Config struct {
	VectorCourt VectorCourtConfig `json:"vectorcourt"`
}

// VectorCourtConfig holds VectorCourt API integration settings.
type VectorCourtConfig struct {
	APIKey   string `json:"api_key"`
	Endpoint string `json:"endpoint"`
}

// legacyOraculConfig is used only for reading old config files.
type legacyOraculConfig struct {
	APIKey   string `json:"api_key"`
	Endpoint string `json:"endpoint"`
}

// DefaultEndpoint returns the default VectorCourt API endpoint.
func DefaultEndpoint() string {
	return "https://vectorcourt.com"
}

// Load reads config from ~/.vectorpad/config.json.
// Returns a zero Config if the file does not exist.
// Migrates legacy "oracul" config keys to "vectorcourt" on read.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Migrate legacy "oracul" key if "vectorcourt" is empty.
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) == nil {
		if oraculRaw, ok := raw["oracul"]; ok {
			var legacy legacyOraculConfig
			if json.Unmarshal(oraculRaw, &legacy) == nil {
				if cfg.VectorCourt.APIKey == "" && legacy.APIKey != "" {
					cfg.VectorCourt.APIKey = legacy.APIKey
				}
				if cfg.VectorCourt.Endpoint == "" && legacy.Endpoint != "" {
					cfg.VectorCourt.Endpoint = legacy.Endpoint
				}
			}
		}
	}

	return &cfg, nil
}

// Save writes config to ~/.vectorpad/config.json with 0600 permissions.
func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// Set updates a config value by dot-path key (e.g., "vectorcourt.api_key").
func Set(key, value string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	switch key {
	case "vectorcourt.api_key":
		cfg.VectorCourt.APIKey = value
	case "vectorcourt.endpoint":
		cfg.VectorCourt.Endpoint = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return Save(cfg)
}

// Get retrieves a config value by dot-path key.
func Get(key string) (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}

	switch key {
	case "vectorcourt.api_key":
		return cfg.VectorCourt.APIKey, nil
	case "vectorcourt.endpoint":
		ep := cfg.VectorCourt.Endpoint
		if ep == "" {
			ep = DefaultEndpoint()
		}
		return ep, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// Endpoint returns the configured VectorCourt endpoint or the default.
func (c *Config) Endpoint() string {
	if c.VectorCourt.Endpoint != "" {
		return c.VectorCourt.Endpoint
	}
	return DefaultEndpoint()
}

func configPath() (string, error) {
	home := strings.TrimSpace(os.Getenv(vectorpadHomeEnv))
	if home != "" {
		return filepath.Join(filepath.Clean(home), configFileName), nil
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return filepath.Join(userHome, defaultVectorpadHome, configFileName), nil
}
