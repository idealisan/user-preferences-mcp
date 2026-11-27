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

// The table is intentionally named preference_descriptions rather than reusing
// the old "preferences" table, so databases created by earlier versions keep
// their data untouched. Old data is not migrated into the new format.
var migrations = []string{
	`CREATE TABLE IF NOT EXISTS preference_descriptions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		profile TEXT NOT NULL DEFAULT 'default',
		description TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_preference_descriptions_profile
		ON preference_descriptions(profile, id)`,
}

func migrate(db *sql.DB) error {
	for _, stmt := range migrations {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
