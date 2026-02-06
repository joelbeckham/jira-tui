package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAppInit(t *testing.T) {
	app := NewApp()
	cmd := app.Init()
	if cmd != nil {
		t.Error("Init() should return nil cmd")
	}
}

func TestAppQuitOnQ(t *testing.T) {
	app := NewApp()
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
	// Bubbletea's Quit returns a special Msg; verify by calling the cmd
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestAppQuitOnCtrlC(t *testing.T) {
	app := NewApp()
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
	app := NewApp()
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
	app := NewApp()
	view := app.View()
	if !strings.Contains(view, "Loading") {
		t.Errorf("expected loading message, got: %s", view)
	}
}

func TestAppViewAfterReady(t *testing.T) {
	app := NewApp()
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
