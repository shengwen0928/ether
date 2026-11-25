//go:build integration

package device

import (
	"context"
	"errors"
	"testing"
	"time"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
	"ether.local/api/internal/user"
)

func TestPG_DeviceLifecycle(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)
	q := postgres.WrapDB(db)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	users := user.NewService(user.NewPGUserStore(q), user.NewPGSettingsStore(q), func() time.Time { return now })
	u, err := users.Provision(ctx, "kc-subject-1", "a@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	u2, err := users.Provision(ctx, "kc-subject-2", "b@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision 2: %v", err)
	}
	svc := NewService(NewPGStore(q), users, func() time.Time { return now })

	hash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	d, err := svc.Register(ctx, u.ID, "我的手機", "android", hash)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if d.DeviceIdentifierHash != hash || d.Revoked {
		t.Fatalf("device = %+v", d)
	}

	list, err := svc.List(ctx, u.ID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != d.ID {
		t.Fatalf("list = %v", list)
	}

	// 非擁有者撤銷 → 不存在（不洩漏）。
	if _, err := svc.Revoke(ctx, u2.ID, d.ID); !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("non-owner revoke err = %v, want ErrDeviceNotFound", err)
	}

	trusted, err := svc.SetTrusted(ctx, u.ID, d.ID, true)
	if err != nil {
		t.Fatalf("SetTrusted: %v", err)
	}
	if !trusted.Trusted {
		t.Fatalf("trusted = %+v", trusted)
	}

	revoked, err := svc.Revoke(ctx, u.ID, d.ID)
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if !revoked.Revoked || revoked.RevokedAt == nil {
		t.Fatalf("revoked = %+v", revoked)
	}
	// 冪等：再次撤銷回傳現狀。
	if _, err := svc.Revoke(ctx, u.ID, d.ID); err != nil {
		t.Fatalf("Revoke again: %v", err)
	}
}
