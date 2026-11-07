package mcpserver

import (
	"context"
	"encoding/json"

	"github.com/idealisan/user-preferences-mcp/internal/preference"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type getPreferenceInput struct {
	UserID  string `json:"user_id,omitempty" jsonschema:"user ID; defaults to the server default user"`
	Key     string `json:"key" jsonschema:"preference key, for example coding.language"`
	Scope   string `json:"scope" jsonschema:"global or project; project requires scope_id"`
	ScopeID string `json:"scope_id,omitempty" jsonschema:"project ID; required when scope is project"`
}

type listPreferencesInput struct {
	UserID  string `json:"user_id,omitempty" jsonschema:"user ID; defaults to the server default user"`
	Scope   string `json:"scope,omitempty" jsonschema:"optional scope filter: global or project"`
	ScopeID string `json:"scope_id,omitempty" jsonschema:"project ID; required when scope is project"`
	Keyword string `json:"keyword,omitempty" jsonschema:"optional case-insensitive substring match on key, description and value text"`
}

type setPreferenceInput struct {
	UserID      string `json:"user_id,omitempty" jsonschema:"user ID; defaults to the server default user"`
	Key         string `json:"key" jsonschema:"preference key using lowercase dot notation, for example coding.language"`
	Value       any    `json:"value" jsonschema:"any valid JSON value; on update the whole previous value is replaced, with no deep merge"`
	Scope       string `json:"scope" jsonschema:"global for every project, or project (requires scope_id) for one project"`
	ScopeID     string `json:"scope_id,omitempty" jsonschema:"project ID; required when scope is project"`
	Description string `json:"description,omitempty" jsonschema:"optional human-readable description shown to agents"`
}

type deletePreferenceInput struct {
	UserID  string `json:"user_id,omitempty" jsonschema:"user ID; defaults to the server default user"`
	Key     string `json:"key" jsonschema:"preference key, for example coding.language"`
	Scope   string `json:"scope" jsonschema:"global or project; project requires scope_id"`
	ScopeID string `json:"scope_id,omitempty" jsonschema:"project ID; required when scope is project"`
}

type effectivePreferencesInput struct {
	UserID    string `json:"user_id,omitempty" jsonschema:"user ID; defaults to the server default user"`
	ProjectID string `json:"project_id,omitempty" jsonschema:"optional project ID; when set, project preferences override global preferences per key"`
	Keyword   string `json:"keyword,omitempty" jsonschema:"optional case-insensitive substring match on key, description and value text"`
}

func boolPtr(v bool) *bool { return &v }

func readOnlyAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:         title,
		ReadOnlyHint:  true,
		OpenWorldHint: boolPtr(false),
	}
}

func writeAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		DestructiveHint: boolPtr(true),
		IdempotentHint:  true,
		OpenWorldHint:   boolPtr(false),
	}
}

func registerTools(server *mcp.Server, svc *preference.Service, defaultUserID string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_preference",
		Title:       "Get Preference",
		Description: "Read one stored user preference by key and scope. Returns {key, value, description}, or a not-found error if the key is absent. Read-only, and it does not merge scopes: use get_effective_preferences when you need the value actually in effect.",
		Annotations: readOnlyAnnotations("Get Preference"),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in getPreferenceInput) (*mcp.CallToolResult, any, error) {
		p, err := svc.Get(ctx, userOr(in.UserID, defaultUserID), in.Key, in.Scope, in.ScopeID)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"key":         p.Key,
			"value":       rawValue(p.Value),
			"description": p.Description,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_preferences",
		Title:       "List Preferences",
		Description: "List stored user preferences, optionally filtered by scope and keyword. The keyword is a case-insensitive substring match over key, description and JSON value text. Returns {preferences: [...], count}. Read-only. Use this to discover which keys exist; use get_effective_preferences for the merged view that is in effect.",
		Annotations: readOnlyAnnotations("List Preferences"),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listPreferencesInput) (*mcp.CallToolResult, any, error) {
		items, err := svc.List(ctx, userOr(in.UserID, defaultUserID), in.Scope, in.ScopeID, in.Keyword)
		if err != nil {
			return nil, nil, err
		}

		out := make([]map[string]any, 0, len(items))
		for _, p := range items {
			out = append(out, map[string]any{
				"key":         p.Key,
				"value":       rawValue(p.Value),
				"scope":       p.Scope,
				"scope_id":    p.ScopeID,
				"description": p.Description,
			})
		}
		return nil, map[string]any{
			"preferences": out,
			"count":       len(out),
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_preference",
		Title:       "Set Preference",
		Description: "Create or update (upsert) a user preference. value may be any JSON value (string, number, boolean, array or object). Calling again with the same key and scope replaces the entire previous value and description, with no deep merge. Use scope \"global\" for every project, or \"project\" plus scope_id for one project. Do not change global preferences unless the user explicitly asks.",
		Annotations: writeAnnotations("Set Preference"),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in setPreferenceInput) (*mcp.CallToolResult, any, error) {
		value, err := json.Marshal(in.Value)
		if err != nil {
			return nil, nil, preference.InvalidJSON("value cannot be encoded as JSON: %v", err)
		}
		p := &preference.Preference{
			UserID:      userOr(in.UserID, defaultUserID),
			Key:         in.Key,
			Value:       value,
			Scope:       in.Scope,
			ScopeID:     in.ScopeID,
			Description: in.Description,
		}
		if err := svc.Set(ctx, p); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"success": true,
			"key":     p.Key,
			"scope":   p.Scope,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_preference",
		Title:       "Delete Preference",
		Description: "Delete a user preference by key and scope. Idempotent: it succeeds with deleted=false when no matching record exists. Project scope requires scope_id. Deletion is irreversible, so confirm intent before removing global preferences.",
		Annotations: writeAnnotations("Delete Preference"),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in deletePreferenceInput) (*mcp.CallToolResult, any, error) {
		deleted, err := svc.Delete(ctx, userOr(in.UserID, defaultUserID), in.Key, in.Scope, in.ScopeID)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"success": true,
			"deleted": deleted,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_effective_preferences",
		Title:       "Get Effective Preferences",
		Description: "Get the user preferences actually in effect, after merging global and project scope (project overrides global per key). Call this at the start of a task to load the user's working context. Returns {preferences: {key: value}, sources: {key: \"global\" | \"project\"}}. Optional keyword filters the result; project_id enables project-level overrides. Read-only.",
		Annotations: readOnlyAnnotations("Get Effective Preferences"),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in effectivePreferencesInput) (*mcp.CallToolResult, any, error) {
		preferences, sources, err := svc.Effective(ctx, userOr(in.UserID, defaultUserID), in.ProjectID, in.Keyword)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"preferences": preferences,
			"sources":     sources,
		}, nil
	})
}

func rawValue(raw json.RawMessage) any {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return value
}

func userOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
