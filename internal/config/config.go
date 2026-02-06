// Package config handles loading and validating jira-tui configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	Jira JiraConfig `yaml:"jira"`
	Tabs []TabConfig `yaml:"tabs"`
	Cache CacheConfig `yaml:"cache"`
}

// JiraConfig holds Jira-specific configuration.
type JiraConfig struct {
	BaseURL  string `yaml:"base_url"`
	Email    string `yaml:"email"`
	APIToken string `yaml:"api_token"`
}

// TabConfig defines a filter-backed tab in the TUI.
type TabConfig struct {
	Label     string   `yaml:"label"`
	FilterID  string   `yaml:"filter_id,omitempty"`
	FilterURL string   `yaml:"filter_url,omitempty"`
	Columns   []string `yaml:"columns"`
	Sort      string   `yaml:"sort,omitempty"`
}

// CacheConfig holds caching configuration.
type CacheConfig struct {
	TTL string `yaml:"ttl"` // duration string, e.g. "5m"
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting config dir: %w", err)
	}
	return filepath.Join(configDir, "jira-tui", "config.yaml"), nil
}

// Load reads and parses the config file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Validate checks that all required config fields are set.
func (c *Config) Validate() error {
	if c.Jira.BaseURL == "" {
		return fmt.Errorf("jira.base_url is required")
	}
	if c.Jira.Email == "" {
		return fmt.Errorf("jira.email is required")
	}
	if c.Jira.APIToken == "" {
		return fmt.Errorf("jira.api_token is required")
	}
	if len(c.Tabs) == 0 {
		return fmt.Errorf("at least one tab is required")
	}
	for i, tab := range c.Tabs {
		if tab.Label == "" {
			return fmt.Errorf("tabs[%d].label is required", i)
		}
		if tab.FilterID == "" && tab.FilterURL == "" {
			return fmt.Errorf("tabs[%d] must have filter_id or filter_url", i)
		}
		if len(tab.Columns) == 0 {
			return fmt.Errorf("tabs[%d].columns must not be empty", i)
		}
	}
	return nil
}
