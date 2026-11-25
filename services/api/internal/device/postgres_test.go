package device

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

func deviceRow() []any {
	return []any{"dev-1", "user-1", "我的手機", "android", "aabbcc", fixedTime, true, false, nil, fixedTime, fixedTime}
}

func TestPGStore_Get_MapsRow(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "FROM user_devices") || !strings.Contains(query, "id = $1::uuid") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: deviceRow()}
	}
	d, err := NewPGStore(fake).Get(context.Background(), "dev-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if d.DeviceName != "我的手機" || !d.Trusted || d.Revoked || d.RevokedAt != nil || d.LastActiveAt == nil {
		t.Fatalf("device = %+v", d)
	}
}

func TestPGStore_ListByUser_IteratesRows(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "ORDER BY created_at ASC") {
			t.Errorf("unexpected query: %s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{deviceRow(), deviceRow()}}, nil
	}
	list, err := NewPGStore(fake).ListByUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
}

func TestPGStore_Create_NullTimes(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if !strings.Contains(query, "INSERT INTO user_devices") {
			t.Errorf("unexpected query: %s", query)
		}
		if args[5] != nil || args[8] != nil {
			t.Errorf("time args = %v, %v, want nil, nil", args[5], args[8])
		}
		return postgrestest.FakeResult{Affected: 1}, nil
	}
	d := &Device{ID: "dev-1", UserID: "user-1", DeviceName: "手機", Platform: "android", DeviceIdentifierHash: "aabbcc", CreatedAt: fixedTime, UpdatedAt: fixedTime}
	if err := NewPGStore(fake).Create(context.Background(), d); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestPGStore_Save_MissingRow(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return postgrestest.FakeResult{Affected: 0}, nil
	}
	d := &Device{ID: "missing", UserID: "user-1", DeviceName: "X", Platform: "android", DeviceIdentifierHash: "aabbcc", UpdatedAt: fixedTime}
	if err := NewPGStore(fake).Save(context.Background(), d); !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("err = %v, want ErrDeviceNotFound", err)
	}
}

func TestPGStore_Get_NotFound(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	if _, err := NewPGStore(fake).Get(context.Background(), "missing"); !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("err = %v, want ErrDeviceNotFound", err)
	}
}
