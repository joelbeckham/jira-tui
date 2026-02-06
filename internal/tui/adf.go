package tui

import (
	"fmt"
	"strings"
)

// extractADFText recursively extracts plain text from a Jira ADF document.
// ADF is a JSON structure with "type" and "content" fields. We walk the tree
// and concatenate all "text" nodes, inserting newlines at paragraph/heading
// boundaries.
func extractADFText(doc interface{}) string {
	if doc == nil {
		return ""
	}

	// If it's already a string, return it directly.
	if s, ok := doc.(string); ok {
		return s
	}

	node, ok := doc.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("%v", doc)
	}

	var b strings.Builder
	extractNode(&b, node, true)
	return strings.TrimSpace(b.String())
}

// extractNode recursively processes an ADF node.
func extractNode(b *strings.Builder, node map[string]interface{}, topLevel bool) {
	nodeType, _ := node["type"].(string)

	// If this is a text node, write the text content.
	if nodeType == "text" {
		if text, ok := node["text"].(string); ok {
			b.WriteString(text)
		}
		return
	}

	// If this is a hardBreak, emit a newline.
	if nodeType == "hardBreak" {
		b.WriteString("\n")
		return
	}

	// Process children.
	content, ok := node["content"].([]interface{})
	if !ok {
		return
	}

	for _, child := range content {
		childNode, ok := child.(map[string]interface{})
		if !ok {
			continue
		}
		extractNode(b, childNode, false)
	}

	// Add newline after block-level elements.
	switch nodeType {
	case "paragraph", "heading", "blockquote", "codeBlock",
		"bulletList", "orderedList", "listItem", "rule",
		"mediaSingle", "mediaGroup", "decisionList", "taskList":
		b.WriteString("\n")
	}
}

// makeADFDocument wraps plain text in a minimal ADF document suitable for
// the Jira API description field.
func makeADFDocument(text string) map[string]interface{} {
	// Split into paragraphs on double newlines, fall back to single line
	paragraphs := strings.Split(text, "\n\n")
	content := make([]interface{}, 0, len(paragraphs))
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		content = append(content, map[string]interface{}{
			"type": "paragraph",
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": p,
				},
			},
		})
	}
	return map[string]interface{}{
		"version": 1,
		"type":    "doc",
		"content": content,
	}
}
