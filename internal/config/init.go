package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// SampleConfig is the default config.yaml written by Init.
const SampleConfig = `# jira-tui configuration
# Edit the values below for your Jira Cloud instance.

jira:
  base_url: https://yourcompany.atlassian.net
  default_project: PROJ  # used by 'c' (create issue) hotkey

tabs:
  - label: "My Sprint"
    filter_id: "10042"
    columns: [key, summary, status, assignee, priority]
    sort: priority

  - label: "Backlog"
    filter_id: "10043"
    columns: [key, summary, status, priority]

  - label: "Bugs"
    filter_id: "10100"
    columns: [key, summary, status, assignee, reporter]
    sort: created
`

// SampleSecrets is the default secrets.yaml written by Init.
const SampleSecrets = `# jira-tui secrets — DO NOT COMMIT
# Generate an API token at:
#   https://id.atlassian.com/manage-profile/security/api-tokens

jira:
  email: you@yourcompany.com
  api_token: your-api-token-here
`

// Init creates the .jira-tui directory with sample config and secrets files.
// It returns the directory path created. If the directory already exists, it
// returns an error.
func Init() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating config dir: %w", err)
	}

	configPath := filepath.Join(dir, "config.yaml")
	if err := writeIfNotExists(configPath, SampleConfig); err != nil {
		return dir, err
	}

	secretsPath := filepath.Join(dir, "secrets.yaml")
	if err := writeIfNotExists(secretsPath, SampleSecrets); err != nil {
		return dir, err
	}

	return dir, nil
}

// DirExists returns true if the .jira-tui config directory exists.
func DirExists() bool {
	dir, err := DefaultConfigDir()
	if err != nil {
		return false
	}
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

func writeIfNotExists(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // already exists — don't overwrite
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", filepath.Base(path), err)
	}
	return nil
}
