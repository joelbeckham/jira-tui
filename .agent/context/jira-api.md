# Jira Cloud REST API v3 Reference

## OpenAPI Spec

The full OpenAPI 3.0 spec for Jira Cloud REST API v3 is available at:

**https://dac-static.atlassian.com/cloud/jira/platform/swagger-v3.v3.json?_v=1.8348.0**

> Do NOT download this into the repo — it's ~5MB of JSON.
> Agents should fetch specific endpoints on-demand using web fetch tools.

## Documentation

- API docs: https://developer.atlassian.com/cloud/jira/platform/rest/v3/intro/
- Auth: https://developer.atlassian.com/cloud/jira/platform/basic-auth-for-rest-apis/

## Key Endpoints We Use

| Endpoint | Purpose |
|---|---|
| `GET /rest/api/3/myself` | Verify auth, get current user |
| `GET /rest/api/3/filter/{id}` | Get a saved filter (includes JQL) |
| `GET /rest/api/3/search?jql=...` | Search issues by JQL |
| `GET /rest/api/3/issue/{issueIdOrKey}` | Get a single issue |
| `POST /rest/api/3/issue` | Create an issue |
| `GET /rest/api/3/issue/{issueIdOrKey}/transitions` | Get available status transitions |
| `POST /rest/api/3/issue/{issueIdOrKey}/transitions` | Transition issue status |

## Authentication

- Method: Basic Auth (email + API token)
- Header: `Authorization: Basic base64(email:apiToken)`
- API tokens: https://id.atlassian.com/manage-profile/security/api-tokens

## Agent Instructions

When implementing a new Jira API endpoint:
1. Fetch the OpenAPI spec URL above for the specific endpoint details
2. Check field types and required parameters
3. Add types to `internal/jira/types.go`
4. Add the client method to the appropriate file in `internal/jira/`
5. Write tests using `httptest.NewServer` with realistic mock responses
