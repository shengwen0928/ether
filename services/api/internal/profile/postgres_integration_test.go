//go:build integration

package profile

import (
	"context"
	"errors"
	"testing"
	"time"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
	"ether.local/api/internal/user"
)

func TestPG_ProfilePersists(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)
	q := postgres.WrapDB(db)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	users := user.NewService(user.NewPGUserStore(q), user.NewPGSettingsStore(q), func() time.Time { return now })
	u, err := users.Provision(ctx, "kc-subject-1", "a@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	svc := NewService(NewPGStore(q), users, func() time.Time { return now })

	p, err := svc.GetOrCreate(ctx, u.ID, "阿哲")
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	if p.DisplayName != "阿哲" || p.UserID != u.ID {
		t.Fatalf("profile = %+v", p)
	}

	name := "阿哲哲"
	pid := "azhe"
	updated, err := svc.Update(ctx, u.ID, Patch{DisplayName: &name, PublicID: &pid})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.DisplayName != "阿哲哲" || updated.PublicID != "azhe" {
		t.Fatalf("updated = %+v", updated)
	}

	// 第二人佔用同一 public_id → ValidationError。
	u2, err := users.Provision(ctx, "kc-subject-2", "b@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision 2: %v", err)
	}
	if _, err := svc.GetOrCreate(ctx, u2.ID, "小明"); err != nil {
		t.Fatalf("GetOrCreate 2: %v", err)
	}
	if _, err := svc.Update(ctx, u2.ID, Patch{PublicID: &pid}); err == nil {
		t.Fatal("duplicate public_id must fail")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) || ve.Fields[0].Field != "public_id" {
			t.Fatalf("duplicate public_id err = %v, want ValidationError public_id", err)
		}
	}

	// 以 public_id 查找。
	found, err := NewPGStore(q).GetByPublicID(ctx, "azhe")
	if err != nil {
		t.Fatalf("GetByPublicID: %v", err)
	}
	if found.UserID != u.ID {
		t.Fatalf("found user = %s, want %s", found.UserID, u.ID)
	}
}
