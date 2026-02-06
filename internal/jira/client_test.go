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
		IsLast: true,
		Issues: []Issue{
			{ID: "10001", Key: "PROJ-1", Fields: IssueFields{Summary: "First issue"}},
			{ID: "10002", Key: "PROJ-2", Fields: IssueFields{Summary: "Second issue"}},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/search/jql" {
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
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
	if result.Issues[0].Key != "PROJ-1" {
		t.Errorf("expected PROJ-1, got %s", result.Issues[0].Key)
	}
	if !result.IsLast {
		t.Error("expected IsLast=true")
	}
}

func TestSearchIssuesDefaultMaxResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/search/jql" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["maxResults"] != float64(50) {
			t.Errorf("expected default maxResults 50, got %v", body["maxResults"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResult{IsLast: true})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	_, err := c.SearchIssues(context.Background(), SearchOptions{JQL: "project = TEST"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetIssue(t *testing.T) {
	expected := Issue{
		ID:  "10001",
		Key: "PROJ-1",
		Fields: IssueFields{
			Summary: "Test issue",
			Labels:  []string{"bug", "urgent"},
			Subtasks: []Issue{
				{ID: "10002", Key: "PROJ-2", Fields: IssueFields{Summary: "Subtask"}},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	issue, err := c.GetIssue(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue.Key != "PROJ-1" {
		t.Errorf("expected PROJ-1, got %s", issue.Key)
	}
	if issue.Fields.Summary != "Test issue" {
		t.Errorf("expected summary 'Test issue', got %s", issue.Fields.Summary)
	}
	if len(issue.Fields.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(issue.Fields.Labels))
	}
	if len(issue.Fields.Subtasks) != 1 {
		t.Errorf("expected 1 subtask, got %d", len(issue.Fields.Subtasks))
	}
}

func TestUpdateIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		fields, ok := body["fields"].(map[string]interface{})
		if !ok {
			t.Fatal("expected fields in body")
		}
		if fields["summary"] != "Updated title" {
			t.Errorf("expected updated title, got %v", fields["summary"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	err := c.UpdateIssue(context.Background(), "PROJ-1", map[string]interface{}{
		"summary": "Updated title",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var body CreateIssueRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Fields["summary"] != "New issue" {
			t.Errorf("expected summary 'New issue', got %v", body.Fields["summary"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateIssueResponse{
			ID:   "10099",
			Key:  "PROJ-99",
			Self: "https://example.atlassian.net/rest/api/3/issue/10099",
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	resp, err := c.CreateIssue(context.Background(), CreateIssueRequest{
		Fields: map[string]interface{}{
			"summary":   "New issue",
			"project":   map[string]string{"key": "PROJ"},
			"issuetype": map[string]string{"name": "Task"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Key != "PROJ-99" {
		t.Errorf("expected PROJ-99, got %s", resp.Key)
	}
}

func TestDeleteIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Query().Get("deleteSubtasks") != "true" {
			t.Error("expected deleteSubtasks=true")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	err := c.DeleteIssue(context.Background(), "PROJ-1", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteIssueNoSubtasks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("deleteSubtasks") == "true" {
			t.Error("did not expect deleteSubtasks=true")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	err := c.DeleteIssue(context.Background(), "PROJ-1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTransitions(t *testing.T) {
	expected := TransitionsResponse{
		Transitions: []Transition{
			{
				ID:   "11",
				Name: "In Progress",
				To: &Status{
					Name:           "In Progress",
					ID:             "3",
					StatusCategory: &StatusCategory{ID: 4, Key: "indeterminate", Name: "In Progress"},
				},
			},
			{
				ID:   "21",
				Name: "Done",
				To: &Status{
					Name:           "Done",
					ID:             "5",
					StatusCategory: &StatusCategory{ID: 3, Key: "done", Name: "Done"},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1/transitions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	transitions, err := c.GetTransitions(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(transitions) != 2 {
		t.Fatalf("expected 2 transitions, got %d", len(transitions))
	}
	if transitions[0].Name != "In Progress" {
		t.Errorf("expected 'In Progress', got %s", transitions[0].Name)
	}
	if transitions[1].To.StatusCategory.Key != "done" {
		t.Errorf("expected done category, got %s", transitions[1].To.StatusCategory.Key)
	}
}

func TestTransitionIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1/transitions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		transition, ok := body["transition"].(map[string]interface{})
		if !ok {
			t.Fatal("expected transition in body")
		}
		if transition["id"] != "21" {
			t.Errorf("expected transition id '21', got %v", transition["id"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	err := c.TransitionIssue(context.Background(), "PROJ-1", "21")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssignIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1/assignee" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["accountId"] != "abc123" {
			t.Errorf("expected accountId 'abc123', got %v", body["accountId"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	err := c.AssignIssue(context.Background(), "PROJ-1", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchAllUsers(t *testing.T) {
	// Return a mix of active and inactive users
	allUsers := []User{
		{AccountID: "u1", DisplayName: "Active User", Active: true},
		{AccountID: "u2", DisplayName: "Inactive User", Active: false},
		{AccountID: "u3", DisplayName: "Another Active", Active: true},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/users/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(allUsers)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test@example.com", "token")
	users, err := c.SearchAllUsers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should only return active users
	if len(users) != 2 {
		t.Fatalf("expected 2 active users, got %d", len(users))
	}
	if users[0].AccountID != "u1" {
		t.Errorf("expected u1, got %s", users[0].AccountID)
	}
	if users[1].AccountID != "u3" {
		t.Errorf("expected u3, got %s", users[1].AccountID)
	}
}
