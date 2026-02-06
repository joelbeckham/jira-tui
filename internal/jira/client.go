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

// SearchIssues performs a JQL search and returns matching issues.
func (c *Client) SearchIssues(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	if opts.MaxResults == 0 {
		opts.MaxResults = 50
	}

	body := map[string]interface{}{
		"jql":        opts.JQL,
		"startAt":    opts.StartAt,
		"maxResults": opts.MaxResults,
	}
	if len(opts.Fields) > 0 {
		body["fields"] = opts.Fields
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling search request: %w", err)
	}

	data, err := c.do(ctx, http.MethodPost, "/rest/api/3/search", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("searching issues: %w", err)
	}

	var result SearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}
	return &result, nil
}
