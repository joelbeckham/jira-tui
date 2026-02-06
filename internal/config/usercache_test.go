package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUserCacheRoundTrip(t *testing.T) {
	// Use a temp dir instead of the real config dir
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "users.json")

	users := []CachedUser{
		{AccountID: "u1", DisplayName: "Alice", Email: "alice@example.com"},
		{AccountID: "u2", DisplayName: "Bob"},
	}

	// Write
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read back
	readData, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var loaded []CachedUser
	if err := json.Unmarshal(readData, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 users, got %d", len(loaded))
	}
	if loaded[0].AccountID != "u1" {
		t.Errorf("expected u1, got %s", loaded[0].AccountID)
	}
	if loaded[0].DisplayName != "Alice" {
		t.Errorf("expected Alice, got %s", loaded[0].DisplayName)
	}
	if loaded[1].AccountID != "u2" {
		t.Errorf("expected u2, got %s", loaded[1].AccountID)
	}
}
