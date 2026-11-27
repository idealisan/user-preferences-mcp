// Command userpref-mcp-stdio serves the same user preference tools as
// userpref-mcp, but over MCP stdio instead of Streamable HTTP. The MCP client
// launches it as a child process, so there is no background server to manage.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/idealisan/user-preferences-mcp/internal/config"
	mcpserver "github.com/idealisan/user-preferences-mcp/internal/mcp"
	"github.com/idealisan/user-preferences-mcp/internal/preference"
	"github.com/idealisan/user-preferences-mcp/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "userpref-mcp-stdio: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.ParseStdio(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// stdout belongs exclusively to the MCP protocol; all logs go to stderr.
	logger := newLogger(cfg.LogLevel)
	logger.Info("starting userpref-mcp-stdio",
		"version", version,
		"db", cfg.DBPath,
		"default_profile", cfg.DefaultProfile,
	)

	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	svc := preference.NewService(storage.NewRepository(db))
	server := mcpserver.New(svc, cfg.DefaultProfile, version)

	// Run blocks until the client closes stdin or otherwise disconnects.
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

func newLogger(level string) *slog.Logger {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: l}))
}
