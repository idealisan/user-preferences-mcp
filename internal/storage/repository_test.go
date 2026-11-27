package storage

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRepository(t *testing.T) {
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "preferences.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	repo := NewRepository(db)

	if err := repo.ReplaceAll(ctx, "default", []string{"a", "b"}); err != nil {
		t.Fatalf("replace: %v", err)
	}
	got, err := repo.List(ctx, "default")
	if err != nil || !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("list = %v, %v", got, err)
	}

	if err := repo.ReplaceAll(ctx, "default", []string{"c"}); err != nil {
		t.Fatalf("replace again: %v", err)
	}
	got, _ = repo.List(ctx, "default")
	if !reflect.DeepEqual(got, []string{"c"}) {
		t.Fatalf("list after replace = %v", got)
	}

	if err := repo.Append(ctx, "default", []string{"d", "e"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	got, _ = repo.List(ctx, "default")
	if !reflect.DeepEqual(got, []string{"c", "d", "e"}) {
		t.Fatalf("list after append = %v", got)
	}

	if err := repo.ReplaceAll(ctx, "work", []string{"w"}); err != nil {
		t.Fatalf("replace work: %v", err)
	}
	got, _ = repo.List(ctx, "work")
	if !reflect.DeepEqual(got, []string{"w"}) {
		t.Fatalf("work list = %v", got)
	}
	got, _ = repo.List(ctx, "default")
	if !reflect.DeepEqual(got, []string{"c", "d", "e"}) {
		t.Fatalf("default list changed: %v", got)
	}

	profiles, err := repo.ListProfiles(ctx)
	if err != nil || !reflect.DeepEqual(profiles, []string{"default", "work"}) {
		t.Fatalf("list profiles = %v, %v", profiles, err)
	}

	if err := repo.ReplaceAll(ctx, "default", nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	got, _ = repo.List(ctx, "default")
	if len(got) != 0 {
		t.Fatalf("list after clear = %v", got)
	}
	profiles, _ = repo.ListProfiles(ctx)
	if !reflect.DeepEqual(profiles, []string{"work"}) {
		t.Fatalf("profiles after clear = %v", profiles)
	}
}

func TestRepositoryPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.db")
	ctx := context.Background()

	db, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := NewRepository(db).Append(ctx, "default", []string{"persisted"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	db.Close()

	db, err = Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db.Close()
	got, err := NewRepository(db).List(ctx, "default")
	if err != nil || !reflect.DeepEqual(got, []string{"persisted"}) {
		t.Fatalf("list after reopen = %v, %v", got, err)
	}
}
