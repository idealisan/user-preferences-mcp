package mcpserver

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/idealisan/user-preferences-mcp/internal/preference"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerResources(server *mcp.Server, svc *preference.Service, defaultUserID string) {
	handler := func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri := req.Params.URI
		u, err := url.Parse(uri)
		if err != nil || u.Scheme != "preferences" || u.Host != "global" {
			return nil, mcp.ResourceNotFoundError(uri)
		}

		keyword := strings.Trim(strings.TrimPrefix(u.Path, "/"), "/")
		items, err := svc.List(ctx, defaultUserID, preference.ScopeGlobal, "", keyword)
		if err != nil {
			return nil, err
		}

		preferences := make(map[string]any, len(items))
		for _, p := range items {
			preferences[p.Key] = rawValue(p.Value)
		}
		data, err := json.Marshal(map[string]any{"preferences": preferences})
		if err != nil {
			return nil, err
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(data),
			}},
		}, nil
	}

	server.AddResource(&mcp.Resource{
		URI:         "preferences://global",
		Name:        "global-preferences",
		Title:       "Global user preferences",
		Description: "All global user preferences as {preferences: {key: value}}. Read-only. Project overrides are not applied; use get_effective_preferences for the merged values in effect.",
		MIMEType:    "application/json",
	}, handler)

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "preferences://global/{keyword}",
		Name:        "global-preferences-keyword",
		Title:       "Global user preferences by keyword",
		Description: "Global user preferences filtered by the URI path segment as a case-insensitive substring over key, description and value text. Returns {preferences: {key: value}}. Read-only.",
		MIMEType:    "application/json",
	}, handler)
}
