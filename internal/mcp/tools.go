package mcpserver

import (
	"context"
	"strings"

	"github.com/idealisan/user-preferences-mcp/internal/preference"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listProfilesInput struct{}

type getPreferencesInput struct {
	Profile string `json:"profile,omitempty" jsonschema:"optional profile name; defaults to 'default'. Use list_profiles to discover available profiles."`
}

type writePreferencesInput struct {
	Profile     string   `json:"profile,omitempty" jsonschema:"optional profile name; defaults to 'default'. Use list_profiles to discover available profiles."`
	Preferences []string `json:"preferences" jsonschema:"plain-text preference descriptions; each string is one complete preference"`
}

func boolPtr(v bool) *bool { return &v }

func readOnlyAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:         title,
		ReadOnlyHint:  true,
		OpenWorldHint: boolPtr(false),
	}
}

func writeAnnotations(title string, destructive bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		DestructiveHint: boolPtr(destructive),
		IdempotentHint:  destructive,
		OpenWorldHint:   boolPtr(false),
	}
}

func registerTools(server *mcp.Server, svc *preference.Service, defaultProfile string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_profiles",
		Title:       "List Preference Profiles",
		Description: "List all existing preference profile names. Most users only have the 'default' profile; others may keep separate profiles for different contexts. Call this first if you are unsure which profile to use, and confirm with the user when more than one exists. Returns {profiles: [name, ...], count}. Takes no arguments. Read-only.",
		Annotations: readOnlyAnnotations("List Preference Profiles"),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ listProfilesInput) (*mcp.CallToolResult, any, error) {
		profiles, err := svc.ListProfiles(ctx)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"profiles": profiles,
			"count":    len(profiles),
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_all_preferences",
		Title:       "Get All Preferences",
		Description: "Read every stored preference description for a profile in one call. There is no keyword, filter, pagination or partial selection: the result is the complete list, so you cannot miss a preference. Call this at the start of a task and follow every returned description. Omit profile to use 'default'. Returns {profile, preferences: [description, ...], count}. Read-only.",
		Annotations: readOnlyAnnotations("Get All Preferences"),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in getPreferencesInput) (*mcp.CallToolResult, any, error) {
		profile := profileOr(in.Profile, defaultProfile)
		items, err := svc.GetAll(ctx, profile)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"profile":     profile,
			"preferences": items,
			"count":       len(items),
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "replace_all_preferences",
		Title:       "Replace All Preferences",
		Description: "Overwrite a profile's preferences with the complete supplied list. Existing descriptions are all removed and replaced by the new ones, preserving the order given. Pass an empty list to clear the profile. This is a full replacement, so include every preference you want to keep. Omit profile to use 'default'. Returns {profile, success, count}.",
		Annotations: writeAnnotations("Replace All Preferences", true),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in writePreferencesInput) (*mcp.CallToolResult, any, error) {
		profile := profileOr(in.Profile, defaultProfile)
		if err := svc.ReplaceAll(ctx, profile, in.Preferences); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"profile": profile,
			"success": true,
			"count":   len(in.Preferences),
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "append_preferences",
		Title:       "Append Preferences",
		Description: "Add one or more preference descriptions to the end of a profile's existing list without changing what is already stored. Omit profile to use 'default'. Returns {profile, success, count, added} where count is the total number of preferences after appending. Prefer this over replace_all_preferences when adding a single preference.",
		Annotations: writeAnnotations("Append Preferences", false),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in writePreferencesInput) (*mcp.CallToolResult, any, error) {
		profile := profileOr(in.Profile, defaultProfile)
		if err := svc.Append(ctx, profile, in.Preferences); err != nil {
			return nil, nil, err
		}
		items, err := svc.GetAll(ctx, profile)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"profile": profile,
			"success": true,
			"count":   len(items),
			"added":   len(in.Preferences),
		}, nil
	})
}

func profileOr(profile, fallback string) string {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return fallback
	}
	return profile
}
