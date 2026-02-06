package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CachedUser is the minimal user data stored in the cache file.
type CachedUser struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	Email       string `json:"emailAddress,omitempty"`
}

// UserCachePath returns the path to the user cache file.
func UserCachePath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "users.json"), nil
}

// LoadUserCache reads the user cache file. Returns nil, nil if the file
// does not exist (caller should fetch from API and call SaveUserCache).
func LoadUserCache() ([]CachedUser, error) {
	path, err := UserCachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // cache miss — not an error
		}
		return nil, fmt.Errorf("reading user cache: %w", err)
	}

	var users []CachedUser
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("parsing user cache: %w", err)
	}
	return users, nil
}

// SaveUserCache writes user data to the cache file.
func SaveUserCache(users []CachedUser) error {
	path, err := UserCachePath()
	if err != nil {
		return err
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling user cache: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing user cache: %w", err)
	}
	return nil
}
