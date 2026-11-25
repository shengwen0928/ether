package message

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeChats 是 ConversationAccess 的測試替身：成員關係、水位與
// 最後訊息快取皆以 map 記錄，不依賴 chat 套件。
type fakeChats struct {
	members map[string]map[string]bool // conversationID -> userID -> active
	direct  map[string]bool
	blocked bool
	readSeq map[string]int64 // conversationID + "\x00" + userID -> seq
	noted   []string         // NoteMessage 收到的 messageID（依序）
	marked  []int64          // MarkRead 收到的 seq（依序）
}

func newFakeChats() *fakeChats {
	return &fakeChats{
		members: map[string]map[string]bool{},
		direct:  map[string]bool{},
		readSeq: map[string]int64{},
	}
}

func (f *fakeChats) addMember(convID, userID string, direct bool) {
	if f.members[convID] == nil {
		f.members[convID] = map[string]bool{}
	}
	f.members[convID][userID] = true
	f.direct[convID] = direct
}

func (f *fakeChats) IsActiveMember(_ context.Context, conversationID, userID string) (bool, error) {
	return f.members[conversationID][userID], nil
}

func (f *fakeChats) IsDirect(_ context.Context, conversationID string) (bool, error) {
	return f.direct[conversationID], nil
}

func (f *fakeChats) DirectBlocked(_ context.Context, _ string, _ string) (bool, error) {
	return f.blocked, nil
}

func (f *fakeChats) NoteMessage(_ context.Context, _ string, messageID string, _ time.Time) error {
	f.noted = append(f.noted, messageID)
	return nil
}

func (f *fakeChats) MarkRead(_ context.Context, conversationID, userID, _ string, seq int64, _ time.Time) error {
	key := conversationID + "\x00" + userID
	if seq > f.readSeq[key] {
		f.readSeq[key] = seq
	}
	f.marked = append(f.marked, seq)
	return nil
}

func (f *fakeChats) ReadSequence(_ context.Context, conversationID, userID string) (int64, bool, error) {
	return f.readSeq[conversationID+"\x00"+userID], f.members[conversationID][userID], nil
}

func testService() (*Service, *fakeChats) {
	fixed := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	chats := newFakeChats()
	svc := NewService(NewMemoryStore(), chats, func() time.Time { return fixed })
	return svc, chats
}

func TestSend_AssignsMonotonicSequences(t *testing.T) {
	svc, chats := testService()
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-1", "u-b", false)
	ctx := context.Background()
	var seqs []int64
	for i := 0; i < 3; i++ {
		res, err := svc.Send(ctx, "u-a", "conv-1", "text", "你好", "", "")
		if err != nil {
			t.Fatalf("Send %d: %v", i, err)
		}
		if !res.IsNew {
			t.Fatalf("Send %d: IsNew = false", i)
		}
		seqs = append(seqs, res.Message.SequenceNumber)
	}
	if seqs[0] != 1 || seqs[1] != 2 || seqs[2] != 3 {
		t.Fatalf("seqs = %v, want [1 2 3]", seqs)
	}
	if len(chats.noted) != 3 {
		t.Fatalf("NoteMessage 呼叫 %d 次，want 3", len(chats.noted))
	}
}

func TestSend_IdempotentByClientMessageID(t *testing.T) {
	svc, chats := testService()
	chats.addMember("conv-1", "u-a", false)
	ctx := context.Background()
	first, err := svc.Send(ctx, "u-a", "conv-1", "text", "重送測試", "client-1", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	second, err := svc.Send(ctx, "u-a", "conv-1", "text", "重送測試（不同內容）", "client-1", "")
	if err != nil {
		t.Fatalf("重送: %v", err)
	}
	if second.IsNew || second.Message.ID != first.Message.ID {
		t.Fatalf("重送應回傳既有訊息：isNew=%v id=%s", second.IsNew, second.Message.ID)
	}
	if second.Message.Content != "重送測試" {
		t.Fatalf("重送回傳內容 = %q, want 原始內容", second.Message.Content)
	}
	if len(chats.noted) != 1 {
		t.Fatalf("冪等命中不得重發 NoteMessage，noted = %d", len(chats.noted))
	}
}

func TestSend_Validation(t *testing.T) {
	svc, chats := testService()
	chats.addMember("conv-1", "u-a", false)
	ctx := context.Background()
	if _, err := svc.Send(ctx, "u-ghost", "conv-1", "text", "x", "", ""); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("非成員 err = %v, want ErrMessageNotFound", err)
	}
	if _, err := svc.Send(ctx, "u-a", "conv-1", "text", "", "", ""); err == nil {
		t.Fatal("空訊息應失敗")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("err = %v, want ValidationError", err)
		}
	}
	if _, err := svc.Send(ctx, "u-a", "conv-1", "image", "x", "", ""); err == nil {
		t.Fatal("非 text kind 應失敗")
	}
	if _, err := svc.Send(ctx, "u-a", "conv-1", "text", "x", "", "ghost-msg"); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("引用不存在的訊息 err = %v, want ErrMessageNotFound", err)
	}
}

func TestSend_BlockedDirectReturnsNotFound(t *testing.T) {
	svc, chats := testService()
	chats.addMember("conv-1", "u-a", true)
	chats.blocked = true
	if _, err := svc.Send(context.Background(), "u-a", "conv-1", "text", "x", "", ""); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("封鎖中 err = %v, want ErrMessageNotFound", err)
	}
}

func TestList_CursorPagination(t *testing.T) {
	svc, chats := testService()
	chats.addMember("conv-1", "u-a", false)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if _, err := svc.Send(ctx, "u-a", "conv-1", "text", "訊息", "", ""); err != nil {
			t.Fatalf("Send: %v", err)
		}
	}
	page, err := svc.List(ctx, "u-a", "conv-1", nil, nil, 2)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Messages) != 2 || page.Messages[0].SequenceNumber != 5 || page.NextCursor != "4" {
		t.Fatalf("最新頁 = %+v, cursor = %q", page.Messages, page.NextCursor)
	}
	before := int64(4)
	page2, err := svc.List(ctx, "u-a", "conv-1", &before, nil, 2)
	if err != nil {
		t.Fatalf("List before: %v", err)
	}
	if len(page2.Messages) != 2 || page2.Messages[0].SequenceNumber != 3 {
		t.Fatalf("before 頁 = %+v", page2.Messages)
	}
	since := int64(3)
	page3, err := svc.List(ctx, "u-a", "conv-1", nil, &since, 10)
	if err != nil {
		t.Fatalf("List since: %v", err)
	}
	if len(page3.Messages) != 2 || page3.Messages[0].SequenceNumber != 4 {
		t.Fatalf("since 頁 = %+v", page3.Messages)
	}
	both := int64(1)
	if _, err := svc.List(ctx, "u-a", "conv-1", &both, &since, 10); err == nil {
		t.Fatal("雙游標應失敗")
	}
}

func TestEditRecallDelete(t *testing.T) {
	svc, chats := testService()
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-1", "u-b", false)
	ctx := context.Background()
	res, err := svc.Send(ctx, "u-a", "conv-1", "text", "原文", "", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	id := res.Message.ID
	if _, err := svc.Edit(ctx, "u-b", id, "竄改"); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("非作者編輯 err = %v, want ErrMessageNotFound", err)
	}
	edited, err := svc.Edit(ctx, "u-a", id, "改後")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if edited.Content != "改後" || edited.EditCount != 1 || edited.OriginalContent != "原文" || edited.EditedAt == nil {
		t.Fatalf("編輯結果 = %+v", edited)
	}
	deleted, err := svc.DeleteForSelf(ctx, "u-a", id)
	if err != nil {
		t.Fatalf("DeleteForSelf: %v", err)
	}
	if deleted.DeletedForSenderAt == nil {
		t.Fatal("刪除標記未寫入")
	}
	if _, err := svc.Edit(ctx, "u-a", id, "再改"); err == nil {
		t.Fatal("已刪除訊息應不可編輯")
	}
	recalled, err := svc.Recall(ctx, "u-a", id)
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if recalled.Status != StatusRecalled || recalled.RecalledAt == nil {
		t.Fatalf("收回結果 = %+v", recalled)
	}
	again, err := svc.Recall(ctx, "u-a", id)
	if err != nil || again.Status != StatusRecalled {
		t.Fatalf("重複收回應冪等成功：%v, %+v", err, again)
	}
}

func TestReceipts_ReadAdvancesWatermark(t *testing.T) {
	svc, chats := testService()
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-1", "u-b", false)
	ctx := context.Background()
	res, err := svc.Send(ctx, "u-a", "conv-1", "text", "你好", "", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	id := res.Message.ID
	if _, err := svc.PostReceipt(ctx, "u-a", id, ReceiptDelivered, ""); err == nil {
		t.Fatal("自己的訊息寫回條應失敗")
	}
	if _, err := svc.PostReceipt(ctx, "u-b", id, "seen", ""); err == nil {
		t.Fatal("非法回條型別應失敗")
	}
	r1, err := svc.PostReceipt(ctx, "u-b", id, ReceiptRead, "")
	if err != nil {
		t.Fatalf("PostReceipt: %v", err)
	}
	r2, err := svc.PostReceipt(ctx, "u-b", id, ReceiptRead, "")
	if err != nil || r2.ID != r1.ID {
		t.Fatalf("重複回條應冪等：%v, %v", err, r2)
	}
	if len(chats.marked) != 1 || chats.marked[0] != 1 {
		t.Fatalf("MarkRead 呼叫 = %v, want [1]", chats.marked)
	}
	list, err := svc.ListReceipts(ctx, "u-b", id)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListReceipts = %d, %v", len(list), err)
	}
	n, err := svc.UnreadCount(ctx, "conv-1", "u-b")
	if err != nil || n != 0 {
		t.Fatalf("已讀後未讀數 = %d, err = %v, want 0", n, err)
	}
	n, err = svc.UnreadCount(ctx, "conv-1", "u-a")
	if err != nil || n != 0 {
		t.Fatalf("自己的訊息不計未讀：n = %d, err = %v", n, err)
	}
}
