package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/idealisan/user-preferences-mcp/internal/preference"
	"github.com/idealisan/user-preferences-mcp/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToolsAndResources(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(filepath.Join(t.TempDir(), "preferences.db"))
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	defer db.Close()

	svc := preference.NewService(storage.NewRepository(db))
	server := New(svc, "default", "test")

	handler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{},
	)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             httpServer.URL,
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	call := func(name string, args map[string]any) string {
		t.Helper()
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("call %s: %v", name, err)
		}
		if res.IsError {
			t.Fatalf("call %s returned tool error: %s", name, contentText(res))
		}
		return contentText(res)
	}

	call("set_preference", map[string]any{
		"key":         "coding.language",
		"value":       "Go",
		"scope":       "global",
		"description": "preferred language",
	})
	call("set_preference", map[string]any{
		"key":     "coding.style",
		"value":   "simple",
		"scope":   "global",
		"user_id": "alice",
	})
	call("set_preference", map[string]any{
		"key":      "coding.language",
		"value":    "Python",
		"scope":    "project",
		"scope_id": "project-a",
	})

	got := call("get_preference", map[string]any{
		"key":   "coding.language",
		"scope": "global",
	})
	if !strings.Contains(got, `"Go"`) {
		t.Fatalf("get_preference = %s, want Go", got)
	}

	effective := call("get_effective_preferences", map[string]any{
		"project_id": "project-a",
		"category":   "coding",
	})
	var effectiveOut struct {
		Preferences map[string]any    `json:"preferences"`
		Sources     map[string]string `json:"sources"`
	}
	if err := json.Unmarshal([]byte(effective), &effectiveOut); err != nil {
		t.Fatalf("decode effective preferences %q: %v", effective, err)
	}
	if effectiveOut.Preferences["coding.language"] != "Python" {
		t.Fatalf("effective coding.language = %v, want Python", effectiveOut.Preferences["coding.language"])
	}
	if effectiveOut.Sources["coding.language"] != "project" {
		t.Fatalf("effective source = %q, want project", effectiveOut.Sources["coding.language"])
	}

	listed := call("list_preferences", map[string]any{"category": "coding"})
	if !strings.Contains(listed, `"count":2`) {
		t.Fatalf("list_preferences = %s, want 2 global items", listed)
	}

	resource, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "preferences://global/coding"})
	if err != nil {
		t.Fatalf("read resource: %v", err)
	}
	if len(resource.Contents) != 1 || !strings.Contains(resource.Contents[0].Text, `"Go"`) {
		t.Fatalf("resource contents = %+v, want Go preference", resource.Contents)
	}

	deleted := call("delete_preference", map[string]any{
		"key":   "coding.language",
		"scope": "global",
	})
	if !strings.Contains(deleted, `"deleted":true`) {
		t.Fatalf("delete_preference = %s, want deleted true", deleted)
	}
}

func contentText(res *mcp.CallToolResult) string {
	var parts []string
	for _, c := range res.Content {
		if text, ok := c.(*mcp.TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}
