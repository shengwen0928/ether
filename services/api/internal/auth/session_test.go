package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func fixedNow() time.Time { return time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC) }

func TestSessionService_RotateAndReplay(t *testing.T) {
	ctx := context.Background()
	svc := NewSessionService(NewMemorySessionStore(), fixedNow)
	h1 := HashToken("refresh-1")
	h2 := HashToken("refresh-2")
	h3 := HashToken("refresh-3")

	sess, err := svc.Create(ctx, "user-1", "device-1", "kc-sid-1", h1, fixedNow().Add(time.Hour))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// 正常輪換。
	rotated, err := svc.Rotate(ctx, sess.ID, h1, h2, fixedNow().Add(2*time.Hour))
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if rotated.RefreshTokenHash != h2 {
		t.Fatalf("rotated hash = %q, want %q", rotated.RefreshTokenHash, h2)
	}
	// 從未見過的舊雜湊：單純無效，不得撤銷家族。
	if _, err := svc.Rotate(ctx, sess.ID, h3, h1, fixedNow().Add(3*time.Hour)); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("unknown old hash: err = %v, want ErrSessionInvalid", err)
	}
	// 重放已輪換的雜湊：盜用，家族撤銷。
	if _, err := svc.Rotate(ctx, sess.ID, h1, h3, fixedNow().Add(3*time.Hour)); !errors.Is(err, ErrRefreshTokenReplayed) {
		t.Fatalf("replay: err = %v, want ErrRefreshTokenReplayed", err)
	}
	got, err := svc.store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.Revoked() || got.RevokedReason != "refresh_token_replayed" {
		t.Fatalf("session after replay: revoked=%v reason=%q", got.Revoked(), got.RevokedReason)
	}
}

func TestSessionService_CreateRejectsPlaintext(t *testing.T) {
	ctx := context.Background()
	svc := NewSessionService(NewMemorySessionStore(), fixedNow)
	if _, err := svc.Create(ctx, "user-1", "", "", "not-a-hash", fixedNow().Add(time.Hour)); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("plaintext refresh value: err = %v, want ErrSessionInvalid", err)
	}
}

func TestSessionService_RevokeIdempotentAndRevokeAll(t *testing.T) {
	ctx := context.Background()
	svc := NewSessionService(NewMemorySessionStore(), fixedNow)
	h := HashToken("refresh-1")
	s1, _ := svc.Create(ctx, "user-1", "", "", h, fixedNow().Add(time.Hour))
	s2, _ := svc.Create(ctx, "user-1", "", "", h, fixedNow().Add(time.Hour))
	if _, err := svc.Revoke(ctx, s1.ID, "logout"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err := svc.Revoke(ctx, s1.ID, "logout"); err != nil {
		t.Fatalf("Revoke again: %v", err)
	}
	if err := svc.RevokeAllForUser(ctx, "user-1", "logout_all"); err != nil {
		t.Fatalf("RevokeAllForUser: %v", err)
	}
	for _, id := range []string{s1.ID, s2.ID} {
		got, _ := svc.store.Get(ctx, id)
		if !got.Revoked() {
			t.Fatalf("session %s not revoked", id)
		}
	}
}

func TestSessionService_TouchExpired(t *testing.T) {
	now := fixedNow()
	svc := NewSessionService(NewMemorySessionStore(), func() time.Time { return now })
	sess, _ := svc.Create(context.Background(), "user-1", "", "", HashToken("r"), now.Add(time.Hour))
	now = now.Add(2 * time.Hour)
	if err := svc.Touch(context.Background(), sess.ID); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("Touch expired: err = %v, want ErrSessionInvalid", err)
	}
}

func TestHashToken_NeverPlaintext(t *testing.T) {
	h := HashToken("secret-refresh-token")
	if h == "secret-refresh-token" || len(h) != 64 {
		t.Fatalf("hash = %q", h)
	}
	if HashToken("secret-refresh-token") != h {
		t.Fatal("hash not deterministic")
	}
}
