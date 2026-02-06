package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func updateOverlay(o overlay, msg tea.Msg) overlay {
	updated, _ := o.Update(msg)
	return updated
}

func keyMsg(key string) tea.KeyMsg {
	if len(key) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

func TestSelectionOverlayFilterAndSelect(t *testing.T) {
	items := []selectionItem{
		{ID: "1", Label: "High"},
		{ID: "2", Label: "Medium"},
		{ID: "3", Label: "Low"},
	}
	var o overlay = newSelectionOverlay("Pick one", items)

	s := o.(*selectionOverlay)
	if len(s.filtered) != 3 {
		t.Errorf("expected 3 filtered items, got %d", len(s.filtered))
	}

	for _, ch := range "med" {
		o = updateOverlay(o, keyMsg(string(ch)))
	}
	s = o.(*selectionOverlay)
	if len(s.filtered) != 1 {
		t.Errorf("expected 1 match for 'med', got %d", len(s.filtered))
	}

	o = updateOverlay(o, keyMsg("enter"))
	isDone, result := o.done()
	if !isDone {
		t.Error("expected done after enter")
	}
	sel, ok := result.(*selectionItem)
	if !ok || sel == nil {
		t.Fatal("expected selectionItem result")
	}
	if sel.Label != "Medium" {
		t.Errorf("expected 'Medium', got '%s'", sel.Label)
	}
}

func TestSelectionOverlayEscCancels(t *testing.T) {
	var o overlay = newSelectionOverlay("Pick one", []selectionItem{{ID: "1", Label: "A"}})
	o = updateOverlay(o, keyMsg("esc"))

	isDone, result := o.done()
	if !isDone {
		t.Error("expected done after esc")
	}
	if result != nil {
		t.Error("expected nil result on cancel")
	}
}

func TestSelectionOverlayCursorNavigation(t *testing.T) {
	items := []selectionItem{
		{ID: "1", Label: "A"},
		{ID: "2", Label: "B"},
		{ID: "3", Label: "C"},
	}
	var o overlay = newSelectionOverlay("Select", items)

	s := o.(*selectionOverlay)
	if s.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", s.cursor)
	}

	o = updateOverlay(o, keyMsg("down"))
	o = updateOverlay(o, keyMsg("down"))
	s = o.(*selectionOverlay)
	if s.cursor != 2 {
		t.Errorf("expected cursor at 2, got %d", s.cursor)
	}

	o = updateOverlay(o, keyMsg("down"))
	s = o.(*selectionOverlay)
	if s.cursor != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", s.cursor)
	}

	o = updateOverlay(o, keyMsg("up"))
	s = o.(*selectionOverlay)
	if s.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", s.cursor)
	}

	o = updateOverlay(o, keyMsg("enter"))
	_, result := o.done()
	sel := result.(*selectionItem)
	if sel.Label != "B" {
		t.Errorf("expected 'B', got '%s'", sel.Label)
	}
}

func TestSelectionOverlayViewContainsTitle(t *testing.T) {
	o := newSelectionOverlay("My Title", []selectionItem{{ID: "1", Label: "Item"}})
	view := o.View(80, 24)
	if !strings.Contains(view, "My Title") {
		t.Error("expected title in view")
	}
	if !strings.Contains(view, "Item") {
		t.Error("expected item in view")
	}
}

func TestSelectionOverlayEmptyFilterShowsMessage(t *testing.T) {
	var o overlay = newSelectionOverlay("Pick", []selectionItem{{ID: "1", Label: "Alpha"}})
	for _, ch := range "zzz" {
		o = updateOverlay(o, keyMsg(string(ch)))
	}
	view := o.View(80, 24)
	if !strings.Contains(view, "No matches") {
		t.Error("expected 'No matches' message")
	}
}

func TestTextInputOverlayPreFilled(t *testing.T) {
	ti := newTextInputOverlay("Edit Title", "Hello World")
	if ti.input.Value() != "Hello World" {
		t.Errorf("expected pre-filled value 'Hello World', got '%s'", ti.input.Value())
	}
}

func TestTextInputOverlayEnterSaves(t *testing.T) {
	ti := newTextInputOverlay("Edit Title", "original")
	ti.input.SetValue("new title")
	var o overlay = ti
	o = updateOverlay(o, keyMsg("enter"))

	isDone, result := o.done()
	if !isDone {
		t.Error("expected done after enter")
	}
	if result != "new title" {
		t.Errorf("expected 'new title', got '%v'", result)
	}
}

func TestTextInputOverlayEscCancels(t *testing.T) {
	var o overlay = newTextInputOverlay("Edit Title", "original")
	o = updateOverlay(o, keyMsg("esc"))

	isDone, result := o.done()
	if !isDone {
		t.Error("expected done after esc")
	}
	if result != nil {
		t.Error("expected nil result on cancel")
	}
}

func TestTextInputOverlayView(t *testing.T) {
	ti := newTextInputOverlay("Edit Title", "hello")
	view := ti.View(80, 24)
	if !strings.Contains(view, "Edit Title") {
		t.Error("expected title in view")
	}
	if !strings.Contains(view, "enter: save") {
		t.Error("expected hint in view")
	}
}

func TestTextEditorOverlayCtrlSSaves(t *testing.T) {
	te := newTextEditorOverlay("Edit Description", "original text", 80, 24)
	te.editor.SetValue("updated text")
	var o overlay = te
	o = updateOverlay(o, keyMsg("ctrl+s"))

	isDone, result := o.done()
	if !isDone {
		t.Error("expected done after ctrl+s")
	}
	if result != "updated text" {
		t.Errorf("expected 'updated text', got '%v'", result)
	}
}

func TestTextEditorOverlayEscCancels(t *testing.T) {
	var o overlay = newTextEditorOverlay("Edit Description", "text", 80, 24)
	o = updateOverlay(o, keyMsg("esc"))

	isDone, result := o.done()
	if !isDone {
		t.Error("expected done after esc")
	}
	if result != nil {
		t.Error("expected nil result on cancel")
	}
}

func TestTextEditorOverlayView(t *testing.T) {
	te := newTextEditorOverlay("Edit Desc", "hello", 80, 24)
	view := te.View(80, 24)
	if !strings.Contains(view, "Edit Desc") {
		t.Error("expected title in view")
	}
	if !strings.Contains(view, "ctrl+s: save") {
		t.Error("expected hint in view")
	}
}

func TestConfirmOverlayYConfirms(t *testing.T) {
	var o overlay = newConfirmOverlay("Delete PROJ-1?")
	o = updateOverlay(o, keyMsg("y"))

	isDone, result := o.done()
	if !isDone {
		t.Error("expected done")
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestConfirmOverlayNDenies(t *testing.T) {
	var o overlay = newConfirmOverlay("Delete PROJ-1?")
	o = updateOverlay(o, keyMsg("n"))

	isDone, result := o.done()
	if !isDone {
		t.Error("expected done")
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestConfirmOverlayEscDenies(t *testing.T) {
	var o overlay = newConfirmOverlay("Delete PROJ-1?")
	o = updateOverlay(o, keyMsg("esc"))

	isDone, result := o.done()
	if !isDone {
		t.Error("expected done")
	}
	if result != nil {
		t.Error("expected nil on esc")
	}
}

func TestConfirmOverlayView(t *testing.T) {
	c := newConfirmOverlay("Are you sure?")
	view := c.View(80, 24)
	if !strings.Contains(view, "Are you sure?") {
		t.Error("expected message in view")
	}
	if !strings.Contains(view, "y: confirm") {
		t.Error("expected hint in view")
	}
}

func TestSelectionOverlayWithDescriptions(t *testing.T) {
	items := []selectionItem{
		{ID: "1", Label: "Alice", Desc: "alice@example.com"},
		{ID: "2", Label: "Bob", Desc: "bob@example.com"},
	}
	var o overlay = newSelectionOverlay("Select User", items)

	for _, ch := range "bob" {
		o = updateOverlay(o, keyMsg(string(ch)))
	}
	s := o.(*selectionOverlay)
	if len(s.filtered) != 1 {
		t.Errorf("expected 1 match for 'bob', got %d", len(s.filtered))
	}

	o2 := newSelectionOverlay("Select User", items)
	view := o2.View(80, 24)
	if !strings.Contains(view, "alice@example.com") {
		t.Error("expected description in view")
	}
}
