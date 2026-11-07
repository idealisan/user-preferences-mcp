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
	Scope   string `json:"scope" jsonschema:"global or project"`
	ScopeID string `json:"scope_id,omitempty" jsonschema:"project ID; required when scope is project"`
}

type listPreferencesInput struct {
	UserID  string `json:"user_id,omitempty" jsonschema:"user ID; defaults to the server default user"`
	Scope   string `json:"scope,omitempty" jsonschema:"optional scope filter: global or project"`
	ScopeID string `json:"scope_id,omitempty" jsonschema:"project ID; required when scope is project"`
	Keyword string `json:"keyword,omitempty" jsonschema:"optional keyword; case-insensitive substring match on key, description and value"`
}

type setPreferenceInput struct {
	UserID      string `json:"user_id,omitempty" jsonschema:"user ID; defaults to the server default user"`
	Key         string `json:"key" jsonschema:"preference key, for example coding.language"`
	Value       any    `json:"value" jsonschema:"preference value; any valid JSON value"`
	Scope       string `json:"scope" jsonschema:"global or project"`
	ScopeID     string `json:"scope_id,omitempty" jsonschema:"project ID; required when scope is project"`
	Description string `json:"description,omitempty" jsonschema:"optional human-readable description"`
}

type deletePreferenceInput struct {
	UserID  string `json:"user_id,omitempty" jsonschema:"user ID; defaults to the server default user"`
	Key     string `json:"key" jsonschema:"preference key, for example coding.language"`
	Scope   string `json:"scope" jsonschema:"global or project"`
	ScopeID string `json:"scope_id,omitempty" jsonschema:"project ID; required when scope is project"`
}

type effectivePreferencesInput struct {
	UserID    string `json:"user_id,omitempty" jsonschema:"user ID; defaults to the server default user"`
	ProjectID string `json:"project_id,omitempty" jsonschema:"optional project ID; when set, project preferences override global preferences"`
	Keyword   string `json:"keyword,omitempty" jsonschema:"optional keyword; case-insensitive substring match on key, description and value"`
}

func registerTools(server *mcp.Server, svc *preference.Service, defaultUserID string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_preference",
		Description: "Get a single user preference by key and scope.",
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
		Description: "List user preferences, optionally filtered by scope and keyword.",
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
		Description: "Create or update a user preference.",
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
		Description: "Delete a user preference by key and scope.",
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
		Description: "Get merged effective preferences, with project scope overriding global scope.",
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
