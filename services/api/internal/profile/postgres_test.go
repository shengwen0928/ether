package profile

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

func TestPGStore_GetByUserID_MapsRow(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "FROM user_profiles") || !strings.Contains(query, "user_id = $1::uuid") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: []any{"user-1", "阿哲", "azhe", "media-1", "你好", "忙碌中", fixedTime, fixedTime}}
	}
	p, err := NewPGStore(fake).GetByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}
	if p.PublicID != "azhe" || p.AvatarMediaID != "media-1" || p.Bio != "你好" || p.StatusMessage != "忙碌中" {
		t.Fatalf("profile = %+v", p)
	}
}

func TestPGStore_GetByUserID_NullsAndNotFound(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Values: []any{"user-1", "阿哲", nil, nil, nil, nil, fixedTime, fixedTime}}
	}
	p, err := NewPGStore(fake).GetByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}
	if p.PublicID != "" || p.Bio != "" {
		t.Fatalf("null mapping = %+v", p)
	}

	fake2 := &postgrestest.FakeQuerier{}
	fake2.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	if _, err := NewPGStore(fake2).GetByUserID(context.Background(), "missing"); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("err = %v, want ErrProfileNotFound", err)
	}
	if _, err := NewPGStore(fake2).GetByPublicID(context.Background(), ""); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("empty public_id err = %v, want ErrProfileNotFound", err)
	}
	if len(fake2.Calls) != 1 {
		t.Fatalf("calls = %d, want 1 (empty public_id must not query)", len(fake2.Calls))
	}
}

func TestPGStore_Create_PublicIDConflict(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if !strings.Contains(query, "INSERT INTO user_profiles") {
			t.Errorf("unexpected query: %s", query)
		}
		if args[2] != nil {
			t.Errorf("public_id arg = %v, want nil (empty writes NULL)", args[2])
		}
		return nil, &pq.Error{Code: "23505", Constraint: "user_profiles_public_id_key"}
	}
	err := NewPGStore(fake).Create(context.Background(), &Profile{UserID: "user-1", DisplayName: "阿哲", CreatedAt: fixedTime, UpdatedAt: fixedTime})
	var ve *ValidationError
	if !errors.As(err, &ve) || len(ve.Fields) != 1 || ve.Fields[0].Field != "public_id" || ve.Fields[0].Issue != "DUPLICATE" {
		t.Fatalf("err = %v (%T), want ValidationError DUPLICATE public_id", err, err)
	}
}

func TestPGStore_Save_MissingRow(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return postgrestest.FakeResult{Affected: 0}, nil
	}
	err := NewPGStore(fake).Save(context.Background(), &Profile{UserID: "missing", DisplayName: "X", UpdatedAt: fixedTime})
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("err = %v, want ErrProfileNotFound", err)
	}
}
