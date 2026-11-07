package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/idealisan/user-preferences-mcp/internal/preference"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context, userID, scope, scopeID, key string) (*preference.Preference, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT user_id, scope, scope_id, key, value, description, created_at, updated_at
		FROM preferences
		WHERE user_id = ? AND scope = ? AND scope_id = ? AND key = ?`,
		userID, scope, scopeID, key)

	p, err := scanPreference(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, preference.ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *Repository) List(ctx context.Context, userID, scope, scopeID, keyword string) ([]*preference.Preference, error) {
	query := `
		SELECT user_id, scope, scope_id, key, value, description, created_at, updated_at
		FROM preferences
		WHERE user_id = ?`
	args := []any{userID}

	if scope != "" {
		query += " AND scope = ? AND scope_id = ?"
		args = append(args, scope, scopeID)
	}
	if keyword != "" {
		pattern := "%" + strings.ToLower(escapeLike(keyword)) + "%"
		query += ` AND (
			LOWER(key) LIKE ? ESCAPE '\'
			OR LOWER(description) LIKE ? ESCAPE '\'
			OR LOWER(value) LIKE ? ESCAPE '\'
		)`
		args = append(args, pattern, pattern, pattern)
	}
	query += " ORDER BY scope, scope_id, key"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []*preference.Preference{}
	for rows.Next() {
		p, err := scanPreference(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) Upsert(ctx context.Context, p *preference.Preference) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO preferences (user_id, scope, scope_id, key, value, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, scope, scope_id, key) DO UPDATE SET
			value = excluded.value,
			description = excluded.description,
			updated_at = CURRENT_TIMESTAMP`,
		p.UserID, p.Scope, p.ScopeID, p.Key, string(p.Value), p.Description)
	return err
}

func (r *Repository) Delete(ctx context.Context, userID, scope, scopeID, key string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM preferences
		WHERE user_id = ? AND scope = ? AND scope_id = ? AND key = ?`,
		userID, scope, scopeID, key)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPreference(s scanner) (*preference.Preference, error) {
	var (
		p     preference.Preference
		value string
	)
	if err := s.Scan(
		&p.UserID, &p.Scope, &p.ScopeID, &p.Key, &value, &p.Description, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if !json.Valid([]byte(value)) {
		return nil, fmt.Errorf("stored value for key %q is not valid JSON", p.Key)
	}
	p.Value = json.RawMessage(value)
	return &p, nil
}

func escapeLike(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}
