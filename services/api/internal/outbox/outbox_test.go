package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
)

func testEvent() *Event {
	return &Event{
		EventName:      "message.received",
		Topic:          "conversation.conv-1",
		ConversationID: "conv-1",
		MessageID:      "m-1",
		Payload:        json.RawMessage(`{"message_id":"m-1"}`),
	}
}

func TestMemoryStore_AppendRejectsInvalidPayload(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	if err := s.Append(ctx, &Event{EventName: "message.received", Topic: "t", Payload: json.RawMessage(`{`)}); err == nil {
		t.Fatal("非法 JSON payload 應被拒絕")
	}
	if err := s.Append(ctx, &Event{Topic: "t", Payload: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("空事件名應被拒絕")
	}
	if err := s.Append(ctx, &Event{EventName: "message.received", Payload: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("空主題應被拒絕")
	}
}

func TestMemoryStore_ClaimPublishFlow(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		ev := testEvent()
		ev.MessageID = "m-1"
		if err := s.Append(ctx, ev); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	claimed, err := s.Claim(ctx, 2)
	if err != nil || len(claimed) != 2 {
		t.Fatalf("Claim = %d, %v", len(claimed), err)
	}
	for _, ev := range claimed {
		if ev.Status != StatusProcessing || ev.ClaimedAt == nil {
			t.Fatalf("認領後狀態 = %q", ev.Status)
		}
	}
	// 第二次認領不得重複拿到同一列。
	claimed2, _ := s.Claim(ctx, 10)
	if len(claimed2) != 1 {
		t.Fatalf("第二次 Claim = %d, want 1", len(claimed2))
	}
	if err := s.MarkPublished(ctx, claimed[0].ID); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}
	if claimed3, _ := s.Claim(ctx, 10); len(claimed3) != 0 {
		t.Fatalf("已發佈後 Claim = %d, want 0", len(claimed3))
	}
	// 重複標記已發佈為冪等成功。
	if err := s.MarkPublished(ctx, claimed[0].ID); err != nil {
		t.Fatalf("重複 MarkPublished: %v", err)
	}
}

func TestMemoryStore_MarkFailedRetriesThenDead(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	ev := testEvent()
	if err := s.Append(ctx, ev); err != nil {
		t.Fatalf("Append: %v", err)
	}
	claimed, _ := s.Claim(ctx, 1)
	dead, err := s.MarkFailed(ctx, claimed[0].ID, 2)
	if err != nil || dead {
		t.Fatalf("首次失敗 dead=%v err=%v, want dead=false", dead, err)
	}
	claimed, _ = s.Claim(ctx, 1)
	if len(claimed) != 1 {
		t.Fatal("失敗後應回到待發佈、可再次認領")
	}
	dead, err = s.MarkFailed(ctx, claimed[0].ID, 2)
	if err != nil || !dead {
		t.Fatalf("第二次失敗 dead=%v err=%v, want dead=true", dead, err)
	}
	if claimed, _ := s.Claim(ctx, 10); len(claimed) != 0 {
		t.Fatalf("死信後 Claim = %d, want 0", len(claimed))
	}
}

func TestMemoryStore_ClaimReclaimsExpiredLease(t *testing.T) {
	s := NewMemoryStore()
	fixed := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return fixed }
	ctx := context.Background()
	if err := s.Append(context.Background(), testEvent()); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if claimed, _ := s.Claim(ctx, 1); len(claimed) != 1 {
		t.Fatal("首次認領應成功")
	}
	// 租約內不得重複認領。
	if claimed, _ := s.Claim(ctx, 1); len(claimed) != 0 {
		t.Fatal("租約內不得重複認領")
	}
	// 租約逾期後可再次認領。
	s.now = func() time.Time { return fixed.Add(ClaimLease + time.Minute) }
	if claimed, _ := s.Claim(ctx, 1); len(claimed) != 1 {
		t.Fatal("租約逾期後應可再次認領")
	}
}

func TestPGStore_AppendSQL(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	ev := testEvent()
	if err := NewPGStore(fake).Append(context.Background(), ev); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if len(fake.Calls) != 1 || !strings.Contains(fake.Calls[0].Query, "INSERT INTO message_outbox") {
		t.Fatalf("Append calls = %+v", fake.Calls)
	}
	if !strings.Contains(fake.Calls[0].Query, "'pending'") {
		t.Fatalf("Append 須以 pending 寫入：%q", fake.Calls[0].Query)
	}
}

func TestPGStore_ClaimSQLUsesSkipLocked(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	if _, err := NewPGStore(fake).Claim(context.Background(), 10); err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if len(fake.Calls) != 1 || !strings.Contains(fake.Calls[0].Query, "FOR UPDATE SKIP LOCKED") {
		t.Fatalf("Claim 須以 SKIP LOCKED 搶佔：%+v", fake.Calls)
	}
	if !strings.Contains(fake.Calls[0].Query, "ORDER BY created_at ASC") {
		t.Fatalf("Claim 須依建立時間排序：%q", fake.Calls[0].Query)
	}
}

func TestPGStore_MarkFailedSQL(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		return postgrestest.FakeRow{Values: []any{"dead"}}
	}
	dead, err := NewPGStore(fake).MarkFailed(context.Background(), "e-1", 3)
	if err != nil || !dead {
		t.Fatalf("MarkFailed dead=%v err=%v", dead, err)
	}
	if len(fake.Calls) != 1 || !strings.Contains(fake.Calls[0].Query, "attempts + 1 >=") {
		t.Fatalf("MarkFailed 須以嘗試次數判斷死信：%+v", fake.Calls)
	}
}

func TestPGStore_MarkFailedNoRowsIsNotDead(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	dead, err := NewPGStore(fake).MarkFailed(context.Background(), "e-1", 3)
	if err != nil || dead {
		t.Fatalf("查無列時 MarkFailed dead=%v err=%v, want dead=false", dead, err)
	}
}
