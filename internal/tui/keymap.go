package tui

import "github.com/charmbracelet/bubbletea"

// Keymap defines the global keybindings for the application.
type Keymap struct {
	Quit    tea.Key
	Help    tea.Key
	Back    tea.Key
	Confirm tea.Key
	Up      tea.Key
	Down    tea.Key
	Left    tea.Key
	Right   tea.Key
}

// DefaultKeymap returns the default keybindings.
func DefaultKeymap() Keymap {
	return Keymap{
		Quit:    tea.Key{Type: tea.KeyRunes, Runes: []rune("q")},
		Help:    tea.Key{Type: tea.KeyRunes, Runes: []rune("?")},
		Back:    tea.Key{Type: tea.KeyEsc},
		Confirm: tea.Key{Type: tea.KeyEnter},
		Up:      tea.Key{Type: tea.KeyRunes, Runes: []rune("k")},
		Down:    tea.Key{Type: tea.KeyRunes, Runes: []rune("j")},
		Left:    tea.Key{Type: tea.KeyRunes, Runes: []rune("h")},
		Right:   tea.Key{Type: tea.KeyRunes, Runes: []rune("l")},
	}
}
