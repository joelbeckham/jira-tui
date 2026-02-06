package config

import (
	"os"
	"path/filepath"
	"testing"
)

const validSecrets = `
jira:
  email: user@example.com
  api_token: secret-token
`

const validConfigWithTabs = `
jira:
  base_url: https://example.atlassian.net
tabs:
  - label: "My Work"
    filter_id: "10100"
    columns: ["key", "summary", "status"]
    sort: "updated DESC"
  - label: "Bugs"
    filter_url: "https://example.atlassian.net/issues/?filter=10102"
    columns: ["key", "summary", "priority"]
`

func TestLoadValidConfig(t *testing.T) {
	cfgPath := writeTestFile(t, "config.yaml", validConfigWithTabs)
	secPath := writeTestFile(t, "secrets.yaml", validSecrets)

	cfg, err := Load(cfgPath, secPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Jira.BaseURL != "https://example.atlassian.net" {
		t.Errorf("unexpected base URL: %s", cfg.Jira.BaseURL)
	}
	if cfg.Jira.Email != "user@example.com" {
		t.Errorf("unexpected email: %s", cfg.Jira.Email)
	}
	if cfg.Jira.APIToken != "secret-token" {
		t.Errorf("unexpected API token: %s", cfg.Jira.APIToken)
	}
	if len(cfg.Tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(cfg.Tabs))
	}
	if cfg.Tabs[0].Label != "My Work" {
		t.Errorf("unexpected tab label: %s", cfg.Tabs[0].Label)
	}
	if cfg.Tabs[0].FilterID != "10100" {
		t.Errorf("unexpected filter ID: %s", cfg.Tabs[0].FilterID)
	}
	if cfg.Tabs[1].FilterURL != "https://example.atlassian.net/issues/?filter=10102" {
		t.Errorf("unexpected filter URL: %s", cfg.Tabs[1].FilterURL)
	}
}

func TestLoadMissingBaseURL(t *testing.T) {
	cfgPath := writeTestFile(t, "config.yaml", `
jira:
  base_url: ""
tabs:
  - label: "Work"
    filter_id: "10100"
    columns: ["key", "summary"]
`)
	secPath := writeTestFile(t, "secrets.yaml", validSecrets)
	_, err := Load(cfgPath, secPath)
	if err == nil {
		t.Fatal("expected validation error for missing base_url")
	}
}

func TestLoadMissingEmail(t *testing.T) {
	cfgPath := writeTestFile(t, "config.yaml", `
jira:
  base_url: https://example.atlassian.net
tabs:
  - label: "Work"
    filter_id: "10100"
    columns: ["key", "summary"]
`)
	secPath := writeTestFile(t, "secrets.yaml", `
jira:
  api_token: secret-token
`)
	_, err := Load(cfgPath, secPath)
	if err == nil {
		t.Fatal("expected validation error for missing email")
	}
}

func TestLoadMissingAPIToken(t *testing.T) {
	cfgPath := writeTestFile(t, "config.yaml", `
jira:
  base_url: https://example.atlassian.net
tabs:
  - label: "Work"
    filter_id: "10100"
    columns: ["key", "summary"]
`)
	secPath := writeTestFile(t, "secrets.yaml", `
jira:
  email: user@example.com
`)
	_, err := Load(cfgPath, secPath)
	if err == nil {
		t.Fatal("expected validation error for missing api_token")
	}
}

func TestLoadNoTabs(t *testing.T) {
	cfgPath := writeTestFile(t, "config.yaml", `
jira:
  base_url: https://example.atlassian.net
`)
	secPath := writeTestFile(t, "secrets.yaml", validSecrets)
	_, err := Load(cfgPath, secPath)
	if err == nil {
		t.Fatal("expected validation error for missing tabs")
	}
}

func TestLoadTabMissingLabel(t *testing.T) {
	cfgPath := writeTestFile(t, "config.yaml", `
jira:
  base_url: https://example.atlassian.net
tabs:
  - filter_id: "10100"
    columns: ["key", "summary"]
`)
	secPath := writeTestFile(t, "secrets.yaml", validSecrets)
	_, err := Load(cfgPath, secPath)
	if err == nil {
		t.Fatal("expected validation error for missing tab label")
	}
}

func TestLoadTabMissingFilter(t *testing.T) {
	cfgPath := writeTestFile(t, "config.yaml", `
jira:
  base_url: https://example.atlassian.net
tabs:
  - label: "Work"
    columns: ["key", "summary"]
`)
	secPath := writeTestFile(t, "secrets.yaml", validSecrets)
	_, err := Load(cfgPath, secPath)
	if err == nil {
		t.Fatal("expected validation error for missing filter_id/filter_url")
	}
}

func TestLoadTabMissingColumns(t *testing.T) {
	cfgPath := writeTestFile(t, "config.yaml", `
jira:
  base_url: https://example.atlassian.net
tabs:
  - label: "Work"
    filter_id: "10100"
`)
	secPath := writeTestFile(t, "secrets.yaml", validSecrets)
	_, err := Load(cfgPath, secPath)
	if err == nil {
		t.Fatal("expected validation error for missing columns")
	}
}

func TestLoadMissingConfigFile(t *testing.T) {
	secPath := writeTestFile(t, "secrets.yaml", validSecrets)
	_, err := Load("/nonexistent/path/config.yaml", secPath)
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestLoadMissingSecretsFile(t *testing.T) {
	cfgPath := writeTestFile(t, "config.yaml", validConfigWithTabs)
	_, err := Load(cfgPath, "/nonexistent/path/secrets.yaml")
	if err == nil {
		t.Fatal("expected error for missing secrets file")
	}
}

func TestValidate(t *testing.T) {
	validTabs := []TabConfig{{
		Label:    "Work",
		FilterID: "10100",
		Columns:  []string{"key", "summary"},
	}}

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid",
			config: Config{
				Jira: JiraConfig{
					BaseURL:  "https://example.atlassian.net",
					Email:    "user@example.com",
					APIToken: "token",
				},
				Tabs: validTabs,
			},
			wantErr: false,
		},
		{
			name:    "empty",
			config:  Config{},
			wantErr: true,
		},
		{
			name: "valid with filter_url",
			config: Config{
				Jira: JiraConfig{
					BaseURL:  "https://example.atlassian.net",
					Email:    "user@example.com",
					APIToken: "token",
				},
				Tabs: []TabConfig{{
					Label:     "Work",
					FilterURL: "https://example.atlassian.net/issues/?filter=10100",
					Columns:   []string{"key", "summary"},
				}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func writeTestFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing test file %s: %v", name, err)
	}
	return path
}
