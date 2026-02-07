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
	Summary     string       `json:"summary"`
	Description interface{}  `json:"description"` // ADF document (map) or string
	Status      *Status      `json:"status"`
	Assignee    *User        `json:"assignee"`
	Reporter    *User        `json:"reporter"`
	Priority    *Named       `json:"priority"`
	IssueType   *Named       `json:"issuetype"`
	Project     *Named       `json:"project"`
	Created     string       `json:"created"`
	Updated     string       `json:"updated"`
	Labels      []string     `json:"labels"`
	Subtasks    []Issue      `json:"subtasks"`
	IssueLinks  []IssueLink  `json:"issuelinks"`
	Parent      *ParentIssue `json:"parent"`
}

// ParentIssue is a minimal issue reference for the parent field.
type ParentIssue struct {
	ID     string       `json:"id"`
	Key    string       `json:"key"`
	Fields *IssueFields `json:"fields,omitempty"`
}

// IssueLink represents a link between two issues.
type IssueLink struct {
	ID           string   `json:"id"`
	Type         LinkType `json:"type"`
	InwardIssue  *Issue   `json:"inwardIssue,omitempty"`
	OutwardIssue *Issue   `json:"outwardIssue,omitempty"`
}

// LinkType describes the type of issue link.
type LinkType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
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

// SearchResult represents the response from a JQL search
// (POST /rest/api/3/search/jql — enhanced search).
type SearchResult struct {
	Issues        []Issue `json:"issues"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
	IsLast        bool    `json:"isLast"`
}

// SearchOptions configures a JQL search request.
type SearchOptions struct {
	JQL           string
	Fields        []string
	MaxResults    int
	NextPageToken string
}

// Transition represents an available workflow transition.
type Transition struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	To   *Status `json:"to"`
}

// TransitionsResponse wraps the list returned by GET transitions.
type TransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

// CreateIssueRequest is the body for POST /rest/api/3/issue.
type CreateIssueRequest struct {
	Fields map[string]interface{} `json:"fields"`
}

// CreateIssueResponse is the response from POST /rest/api/3/issue.
type CreateIssueResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

// Comment represents a single Jira issue comment.
type Comment struct {
	ID      string      `json:"id"`
	Author  *User       `json:"author"`
	Body    interface{} `json:"body"` // ADF document (same format as issue description)
	Created string      `json:"created"`
	Updated string      `json:"updated"`
}

// CommentsResponse is the paginated response from GET issue comments.
type CommentsResponse struct {
	Comments   []Comment `json:"comments"`
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
}
