package storage

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if info, err := os.Stat(filepath.Dir(abs)); err != nil {
		return nil, fmt.Errorf("database directory %q is not accessible: %w", filepath.Dir(abs), err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("database directory %q is not a directory", filepath.Dir(abs))
	}

	db, err := sql.Open("sqlite", buildDSN(abs))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize database: %w", err)
	}
	return db, nil
}

func buildDSN(absPath string) string {
	p := filepath.ToSlash(absPath)
	if runtime.GOOS == "windows" && !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	u := url.URL{Scheme: "file", Path: p}
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(NORMAL)")
	u.RawQuery = q.Encode()
	return u.String()
}

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS preferences (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL DEFAULT 'default',
		scope TEXT NOT NULL DEFAULT 'global',
		scope_id TEXT NOT NULL DEFAULT '',
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		CHECK (scope IN ('global', 'project')),
		CHECK (
			(scope = 'global' AND scope_id = '')
			OR
			(scope = 'project' AND scope_id != '')
		),
		UNIQUE(user_id, scope, scope_id, key)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_preferences_lookup ON preferences(user_id, scope, scope_id)`,
	`CREATE INDEX IF NOT EXISTS idx_preferences_key ON preferences(user_id, key)`,
}

func migrate(db *sql.DB) error {
	for _, stmt := range migrations {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
