//go:build integration

package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
	"ether.local/api/internal/user"
)

func TestPG_SessionLifecycleAndReplayPersists(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)
	q := postgres.WrapDB(db)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	users := user.NewService(user.NewPGUserStore(q), user.NewPGSettingsStore(q), func() time.Time { return now })
	u, err := users.Provision(ctx, "kc-subject-1", "a@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}

	store := NewPGSessionStore(q)
	svc := NewSessionService(store, func() time.Time { return now })
	h1 := HashToken("refresh-1")
	h2 := HashToken("refresh-2")
	h3 := HashToken("refresh-3")

	sess, err := svc.Create(ctx, u.ID, "", "kc-sid-1", h1, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// 以 sid 查找。
	bySID, err := store.GetByKeycloakSessionID(ctx, "kc-sid-1")
	if err != nil {
		t.Fatalf("GetByKeycloakSessionID: %v", err)
	}
	if bySID.ID != sess.ID {
		t.Fatalf("by sid id = %s, want %s", bySID.ID, sess.ID)
	}

	// 正常輪換。
	if _, err := svc.Rotate(ctx, sess.ID, h1, h2, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	// 「重啟」：全新的 Service 實例、同一 PostgreSQL 儲存，舊雜湊
	// 重放仍須被偵測為盜用（持久化重放偵測的核心斷言）。
	svc2 := NewSessionService(NewPGSessionStore(q), func() time.Time { return now })
	if _, err := svc2.Rotate(ctx, sess.ID, h1, h3, now.Add(3*time.Hour)); !errors.Is(err, ErrRefreshTokenReplayed) {
		t.Fatalf("replay after restart err = %v, want ErrRefreshTokenReplayed", err)
	}
	got, err := store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.Revoked() || got.RevokedReason != "refresh_token_replayed" {
		t.Fatalf("session after replay: revoked=%v reason=%q", got.Revoked(), got.RevokedReason)
	}

	// Touch 與 RevokeAll。
	sess2, err := svc.Create(ctx, u.ID, "", "kc-sid-2", HashToken("refresh-9"), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}
	if err := svc.Touch(ctx, sess2.ID); err != nil {
		t.Fatalf("Touch: %v", err)
	}
	if err := svc.RevokeAllForUser(ctx, u.ID, "logout_all"); err != nil {
		t.Fatalf("RevokeAllForUser: %v", err)
	}
	list, err := store.ListByUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("sessions = %d, want 2", len(list))
	}
	for _, s := range list {
		if !s.Revoked() {
			t.Fatalf("session %s not revoked after RevokeAll", s.ID)
		}
	}
}
