//go:build integration

package friend

import (
	"context"
	"testing"
	"time"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
	"ether.local/api/internal/profile"
	"ether.local/api/internal/user"
)

// TestPG_FriendFlowPersists 驗證好友完整流程經 PostgreSQL 持久化：
// 邀請→接受→好友清單→封鎖→解除封鎖→重新邀請。無隔離資料庫時
// 跳過並標示「環境待驗證」。
func TestPG_FriendFlowPersists(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)
	q := postgres.WrapDB(db)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	users := user.NewService(user.NewPGUserStore(q), user.NewPGSettingsStore(q), func() time.Time { return now })
	ua, err := users.Provision(ctx, "kc-subject-a", "a@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision a: %v", err)
	}
	ub, err := users.Provision(ctx, "kc-subject-b", "b@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision b: %v", err)
	}
	profiles := profile.NewService(profile.NewPGStore(q), users, func() time.Time { return now })
	nameB := "小彤"
	pidB := "xiaotong"
	if _, err := profiles.GetOrCreate(ctx, ub.ID, "小彤"); err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	if _, err := profiles.Update(ctx, ub.ID, profile.Patch{DisplayName: &nameB, PublicID: &pidB}); err != nil {
		t.Fatalf("Update profile: %v", err)
	}
	nameA := "阿哲"
	pidA := "azhe"
	if _, err := profiles.GetOrCreate(ctx, ua.ID, "阿哲"); err != nil {
		t.Fatalf("GetOrCreate a: %v", err)
	}
	if _, err := profiles.Update(ctx, ua.ID, profile.Patch{DisplayName: &nameA, PublicID: &pidA}); err != nil {
		t.Fatalf("Update profile a: %v", err)
	}

	svc := NewService(NewPGStore(q), users, profiles, func() time.Time { return now })
	req, err := svc.SendRequest(ctx, ua.ID, ub.ID, "你好")
	if err != nil {
		t.Fatalf("SendRequest: %v", err)
	}
	if _, err := svc.RespondRequest(ctx, ub.ID, req.ID, true); err != nil {
		t.Fatalf("RespondRequest: %v", err)
	}

	// 第二個 Service 實例共用同一資料庫（模擬重啟）：好友關係仍在。
	svc2 := NewService(NewPGStore(q), users, profiles, func() time.Time { return now })
	friends, err := svc2.ListFriends(ctx, ua.ID, false)
	if err != nil {
		t.Fatalf("ListFriends: %v", err)
	}
	if len(friends) != 1 || friends[0].Friend.DisplayName != "小彤" || friends[0].Friend.PublicID != "xiaotong" {
		t.Fatalf("friends = %+v", friends)
	}

	// 封鎖後對方視角的搜尋與邀請皆為 404（PostgreSQL 持久化的封鎖列）。
	if _, _, err := svc2.Block(ctx, ua.ID, ub.ID, ""); err != nil {
		t.Fatalf("Block: %v", err)
	}
	if _, err := svc2.SearchByPublicID(ctx, ub.ID, "azhe"); err != ErrUserNotFound {
		t.Fatalf("search after block err = %v, want ErrUserNotFound", err)
	}
	if _, err := svc2.Unblock(ctx, ua.ID, ub.ID); err != nil {
		t.Fatalf("Unblock: %v", err)
	}
	if _, err := svc2.SendRequest(ctx, ua.ID, ub.ID, ""); err != nil {
		t.Fatalf("re-request after unblock: %v", err)
	}
}
