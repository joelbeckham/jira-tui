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

func TestGetFilter(t *testing.T) {
	expected := Filter{
		ID:        "10042",
		Name:      "My Sprint Work",
		JQL:       "assignee = currentUser() AND sprint in openSprints()",
		ViewURL:   "https://example.atlassian.net/issues/?filter=10042",
		SearchURL: "https://example.atlassian.net/rest/api/3/search?jql=assignee+%3D+currentUser()",
		Favourite: true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/filter/10042" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		_, _, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	filter, err := c.GetFilter(context.Background(), "10042")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filter.ID != expected.ID {
		t.Errorf("expected filter ID %s, got %s", expected.ID, filter.ID)
	}
	if filter.Name != expected.Name {
		t.Errorf("expected filter name %s, got %s", expected.Name, filter.Name)
	}
	if filter.JQL != expected.JQL {
		t.Errorf("expected JQL %s, got %s", expected.JQL, filter.JQL)
	}
}

func TestGetFilterNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errorMessages":["The filter with id '99999' does not exist."]}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	_, err := c.GetFilter(context.Background(), "99999")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestSearchIssues(t *testing.T) {
	expected := SearchResult{
		StartAt:    0,
		MaxResults: 50,
		Total:      2,
		Issues: []Issue{
			{ID: "10001", Key: "PROJ-1", Fields: IssueFields{Summary: "First issue"}},
			{ID: "10002", Key: "PROJ-2", Fields: IssueFields{Summary: "Second issue"}},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["jql"] != "project = PROJ" {
			t.Errorf("unexpected JQL: %v", body["jql"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	result, err := c.SearchIssues(context.Background(), SearchOptions{
		JQL:    "project = PROJ",
		Fields: []string{"summary", "status", "assignee"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 total issues, got %d", result.Total)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
	if result.Issues[0].Key != "PROJ-1" {
		t.Errorf("expected PROJ-1, got %s", result.Issues[0].Key)
	}
}

func TestSearchIssuesDefaultMaxResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["maxResults"] != float64(50) {
			t.Errorf("expected default maxResults 50, got %v", body["maxResults"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResult{})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	_, err := c.SearchIssues(context.Background(), SearchOptions{JQL: "project = TEST"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
