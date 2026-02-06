package jira

// User represents a Jira user.
type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	Email       string `json:"emailAddress"`
	Active      bool   `json:"active"`
}

// Issue represents a Jira issue.
type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Self   string      `json:"self"`
	Fields IssueFields `json:"fields"`
}

// IssueFields contains the fields of a Jira issue.
type IssueFields struct {
	Summary     string  `json:"summary"`
	Description string  `json:"description"`
	Status      *Status `json:"status"`
	Assignee    *User   `json:"assignee"`
	Reporter    *User   `json:"reporter"`
	Priority    *Named  `json:"priority"`
	IssueType   *Named  `json:"issuetype"`
	Project     *Named  `json:"project"`
	Created     string  `json:"created"`
	Updated     string  `json:"updated"`
}

// Status represents a Jira status.
type Status struct {
	Name           string          `json:"name"`
	ID             string          `json:"id"`
	StatusCategory *StatusCategory `json:"statusCategory"`
}

// StatusCategory represents a Jira status category.
type StatusCategory struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// Named is a generic type for Jira entities that have an ID and Name.
type Named struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Filter represents a saved Jira filter.
type Filter struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	JQL         string `json:"jql"`
	ViewURL     string `json:"viewUrl"`
	SearchURL   string `json:"searchUrl"`
	Favourite   bool   `json:"favourite"`
}

// Board represents a Jira board.
type Board struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Sprint represents a Jira sprint.
type Sprint struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
	Goal  string `json:"goal"`
}

// SearchResult represents the response from a JQL search.
type SearchResult struct {
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

// SearchOptions configures a JQL search request.
type SearchOptions struct {
	JQL        string
	Fields     []string
	StartAt    int
	MaxResults int
}
