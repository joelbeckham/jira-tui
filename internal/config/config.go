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
	Jira  JiraConfig  `yaml:"jira"`
	Tabs  []TabConfig `yaml:"tabs"`
	Cache CacheConfig `yaml:"cache"`
}

// JiraConfig holds Jira-specific configuration.
// Connection details live in config.yaml; credentials live in a separate
// secrets.yaml file that is gitignored.
type JiraConfig struct {
	BaseURL        string `yaml:"base_url"`
	Email          string `yaml:"email"`
	APIToken       string `yaml:"api_token"` // loaded from secrets file, not config
	DefaultProject string `yaml:"default_project,omitempty"`
}

// SecretsConfig holds sensitive credentials loaded from a separate file.
type SecretsConfig struct {
	Jira JiraSecrets `yaml:"jira"`
}

// JiraSecrets holds the Jira credentials.
type JiraSecrets struct {
	Email    string `yaml:"email"`
	APIToken string `yaml:"api_token"`
}

// TabConfig defines a filter-backed tab in the TUI.
// Exactly one of FilterID, FilterURL, or JQL must be provided.
type TabConfig struct {
	Label     string   `yaml:"label"`
	FilterID  string   `yaml:"filter_id,omitempty"`
	FilterURL string   `yaml:"filter_url,omitempty"`
	JQL       string   `yaml:"jql,omitempty"`
	Columns   []string `yaml:"columns"`
	Sort      string   `yaml:"sort,omitempty"`
}

// CacheConfig holds caching configuration.
type CacheConfig struct {
	TTL string `yaml:"ttl"` // duration string, e.g. "5m"
}

// DefaultConfigDir returns the .jira-tui directory next to the executable.
func DefaultConfigDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("finding executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolving executable symlinks: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), ".jira-tui"), nil
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// DefaultSecretsPath returns the default secrets file path.
func DefaultSecretsPath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "secrets.yaml"), nil
}

// Load reads and parses the config and secrets files.
// configPath is the path to config.yaml, secretsPath is the path to secrets.yaml.
func Load(configPath, secretsPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Load secrets from separate file
	secretsData, err := os.ReadFile(secretsPath)
	if err != nil {
		return nil, fmt.Errorf("reading secrets file: %w", err)
	}

	var secrets SecretsConfig
	if err := yaml.Unmarshal(secretsData, &secrets); err != nil {
		return nil, fmt.Errorf("parsing secrets file: %w", err)
	}

	// Merge secrets into config
	cfg.Jira.Email = secrets.Jira.Email
	cfg.Jira.APIToken = secrets.Jira.APIToken

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
		// Count how many data sources are specified
		sources := 0
		if tab.FilterID != "" {
			sources++
		}
		if tab.FilterURL != "" {
			sources++
		}
		if tab.JQL != "" {
			sources++
		}
		if sources == 0 {
			return fmt.Errorf("tabs[%d] must have filter_id, filter_url, or jql", i)
		}
		if sources > 1 {
			return fmt.Errorf("tabs[%d] must have only one of filter_id, filter_url, or jql", i)
		}
		if len(tab.Columns) == 0 {
			return fmt.Errorf("tabs[%d].columns must not be empty", i)
		}
	}
	return nil
}
