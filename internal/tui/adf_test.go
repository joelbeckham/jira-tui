package tui

import (
	"testing"
)

func TestExtractADFText_NilInput(t *testing.T) {
	result := extractADFText(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestExtractADFText_StringInput(t *testing.T) {
	result := extractADFText("plain text")
	if result != "plain text" {
		t.Errorf("expected 'plain text', got %q", result)
	}
}

func TestExtractADFText_SingleParagraph(t *testing.T) {
	doc := map[string]interface{}{
		"type":    "doc",
		"version": float64(1),
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Hello world",
					},
				},
			},
		},
	}
	result := extractADFText(doc)
	if result != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", result)
	}
}

func TestExtractADFText_MultipleParagraphs(t *testing.T) {
	doc := map[string]interface{}{
		"type":    "doc",
		"version": float64(1),
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "First paragraph",
					},
				},
			},
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Second paragraph",
					},
				},
			},
		},
	}
	result := extractADFText(doc)
	expected := "First paragraph\nSecond paragraph"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExtractADFText_HeadingAndParagraph(t *testing.T) {
	doc := map[string]interface{}{
		"type": "doc",
		"content": []interface{}{
			map[string]interface{}{
				"type": "heading",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Title",
					},
				},
			},
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Body text",
					},
				},
			},
		},
	}
	result := extractADFText(doc)
	expected := "Title\nBody text"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExtractADFText_InlineFormatting(t *testing.T) {
	// ADF with bold/italic marks — we just extract the text nodes
	doc := map[string]interface{}{
		"type": "doc",
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Normal ",
					},
					map[string]interface{}{
						"type": "text",
						"text": "bold",
						"marks": []interface{}{
							map[string]interface{}{"type": "strong"},
						},
					},
					map[string]interface{}{
						"type": "text",
						"text": " text",
					},
				},
			},
		},
	}
	result := extractADFText(doc)
	if result != "Normal bold text" {
		t.Errorf("expected 'Normal bold text', got %q", result)
	}
}

func TestExtractADFText_EmptyDoc(t *testing.T) {
	doc := map[string]interface{}{
		"type":    "doc",
		"content": []interface{}{},
	}
	result := extractADFText(doc)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestMakeADFDocument_SingleParagraph(t *testing.T) {
	doc := makeADFDocument("Hello world")
	content, ok := doc["content"].([]interface{})
	if !ok {
		t.Fatal("expected content array")
	}
	if len(content) != 1 {
		t.Fatalf("expected 1 paragraph, got %d", len(content))
	}
	para := content[0].(map[string]interface{})
	if para["type"] != "paragraph" {
		t.Errorf("expected paragraph type, got %v", para["type"])
	}
	paraContent := para["content"].([]interface{})
	textNode := paraContent[0].(map[string]interface{})
	if textNode["text"] != "Hello world" {
		t.Errorf("expected 'Hello world', got %v", textNode["text"])
	}
}

func TestMakeADFDocument_MultipleParagraphs(t *testing.T) {
	doc := makeADFDocument("First\n\nSecond")
	content := doc["content"].([]interface{})
	if len(content) != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", len(content))
	}
}

func TestMakeADFDocument_Empty(t *testing.T) {
	doc := makeADFDocument("")
	content := doc["content"].([]interface{})
	if len(content) != 0 {
		t.Fatalf("expected 0 paragraphs, got %d", len(content))
	}
}
