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

func TestTools(t *testing.T) {
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

	init := session.InitializeResult()
	if init == nil || !strings.Contains(init.Instructions, "get_all_preferences") {
		t.Fatalf("initialize result should carry usage instructions, got %+v", init)
	}
	if strings.Contains(init.Instructions, "keyword") {
		t.Fatalf("instructions should not mention keyword filtering: %q", init.Instructions)
	}

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	wantTools := map[string]bool{
		"list_profiles":           true,
		"get_all_preferences":     true,
		"replace_all_preferences": true,
		"append_preferences":      true,
	}
	if len(tools.Tools) != len(wantTools) {
		t.Fatalf("got %d tools, want %d", len(tools.Tools), len(wantTools))
	}
	readOnly := map[string]bool{
		"list_profiles":       true,
		"get_all_preferences": true,
	}
	for _, tool := range tools.Tools {
		if !wantTools[tool.Name] {
			t.Fatalf("unexpected tool %q", tool.Name)
		}
		if tool.Title == "" || len(tool.Description) < 80 {
			t.Errorf("tool %s has insufficient metadata: %+v", tool.Name, tool)
		}
		if tool.Annotations == nil || tool.Annotations.ReadOnlyHint != readOnly[tool.Name] {
			t.Errorf("tool %s annotations = %+v, readOnly want %v", tool.Name, tool.Annotations, readOnly[tool.Name])
		}
	}

	call := func(name string, args map[string]any) (string, bool) {
		t.Helper()
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("call %s: %v", name, err)
		}
		return contentText(res), res.IsError
	}
	mustCall := func(name string, args map[string]any) string {
		t.Helper()
		text, isErr := call(name, args)
		if isErr {
			t.Fatalf("call %s returned tool error: %s", name, text)
		}
		return text
	}

	decode := func(text string) map[string]any {
		t.Helper()
		var out map[string]any
		if err := json.Unmarshal([]byte(text), &out); err != nil {
			t.Fatalf("decode %q: %v", text, err)
		}
		return out
	}

	// Fresh profiles are empty.
	out := decode(mustCall("get_all_preferences", nil))
	if out["count"] != float64(0) || out["profile"] != "default" {
		t.Fatalf("empty get = %v", out)
	}

	// Full replacement stores the exact list in order.
	mustCall("replace_all_preferences", map[string]any{
		"preferences": []string{"用中文交流", "改完代码要跑测试"},
	})
	out = decode(mustCall("get_all_preferences", nil))
	if out["count"] != float64(2) {
		t.Fatalf("get after replace = %v", out)
	}
	items := out["preferences"].([]any)
	if items[0] != "用中文交流" || items[1] != "改完代码要跑测试" {
		t.Fatalf("preferences out of order: %v", items)
	}

	// A second replacement overwrites everything.
	mustCall("replace_all_preferences", map[string]any{
		"preferences": []string{"只保留这一条"},
	})
	out = decode(mustCall("get_all_preferences", nil))
	if out["count"] != float64(1) || out["preferences"].([]any)[0] != "只保留这一条" {
		t.Fatalf("get after second replace = %v", out)
	}

	// Append adds to the end without touching existing items.
	out = decode(mustCall("append_preferences", map[string]any{
		"preferences": []string{"新增一条"},
	}))
	if out["count"] != float64(2) || out["added"] != float64(1) {
		t.Fatalf("append result = %v", out)
	}
	out = decode(mustCall("get_all_preferences", nil))
	items = out["preferences"].([]any)
	if items[0] != "只保留这一条" || items[1] != "新增一条" {
		t.Fatalf("preferences after append = %v", items)
	}

	// Profiles are isolated and discoverable.
	mustCall("replace_all_preferences", map[string]any{
		"profile":     "work",
		"preferences": []string{"工作偏好"},
	})
	out = decode(mustCall("get_all_preferences", map[string]any{"profile": "work"}))
	if out["count"] != float64(1) || out["preferences"].([]any)[0] != "工作偏好" {
		t.Fatalf("work profile = %v", out)
	}
	out = decode(mustCall("get_all_preferences", nil))
	if out["count"] != float64(2) {
		t.Fatalf("default profile was modified: %v", out)
	}
	out = decode(mustCall("list_profiles", nil))
	profiles := out["profiles"].([]any)
	if len(profiles) != 2 || profiles[0] != "default" || profiles[1] != "work" {
		t.Fatalf("list_profiles = %v", out)
	}

	// An empty list clears a profile; other profiles are unaffected.
	out = decode(mustCall("replace_all_preferences", map[string]any{"preferences": []string{}}))
	if out["count"] != float64(0) {
		t.Fatalf("clear result = %v", out)
	}
	out = decode(mustCall("get_all_preferences", nil))
	if out["count"] != float64(0) {
		t.Fatalf("get after clear = %v", out)
	}

	// Blank descriptions are rejected.
	if _, isErr := call("append_preferences", map[string]any{"preferences": []string{"  "}}); !isErr {
		t.Fatal("append with a blank description should return a tool error")
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
