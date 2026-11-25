package message

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
)

func TestPGStore_SetReaction_ReplacesAndReadsBack(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	var sawUpdate, sawInsert bool
	fake.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if strings.Contains(query, "UPDATE message_reactions") {
			sawUpdate = true
			if !strings.Contains(query, "removed_at IS NULL AND reaction <> $3") {
				t.Errorf("更換反應須只標記其他表情的生效列：%s", query)
			}
		}
		if strings.Contains(query, "INSERT INTO message_reactions") {
			sawInsert = true
			if !strings.Contains(query, "ON CONFLICT (message_id, user_id, reaction) WHERE removed_at IS NULL DO NOTHING") {
				t.Errorf("寫入須以部分唯一索引收斂併發重複：%s", query)
			}
		}
		return postgrestest.FakeResult{Affected: 1}, nil
	}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		return postgrestest.FakeRow{Values: []any{"r-1", "m-1", "u-b", "😂", fixedTime, fixedTime, nil}}
	}
	r := &Reaction{
		ID: "r-1", MessageID: "m-1", UserID: "u-b", Emoji: "😂",
		CreatedAt: fixedTime, UpdatedAt: fixedTime,
	}
	got, err := NewPGStore(fake).SetReaction(context.Background(), r)
	if err != nil {
		t.Fatalf("SetReaction: %v", err)
	}
	if !sawUpdate || !sawInsert || got == nil || got.Emoji != "😂" {
		t.Fatalf("update=%v insert=%v got=%+v", sawUpdate, sawInsert, got)
	}
}

func TestPGStore_RemoveReaction_MarksRemoved(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	var sawUpdate bool
	fake.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if strings.Contains(query, "UPDATE message_reactions") && strings.Contains(query, "removed_at = now()") {
			sawUpdate = true
		}
		return postgrestest.FakeResult{Affected: 1}, nil
	}
	if err := NewPGStore(fake).RemoveReaction(context.Background(), "m-1", "u-b"); err != nil {
		t.Fatalf("RemoveReaction: %v", err)
	}
	if !sawUpdate {
		t.Fatal("取消反應須以標記寫入，不得物理刪除")
	}
}

func TestPGStore_ListActiveReactions_OnlyActive(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "removed_at IS NULL") {
			t.Errorf("彙總須只讀生效列：%s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{
			{"r-1", "m-1", "u-b", "👍", fixedTime, fixedTime, nil},
			{"r-2", "m-1", "u-a", "👍", fixedTime, fixedTime, nil},
		}}, nil
	}
	rows, err := NewPGStore(fake).ListActiveReactions(context.Background(), "m-1")
	if err != nil || len(rows) != 2 {
		t.Fatalf("ListActiveReactions = %d, %v", len(rows), err)
	}
}

func TestPGStore_PinMessage_InsertsNewAndReturnsExisting(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	var sawInsert bool
	fake.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if strings.Contains(query, "INSERT INTO message_pins") {
			sawInsert = true
		}
		return postgrestest.FakeResult{Affected: 1}, nil
	}
	// 第一輪：查無生效列 → 寫入新列。
	fake.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	p := &Pin{
		ID: "p-1", ConversationID: "c-1", MessageID: "m-1", PinnedBy: "u-a",
		PinnedAt: fixedTime, CreatedAt: fixedTime, UpdatedAt: fixedTime,
	}
	got, isNew, err := NewPGStore(fake).PinMessage(context.Background(), p)
	if err != nil || !isNew || !sawInsert || got != p {
		t.Fatalf("新建置頂：got=%+v isNew=%v sawInsert=%v err=%v", got, isNew, sawInsert, err)
	}
	// 第二輪：生效列已存在 → 回傳既有列，不寫入。
	fake.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Values: []any{"p-0", "c-1", "m-1", "u-b", fixedTime, nil, fixedTime, fixedTime}}
	}
	sawInsert = false
	got, isNew, err = NewPGStore(fake).PinMessage(context.Background(), p)
	if err != nil || isNew || sawInsert || got == nil || got.ID != "p-0" {
		t.Fatalf("重複置頂：got=%+v isNew=%v sawInsert=%v err=%v", got, isNew, sawInsert, err)
	}
}

func TestPGStore_UnpinMessage_ReturnsCancelledRow(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "RETURNING") || !strings.Contains(query, "unpinned_at IS NULL") {
			t.Errorf("取消置頂須以 RETURNING 讀回被取消的列：%s", query)
		}
		return postgrestest.FakeRow{Values: []any{"p-1", "c-1", "m-1", "u-a", fixedTime, fixedTime, fixedTime, fixedTime}}
	}
	got, err := NewPGStore(fake).UnpinMessage(context.Background(), "c-1", "m-1", fixedTime)
	if err != nil || got == nil || got.UnpinnedAt == nil {
		t.Fatalf("UnpinMessage = %+v, %v", got, err)
	}
}

func TestPGStore_ListActivePins_OrdersByPinnedAt(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "ORDER BY pinned_at") || !strings.Contains(query, "unpinned_at IS NULL") {
			t.Errorf("置頂清單須依置頂時間順序且只含生效列：%s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{
			{"p-1", "c-1", "m-1", "u-a", fixedTime, nil, fixedTime, fixedTime},
		}}, nil
	}
	pins, err := NewPGStore(fake).ListActivePins(context.Background(), "c-1")
	if err != nil || len(pins) != 1 || pins[0].MessageID != "m-1" {
		t.Fatalf("ListActivePins = %+v, %v", pins, err)
	}
}
