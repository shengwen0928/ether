//go:build integration

package message

import (
	"context"
	"errors"
	"testing"
	"time"

	"ether.local/api/internal/chat"
	"ether.local/api/internal/friend"
	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
	"ether.local/api/internal/profile"
	"ether.local/api/internal/user"
)

// TestPG_ReactionsPinsPersist 驗證反應與置頂經 PostgreSQL 持久化：
// 建對話→發送訊息→反應／更換／取消→置頂／上限／取消→第二個
// Service 實例（模擬重啟）讀取彙總與置頂清單→非成員 404。
// 無隔離資料庫時跳過並標示「環境待驗證」。
func TestPG_ReactionsPinsPersist(t *testing.T) {
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
	uc, err := users.Provision(ctx, "kc-subject-c", "c@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision c: %v", err)
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
	sent := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		res, err := svc.Send(ctx, ua.ID, conv.ID, "text", "訊息", "", "")
		if err != nil {
			t.Fatalf("Send: %v", err)
		}
		sent = append(sent, res.Message.ID)
	}

	// 反應：u-b 讚、u-a 愛心；u-b 更換為大笑；u-a 取消。
	if _, err := svc.SetReaction(ctx, ub.ID, sent[0], "👍"); err != nil {
		t.Fatalf("SetReaction: %v", err)
	}
	if _, err := svc.SetReaction(ctx, ua.ID, sent[0], "❤️"); err != nil {
		t.Fatalf("SetReaction: %v", err)
	}
	summary, err := svc.SetReaction(ctx, ub.ID, sent[0], "😂")
	if err != nil || len(summary.Reactions) != 2 {
		t.Fatalf("更換後彙總 = %+v, %v", summary, err)
	}
	if _, err := svc.RemoveReaction(ctx, ua.ID, sent[0]); err != nil {
		t.Fatalf("RemoveReaction: %v", err)
	}

	// 置頂前五則，第六則達上限拒絕；取消第一則後可置頂第六則。
	for _, id := range sent[:5] {
		if _, _, err := svc.PinMessage(ctx, ub.ID, conv.ID, id); err != nil {
			t.Fatalf("PinMessage: %v", err)
		}
	}
	if _, _, err := svc.PinMessage(ctx, ub.ID, conv.ID, sent[5]); err == nil {
		t.Fatal("超過置頂上限應失敗")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("err = %v, want ValidationError", err)
		}
	}
	if _, err := svc.UnpinMessage(ctx, ub.ID, conv.ID, sent[0]); err != nil {
		t.Fatalf("UnpinMessage: %v", err)
	}
	if _, _, err := svc.PinMessage(ctx, ub.ID, conv.ID, sent[5]); err != nil {
		t.Fatalf("取消後再置頂應成功：%v", err)
	}

	// 第二個 Service 實例共用同一資料庫（模擬重啟）：彙總與置頂皆持久化。
	svc2 := NewService(NewPGStore(q), chats, func() time.Time { return now })
	summary2, err := svc2.ListReactions(ctx, ub.ID, sent[0])
	if err != nil || len(summary2.Reactions) != 1 || summary2.Reactions[0].Emoji != "😂" {
		t.Fatalf("重啟後彙總 = %+v, %v", summary2, err)
	}
	pins, err := svc2.ListPins(ctx, ua.ID, conv.ID)
	if err != nil || len(pins) != 5 {
		t.Fatalf("重啟後置頂 = %d 筆，%v", len(pins), err)
	}
	if _, err := svc2.SetReaction(ctx, uc.ID, sent[0], "👍"); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("非成員反應 err = %v, want ErrMessageNotFound", err)
	}
	if _, err := svc2.ListPins(ctx, uc.ID, conv.ID); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("非成員列出置頂 err = %v, want ErrMessageNotFound", err)
	}
}
