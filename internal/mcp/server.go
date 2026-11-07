package mcpserver

import (
	"github.com/idealisan/user-preferences-mcp/internal/preference"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func New(svc *preference.Service, defaultUserID, version string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "userpref-mcp",
		Title:   "User Preferences MCP Server",
		Version: version,
	}, nil)

	registerTools(server, svc, defaultUserID)
	registerResources(server, svc, defaultUserID)
	return server
}
