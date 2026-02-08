package tui

import (
	"strings"
	"testing"
)

func TestPriorityIconKnown(t *testing.T) {
	tests := []struct {
		name     string
		wantIcon string
	}{
		{"Blocked", "⊘"},
		{"Blocker", "⊘"},
		{"Critical", "↑↑"},
		{"Highest", "↑↑"},
		{"High", "↑"},
		{"Medium", "≡"},
		{"Medium-Rare", "↓"},
		{"Low", "↓↓"},
		{"Lowest", "↓↓"},
	}

	for _, tt := range tests {
		got := priorityIcon(tt.name)
		// priorityIcon returns plain text (no ANSI codes) for safe use in table cells
		if got != tt.wantIcon {
			t.Errorf("priorityIcon(%q) = %q, want %q", tt.name, got, tt.wantIcon)
		}
	}
}

func TestPriorityIconUnknown(t *testing.T) {
	got := priorityIcon("SuperCustom")
	if got != "SuperCustom" {
		t.Errorf("priorityIcon(unknown) = %q, want %q", got, "SuperCustom")
	}
}

func TestPriorityIconNotPrioritizedBlank(t *testing.T) {
	got := priorityIcon("Not Prioritized")
	if got != "" {
		t.Errorf("priorityIcon(Not Prioritized) = %q, want blank", got)
	}
}

func TestPriorityLabelKnown(t *testing.T) {
	got := priorityLabel("High")
	// Should contain both the icon and the text name
	if !strings.Contains(got, "↑") {
		t.Errorf("priorityLabel(High) missing icon, got: %q", got)
	}
	if !strings.Contains(got, "High") {
		t.Errorf("priorityLabel(High) missing name, got: %q", got)
	}
}

func TestPriorityLabelUnknown(t *testing.T) {
	got := priorityLabel("Whatever")
	if got != "Whatever" {
		t.Errorf("priorityLabel(unknown) = %q, want %q", got, "Whatever")
	}
}

func TestPriorityMapCoversAllEntries(t *testing.T) {
	// Ensure every entry in priorityMap has both an icon and a color
	for name, def := range priorityMap {
		if def.icon == "" {
			t.Errorf("priorityMap[%q] has empty icon", name)
		}
		if def.color == "" {
			t.Errorf("priorityMap[%q] has empty color", name)
		}
	}
}

func TestHexToRGB(t *testing.T) {
	tests := []struct {
		hex     string
		r, g, b uint8
	}{
		{"#FF5630", 255, 86, 48},
		{"#2684FF", 38, 132, 255},
		{"#FFAB00", 255, 171, 0},
		{"#000000", 0, 0, 0},
		{"#FFFFFF", 255, 255, 255},
	}
	for _, tt := range tests {
		r, g, b := hexToRGB(tt.hex)
		if r != tt.r || g != tt.g || b != tt.b {
			t.Errorf("hexToRGB(%q) = (%d,%d,%d), want (%d,%d,%d)", tt.hex, r, g, b, tt.r, tt.g, tt.b)
		}
	}
}

func TestColorizePrioritiesContainsANSI(t *testing.T) {
	// Each known icon should get ANSI color codes after colorization
	tests := []struct {
		icon string
	}{
		{"⊘"},
		{"↑↑"},
		{"↑"},
		{"≡"},
		{"↓↓"},
		{"↓"},
	}
	for _, tt := range tests {
		input := "  " + tt.icon + "  "
		got := colorizePriorities(input)
		if !strings.Contains(got, "\x1b[38;2;") {
			t.Errorf("colorizePriorities(%q) missing ANSI color, got %q", input, got)
		}
		if !strings.Contains(got, tt.icon) {
			t.Errorf("colorizePriorities(%q) missing icon text, got %q", input, got)
		}
		if !strings.Contains(got, "\x1b[39m") {
			t.Errorf("colorizePriorities(%q) missing fg reset, got %q", input, got)
		}
	}
}

func TestColorizePrioritiesNoIconUnchanged(t *testing.T) {
	input := "PROJ-123  Fix the login bug      Done"
	got := colorizePriorities(input)
	if got != input {
		t.Errorf("colorizePriorities should not modify text without icons\ninput: %q\ngot:   %q", input, got)
	}
}

func TestColorizePrioritiesDoubleArrowBeforeSingle(t *testing.T) {
	// "↑↑" should be colored as one unit (red), not as two "↑" (orange)
	input := "  ↑↑  "
	got := colorizePriorities(input)
	// Should contain the double-arrow colored red (#FF5630 = 255,86,48)
	if !strings.Contains(got, "\x1b[38;2;255;86;48m↑↑\x1b[39m") {
		t.Errorf("colorizePriorities(%q) should color ↑↑ as red unit, got %q", input, got)
	}
	// Should NOT contain the single-arrow orange color (#FF7452 = 255,116,82)
	if strings.Contains(got, "255;116;82") {
		t.Errorf("colorizePriorities(%q) should not use single-↑ color for ↑↑, got %q", input, got)
	}
}
