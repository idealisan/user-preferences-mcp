package main

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestStdioServer verifies that the stdio binary exposes exactly the same
// tools and behaviour as the HTTP binary. It drives the real compiled binary
// through CommandTransport, which is how MCP clients launch stdio servers.
func TestStdioServer(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "userpref-mcp-stdio")
	build := exec.Command("go", "build", "-o", bin, "./cmd/userpref-mcp-stdio")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build stdio binary: %v\n%s", err, out)
	}

	ctx := context.Background()
	cmd := exec.Command(bin, "--db", filepath.Join(t.TempDir(), "preferences.db"))
	client := mcp.NewClient(&mcp.Implementation{Name: "stdio-test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	init := session.InitializeResult()
	if init == nil || !strings.Contains(init.Instructions, "get_all_preferences") {
		t.Fatalf("instructions = %+v", init)
	}

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	want := map[string]bool{
		"list_profiles":           true,
		"get_all_preferences":     true,
		"replace_all_preferences": true,
		"append_preferences":      true,
	}
	if len(tools.Tools) != len(want) {
		t.Fatalf("got %d tools, want %d", len(tools.Tools), len(want))
	}
	for _, tool := range tools.Tools {
		if !want[tool.Name] {
			t.Fatalf("unexpected tool %q", tool.Name)
		}
	}

	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("call %s: %v", name, err)
		}
		if res.IsError || len(res.Content) == 0 {
			t.Fatalf("call %s failed: %+v", name, res)
		}
		text, ok := res.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("call %s: unexpected content %T", name, res.Content[0])
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(text.Text), &out); err != nil {
			t.Fatalf("decode %q: %v", text.Text, err)
		}
		return out
	}

	if out := call("get_all_preferences", nil); out["count"] != float64(0) {
		t.Fatalf("empty get = %v", out)
	}
	call("replace_all_preferences", map[string]any{
		"preferences": []string{"用中文交流", "改完代码跑测试"},
	})
	out := call("get_all_preferences", nil)
	items := out["preferences"].([]any)
	if out["count"] != float64(2) || items[0] != "用中文交流" {
		t.Fatalf("get after replace = %v", out)
	}
	out = call("append_preferences", map[string]any{"preferences": []string{"新增一条"}})
	if out["count"] != float64(3) || out["added"] != float64(1) {
		t.Fatalf("append = %v", out)
	}
	if out := call("list_profiles", nil); out["count"] != float64(1) || out["profiles"].([]any)[0] != "default" {
		t.Fatalf("list profiles = %v", out)
	}
}
