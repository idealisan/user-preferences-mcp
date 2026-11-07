package mcpserver

import (
	"github.com/idealisan/user-preferences-mcp/internal/preference"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// instructions is sent to clients during initialization. It is the only
// channel through which the server can tell an agent what this service is for
// and when to use it; tool descriptions alone do not establish behavior.
const instructions = `User Preference Service: one user, one preference profile, shared by every AI agent.

The preferences stored here are explicit user preferences. They take precedence over your own defaults and over project conventions.

Before starting a task, call get_effective_preferences (pass project_id when the current project has a stable ID) and follow the returned preferences. If the user changes a preference during the session, read the effective preferences again.

Scopes:
- "global" applies to every project.
- "project" applies to one project only and requires scope_id.
- For the same key, a project value overrides the global value. Merging is key-level: the project value replaces the whole global value, with no deep merge for objects or arrays.

Values are arbitrary JSON (string, number, boolean, array, object). Keys use lowercase dot notation, for example coding.language, workflow.run_tests_after_changes, filesystem.temporary_files.

Use get_preference for a single key, list_preferences to browse or search stored keys by keyword, set_preference to create or update a value (upsert), and delete_preference to remove a key.

Do not modify global preferences unless the user explicitly asks. Do not invent preferences that are not stored here.`

func New(svc *preference.Service, defaultUserID, version string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "userpref-mcp",
		Title:   "User Preferences MCP Server",
		Version: version,
	}, &mcp.ServerOptions{
		Instructions: instructions,
	})

	registerTools(server, svc, defaultUserID)
	registerResources(server, svc, defaultUserID)
	return server
}
