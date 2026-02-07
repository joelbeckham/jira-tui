package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jbeckham/jira-tui/internal/config"
	"github.com/jbeckham/jira-tui/internal/jira"
	"github.com/jbeckham/jira-tui/internal/tui"
)

func main() {
	// Handle "init" subcommand
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runInit()
		return
	}

	// Auto-init if .jira-tui directory doesn't exist
	if !config.DirExists() {
		dir, err := config.Init()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created %s/\n\n", dir)
		fmt.Println("To get started:")
		fmt.Printf("  1. Edit %s with your Jira URL and filters\n", filepath.Join(dir, "config.yaml"))
		fmt.Printf("  2. Edit %s with your email and API token\n", filepath.Join(dir, "secrets.yaml"))
		fmt.Printf("     (generate a token at https://id.atlassian.com/manage-profile/security/api-tokens)\n")
		fmt.Println("  3. Run jira-tui again")
		os.Exit(0)
	}

	configDir, err := config.DefaultConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load(
		filepath.Join(configDir, "config.yaml"),
		filepath.Join(configDir, "secrets.yaml"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	client := jira.NewClient(cfg.Jira.BaseURL, cfg.Jira.Email, cfg.Jira.APIToken)

	p := tea.NewProgram(tui.NewApp(client, cfg.Tabs, cfg.Jira.DefaultProject), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runInit() {
	if config.DirExists() {
		dir, _ := config.DefaultConfigDir()
		fmt.Printf("%s/ already exists\n", dir)
		os.Exit(0)
	}
	dir, err := config.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created %s/\n", dir)
	fmt.Printf("  config.yaml  — Jira URL, tabs, columns\n")
	fmt.Printf("  secrets.yaml — email and API token\n")
}
