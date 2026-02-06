package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jbeckham/jira-tui/internal/jira"
)

func TestAppInit(t *testing.T) {
	app := NewApp(nil)
	cmd := app.Init()
	if cmd != nil {
		t.Error("Init() should return nil cmd when no client")
	}
}

func TestAppInitWithClient(t *testing.T) {
	client := jira.NewClient("https://example.atlassian.net", "test@example.com", "token")
	app := NewApp(client)
	cmd := app.Init()
	if cmd == nil {
		t.Error("Init() should return a cmd when client is set")
	}
	if !app.checking {
		t.Error("expected checking=true when client is set")
	}
}

func TestAppQuitOnQ(t *testing.T) {
	app := NewApp(nil)
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestAppQuitOnCtrlC(t *testing.T) {
	app := NewApp(nil)
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestAppHandlesWindowSize(t *testing.T) {
	app := NewApp(nil)
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	updated := model.(App)
	if updated.width != 80 || updated.height != 24 {
		t.Errorf("expected 80x24, got %dx%d", updated.width, updated.height)
	}
	if !updated.ready {
		t.Error("expected ready=true after WindowSizeMsg")
	}
}

func TestAppViewBeforeReady(t *testing.T) {
	app := NewApp(nil)
	view := app.View()
	if !strings.Contains(view, "Loading") {
		t.Errorf("expected loading message, got: %s", view)
	}
}

func TestAppViewAfterReady(t *testing.T) {
	app := NewApp(nil)
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	updated := model.(App)
	view := updated.View()
	if !strings.Contains(view, "jira-tui") {
		t.Errorf("expected title in view, got: %s", view)
	}
	if !strings.Contains(view, "quit") {
		t.Errorf("expected help text in view, got: %s", view)
	}
}

func TestAppConnStatusSuccess(t *testing.T) {
	app := NewApp(nil)
	app.ready = true
	app.checking = true

	model, _ := app.Update(connStatusMsg{
		user: &jira.User{DisplayName: "Test User"},
	})
	updated := model.(App)
	if updated.checking {
		t.Error("expected checking=false after connStatusMsg")
	}
	if updated.user == nil {
		t.Fatal("expected user to be set")
	}
	if updated.user.DisplayName != "Test User" {
		t.Errorf("expected 'Test User', got %s", updated.user.DisplayName)
	}
	view := updated.View()
	if !strings.Contains(view, "Connected as Test User") {
		t.Errorf("expected connected message, got: %s", view)
	}
}

func TestAppConnStatusError(t *testing.T) {
	app := NewApp(nil)
	app.ready = true
	app.checking = true

	model, _ := app.Update(connStatusMsg{
		err: fmt.Errorf("401 Unauthorized"),
	})
	updated := model.(App)
	if updated.checking {
		t.Error("expected checking=false after connStatusMsg")
	}
	if updated.connErr == nil {
		t.Fatal("expected connErr to be set")
	}
	view := updated.View()
	if !strings.Contains(view, "Connection failed") {
		t.Errorf("expected error message, got: %s", view)
	}
}

func TestAppViewConnecting(t *testing.T) {
	app := NewApp(nil)
	app.ready = true
	app.checking = true
	view := app.View()
	if !strings.Contains(view, "Connecting to Jira") {
		t.Errorf("expected connecting message, got: %s", view)
	}
}
