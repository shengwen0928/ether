package friend

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

func TestPGStore_GetFriendship_OrdersPair(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "FROM friendships") {
			t.Errorf("unexpected query: %s", query)
		}
		if args[0] != "u-a" || args[1] != "u-b" {
			t.Errorf("args = %v, want ordered pair u-a/u-b", args)
		}
		return postgrestest.FakeRow{Values: []any{"f-1", "u-a", "u-b", "active", fixedTime, nil, fixedTime, fixedTime}}
	}
	f, err := NewPGStore(fake).GetFriendship(context.Background(), "u-b", "u-a")
	if err != nil {
		t.Fatalf("GetFriendship: %v", err)
	}
	if f == nil || f.FirstID != "u-a" || f.OtherID("u-a") != "u-b" || f.EstablishedAt == nil {
		t.Fatalf("friendship = %+v", f)
	}

	fake2 := &postgrestest.FakeQuerier{}
	fake2.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	f, err = NewPGStore(fake2).GetFriendship(context.Background(), "u-a", "u-b")
	if err != nil || f != nil {
		t.Fatalf("missing = %+v, %v, want nil, nil", f, err)
	}
}

func TestPGStore_CreateRequest_MapsPendingUnique(t *testing.T) {
	pqErr := &pq.Error{Code: "23505", Constraint: "friend_requests_pending_unique_idx"}
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return nil, pqErr
	}
	err := NewPGStore(fake).CreateRequest(context.Background(), &FriendRequest{
		ID: "r-1", RequesterID: "u-a", AddresseeID: "u-b",
		Status: RequestPending, CreatedAt: fixedTime, UpdatedAt: fixedTime,
	})
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Fields[0].Field != "addressee_id" {
		t.Fatalf("err = %v, want ValidationError on addressee_id", err)
	}
}

func TestPGStore_SaveRequest_NotFound(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return postgrestest.FakeResult{Affected: 0}, nil
	}
	err := NewPGStore(fake).SaveRequest(context.Background(), &FriendRequest{
		ID: "r-missing", Status: RequestAccepted, UpdatedAt: fixedTime,
	})
	if !errors.Is(err, ErrRequestNotFound) {
		t.Fatalf("err = %v, want ErrRequestNotFound", err)
	}
}

func TestPGStore_BlockedBetween_MapsBool(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "SELECT EXISTS") || !strings.Contains(query, "blocked_users") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: []any{true}}
	}
	blocked, err := NewPGStore(fake).BlockedBetween(context.Background(), "u-a", "u-b")
	if err != nil || !blocked {
		t.Fatalf("blocked = %v, %v", blocked, err)
	}
}

func TestPGStore_ListFriendships_MapsRows(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "FROM friendships") || !strings.Contains(query, "status = 'active'") {
			t.Errorf("unexpected query: %s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{
			{"f-1", "u-a", "u-b", "active", fixedTime, nil, fixedTime, fixedTime},
		}}, nil
	}
	rows, err := NewPGStore(fake).ListFriendships(context.Background(), "u-b")
	if err != nil {
		t.Fatalf("ListFriendships: %v", err)
	}
	if len(rows) != 1 || rows[0].OtherID("u-b") != "u-a" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestPGStore_GetRequest_NotFound(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	if _, err := NewPGStore(fake).GetRequest(context.Background(), "missing"); !errors.Is(err, ErrRequestNotFound) {
		t.Fatalf("err = %v, want ErrRequestNotFound", err)
	}
}
