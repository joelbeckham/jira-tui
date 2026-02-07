// Package jira provides a client for the Jira REST API.
package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a Jira REST API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	email      string
	apiToken   string
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// NewClient creates a new Jira API client.
func NewClient(baseURL, email, apiToken string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL:  baseURL,
		email:    email,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// BaseURL returns the Jira instance base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// BrowseURL returns the Jira web URL for the given issue key.
func (c *Client) BrowseURL(issueKey string) string {
	return c.baseURL + "/browse/" + issueKey
}

// do executes an HTTP request with authentication and returns the response body.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// GetMyself returns the currently authenticated user.
func (c *Client) GetMyself(ctx context.Context) (*User, error) {
	data, err := c.do(ctx, http.MethodGet, "/rest/api/3/myself", nil)
	if err != nil {
		return nil, fmt.Errorf("getting myself: %w", err)
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("parsing user: %w", err)
	}
	return &user, nil
}

// GetFilter returns a saved Jira filter by ID.
func (c *Client) GetFilter(ctx context.Context, filterID string) (*Filter, error) {
	path := fmt.Sprintf("/rest/api/3/filter/%s", filterID)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting filter %s: %w", filterID, err)
	}

	var filter Filter
	if err := json.Unmarshal(data, &filter); err != nil {
		return nil, fmt.Errorf("parsing filter: %w", err)
	}
	return &filter, nil
}

// SearchIssues performs a JQL search using the enhanced search endpoint
// (POST /rest/api/3/search/jql) and returns matching issues.
func (c *Client) SearchIssues(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	if opts.MaxResults == 0 {
		opts.MaxResults = 50
	}

	body := map[string]interface{}{
		"jql":        opts.JQL,
		"maxResults": opts.MaxResults,
	}
	if len(opts.Fields) > 0 {
		body["fields"] = opts.Fields
	}
	if opts.NextPageToken != "" {
		body["nextPageToken"] = opts.NextPageToken
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling search request: %w", err)
	}

	data, err := c.do(ctx, http.MethodPost, "/rest/api/3/search/jql", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("searching issues: %w", err)
	}

	var result SearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}
	return &result, nil
}

// GetIssue returns the full details for a single issue by key or ID.
func (c *Client) GetIssue(ctx context.Context, issueKeyOrID string) (*Issue, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s", issueKeyOrID)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting issue %s: %w", issueKeyOrID, err)
	}
	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue: %w", err)
	}
	return &issue, nil
}

// GetComments returns the comments for a Jira issue, newest first.
func (c *Client) GetComments(ctx context.Context, issueKeyOrID string) ([]Comment, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/comment?orderBy=-created&maxResults=50", issueKeyOrID)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting comments for %s: %w", issueKeyOrID, err)
	}
	var resp CommentsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing comments: %w", err)
	}
	return resp.Comments, nil
}

// AddComment adds a comment to a Jira issue. The body is an ADF document.
func (c *Client) AddComment(ctx context.Context, issueKeyOrID string, body map[string]interface{}) (*Comment, error) {
	jsonBody, err := json.Marshal(map[string]interface{}{"body": body})
	if err != nil {
		return nil, fmt.Errorf("marshaling comment: %w", err)
	}
	path := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKeyOrID)
	data, err := c.do(ctx, http.MethodPost, path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("adding comment to %s: %w", issueKeyOrID, err)
	}
	var comment Comment
	if err := json.Unmarshal(data, &comment); err != nil {
		return nil, fmt.Errorf("parsing comment response: %w", err)
	}
	return &comment, nil
}

// UpdateIssue updates an issue's fields (summary, description, priority, etc.).
func (c *Client) UpdateIssue(ctx context.Context, issueKeyOrID string, fields map[string]interface{}) error {
	body := map[string]interface{}{
		"fields": fields,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling update: %w", err)
	}
	path := fmt.Sprintf("/rest/api/3/issue/%s", issueKeyOrID)
	_, err = c.do(ctx, http.MethodPut, path, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("updating issue %s: %w", issueKeyOrID, err)
	}
	return nil
}

// CreateIssue creates a new issue and returns the created issue reference.
func (c *Client) CreateIssue(ctx context.Context, req CreateIssueRequest) (*CreateIssueResponse, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling create request: %w", err)
	}
	data, err := c.do(ctx, http.MethodPost, "/rest/api/3/issue", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating issue: %w", err)
	}
	var resp CreateIssueResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing create response: %w", err)
	}
	return &resp, nil
}

// DeleteIssue deletes an issue, optionally cascading to subtasks.
func (c *Client) DeleteIssue(ctx context.Context, issueKeyOrID string, deleteSubtasks bool) error {
	path := fmt.Sprintf("/rest/api/3/issue/%s", issueKeyOrID)
	if deleteSubtasks {
		path += "?deleteSubtasks=true"
	}
	_, err := c.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("deleting issue %s: %w", issueKeyOrID, err)
	}
	return nil
}

// GetTransitions returns the available transitions for an issue.
func (c *Client) GetTransitions(ctx context.Context, issueKeyOrID string) ([]Transition, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKeyOrID)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting transitions for %s: %w", issueKeyOrID, err)
	}
	var resp TransitionsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing transitions: %w", err)
	}
	return resp.Transitions, nil
}

// TransitionIssue executes a workflow transition on an issue.
func (c *Client) TransitionIssue(ctx context.Context, issueKeyOrID, transitionID string) error {
	body := map[string]interface{}{
		"transition": map[string]string{
			"id": transitionID,
		},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling transition: %w", err)
	}
	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKeyOrID)
	_, err = c.do(ctx, http.MethodPost, path, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("transitioning issue %s: %w", issueKeyOrID, err)
	}
	return nil
}

// AssignIssue assigns an issue to a user by account ID.
// Pass an empty accountID to unassign.
func (c *Client) AssignIssue(ctx context.Context, issueKeyOrID, accountID string) error {
	body := map[string]interface{}{
		"accountId": accountID,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling assign: %w", err)
	}
	path := fmt.Sprintf("/rest/api/3/issue/%s/assignee", issueKeyOrID)
	_, err = c.do(ctx, http.MethodPut, path, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("assigning issue %s: %w", issueKeyOrID, err)
	}
	return nil
}

// Priority represents a Jira priority with its ID and name.
type Priority struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetPriorities fetches all available priorities from the Jira instance.
func (c *Client) GetPriorities(ctx context.Context) ([]Priority, error) {
	data, err := c.do(ctx, http.MethodGet, "/rest/api/3/priority", nil)
	if err != nil {
		return nil, fmt.Errorf("getting priorities: %w", err)
	}
	var priorities []Priority
	if err := json.Unmarshal(data, &priorities); err != nil {
		return nil, fmt.Errorf("parsing priorities: %w", err)
	}
	return priorities, nil
}

// IssueType represents a Jira issue type for a specific project.
type IssueType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Subtask     bool   `json:"subtask"`
	Description string `json:"description,omitempty"`
}

// GetProjectIssueTypes fetches available issue types for a project.
func (c *Client) GetProjectIssueTypes(ctx context.Context, projectKey string) ([]IssueType, error) {
	path := fmt.Sprintf("/rest/api/3/project/%s/statuses", projectKey)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting issue types for %s: %w", projectKey, err)
	}
	// The /project/{key}/statuses endpoint returns [{id, name, subtask, statuses: [...]}]
	var result []IssueType
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing issue types: %w", err)
	}
	// Filter out subtask types — create flow should only offer standard types
	var types []IssueType
	for _, t := range result {
		if !t.Subtask {
			types = append(types, t)
		}
	}
	return types, nil
}

// SearchAllUsers fetches all active users from the instance.
// The Jira API returns users in pages; this method paginates through all results.
func (c *Client) SearchAllUsers(ctx context.Context) ([]User, error) {
	var all []User
	startAt := 0
	maxResults := 1000

	for {
		path := fmt.Sprintf("/rest/api/3/users/search?startAt=%d&maxResults=%d", startAt, maxResults)
		data, err := c.do(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, fmt.Errorf("searching users (startAt=%d): %w", startAt, err)
		}
		var page []User
		if err := json.Unmarshal(data, &page); err != nil {
			return nil, fmt.Errorf("parsing users: %w", err)
		}
		if len(page) == 0 {
			break
		}
		for _, u := range page {
			if u.Active {
				all = append(all, u)
			}
		}
		if len(page) < maxResults {
			break
		}
		startAt += len(page)
	}
	return all, nil
}
