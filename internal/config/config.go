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
	DefaultUserID   = "default"
	DefaultLogLevel = "info"
	DefaultDBName   = "preferences.db"
)

type Config struct {
	DBPath   string
	Host     string
	Port     int
	UserID   string
	LogLevel string
}

func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	return filepath.Join(home, DefaultDBName), nil
}

func Parse(args []string) (*Config, error) {
	fs := flag.NewFlagSet("userpref-mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	dbPath := fs.String("db", "", "SQLite database path (default: user home directory/preferences.db)")
	host := fs.String("host", DefaultHost, "HTTP listen address")
	port := fs.Int("port", DefaultPort, "HTTP listen port")
	userID := fs.String("user-id", DefaultUserID, "default user ID")
	logLevel := fs.String("log-level", DefaultLogLevel, "log level (debug, info, warn, error)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if *port < 1 || *port > 65535 {
		return nil, fmt.Errorf("invalid port %d: must be between 1 and 65535", *port)
	}
	if !validLogLevel(*logLevel) {
		return nil, fmt.Errorf("invalid log level %q: expected debug, info, warn or error", *logLevel)
	}

	cfg := &Config{
		Host:     *host,
		Port:     *port,
		UserID:   *userID,
		LogLevel: strings.ToLower(*logLevel),
	}

	if *dbPath == "" {
		path, err := DefaultDBPath()
		if err != nil {
			return nil, err
		}
		cfg.DBPath = path
	} else {
		cfg.DBPath = filepath.Clean(*dbPath)
	}

	return cfg, nil
}

func validLogLevel(level string) bool {
	switch strings.ToLower(level) {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}
