package auth

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

func sessionRow() []any {
	// id, user_id, device_id, keycloak_session_id, refresh_token_hash,
	// expires_at, last_active_at, is_revoked, revoked_at, revoked_reason,
	// created_at, updated_at
	return []any{"sess-1", "user-1", "dev-1", "kc-sid-1", "hash-1", fixedTime, fixedTime, false, nil, nil, fixedTime, fixedTime}
}

func TestPGSessionStore_Get_MapsRow(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "FROM user_sessions") || !strings.Contains(query, "id = $1::uuid") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: sessionRow()}
	}
	s, err := NewPGSessionStore(fake).Get(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.DeviceID != "dev-1" || s.KeycloakSessionID != "kc-sid-1" || s.Revoked() || s.LastActiveAt.IsZero() {
		t.Fatalf("session = %+v", s)
	}
}

func TestPGSessionStore_Get_RevokedFlagWithoutTimestamp(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	row := sessionRow()
	row[7] = true // is_revoked
	fake.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Values: row}
	}
	s, err := NewPGSessionStore(fake).Get(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !s.Revoked() || s.RevokedAt == nil || !s.RevokedAt.Equal(fixedTime) {
		t.Fatalf("revoked synthesis = revoked %v at %v, want revoked with updated_at", s.Revoked(), s.RevokedAt)
	}
}

func TestPGSessionStore_GetByKeycloakSessionID_EmptySkipsQuery(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	if _, err := NewPGSessionStore(fake).GetByKeycloakSessionID(context.Background(), ""); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("err = %v, want ErrSessionNotFound", err)
	}
	if len(fake.Calls) != 0 {
		t.Fatalf("calls = %d, want 0", len(fake.Calls))
	}
}

func TestPGSessionStore_Create_Nullables(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if !strings.Contains(query, "INSERT INTO user_sessions") {
			t.Errorf("unexpected query: %s", query)
		}
		// device_id, keycloak_session_id, revoked_at, revoked_reason 皆為 NULL。
		if args[2] != nil || args[3] != nil || args[8] != nil || args[9] != nil {
			t.Errorf("nullable args = %v %v %v %v, want all nil", args[2], args[3], args[8], args[9])
		}
		return postgrestest.FakeResult{Affected: 1}, nil
	}
	sess := &Session{ID: "sess-1", UserID: "user-1", RefreshTokenHash: "hash-1", ExpiresAt: fixedTime, LastActiveAt: fixedTime, CreatedAt: fixedTime, UpdatedAt: fixedTime}
	if err := NewPGSessionStore(fake).Create(context.Background(), sess); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestPGSessionStore_ConsumedHashes(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "session_refresh_hashes") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: []any{true}}
	}
	consumed, err := NewPGSessionStore(fake).IsHashConsumed(context.Background(), "sess-1", "hash-1")
	if err != nil || !consumed {
		t.Fatalf("IsHashConsumed = %v, %v, want true, nil", consumed, err)
	}

	fake2 := &postgrestest.FakeQuerier{}
	fake2.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if !strings.Contains(query, "ON CONFLICT DO NOTHING") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeResult{Affected: 0}, nil
	}
	inserted, err := NewPGSessionStore(fake2).MarkHashConsumed(context.Background(), "sess-1", "hash-1")
	if err != nil || inserted {
		t.Fatalf("MarkHashConsumed duplicate = %v, %v, want false, nil", inserted, err)
	}
}

func TestPGSessionStore_Save_MissingRow(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return postgrestest.FakeResult{Affected: 0}, nil
	}
	sess := &Session{ID: "missing", UserID: "user-1", RefreshTokenHash: "hash-1", ExpiresAt: fixedTime, UpdatedAt: fixedTime}
	if err := NewPGSessionStore(fake).Save(context.Background(), sess); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("err = %v, want ErrSessionNotFound", err)
	}
}
