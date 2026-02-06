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
