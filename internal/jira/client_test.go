package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://example.atlassian.net", "user@example.com", "token")
	if c.baseURL != "https://example.atlassian.net" {
		t.Errorf("expected base URL to be set, got %s", c.baseURL)
	}
	if c.email != "user@example.com" {
		t.Errorf("expected email to be set, got %s", c.email)
	}
}

func TestGetMyself(t *testing.T) {
	expected := User{
		AccountID:   "abc123",
		DisplayName: "Test User",
		Email:       "test@example.com",
		Active:      true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/myself" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Verify basic auth is set
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth")
		}
		if user != "test@example.com" || pass != "token" {
			t.Error("unexpected credentials")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	user, err := c.GetMyself(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.AccountID != expected.AccountID {
		t.Errorf("expected account ID %s, got %s", expected.AccountID, user.AccountID)
	}
	if user.DisplayName != expected.DisplayName {
		t.Errorf("expected display name %s, got %s", expected.DisplayName, user.DisplayName)
	}
}

func TestClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Unauthorized"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, "bad@example.com", "wrong")
	_, err := c.GetMyself(context.Background())
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
