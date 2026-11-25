package message

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
)

var fixedTime = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

func TestPGStore_CreateMessage_AssignsSequenceAndReadsItBack(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	var sawInsert, sawSelect bool
	fake.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if strings.Contains(query, "INSERT INTO messages") {
			sawInsert = true
			if !strings.Contains(query, "COALESCE(MAX(sequence_number), 0) + 1") {
				t.Errorf("序號須以子查詢分配：%s", query)
			}
			return postgrestest.FakeResult{Affected: 1}, nil
		}
		return postgrestest.FakeResult{Affected: 1}, nil
	}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if strings.Contains(query, "SELECT sequence_number FROM messages") {
			sawSelect = true
			return postgrestest.FakeRow{Values: []any{int64(7)}}
		}
		return postgrestest.FakeRow{Values: []any{int64(7)}}
	}
	m := &Message{
		ID: "m-1", ConversationID: "c-1", SenderID: "u-a",
		Kind: KindText, Content: "你好", Status: StatusSent,
		CreatedAt: fixedTime, UpdatedAt: fixedTime,
	}
	if err := NewPGStore(fake).CreateMessage(context.Background(), m); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	if !sawInsert || !sawSelect || m.SequenceNumber != 7 {
		t.Fatalf("insert=%v select=%v seq=%d", sawInsert, sawSelect, m.SequenceNumber)
	}
}

func TestPGStore_CreateMessage_RetriesSequenceConflict(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	attempts := 0
	seqErr := &pq.Error{Code: "23505", Constraint: "messages_conversation_id_sequence_number_key"}
	fake.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if strings.Contains(query, "INSERT INTO messages") {
			attempts++
			if attempts < 3 {
				return nil, seqErr
			}
			return postgrestest.FakeResult{Affected: 1}, nil
		}
		return postgrestest.FakeResult{Affected: 1}, nil
	}
	fake.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Values: []any{int64(3)}}
	}
	m := &Message{
		ID: "m-1", ConversationID: "c-1", SenderID: "u-a",
		Kind: KindText, Content: "你好", Status: StatusSent,
		CreatedAt: fixedTime, UpdatedAt: fixedTime,
	}
	if err := NewPGStore(fake).CreateMessage(context.Background(), m); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestPGStore_CreateMessage_MapsClientMessageConflict(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return nil, &pq.Error{Code: "23505", Constraint: "messages_client_message_unique_idx"}
	}
	m := &Message{
		ID: "m-1", ConversationID: "c-1", SenderID: "u-a",
		Kind: KindText, Content: "你好", Status: StatusSent, ClientMessageID: "client-1",
		CreatedAt: fixedTime, UpdatedAt: fixedTime,
	}
	err := NewPGStore(fake).CreateMessage(context.Background(), m)
	if !errors.Is(err, ErrDuplicateClientMessage) {
		t.Fatalf("err = %v, want ErrDuplicateClientMessage", err)
	}
}

func TestPGStore_ListBefore_OrdersDesc(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "ORDER BY m.sequence_number DESC") {
			t.Errorf("ListBefore 須倒序：%s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{
			{"m-2", "c-1", "u-a", "text", "二", "sent", nil, int64(2), nil, nil, nil, nil, 0, fixedTime, fixedTime, nil},
		}}, nil
	}
	msgs, err := NewPGStore(fake).ListBefore(context.Background(), "c-1", 5, 10)
	if err != nil {
		t.Fatalf("ListBefore: %v", err)
	}
	if len(msgs) != 1 || msgs[0].SequenceNumber != 2 {
		t.Fatalf("msgs = %+v", msgs)
	}
}

func TestPGStore_CountAfter_ExcludesOwnAndRecalled(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "status <> 'recalled'") || !strings.Contains(query, "IS DISTINCT FROM") {
			t.Errorf("CountAfter 須排除自己與收回訊息：%s", query)
		}
		return postgrestest.FakeRow{Values: []any{3}}
	}
	n, err := NewPGStore(fake).CountAfter(context.Background(), "c-1", "u-b", 2)
	if err != nil || n != 3 {
		t.Fatalf("CountAfter = %d, %v", n, err)
	}
}
