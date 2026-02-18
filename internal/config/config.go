// Package config manages persistent user configuration.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds all user-configurable settings.
type Config struct {
	WorkspaceGID      string            `json:"workspace_gid"`
	ProjectGID        string            `json:"project_gid"`
	Provider          string            `json:"provider"`
	Model             string            `json:"model"`
	OutputDir         string            `json:"output_dir"`
	APIKeys           map[string]string `json:"api_keys"`
	AsanaCLIPath      string            `json:"asana_cli_path"`
	AutoCompleteTasks bool              `json:"auto_complete_tasks"`
	Theme             string            `json:"theme"`
}

var configDir = filepath.Join(mustHomeDir(), ".task-agent")
var configFile = filepath.Join(configDir, "config.json")

func mustHomeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

// Defaults returns a Config with sensible defaults.
func Defaults() *Config {
	return &Config{
		Provider:  "anthropic",
		Model:     "claude-sonnet-4-6",
		OutputDir: "./task-outputs",
		Theme:     "dark",
		APIKeys:   map[string]string{},
	}
}

// Load reads config from disk, merging with defaults.
func Load() (*Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(configFile)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, err
	}
	if cfg.APIKeys == nil {
		cfg.APIKeys = map[string]string{}
	}
	return cfg, nil
}

// Save writes config to disk with restricted permissions.
func Save(cfg *Config) error {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return err
	}
	return nil
}

// GetAPIKey returns the API key for a provider, checking env vars first.
func GetAPIKey(cfg *Config, providerID string) string {
	envVars := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY",
		"openai":    "OPENAI_API_KEY",
		"groq":      "GROQ_API_KEY",
	}
	if envKey, ok := envVars[providerID]; ok {
		if val := os.Getenv(envKey); val != "" {
			return val
		}
	}
	return cfg.APIKeys[providerID]
}

// SetAPIKey stores an API key in the config.
func SetAPIKey(cfg *Config, providerID, key string) {
	if cfg.APIKeys == nil {
		cfg.APIKeys = map[string]string{}
	}
	cfg.APIKeys[providerID] = key
}

// ConfigPath returns the path to the config file (for display).
func ConfigPath() string {
	return configFile
}