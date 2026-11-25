//go:build integration

package group

import (
	"context"
	"testing"
	"time"

	"ether.local/api/internal/chat"
	"ether.local/api/internal/friend"
	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
	"ether.local/api/internal/profile"
	"ether.local/api/internal/user"
)

// TestPG_GroupFlowPersists 驗證群組完整流程經 PostgreSQL 持久化：
// 建立（含對話容器）→ 邀請／接受（含對話成員同步）→ 公告單一現行
// → 所有權轉移 → 解散 → 第二個 Service 實例（模擬重啟）讀取。
// 無隔離資料庫時跳過並標示「環境待驗證」。
func TestPG_GroupFlowPersists(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)
	q := postgres.WrapDB(db)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	users := user.NewService(user.NewPGUserStore(q), user.NewPGSettingsStore(q), func() time.Time { return now })
	ua, err := users.Provision(ctx, "kc-subject-ga", "ga@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision a: %v", err)
	}
	ub, err := users.Provision(ctx, "kc-subject-gb", "gb@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision b: %v", err)
	}
	uc, err := users.Provision(ctx, "kc-subject-gc", "gc@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision c: %v", err)
	}
	profiles := profile.NewService(profile.NewPGStore(q), users, func() time.Time { return now })
	friendSvc := friend.NewService(friend.NewPGStore(q), users, profiles, func() time.Time { return now })
	chatSvc := chat.NewService(chat.NewPGStore(q), users, friendSvc, func() time.Time { return now })

	svc := NewService(NewPGStore(q), users, friendSvc, chatSvc, func() time.Time { return now })
	entry, err := svc.CreateGroup(ctx, ua.ID, "登山小隊", "", "", []string{ub.ID})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if entry.Group.ConversationID == "" {
		t.Fatal("ConversationID 為空，對話容器未建立")
	}
	if _, err := chatSvc.GetConversation(ctx, ua.ID, entry.Group.ConversationID); err != nil {
		t.Fatalf("GetConversation: %v", err)
	}

	inv, err := svc.CreateInvitation(ctx, ua.ID, entry.Group.ID, uc.ID)
	if err != nil {
		t.Fatalf("CreateInvitation: %v", err)
	}
	if _, err := svc.RespondInvitation(ctx, uc.ID, inv.Invitation.ID, true); err != nil {
		t.Fatalf("RespondInvitation: %v", err)
	}
	active, err := chatSvc.IsActiveMember(ctx, entry.Group.ConversationID, uc.ID)
	if err != nil || !active {
		t.Fatalf("conversation member after accept: active=%v, err=%v", active, err)
	}

	// 第二個 Service 實例共用同一資料庫（模擬重啟）：群組、成員、
	// 邀請狀態皆持久化。
	svc2 := NewService(NewPGStore(q), users, friendSvc, chatSvc, func() time.Time { return now })
	got, err := svc2.GetGroup(ctx, uc.ID, entry.Group.ID)
	if err != nil {
		t.Fatalf("GetGroup after restart: %v", err)
	}
	if got.MemberCount != 3 {
		t.Fatalf("MemberCount = %d, want 3", got.MemberCount)
	}
	if _, err := svc2.CreateAnnouncement(ctx, ua.ID, entry.Group.ID, "第一則"); err != nil {
		t.Fatalf("CreateAnnouncement: %v", err)
	}
	if _, err := svc2.CreateAnnouncement(ctx, ua.ID, entry.Group.ID, "第二則"); err != nil {
		t.Fatalf("CreateAnnouncement 2: %v", err)
	}
	rows, err := svc2.ListAnnouncements(ctx, ub.ID, entry.Group.ID)
	if err != nil {
		t.Fatalf("ListAnnouncements: %v", err)
	}
	activeCount := 0
	for _, a := range rows {
		if a.IsActive {
			activeCount++
			if a.Content != "第二則" {
				t.Fatalf("active announcement = %q", a.Content)
			}
		}
	}
	if activeCount != 1 || len(rows) != 2 {
		t.Fatalf("announcements = %d rows, %d active", len(rows), activeCount)
	}

	p := InvitePolicyAll
	if _, err := svc2.UpdateSettings(ctx, ua.ID, entry.Group.ID, &p); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	st, err := svc2.GetSettings(ctx, ub.ID, entry.Group.ID)
	if err != nil || st.InvitePolicy != InvitePolicyAll {
		t.Fatalf("settings = %+v, %v", st, err)
	}

	if _, err := svc2.TransferOwnership(ctx, ua.ID, entry.Group.ID, ub.ID); err != nil {
		t.Fatalf("TransferOwnership: %v", err)
	}
	if err := svc2.Disband(ctx, ua.ID, entry.Group.ID); err == nil {
		t.Fatal("原擁有者解散應失敗")
	}
	if err := svc2.Disband(ctx, ub.ID, entry.Group.ID); err != nil {
		t.Fatalf("Disband: %v", err)
	}
	if _, err := svc2.GetGroup(ctx, ua.ID, entry.Group.ID); err != ErrGroupNotFound {
		t.Fatalf("disbanded GetGroup err = %v, want ErrGroupNotFound", err)
	}
}
