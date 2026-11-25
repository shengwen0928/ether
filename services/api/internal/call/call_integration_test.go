//go:build integration

package call

import (
	"context"
	"testing"
	"time"

	"ether.local/api/internal/chat"
	"ether.local/api/internal/friend"
	"ether.local/api/internal/group"
	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
	"ether.local/api/internal/profile"
	"ether.local/api/internal/user"
)

// TestPG_CallFlowPersists 驗證通話完整流程經 PostgreSQL 持久化：
// 一對一發起 → 接聽 → 結束 → 第二個 Service 實例（模擬重啟）讀取
// 紀錄、參與者與事件；群組通話拒接彙總亦持久化。
// 無隔離資料庫時跳過並標示「環境待驗證」。
func TestPG_CallFlowPersists(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)
	q := postgres.WrapDB(db)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	users := user.NewService(user.NewPGUserStore(q), user.NewPGSettingsStore(q), func() time.Time { return now })
	ua, err := users.Provision(ctx, "kc-subject-ca", "ca@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision a: %v", err)
	}
	ub, err := users.Provision(ctx, "kc-subject-cb", "cb@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision b: %v", err)
	}
	uc, err := users.Provision(ctx, "kc-subject-cc", "cc@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision c: %v", err)
	}
	profiles := profile.NewService(profile.NewPGStore(q), users, func() time.Time { return now })
	friendSvc := friend.NewService(friend.NewPGStore(q), users, profiles, func() time.Time { return now })
	chatSvc := chat.NewService(chat.NewPGStore(q), users, friendSvc, func() time.Time { return now })
	groupSvc := group.NewService(group.NewPGStore(q), users, friendSvc, chatSvc, func() time.Time { return now })

	if _, err := friendSvc.SendRequest(ctx, ua.ID, ub.ID, ""); err != nil {
		t.Fatalf("SendRequest: %v", err)
	}
	requests, err := friendSvc.ListRequests(ctx, ub.ID, true)
	if err != nil || len(requests) != 1 {
		t.Fatalf("ListRequests = %v, %v", requests, err)
	}
	if _, err := friendSvc.RespondRequest(ctx, ub.ID, requests[0].Request.ID, true); err != nil {
		t.Fatalf("RespondRequest: %v", err)
	}
	conv, isNew, err := chatSvc.CreateDirect(ctx, ua.ID, ub.ID)
	if err != nil || !isNew {
		t.Fatalf("CreateDirect = %v, %v, %v", conv, isNew, err)
	}
	groupEntry, err := groupSvc.CreateGroup(ctx, ua.ID, "通話小隊", "", "", []string{ub.ID, uc.ID})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	svc := NewService(NewPGStore(q), users, chatSvc, groupSvc, func() time.Time { return now })
	entry, err := svc.Initiate(ctx, ua.ID, InitiateInput{ConversationID: conv.ID, Type: CallTypeVoice})
	if err != nil {
		t.Fatalf("Initiate: %v", err)
	}
	if _, err := svc.Answer(ctx, ub.ID, entry.Call.ID); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	ended, err := svc.End(ctx, ua.ID, entry.Call.ID)
	if err != nil {
		t.Fatalf("End: %v", err)
	}
	if ended.Call.Status != StatusEnded {
		t.Fatalf("status = %s, want ended", ended.Call.Status)
	}

	gcall, err := svc.Initiate(ctx, ua.ID, InitiateInput{GroupID: groupEntry.Group.ID, Type: CallTypeVideo})
	if err != nil {
		t.Fatalf("Initiate group: %v", err)
	}
	if _, err := svc.Decline(ctx, ub.ID, gcall.Call.ID); err != nil {
		t.Fatalf("Decline: %v", err)
	}
	declined, err := svc.Decline(ctx, uc.ID, gcall.Call.ID)
	if err != nil {
		t.Fatalf("Decline 2: %v", err)
	}
	if declined.Call.Status != StatusDeclined {
		t.Fatalf("group status = %s, want declined", declined.Call.Status)
	}

	// 第二個 Service 實例共用同一資料庫（模擬重啟）：通話、參與者、
	// 事件皆持久化。
	svc2 := NewService(NewPGStore(q), users, chatSvc, groupSvc, func() time.Time { return now })
	got, err := svc2.GetCall(ctx, ub.ID, entry.Call.ID)
	if err != nil {
		t.Fatalf("GetCall after restart: %v", err)
	}
	if got.Call.Status != StatusEnded || got.ParticipantCount != 2 {
		t.Fatalf("after restart = %+v", got.Call)
	}
	participants, err := svc2.ListParticipants(ctx, ub.ID, entry.Call.ID)
	if err != nil || len(participants) != 2 {
		t.Fatalf("participants = %v, %v", participants, err)
	}
	events, err := svc2.ListEvents(ctx, ub.ID, entry.Call.ID)
	if err != nil || len(events) != 3 {
		t.Fatalf("events = %v, %v", events, err)
	}
	calls, _, err := svc2.ListCalls(ctx, ua.ID, "", 20)
	if err != nil || len(calls) != 2 {
		t.Fatalf("ListCalls = %v, %v", calls, err)
	}
}
