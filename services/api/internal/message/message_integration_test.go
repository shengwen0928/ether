//go:build integration

package message

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

// TestPG_MessageFlowPersists 驗證訊息完整流程經 PostgreSQL 持久化：
// 建對話→發送三則（含引用與去重鍵）→ 第二個 Service 實例（模擬
// 重啟）讀取歷史與游標續拉 → 已讀回條推進水位 → 編輯／收回。
// 無隔離資料庫時跳過並標示「環境待驗證」。
func TestPG_MessageFlowPersists(t *testing.T) {
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
	friends := friend.NewService(friend.NewPGStore(q), users, profiles, func() time.Time { return now })
	if _, err := friends.SendRequest(ctx, ua.ID, ub.ID, ""); err != nil {
		t.Fatalf("SendRequest: %v", err)
	}
	reqs, err := friends.ListRequests(ctx, ub.ID, true)
	if err != nil || len(reqs) != 1 {
		t.Fatalf("ListRequests = %d, %v", len(reqs), err)
	}
	if _, err := friends.RespondRequest(ctx, ub.ID, reqs[0].Request.ID, true); err != nil {
		t.Fatalf("RespondRequest: %v", err)
	}

	chats := chat.NewService(chat.NewPGStore(q), users, friends, func() time.Time { return now })
	conv, isNew, err := chats.CreateDirect(ctx, ua.ID, ub.ID)
	if err != nil || !isNew {
		t.Fatalf("CreateDirect = %+v, %v, %v", conv, isNew, err)
	}

	svc := NewService(NewPGStore(q), chats, func() time.Time { return now })
	first, err := svc.Send(ctx, ua.ID, conv.ID, "text", "第一則", "client-1", "")
	if err != nil || !first.IsNew || first.Message.SequenceNumber != 1 {
		t.Fatalf("Send 1 = %+v, %v", first, err)
	}
	second, err := svc.Send(ctx, ub.ID, conv.ID, "text", "回覆第一則", "client-2", first.Message.ID)
	if err != nil || !second.IsNew || second.Message.SequenceNumber != 2 {
		t.Fatalf("Send 2 = %+v, %v", second, err)
	}
	if _, err := svc.Send(ctx, ua.ID, conv.ID, "text", "第三則", "client-3", ""); err != nil {
		t.Fatalf("Send 3: %v", err)
	}
	replay, err := svc.Send(ctx, ua.ID, conv.ID, "text", "重送", "client-1", "")
	if err != nil || replay.IsNew || replay.Message.ID != first.Message.ID {
		t.Fatalf("重送應冪等命中：%+v, %v", replay, err)
	}

	// 第二個 Service 實例共用同一資料庫（模擬重啟）：歷史、序號、
	// 引用關聯皆持久化；續拉只回傳游標之後的訊息。
	svc2 := NewService(NewPGStore(q), chats, func() time.Time { return now })
	page, err := svc2.List(ctx, ub.ID, conv.ID, nil, nil, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Messages) != 3 || page.Messages[0].SequenceNumber != 3 {
		t.Fatalf("歷史頁 = %d 筆，首筆序號 %d", len(page.Messages), page.Messages[0].SequenceNumber)
	}
	if page.Messages[1].ReplyToMessageID != first.Message.ID {
		t.Fatalf("引用關聯未持久化：%+v", page.Messages[1])
	}
	since := int64(1)
	resync, err := svc2.List(ctx, ub.ID, conv.ID, nil, &since, 10)
	if err != nil || len(resync.Messages) != 2 || resync.Messages[0].SequenceNumber != 2 {
		t.Fatalf("續拉 = %+v, %v", resync, err)
	}

	if _, err := svc2.PostReceipt(ctx, ub.ID, first.Message.ID, ReceiptRead, ""); err != nil {
		t.Fatalf("PostReceipt: %v", err)
	}
	n, err := svc2.UnreadCount(ctx, conv.ID, ub.ID)
	if err != nil || n != 1 {
		t.Fatalf("已讀一則後未讀數 = %d, err = %v, want 1（第三則）", n, err)
	}

	edited, err := svc2.Edit(ctx, ua.ID, first.Message.ID, "第一則（已編輯）")
	if err != nil || edited.EditCount != 1 || edited.OriginalContent != "第一則" {
		t.Fatalf("Edit = %+v, %v", edited, err)
	}
	recalled, err := svc2.Recall(ctx, ub.ID, second.Message.ID)
	if err != nil || recalled.Status != StatusRecalled {
		t.Fatalf("Recall = %+v, %v", recalled, err)
	}
}
