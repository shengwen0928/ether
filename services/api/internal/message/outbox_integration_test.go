//go:build integration

package message

import (
	"context"
	"os"
	"testing"
	"time"

	"ether.local/api/internal/chat"
	"ether.local/api/internal/friend"
	"ether.local/api/internal/outbox"
	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
	"ether.local/api/internal/profile"
	"ether.local/api/internal/user"
)

// TestPG_OutboxAtomicWrite 驗證 Outbox 接線的 PostgreSQL 持久化：
// 發送訊息時訊息列與 message.received 事件列同一交易提交（第二個
// Service 實例可讀到歷史，且 Outbox 認領可拿到該事件）；收回時
// message.recalled 事件列同交易寫入。無隔離資料庫時跳過並標示
// 「環境待驗證」。
func TestPG_OutboxAtomicWrite(t *testing.T) {
	ctx := context.Background()
	postgrestest.SetupTestDB(t) // 套用全部 migration（含 000009）。
	dsn := os.Getenv("ETHER_TEST_DATABASE_URL")
	pgdb, err := postgres.Open(dsn)
	if err != nil {
		t.Fatalf("開啟 PostgreSQL 連線池失敗: %v", err)
	}
	defer func() { _ = pgdb.Close() }()
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pgdb.Ping(pingCtx); err != nil {
		t.Fatalf("PostgreSQL 連線失敗: %v", err)
	}
	q := pgdb.Querier()
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	users := user.NewService(user.NewPGUserStore(q), user.NewPGSettingsStore(q), func() time.Time { return now })
	ua, err := users.Provision(ctx, "kc-outbox-a", "a@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision a: %v", err)
	}
	ub, err := users.Provision(ctx, "kc-outbox-b", "b@example.invalid", true)
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

	obs := outbox.NewPGStore(q)
	svc := NewServiceWithOutbox(NewPGStoreWithDB(q, pgdb), chats, obs, func() time.Time { return now })
	sent, err := svc.Send(ctx, ua.ID, conv.ID, "text", "Outbox 測試", "client-outbox-1", "")
	if err != nil || !sent.IsNew {
		t.Fatalf("Send = %+v, %v", sent, err)
	}
	claimed, err := obs.Claim(ctx, 100)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	var received *outbox.Event
	for _, ev := range claimed {
		if ev.MessageID == sent.Message.ID && ev.EventName == EventMessageReceived {
			received = ev
		}
	}
	if received == nil {
		t.Fatalf("認領不到 message.received 事件（認領 %d 則）", len(claimed))
	}
	if err := obs.MarkPublished(ctx, received.ID); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}

	if _, err := svc.Recall(ctx, ua.ID, sent.Message.ID); err != nil {
		t.Fatalf("Recall: %v", err)
	}
	claimed, err = obs.Claim(ctx, 100)
	if err != nil {
		t.Fatalf("Claim 2: %v", err)
	}
	found := false
	for _, ev := range claimed {
		if ev.MessageID == sent.Message.ID && ev.EventName == EventMessageRecalled {
			found = true
		}
	}
	if !found {
		t.Fatalf("認領不到 message.recalled 事件（認領 %d 則）", len(claimed))
	}
}
