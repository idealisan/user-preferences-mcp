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

		category := strings.Trim(strings.TrimPrefix(u.Path, "/"), "/")
		items, err := svc.List(ctx, defaultUserID, preference.ScopeGlobal, "", category)
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
		Description: "All global user preferences as a JSON object.",
		MIMEType:    "application/json",
	}, handler)

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "preferences://global/{category}",
		Name:        "global-preferences-category",
		Title:       "Global user preferences by category",
		Description: "Global user preferences filtered by key prefix category.",
		MIMEType:    "application/json",
	}, handler)
}
