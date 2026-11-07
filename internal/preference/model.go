package preference

import (
	"encoding/json"
	"strings"
	"time"
)

const (
	ScopeGlobal  = "global"
	ScopeProject = "project"
)

type Preference struct {
	UserID      string          `json:"user_id"`
	Scope       string          `json:"scope"`
	ScopeID     string          `json:"scope_id"`
	Key         string          `json:"key"`
	Value       json.RawMessage `json:"value"`
	Description string          `json:"description"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func IsValidScope(scope string) bool {
	return scope == ScopeGlobal || scope == ScopeProject
}

func TrimKey(key string) string {
	return strings.TrimSpace(key)
}
