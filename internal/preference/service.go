package preference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Repository interface {
	Get(ctx context.Context, userID, scope, scopeID, key string) (*Preference, error)
	List(ctx context.Context, userID, scope, scopeID, keyword string) ([]*Preference, error)
	Upsert(ctx context.Context, p *Preference) error
	Delete(ctx context.Context, userID, scope, scopeID, key string) (bool, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, userID, key, scope, scopeID string) (*Preference, error) {
	userID, key, scope, scopeID, err := normalize(userID, key, scope, scopeID)
	if err != nil {
		return nil, err
	}

	p, err := s.repo.Get(ctx, userID, scope, scopeID, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, &CodedError{Code: "NOT_FOUND", Message: fmt.Sprintf("preference %q not found", key)}
		}
		return nil, DatabaseError(err)
	}
	return p, nil
}

func (s *Service) List(ctx context.Context, userID, scope, scopeID, keyword string) ([]*Preference, error) {
	userID = defaultUserID(userID)
	keyword = strings.TrimSpace(keyword)

	if scope != "" {
		if !IsValidScope(scope) {
			return nil, InvalidScope("scope must be global or project")
		}
		if scope == ScopeGlobal {
			scopeID = ""
		} else if strings.TrimSpace(scopeID) == "" {
			return nil, InvalidScope("scope_id is required for project scope")
		}
	}

	var (
		items []*Preference
		err   error
	)
	if scope == "" {
		items, err = s.repo.List(ctx, userID, "", "", keyword)
	} else {
		items, err = s.repo.List(ctx, userID, scope, scopeID, keyword)
	}
	if err != nil {
		return nil, DatabaseError(err)
	}
	return items, nil
}

func (s *Service) Set(ctx context.Context, p *Preference) error {
	userID, key, scope, scopeID, err := normalize(p.UserID, p.Key, p.Scope, p.ScopeID)
	if err != nil {
		return err
	}
	if err := validateValue(p.Value); err != nil {
		return err
	}

	p.UserID = userID
	p.Key = key
	p.Scope = scope
	p.ScopeID = scopeID
	p.Description = strings.TrimSpace(p.Description)

	if err := s.repo.Upsert(ctx, p); err != nil {
		return DatabaseError(err)
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, userID, key, scope, scopeID string) (bool, error) {
	userID, key, scope, scopeID, err := normalize(userID, key, scope, scopeID)
	if err != nil {
		return false, err
	}

	deleted, err := s.repo.Delete(ctx, userID, scope, scopeID, key)
	if err != nil {
		return false, DatabaseError(err)
	}
	return deleted, nil
}

func (s *Service) Effective(ctx context.Context, userID, projectID, keyword string) (map[string]any, map[string]string, error) {
	userID = defaultUserID(userID)
	projectID = strings.TrimSpace(projectID)
	keyword = strings.TrimSpace(keyword)

	globals, err := s.repo.List(ctx, userID, ScopeGlobal, "", keyword)
	if err != nil {
		return nil, nil, DatabaseError(err)
	}

	preferences := make(map[string]any, len(globals))
	sources := make(map[string]string, len(globals))
	apply := func(items []*Preference, source string) {
		for _, p := range items {
			var value any
			if err := json.Unmarshal(p.Value, &value); err != nil {
				continue
			}
			preferences[p.Key] = value
			sources[p.Key] = source
		}
	}
	apply(globals, ScopeGlobal)

	if projectID != "" {
		projects, err := s.repo.List(ctx, userID, ScopeProject, projectID, keyword)
		if err != nil {
			return nil, nil, DatabaseError(err)
		}
		apply(projects, ScopeProject)
	}

	return preferences, sources, nil
}

func normalize(userID, key, scope, scopeID string) (string, string, string, string, error) {
	userID = defaultUserID(userID)
	key = TrimKey(key)
	scope = strings.TrimSpace(scope)
	scopeID = strings.TrimSpace(scopeID)

	if key == "" {
		return "", "", "", "", InvalidArgument("key must not be empty")
	}
	if !IsValidScope(scope) {
		return "", "", "", "", InvalidScope("scope must be global or project")
	}
	if scope == ScopeGlobal {
		scopeID = ""
	} else if scopeID == "" {
		return "", "", "", "", InvalidScope("scope_id is required for project scope")
	}

	return userID, key, scope, scopeID, nil
}

func defaultUserID(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "default"
	}
	return userID
}

func validateValue(value json.RawMessage) error {
	if len(value) == 0 {
		return InvalidJSON("value must be valid JSON")
	}
	if !json.Valid(value) {
		return InvalidJSON("value must be valid JSON")
	}
	return nil
}
