package message

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
)

func TestPGStore_CreateAttachments_UsesConflictConvergence(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	var sawInsert bool
	fake.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if strings.Contains(query, "INSERT INTO message_attachments") {
			sawInsert = true
			if !strings.Contains(query, "ON CONFLICT (message_id, media_file_id) DO NOTHING") {
				t.Errorf("附件寫入須以唯一索引收斂重複關聯：%s", query)
			}
		}
		return postgrestest.FakeResult{Affected: 1}, nil
	}
	atts := []*Attachment{{
		ID: "a-1", MessageID: "m-1", MediaFileID: "f-1",
		MIMEType: "image/jpeg", SizeBytes: 100, FileName: "photo.jpg", CreatedAt: fixedTime,
	}}
	if err := NewPGStore(fake).CreateAttachments(context.Background(), "m-1", atts); err != nil {
		t.Fatalf("CreateAttachments: %v", err)
	}
	if !sawInsert {
		t.Fatal("未見 message_attachments 寫入")
	}
}

func TestPGStore_ListAttachments_SelectsNamedColumns(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "FROM message_attachments") || !strings.Contains(query, "ORDER BY created_at, id::text") {
			t.Errorf("列出附件須具名欄位並依建立時間排序：%s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{
			{"a-1", "m-1", "f-1", "image/jpeg", int64(100), "photo.jpg", fixedTime},
		}}, nil
	}
	got, err := NewPGStore(fake).ListAttachments(context.Background(), "m-1")
	if err != nil {
		t.Fatalf("ListAttachments: %v", err)
	}
	if len(got) != 1 || got[0].MediaFileID != "f-1" || got[0].SizeBytes != 100 {
		t.Fatalf("got = %+v", got)
	}
}

func TestPGStore_GetAttachment_NotFoundReturnsNil(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	got, err := NewPGStore(fake).GetAttachment(context.Background(), "m-1", "f-1")
	if err != nil || got != nil {
		t.Fatalf("got = %+v, err = %v", got, err)
	}
}

func TestPGStore_CreateForward_UsesConflictConvergence(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	var sawInsert bool
	fake.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if strings.Contains(query, "INSERT INTO message_forwards") {
			sawInsert = true
			if !strings.Contains(query, "ON CONFLICT (forwarded_message_id) DO NOTHING") {
				t.Errorf("轉傳紀錄寫入須以唯一索引收斂重複寫入：%s", query)
			}
		}
		return postgrestest.FakeResult{Affected: 1}, nil
	}
	fwd := &Forward{
		ID: "fw-1", SourceMessageID: "m-1", ForwardedMessageID: "m-2",
		SourceConversationID: "c-1", TargetConversationID: "c-2",
		ForwardedBy: "u-a", CreatedAt: fixedTime,
	}
	if err := NewPGStore(fake).CreateForward(context.Background(), fwd); err != nil {
		t.Fatalf("CreateForward: %v", err)
	}
	if !sawInsert {
		t.Fatal("未見 message_forwards 寫入")
	}
}

func TestPGStore_GetForwardByForwardedID_NotFoundReturnsNil(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	got, err := NewPGStore(fake).GetForwardByForwardedID(context.Background(), "m-2")
	if err != nil || got != nil {
		t.Fatalf("got = %+v, err = %v", got, err)
	}
}
