package preference

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

type fakeRepository struct {
	data map[string][]string
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{data: map[string][]string{}}
}

func (f *fakeRepository) ListProfiles(_ context.Context) ([]string, error) {
	out := []string{}
	for _, profile := range sortedKeys(f.data) {
		if len(f.data[profile]) > 0 {
			out = append(out, profile)
		}
	}
	return out, nil
}

func (f *fakeRepository) List(_ context.Context, profile string) ([]string, error) {
	return append([]string{}, f.data[profile]...), nil
}

func (f *fakeRepository) ReplaceAll(_ context.Context, profile string, descriptions []string) error {
	f.data[profile] = append([]string{}, descriptions...)
	return nil
}

func (f *fakeRepository) Append(_ context.Context, profile string, descriptions []string) error {
	f.data[profile] = append(f.data[profile], descriptions...)
	return nil
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func TestService(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepository()
	svc := NewService(repo)

	if err := svc.ReplaceAll(ctx, "", []string{" 第一条 ", "第二条"}); err != nil {
		t.Fatalf("replace: %v", err)
	}
	got, err := svc.GetAll(ctx, "")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if want := []string{"第一条", "第二条"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("get = %v, want %v", got, want)
	}
	if _, ok := repo.data[DefaultProfile]; !ok {
		t.Fatalf("empty profile should map to %q", DefaultProfile)
	}

	if err := svc.Append(ctx, "  ", []string{"第三条"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	got, _ = svc.GetAll(ctx, DefaultProfile)
	if want := []string{"第一条", "第二条", "第三条"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("get after append = %v, want %v", got, want)
	}

	profiles, err := svc.ListProfiles(ctx)
	if err != nil || !reflect.DeepEqual(profiles, []string{DefaultProfile}) {
		t.Fatalf("list profiles = %v, %v", profiles, err)
	}
}

func TestServiceRejectsBlankDescriptions(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newFakeRepository())

	for _, tc := range []struct {
		name          string
		descriptions  []string
		wantErrorCode string
	}{
		{name: "blank", descriptions: []string{"ok", ""}, wantErrorCode: "INVALID_ARGUMENT"},
		{name: "whitespace", descriptions: []string{"   "}, wantErrorCode: "INVALID_ARGUMENT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, write := range []func(context.Context, string, []string) error{
				svc.ReplaceAll, svc.Append,
			} {
				err := write(ctx, "", tc.descriptions)
				if err == nil || !strings.Contains(err.Error(), tc.wantErrorCode) {
					t.Fatalf("write error = %v, want %s", err, tc.wantErrorCode)
				}
			}
		})
	}
}
