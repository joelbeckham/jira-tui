package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/table"
)

func TestBuildColumnsBasic(t *testing.T) {
	cols := buildColumns([]string{"key", "summary", "status"}, 100)

	if len(cols) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(cols))
	}

	// Verify titles
	expected := []string{"Key", "Summary", "Status"}
	for i, col := range cols {
		if col.Title != expected[i] {
			t.Errorf("column %d: expected title %q, got %q", i, expected[i], col.Title)
		}
	}

	// Total width should be <= totalWidth (padding per column may reduce it)
	totalW := 0
	for _, col := range cols {
		totalW += col.Width
	}
	if totalW > 100 {
		t.Errorf("expected total width <= 100, got %d", totalW)
	}
	if totalW < 50 {
		t.Errorf("expected total width >= 50, got %d (too narrow)", totalW)
	}
}

func TestBuildColumnsFlexDistribution(t *testing.T) {
	// summary is flex, key and status are fixed
	cols := buildColumns([]string{"key", "summary", "status"}, 100)

	keyCol := findCol(cols, "Key")
	summaryCol := findCol(cols, "Summary")

	if keyCol == nil || summaryCol == nil {
		t.Fatal("expected to find Key and Summary columns")
	}

	// Summary (flex) should get the remaining width after fixed columns
	if summaryCol.Width <= keyCol.Width {
		t.Errorf("expected summary width (%d) > key width (%d)", summaryCol.Width, keyCol.Width)
	}
}

func TestBuildColumnsUnknownColumn(t *testing.T) {
	cols := buildColumns([]string{"key", "custom_field"}, 80)

	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cols))
	}

	// Unknown column should use its name as title
	custom := cols[1]
	if custom.Title != "custom_field" {
		t.Errorf("expected title 'custom_field', got %q", custom.Title)
	}
}

func TestBuildColumnsEmpty(t *testing.T) {
	cols := buildColumns(nil, 80)
	if len(cols) != 0 {
		t.Errorf("expected 0 columns, got %d", len(cols))
	}
}

func TestBuildColumnsNarrowWidth(t *testing.T) {
	// When totalWidth is very narrow, columns should get at least minWidth
	cols := buildColumns([]string{"key", "summary", "status", "priority"}, 20)

	if len(cols) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(cols))
	}

	// Even with narrow width, each column should have a positive width
	for _, col := range cols {
		if col.Width < 1 {
			t.Errorf("column %q has non-positive width: %d", col.Title, col.Width)
		}
	}
}

// findCol finds a column by title in a slice.
func findCol(cols []table.Column, title string) *table.Column {
	for i := range cols {
		if cols[i].Title == title {
			return &cols[i]
		}
	}
	return nil
}
