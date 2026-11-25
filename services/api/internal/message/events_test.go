package message

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"ether.local/api/internal/outbox"
)

func testServiceWithOutbox() (*Service, *fakeChats, *outbox.MemoryStore) {
	fixed := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	chats := newFakeChats()
	obs := outbox.NewMemoryStore()
	store := NewMemoryStoreWithOutbox(obs)
	svc := NewServiceWithOutbox(store, chats, obs, func() time.Time { return fixed })
	return svc, chats, obs
}

func claimAll(t *testing.T, obs *outbox.MemoryStore) []*outbox.Event {
	t.Helper()
	claimed, err := obs.Claim(context.Background(), 100)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	return claimed
}

func TestSend_WithOutboxWritesReceivedEvent(t *testing.T) {
	svc, chats, obs := testServiceWithOutbox()
	chats.addMember("conv-1", "u-a", false)
	ctx := context.Background()
	res, err := svc.Send(ctx, "u-a", "conv-1", "text", "你好", "client-1", "")
	if err != nil || !res.IsNew {
		t.Fatalf("Send = %+v, %v", res, err)
	}
	claimed := claimAll(t, obs)
	if len(claimed) != 1 {
		t.Fatalf("事件數 = %d, want 1", len(claimed))
	}
	ev := claimed[0]
	if ev.EventName != EventMessageReceived {
		t.Fatalf("事件名 = %q, want message.received", ev.EventName)
	}
	if ev.Topic != "conversation.conv-1" || ev.MessageID != res.Message.ID {
		t.Fatalf("事件 topic/message = %q/%q", ev.Topic, ev.MessageID)
	}
	var payload receivedPayload
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		t.Fatalf("payload 解析: %v", err)
	}
	if payload.MessageID != res.Message.ID || payload.Sequence != res.Message.SequenceNumber ||
		payload.Content != "你好" || payload.SenderID != "u-a" ||
		payload.ClientMessageID != "client-1" || payload.Status != "sent" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestSend_IdempotentHitWritesNoSecondEvent(t *testing.T) {
	svc, chats, obs := testServiceWithOutbox()
	chats.addMember("conv-1", "u-a", false)
	ctx := context.Background()
	if _, err := svc.Send(ctx, "u-a", "conv-1", "text", "你好", "client-1", ""); err != nil {
		t.Fatalf("Send 1: %v", err)
	}
	res, err := svc.Send(ctx, "u-a", "conv-1", "text", "你好", "client-1", "")
	if err != nil || res.IsNew {
		t.Fatalf("重送應冪等命中：%+v, %v", res, err)
	}
	if claimed := claimAll(t, obs); len(claimed) != 1 {
		t.Fatalf("冪等命中後事件數 = %d, want 1", len(claimed))
	}
}

func TestRecall_WithOutboxWritesRecalledEventWithoutContent(t *testing.T) {
	svc, chats, obs := testServiceWithOutbox()
	chats.addMember("conv-1", "u-a", false)
	ctx := context.Background()
	res, err := svc.Send(ctx, "u-a", "conv-1", "text", "秘密內容", "", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if _, err := svc.Recall(ctx, "u-a", res.Message.ID); err != nil {
		t.Fatalf("Recall: %v", err)
	}
	claimed := claimAll(t, obs)
	if len(claimed) != 2 {
		t.Fatalf("事件數 = %d, want 2（received＋recalled）", len(claimed))
	}
	ev := claimed[1]
	if ev.EventName != EventMessageRecalled {
		t.Fatalf("第二則事件名 = %q, want message.recalled", ev.EventName)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(ev.Payload, &raw); err != nil {
		t.Fatalf("payload 解析: %v", err)
	}
	if _, hasContent := raw["content"]; hasContent {
		t.Fatal("收回事件不得含原內容")
	}
	var payload recalledPayload
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		t.Fatalf("payload 解析: %v", err)
	}
	if payload.RecalledBy != "u-a" || payload.MessageID != res.Message.ID {
		t.Fatalf("payload = %+v", payload)
	}
	// 重複收回冪等，不得再寫事件。
	if _, err := svc.Recall(ctx, "u-a", res.Message.ID); err != nil {
		t.Fatalf("重複 Recall: %v", err)
	}
	if claimed := claimAll(t, obs); len(claimed) != 0 {
		t.Fatalf("重複收回後新增事件數 = %d, want 0", len(claimed))
	}
}

func TestEditAndDeleteForSelf_WriteNoEvent(t *testing.T) {
	svc, chats, obs := testServiceWithOutbox()
	chats.addMember("conv-1", "u-a", false)
	ctx := context.Background()
	res, err := svc.Send(ctx, "u-a", "conv-1", "text", "原文", "", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if _, err := svc.Edit(ctx, "u-a", res.Message.ID, "改後"); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if _, err := svc.DeleteForSelf(ctx, "u-a", res.Message.ID); err != nil {
		t.Fatalf("DeleteForSelf: %v", err)
	}
	claimed := claimAll(t, obs)
	if len(claimed) != 1 || claimed[0].EventName != EventMessageReceived {
		t.Fatalf("編輯／刪除不得產生事件，事件數 = %d", len(claimed))
	}
}

func TestSend_OutboxWithoutAtomicWriterFailsClosed(t *testing.T) {
	fixed := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	chats := newFakeChats()
	chats.addMember("conv-1", "u-a", false)
	// 記憶體儲存未綁定 Outbox 儲存：Outbox 已接線時須明確失敗，
	// 不得以非原子寫入假裝事件不丟失。
	svc := NewServiceWithOutbox(NewMemoryStore(), chats, outbox.NewMemoryStore(), func() time.Time { return fixed })
	_, err := svc.Send(context.Background(), "u-a", "conv-1", "text", "你好", "", "")
	if err == nil || !errors.Is(err, errOutboxUnsupported) {
		t.Fatalf("應回傳 Outbox 不支援錯誤，實際：%v", err)
	}
}
