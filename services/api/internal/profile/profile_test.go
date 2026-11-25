package profile

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubUsers struct{ exists bool }

func (s stubUsers) Exists(context.Context, string) (bool, error) { return s.exists, nil }

func TestGetOrCreate_AndUpdate(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemoryStore(), stubUsers{exists: true}, func() time.Time {
		return time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	})

	p, err := svc.GetOrCreate(ctx, "user-1", "")
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	if p.DisplayName == "" {
		t.Fatal("display name must default to non-empty")
	}
	name := "林曉彤"
	updated, err := svc.Update(ctx, "user-1", Patch{DisplayName: &name})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.DisplayName != "林曉彤" {
		t.Fatalf("display_name = %q", updated.DisplayName)
	}
	// public_id 唯一性。
	pub := "xiaotong"
	if _, err := svc.Update(ctx, "user-1", Patch{PublicID: &pub}); err != nil {
		t.Fatalf("set public_id: %v", err)
	}
	p2, _ := svc.GetOrCreate(ctx, "user-2", "User Two")
	if _, err := svc.Update(ctx, "user-2", Patch{PublicID: &pub}); err == nil {
		t.Fatal("want duplicate public_id error")
	}
	_ = p2
}

func TestGetOrCreate_UnknownUser(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemoryStore(), stubUsers{exists: false}, nil)
	if _, err := svc.GetOrCreate(ctx, "ghost", ""); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("err = %v, want ErrProfileNotFound", err)
	}
}

func TestValidatePatch(t *testing.T) {
	empty := "  "
	if fields := ValidatePatch(Patch{DisplayName: &empty}); len(fields) == 0 {
		t.Fatal("want error for blank display name")
	}
	bad := "Bad ID!"
	if fields := ValidatePatch(Patch{PublicID: &bad}); len(fields) == 0 {
		t.Fatal("want error for bad public_id")
	}
	long := string(make([]rune, 501))
	if fields := ValidatePatch(Patch{Bio: &long}); len(fields) == 0 {
		t.Fatal("want error for long bio")
	}
	ok := "ok_name-1"
	if fields := ValidatePatch(Patch{PublicID: &ok}); len(fields) != 0 {
		t.Fatalf("valid patch rejected: %+v", fields)
	}
}

func TestGetByUserID_AndGetByPublicID_ReadOnly(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemoryStore(), stubUsers{exists: true}, nil)

	// 唯讀方法不得建立檔案。
	if _, err := svc.GetByUserID(ctx, "user-9"); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("GetByUserID missing err = %v", err)
	}
	if _, err := svc.GetByPublicID(ctx, "nobody"); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("GetByPublicID missing err = %v", err)
	}
	if _, err := svc.GetByPublicID(ctx, ""); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("GetByPublicID empty err = %v", err)
	}

	name := "林曉彤"
	pub := "xiaotong"
	p, err := svc.GetOrCreate(ctx, "user-1", "")
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	if _, err := svc.Update(ctx, "user-1", Patch{DisplayName: &name, PublicID: &pub}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err := svc.GetByUserID(ctx, "user-1")
	if err != nil || got.UserID != p.UserID {
		t.Fatalf("GetByUserID = %+v, %v", got, err)
	}
	byPub, err := svc.GetByPublicID(ctx, "xiaotong")
	if err != nil || byPub.UserID != "user-1" {
		t.Fatalf("GetByPublicID = %+v, %v", byPub, err)
	}
}
