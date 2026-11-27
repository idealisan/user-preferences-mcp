package preference

import (
	"context"
	"strings"
)

// DefaultProfile is used whenever a caller omits the optional profile argument.
const DefaultProfile = "default"

type Repository interface {
	ListProfiles(ctx context.Context) ([]string, error)
	List(ctx context.Context, profile string) ([]string, error)
	ReplaceAll(ctx context.Context, profile string, descriptions []string) error
	Append(ctx context.Context, profile string, descriptions []string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListProfiles(ctx context.Context) ([]string, error) {
	profiles, err := s.repo.ListProfiles(ctx)
	if err != nil {
		return nil, DatabaseError(err)
	}
	return profiles, nil
}

func (s *Service) GetAll(ctx context.Context, profile string) ([]string, error) {
	items, err := s.repo.List(ctx, normalizeProfile(profile))
	if err != nil {
		return nil, DatabaseError(err)
	}
	return items, nil
}

func (s *Service) ReplaceAll(ctx context.Context, profile string, descriptions []string) error {
	items, err := normalizeDescriptions(descriptions)
	if err != nil {
		return err
	}
	if err := s.repo.ReplaceAll(ctx, normalizeProfile(profile), items); err != nil {
		return DatabaseError(err)
	}
	return nil
}

func (s *Service) Append(ctx context.Context, profile string, descriptions []string) error {
	items, err := normalizeDescriptions(descriptions)
	if err != nil {
		return err
	}
	if err := s.repo.Append(ctx, normalizeProfile(profile), items); err != nil {
		return DatabaseError(err)
	}
	return nil
}

func normalizeProfile(profile string) string {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return DefaultProfile
	}
	return profile
}

// normalizeDescriptions trims every description and rejects blank entries so
// the stored list never contains invisible or meaningless items. An empty list
// is valid: replacing with it clears the profile, appending with it is a no-op.
func normalizeDescriptions(descriptions []string) ([]string, error) {
	items := make([]string, 0, len(descriptions))
	for i, description := range descriptions {
		description = strings.TrimSpace(description)
		if description == "" {
			return nil, InvalidArgument("preferences[%d] must not be empty", i)
		}
		items = append(items, description)
	}
	return items, nil
}
