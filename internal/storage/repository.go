package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// Repository stores preference descriptions as plain text, grouped by profile.
// Ordering is significant: descriptions are always returned in insertion order.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ListProfiles returns the names of all profiles that currently store at least
// one description, in alphabetical order.
func (r *Repository) ListProfiles(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT profile FROM preference_descriptions ORDER BY profile`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := []string{}
	for rows.Next() {
		var profile string
		if err := rows.Scan(&profile); err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (r *Repository) List(ctx context.Context, profile string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT description FROM preference_descriptions
		WHERE profile = ? ORDER BY id`, profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []string{}
	for rows.Next() {
		var description string
		if err := rows.Scan(&description); err != nil {
			return nil, err
		}
		items = append(items, description)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// ReplaceAll deletes every description in the profile and inserts the given
// ones in order. The whole operation is a single transaction.
func (r *Repository) ReplaceAll(ctx context.Context, profile string, descriptions []string) error {
	return r.write(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM preference_descriptions WHERE profile = ?`, profile); err != nil {
			return err
		}
		return insertAll(ctx, tx, profile, descriptions)
	})
}

// Append adds descriptions to the end of the profile's existing list.
func (r *Repository) Append(ctx context.Context, profile string, descriptions []string) error {
	return r.write(ctx, func(tx *sql.Tx) error {
		return insertAll(ctx, tx, profile, descriptions)
	})
}

func (r *Repository) write(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func insertAll(ctx context.Context, tx *sql.Tx, profile string, descriptions []string) error {
	if len(descriptions) == 0 {
		return nil
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO preference_descriptions (profile, description) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, description := range descriptions {
		if _, err := stmt.ExecContext(ctx, profile, description); err != nil {
			return err
		}
	}
	return nil
}
