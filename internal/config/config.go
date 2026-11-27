package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultHost     = "127.0.0.1"
	DefaultPort     = 8765
	DefaultProfile  = "default"
	DefaultLogLevel = "info"
	DefaultDBName   = "preferences.db"
)

type Config struct {
	DBPath         string
	Host           string
	Port           int
	DefaultProfile string
	LogLevel       string
}

func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	return filepath.Join(home, DefaultDBName), nil
}

// Parse reads flags for the Streamable HTTP binary.
func Parse(args []string) (*Config, error) {
	fs := flag.NewFlagSet("userpref-mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	dbPath := fs.String("db", "", "SQLite database path (default: user home directory/preferences.db)")
	host := fs.String("host", DefaultHost, "HTTP listen address")
	port := fs.Int("port", DefaultPort, "HTTP listen port")
	profile := fs.String("profile", DefaultProfile, "default preference profile name")
	// Accepted as an alias so existing launch scripts keep working.
	fs.StringVar(profile, "user-id", DefaultProfile, "alias for --profile")
	logLevel := fs.String("log-level", DefaultLogLevel, "log level (debug, info, warn, error)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, unexpectedArgs(fs.Args())
	}
	if *port < 1 || *port > 65535 {
		return nil, fmt.Errorf("invalid port %d: must be between 1 and 65535", *port)
	}
	if err := validateCommon(*logLevel, *profile); err != nil {
		return nil, err
	}

	db, err := resolveDBPath(*dbPath)
	if err != nil {
		return nil, err
	}

	return &Config{
		DBPath:         db,
		Host:           *host,
		Port:           *port,
		DefaultProfile: strings.TrimSpace(*profile),
		LogLevel:       strings.ToLower(*logLevel),
	}, nil
}

// ParseStdio reads flags for the stdio binary. There is no host or port: the
// MCP client launches the process and talks to it over stdin/stdout.
func ParseStdio(args []string) (*Config, error) {
	fs := flag.NewFlagSet("userpref-mcp-stdio", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	dbPath := fs.String("db", "", "SQLite database path (default: user home directory/preferences.db)")
	profile := fs.String("profile", DefaultProfile, "default preference profile name")
	fs.StringVar(profile, "user-id", DefaultProfile, "alias for --profile")
	logLevel := fs.String("log-level", DefaultLogLevel, "log level (debug, info, warn, error)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, unexpectedArgs(fs.Args())
	}
	if err := validateCommon(*logLevel, *profile); err != nil {
		return nil, err
	}

	db, err := resolveDBPath(*dbPath)
	if err != nil {
		return nil, err
	}

	return &Config{
		DBPath:         db,
		DefaultProfile: strings.TrimSpace(*profile),
		LogLevel:       strings.ToLower(*logLevel),
	}, nil
}

func validateCommon(logLevel, profile string) error {
	if !validLogLevel(logLevel) {
		return fmt.Errorf("invalid log level %q: expected debug, info, warn or error", logLevel)
	}
	if strings.TrimSpace(profile) == "" {
		return fmt.Errorf("profile name must not be empty")
	}
	return nil
}

func unexpectedArgs(args []string) error {
	return fmt.Errorf("unexpected arguments: %s", strings.Join(args, " "))
}

func resolveDBPath(dbPath string) (string, error) {
	if dbPath != "" {
		return filepath.Clean(dbPath), nil
	}
	return DefaultDBPath()
}

func validLogLevel(level string) bool {
	switch strings.ToLower(level) {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}
