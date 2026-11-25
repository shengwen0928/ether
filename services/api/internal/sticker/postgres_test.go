package sticker

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

func TestPGStore_GetPack_MapsNulls(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "FROM sticker_packs") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: []any{
			"p-1", "u-a", "包", "描述", nil, nil, true, "published", fixedTime, fixedTime,
		}}
	}
	p, err := NewPGStore(fake).GetPack(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("GetPack: %v", err)
	}
	if p == nil || p.Name != "包" || p.Category != "" || p.CoverMediaID != "" || p.Status != PackStatusPublished {
		t.Fatalf("pack = %+v", p)
	}

	fake2 := &postgrestest.FakeQuerier{}
	fake2.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	p, err = NewPGStore(fake2).GetPack(context.Background(), "missing")
	if err != nil || p != nil {
		t.Fatalf("missing = %+v, %v, want nil, nil", p, err)
	}
}

func TestPGStore_SavePack_ZeroRowsReturnsNotFound(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return postgrestest.FakeResult{Affected: 0}, nil
	}
	err := NewPGStore(fake).SavePack(context.Background(), &Pack{ID: "p-1"})
	if !errors.Is(err, ErrPackNotFound) {
		t.Fatalf("SavePack 0 列應回 ErrPackNotFound，現為 %v", err)
	}
}

func TestPGStore_ListPacks_CursorQuery(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "ORDER BY created_at DESC, id DESC") {
			t.Errorf("unexpected query: %s", query)
		}
		if !strings.Contains(query, "status <> 'archived'") {
			t.Errorf("清單須排除已封存包：%s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{}}, nil
	}
	packs, err := NewPGStore(fake).ListPacks(context.Background(), "", 10)
	if err != nil {
		t.Fatalf("ListPacks: %v", err)
	}
	if len(packs) != 0 {
		t.Fatalf("packs = %d, want 0", len(packs))
	}
}

func TestPGStore_NextStickerPosition_Empty(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "MAX(position)") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: []any{nil}}
	}
	pos, err := NewPGStore(fake).NextStickerPosition(context.Background(), "p-1")
	if err != nil || pos != 0 {
		t.Fatalf("空包序號 = %d, %v, want 0", pos, err)
	}
}

func TestPGStore_SaveSticker_ZeroRowsReturnsNotFound(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return postgrestest.FakeResult{Affected: 0}, nil
	}
	err := NewPGStore(fake).SaveSticker(context.Background(), &Sticker{ID: "s-1"})
	if !errors.Is(err, ErrStickerNotFound) {
		t.Fatalf("SaveSticker 0 列應回 ErrStickerNotFound，現為 %v", err)
	}
}

func TestPGStore_GetUserSticker_MapsNulls(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "FROM user_stickers") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: []any{
			"u-1", "u-a", "p-1", nil, "granted", nil, false, fixedTime, fixedTime,
		}}
	}
	row, err := NewPGStore(fake).GetUserStickerByPack(context.Background(), "u-a", "p-1")
	if err != nil {
		t.Fatalf("GetUserStickerByPack: %v", err)
	}
	if row == nil || row.Source != SourceGranted || row.StickerID != "" || row.LastUsedAt != nil {
		t.Fatalf("row = %+v", row)
	}
}

func TestPGStore_PruneRecent_SQL(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if !strings.Contains(query, "OFFSET") {
			t.Errorf("裁剪須以 OFFSET 保留最新列：%s", query)
		}
		return postgrestest.FakeResult{Affected: 1}, nil
	}
	if err := NewPGStore(fake).PruneRecent(context.Background(), "u-a", 30); err != nil {
		t.Fatalf("PruneRecent: %v", err)
	}
}
