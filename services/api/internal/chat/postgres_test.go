package chat

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
)

var fixedTime = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

func TestPGStore_GetConversation_MapsNulls(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "FROM conversations") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: []any{
			"c-1", "direct", nil, "u-a", nil, nil, fixedTime, fixedTime,
		}}
	}
	c, err := NewPGStore(fake).GetConversation(context.Background(), "c-1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if c == nil || c.Type != TypeDirect || c.Title != "" || c.LastMessageID != "" || c.LastMessageAt != nil {
		t.Fatalf("conversation = %+v", c)
	}

	fake2 := &postgrestest.FakeQuerier{}
	fake2.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	c, err = NewPGStore(fake2).GetConversation(context.Background(), "missing")
	if err != nil || c != nil {
		t.Fatalf("missing = %+v, %v, want nil, nil", c, err)
	}
}

func TestPGStore_SaveConversation_ZeroRows(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return postgrestest.FakeResult{Affected: 0}, nil
	}
	err := NewPGStore(fake).SaveConversation(context.Background(), &Conversation{
		ID: "c-1", Type: TypeDirect, CreatedAt: fixedTime, UpdatedAt: fixedTime,
	})
	if err != ErrConversationNotFound {
		t.Fatalf("err = %v, want ErrConversationNotFound", err)
	}
}

func TestPGStore_GetDirectBetween_UsesTwoMemberChecks(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "conversation_members m1") || !strings.Contains(query, "conversation_members m2") {
			t.Errorf("query 須以兩名成員反查：%s", query)
		}
		if args[0] != "u-a" || args[1] != "u-b" {
			t.Errorf("args = %v, want [u-a u-b]", args)
		}
		return postgrestest.FakeRow{Values: []any{
			"c-1", "direct", nil, "u-a", nil, nil, fixedTime, fixedTime,
		}}
	}
	c, err := NewPGStore(fake).GetDirectBetween(context.Background(), "u-a", "u-b")
	if err != nil || c == nil || c.ID != "c-1" {
		t.Fatalf("GetDirectBetween = %+v, %v", c, err)
	}
}

func TestPGStore_SaveMember_MapsReadWatermark(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	var gotArgs []any
	fake.ExecFunc = func(_ string, args []any) (sql.Result, error) {
		gotArgs = args
		return postgrestest.FakeResult{Affected: 1}, nil
	}
	readAt := fixedTime
	err := NewPGStore(fake).SaveMember(context.Background(), &Member{
		ConversationID: "c-1", UserID: "u-a", Role: RoleAdmin,
		JoinedAt: fixedTime, LastReadMessageID: "m-9", LastReadSequence: 9,
		LastReadAt: &readAt, UpdatedAt: fixedTime,
	})
	if err != nil {
		t.Fatalf("SaveMember: %v", err)
	}
	if gotArgs[6] != int64(9) || gotArgs[5] != "m-9" {
		t.Fatalf("args = %v, want 水位欄位在序號前", gotArgs)
	}
}
