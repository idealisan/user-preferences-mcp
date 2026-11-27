package mcpserver

import (
	"github.com/idealisan/user-preferences-mcp/internal/preference"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// instructions is sent to clients during initialization. It is the only
// channel through which the server can tell an agent what this service is for
// and when to use it; tool descriptions alone do not establish behavior.
const instructions = `User Preference Service: one shared profile per user, accessible by every AI agent.

Preferences are plain-text descriptions of how the user wants you to work. They are explicit user instructions and take precedence over your own defaults and over project conventions.

Before starting any task, call get_all_preferences once and read the ENTIRE returned list. There is no search or filter: read everything, then follow every description. Do not try to guess which preferences are relevant and query only those. If the user mentions a different profile, call list_profiles first to discover available names; if more than one exists, confirm the choice with the user. Omitting profile always uses "default".

Write preferences only when the user explicitly asks. Use append_preferences to add items, and replace_all_preferences only when the user wants the whole list rewritten.

Never invent preferences that are not stored here. If a stored description is unclear, ask the user instead of guessing.`

func New(svc *preference.Service, defaultProfile, version string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "userpref-mcp",
		Title:   "User Preferences MCP Server",
		Version: version,
	}, &mcp.ServerOptions{
		Instructions: instructions,
	})

	registerTools(server, svc, defaultProfile)
	return server
}
