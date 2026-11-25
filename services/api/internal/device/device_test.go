package device

import (
	"context"
	"errors"
	"testing"
	"time"

	"ether.local/api/internal/auth"
)

type stubUsers struct{ exists bool }

func (s stubUsers) Exists(context.Context, string) (bool, error) { return s.exists, nil }

func TestRegister_RequiresHash(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemoryStore(), stubUsers{exists: true}, nil)

	// 原始識別碼必須拒絕。
	if _, err := svc.Register(ctx, "user-1", "Phone", "android", "raw-identifier"); err == nil {
		t.Fatal("want error for raw identifier")
	}
	hash := auth.HashToken("device-identifier-1")
	d, err := svc.Register(ctx, "user-1", "Phone", "android", hash)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if d.DeviceIdentifierHash != hash || d.Trusted || d.Revoked {
		t.Fatalf("device = %+v", d)
	}
}

func TestRegister_UnknownUser(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemoryStore(), stubUsers{exists: false}, nil)
	if _, err := svc.Register(ctx, "ghost", "Phone", "android", auth.HashToken("x")); !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("err = %v, want ErrDeviceNotFound", err)
	}
}

func TestRevoke_OwnershipAndTrust(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	svc := NewService(NewMemoryStore(), stubUsers{exists: true}, func() time.Time { return now })
	d, _ := svc.Register(ctx, "user-1", "Phone", "android", auth.HashToken("x"))

	// 非擁有者撤銷 → 不存在（不洩漏）。
	if _, err := svc.Revoke(ctx, "user-2", d.ID); !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("foreign revoke: err = %v, want ErrDeviceNotFound", err)
	}
	revoked, err := svc.Revoke(ctx, "user-1", d.ID)
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if !revoked.Revoked || revoked.RevokedAt == nil {
		t.Fatal("device must be revoked with timestamp")
	}
	// 已撤銷裝置不得設為信任。
	if _, err := svc.SetTrusted(ctx, "user-1", d.ID, true); err == nil {
		t.Fatal("want error trusting a revoked device")
	}
	// 撤銷冪等。
	if _, err := svc.Revoke(ctx, "user-1", d.ID); err != nil {
		t.Fatalf("Revoke again: %v", err)
	}
}
