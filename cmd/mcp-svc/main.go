package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	apiURL string
	token  string
	hc     = &http.Client{}
)

var statusMap = map[string]int{
	"open": 1, "in_progress": 2, "waiting": 3, "resolved": 4, "closed": 5,
}
var priorityMap = map[string]int{
	"low": 1, "medium": 2, "high": 3, "critical": 4,
}
var categoryMap = map[string]int{
	"bug": 1, "feature": 2, "support": 3, "docs": 4, "infra": 5,
}

func main() {
	apiURL = strings.TrimRight(os.Getenv("TRIAGE_API_URL"), "/")
	if apiURL == "" {
		fmt.Fprintln(os.Stderr, "TRIAGE_API_URL must be set")
		os.Exit(1)
	}

	// Auto-login preferred; static token as fallback.
	username := os.Getenv("TRIAGE_USERNAME")
	password := os.Getenv("TRIAGE_PASSWORD")
	if username != "" && password != "" {
		t, err := login(username, password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
			os.Exit(1)
		}
		token = t
	} else {
		token = os.Getenv("TRIAGE_TOKEN")
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "set TRIAGE_USERNAME+TRIAGE_PASSWORD or TRIAGE_TOKEN")
		os.Exit(1)
	}

	s := server.NewMCPServer("triage", "1.0.0")

	s.AddTool(mcp.NewTool("list_tickets",
		mcp.WithDescription("List Triage tickets. Returns id, title, status, priority, category, assignedTo, and createdAt for each ticket."),
		mcp.WithString("status",
			mcp.Description("Filter by status: open, in_progress, waiting, resolved, closed"),
		),
		mcp.WithString("priority",
			mcp.Description("Filter by priority: low, medium, high, critical"),
		),
		mcp.WithString("search",
			mcp.Description("Full-text search across title and description"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number (default 1, page size 50)"),
		),
	), listTickets)

	s.AddTool(mcp.NewTool("get_ticket",
		mcp.WithDescription("Get full detail for a single ticket including all comments."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Ticket UUID"),
		),
	), getTicket)

	s.AddTool(mcp.NewTool("create_ticket",
		mcp.WithDescription("Create a new Triage ticket."),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Short summary of the issue"),
		),
		mcp.WithString("description",
			mcp.Description("Full description"),
		),
		mcp.WithString("priority",
			mcp.Description("low, medium (default), high, critical"),
		),
		mcp.WithString("category",
			mcp.Description("bug, feature, support (default), docs, infra"),
		),
		mcp.WithString("assigned_to",
			mcp.Description("Username to assign the ticket to"),
		),
	), createTicket)

	s.AddTool(mcp.NewTool("update_ticket",
		mcp.WithDescription("Update a ticket's status, priority, assignment, title, or description. Only provided fields are changed."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Ticket UUID"),
		),
		mcp.WithString("status",
			mcp.Description("open, in_progress, waiting, resolved, closed"),
		),
		mcp.WithString("priority",
			mcp.Description("low, medium, high, critical"),
		),
		mcp.WithString("assigned_to",
			mcp.Description("Username to assign to"),
		),
		mcp.WithString("title",
			mcp.Description("New title"),
		),
		mcp.WithString("description",
			mcp.Description("New description"),
		),
	), updateTicket)

	s.AddTool(mcp.NewTool("add_comment",
		mcp.WithDescription("Add a comment to a ticket."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Ticket UUID"),
		),
		mcp.WithString("body",
			mcp.Required(),
			mcp.Description("Comment text"),
		),
	), addComment)

	s.AddTool(mcp.NewTool("get_dashboard_stats",
		mcp.WithDescription("Get ticket counts by status, average resolution time, and daily ticket activity."),
	), getDashboardStats)

	if os.Getenv("MCP_TRANSPORT") == "http" {
		port := os.Getenv("MCP_PORT")
		if port == "" {
			port = "8090"
		}

		cfg, err := loadOAuthConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		mcpHandler := server.NewStreamableHTTPServer(s)

		mux := http.NewServeMux()
		mux.Handle("/.well-known/oauth-protected-resource", server.NewProtectedResourceMetadataHandler(
			server.ProtectedResourceMetadataConfig{
				Resource:               cfg.serverURL,
				AuthorizationServers:   []string{cfg.serverURL},
				ScopesSupported:        []string{"tickets:read", "tickets:write"},
				BearerMethodsSupported: []string{"header"},
			},
		))
		mux.Handle("/.well-known/oauth-authorization-server", authServerMetadataHandler(cfg))
		mux.Handle("/oauth/authorize", authorizeHandler(cfg))
		mux.Handle("/oauth/login", loginHandler(cfg))
		mux.Handle("/oauth/token", tokenHandler(cfg))
		mux.Handle("/mcp", requireBearerToken(cfg, mcpHandler))

		fmt.Fprintf(os.Stderr, "mcp-svc listening on :%s\n", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadOAuthConfig() (oauthConfig, error) {
	serverURL := strings.TrimRight(os.Getenv("MCP_SERVER_URL"), "/")
	clientID := os.Getenv("OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("OAUTH_CLIENT_SECRET")
	jwtSecret := os.Getenv("MCP_JWT_SECRET")
	adminUser := os.Getenv("TRIAGE_ADMIN_USER")
	adminPassHash := os.Getenv("TRIAGE_ADMIN_PASS_HASH")

	var missing []string
	if serverURL == "" {
		missing = append(missing, "MCP_SERVER_URL")
	}
	if clientID == "" {
		missing = append(missing, "OAUTH_CLIENT_ID")
	}
	if clientSecret == "" {
		missing = append(missing, "OAUTH_CLIENT_SECRET")
	}
	if jwtSecret == "" {
		missing = append(missing, "MCP_JWT_SECRET")
	}
	if adminUser == "" {
		missing = append(missing, "TRIAGE_ADMIN_USER")
	}
	if adminPassHash == "" {
		missing = append(missing, "TRIAGE_ADMIN_PASS_HASH")
	}
	if len(missing) > 0 {
		return oauthConfig{}, fmt.Errorf("missing required env vars for HTTP transport: %s", strings.Join(missing, ", "))
	}

	return oauthConfig{
		serverURL:     serverURL,
		clientID:      clientID,
		clientSecret:  clientSecret,
		jwtSecret:     []byte(jwtSecret),
		adminUser:     adminUser,
		adminPassHash: adminPassHash,
	}, nil
}

// ── Auth ─────────────────────────────────────────────────────────────────────

func login(username, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := hc.Post(apiURL+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Token == "" {
		return "", fmt.Errorf("login returned no token (status %d)", resp.StatusCode)
	}
	return result.Token, nil
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

func apiGet(path string) ([]byte, error) {
	req, _ := http.NewRequest(http.MethodGet, apiURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API %s: %s", resp.Status, b)
	}
	return b, nil
}

func apiSend(method, path string, body any) ([]byte, error) {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, apiURL+path, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API %s: %s", resp.Status, rb)
	}
	return rb, nil
}

func result(b []byte, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultText("Error: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func args(req mcp.CallToolRequest) map[string]any {
	m, _ := req.Params.Arguments.(map[string]any)
	return m
}

func strArg(req mcp.CallToolRequest, key string) string {
	v, _ := args(req)[key].(string)
	return v
}

// ── Tool handlers ─────────────────────────────────────────────────────────────

func listTickets(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	q := url.Values{}
	if s := strArg(req, "status"); s != "" {
		if n, ok := statusMap[s]; ok {
			q.Set("status", strconv.Itoa(n))
		}
	}
	if p := strArg(req, "priority"); p != "" {
		if n, ok := priorityMap[p]; ok {
			q.Set("priority", strconv.Itoa(n))
		}
	}
	if s := strArg(req, "search"); s != "" {
		q.Set("search", s)
	}
	if pg, ok := args(req)["page"].(float64); ok {
		q.Set("page", strconv.Itoa(int(pg)))
	}
	path := "/api/tickets"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	return result(apiGet(path))
}

func getTicket(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return result(apiGet("/api/tickets/" + strArg(req, "id")))
}

func createTicket(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	priority := 2 // medium
	if p := strArg(req, "priority"); p != "" {
		if n, ok := priorityMap[p]; ok {
			priority = n
		}
	}
	category := 3 // support
	if c := strArg(req, "category"); c != "" {
		if n, ok := categoryMap[c]; ok {
			category = n
		}
	}
	return result(apiSend(http.MethodPost, "/api/tickets", map[string]any{
		"title":       strArg(req, "title"),
		"description": strArg(req, "description"),
		"priority":    priority,
		"category":    category,
		"assignedTo":  strArg(req, "assigned_to"),
	}))
}

func updateTicket(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := strArg(req, "id")
	body := map[string]any{}
	if s := strArg(req, "status"); s != "" {
		if n, ok := statusMap[s]; ok {
			body["status"] = n
		}
	}
	if p := strArg(req, "priority"); p != "" {
		if n, ok := priorityMap[p]; ok {
			body["priority"] = n
		}
	}
	if a := strArg(req, "assigned_to"); a != "" {
		body["assignedTo"] = a
	}
	if t := strArg(req, "title"); t != "" {
		body["title"] = t
	}
	if d := strArg(req, "description"); d != "" {
		body["description"] = d
	}
	return result(apiSend(http.MethodPut, "/api/tickets/"+id, body))
}

func addComment(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return result(apiSend(http.MethodPost, "/api/tickets/"+strArg(req, "id")+"/comments",
		map[string]string{"body": strArg(req, "body")},
	))
}

func getDashboardStats(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return result(apiGet("/api/dashboard"))
}
