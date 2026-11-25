package timeline

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
)

var fixedTime = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

func TestPGStore_GetPost_MapsNulls(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "FROM posts") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: []any{
			"p-1", "u-a", "內容", "public", nil, false, nil, fixedTime, fixedTime,
		}}
	}
	p, err := NewPGStore(fake).GetPost(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("GetPost: %v", err)
	}
	if p == nil || p.Content != "內容" || p.Visibility != VisibilityPublic || p.EditedAt != nil || p.DeletedAt != nil {
		t.Fatalf("post = %+v", p)
	}

	fake2 := &postgrestest.FakeQuerier{}
	fake2.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	p, err = NewPGStore(fake2).GetPost(context.Background(), "missing")
	if err != nil || p != nil {
		t.Fatalf("missing = %+v, %v, want nil, nil", p, err)
	}
}

func TestPGStore_SavePost_ZeroRowsReturnsNotFound(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return postgrestest.FakeResult{Affected: 0}, nil
	}
	err := NewPGStore(fake).SavePost(context.Background(), &Post{ID: "p-1"})
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("SavePost 0 列應回 ErrPostNotFound，現為 %v", err)
	}
}

func TestPGStore_ListRecentPosts_CursorQuery(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "ORDER BY created_at DESC, id DESC") {
			t.Errorf("unexpected query: %s", query)
		}
		if !strings.Contains(query, "(created_at, id) < ") {
			t.Errorf("游標查詢須以元組比較分頁: %s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{
			{"p-2", "u-a", "二", "public", nil, false, nil, fixedTime, fixedTime},
		}}, nil
	}
	cursor := encodeCursor(fixedTime, "p-9")
	rows, err := NewPGStore(fake).ListRecentPosts(context.Background(), cursor, 10)
	if err != nil {
		t.Fatalf("ListRecentPosts: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "p-2" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestPGStore_GetLike_MissingReturnsNil(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	l, err := NewPGStore(fake).GetLike(context.Background(), "p-1", "u-b")
	if err != nil || l != nil {
		t.Fatalf("like = %+v, %v, want nil, nil", l, err)
	}
}

func TestPGStore_ListComments_AscendingCursor(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "ORDER BY created_at ASC, id ASC") {
			t.Errorf("unexpected query: %s", query)
		}
		if !strings.Contains(query, "(created_at, id) > ") {
			t.Errorf("留言游標查詢須以元組比較分頁: %s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{
			{"c-1", "p-1", "u-b", nil, "hi", nil, false, nil, fixedTime, fixedTime},
		}}, nil
	}
	cursor := encodeCursor(fixedTime, "c-0")
	rows, err := NewPGStore(fake).ListComments(context.Background(), "p-1", cursor, 10)
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if len(rows) != 1 || rows[0].ParentCommentID != "" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestPGStore_CountQueries(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if strings.Contains(query, "FROM post_likes") && !strings.Contains(query, "unliked_at IS NULL") {
			t.Errorf("按讚計數須只計生效列: %s", query)
		}
		if strings.Contains(query, "FROM post_comments") && !strings.Contains(query, "NOT is_deleted") {
			t.Errorf("留言計數須排除已刪除: %s", query)
		}
		return postgrestest.FakeRow{Values: []any{3}}
	}
	n, err := NewPGStore(fake).CountActiveLikes(context.Background(), "p-1")
	if err != nil || n != 3 {
		t.Fatalf("CountActiveLikes = %d, %v", n, err)
	}
	n, err = NewPGStore(fake).CountVisibleComments(context.Background(), "p-1")
	if err != nil || n != 3 {
		t.Fatalf("CountVisibleComments = %d, %v", n, err)
	}
}
