package user

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

func userRow() []any {
	return []any{"user-id-1", "kc-subject-1", "a@example.invalid", true, "active", fixedTime, fixedTime, fixedTime, fixedTime}
}

func TestPGUserStore_GetByID_MapsRow(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "FROM users") || !strings.Contains(query, "id = $1::uuid") {
			t.Errorf("unexpected query: %s", query)
		}
		if len(args) != 1 || args[0] != "user-id-1" {
			t.Errorf("args = %v, want [user-id-1]", args)
		}
		return postgrestest.FakeRow{Values: userRow()}
	}
	u, err := NewPGUserStore(fake).GetByID(context.Background(), "user-id-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if u.ID != "user-id-1" || u.KeycloakSubject != "kc-subject-1" || u.Email != "a@example.invalid" ||
		!u.EmailVerified || u.Status != StatusActive || u.LastLoginAt == nil {
		t.Fatalf("mapped user = %+v", u)
	}
}

func TestPGUserStore_GetByID_NullableColumns(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Values: []any{"user-id-1", "kc-subject-1", nil, false, "disabled", fixedTime, nil, fixedTime, fixedTime}}
	}
	u, err := NewPGUserStore(fake).GetByID(context.Background(), "user-id-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if u.Email != "" || u.LastLoginAt != nil || u.Status != StatusDisabled {
		t.Fatalf("nullable mapping = email %q lastLogin %v status %q", u.Email, u.LastLoginAt, u.Status)
	}
}

func TestPGUserStore_GetByID_NotFound(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	if _, err := NewPGUserStore(fake).GetByID(context.Background(), "missing"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}

func TestPGUserStore_GetByEmail_EmptySkipsQuery(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	if _, err := NewPGUserStore(fake).GetByEmail(context.Background(), ""); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
	if len(fake.Calls) != 0 {
		t.Fatalf("calls = %d, want 0 (empty email must not query)", len(fake.Calls))
	}
}

func TestPGUserStore_Create_MapsEmailConflict(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if !strings.Contains(query, "INSERT INTO users") {
			t.Errorf("unexpected query: %s", query)
		}
		if args[2] != nil {
			t.Errorf("email arg = %v, want nil (empty email writes NULL)", args[2])
		}
		return nil, &pq.Error{Code: "23505", Constraint: "users_email_key"}
	}
	u := &User{ID: "user-id-1", KeycloakSubject: "kc-1", Status: StatusActive, RegisteredAt: fixedTime, CreatedAt: fixedTime, UpdatedAt: fixedTime}
	err := NewPGUserStore(fake).Create(context.Background(), u)
	var ve *ValidationError
	if !errors.As(err, &ve) || len(ve.Fields) != 1 || ve.Fields[0].Field != "email" || ve.Fields[0].Issue != "DUPLICATE" {
		t.Fatalf("err = %v (%T), want ValidationError DUPLICATE email", err, err)
	}
}

func TestPGUserStore_Create_SubjectConflictIsServerError(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return nil, &pq.Error{Code: "23505", Constraint: "users_keycloak_subject_key"}
	}
	u := &User{ID: "user-id-1", KeycloakSubject: "kc-1", Status: StatusActive, RegisteredAt: fixedTime, CreatedAt: fixedTime, UpdatedAt: fixedTime}
	err := NewPGUserStore(fake).Create(context.Background(), u)
	var ve *ValidationError
	if errors.As(err, &ve) {
		t.Fatalf("err = %v, subject conflict must not be a ValidationError", err)
	}
	if err == nil {
		t.Fatal("err = nil, want subject conflict error")
	}
}

func TestPGUserStore_Save_MissingRowIsNotFound(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return postgrestest.FakeResult{Affected: 0}, nil
	}
	u := &User{ID: "missing", KeycloakSubject: "kc-1", Status: StatusActive, UpdatedAt: fixedTime}
	if err := NewPGUserStore(fake).Save(context.Background(), u); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}

func TestPGSettingsStore_RoundTrip(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Values: []any{"user-id-1", "zh-Hant", "system", nil, nil, fixedTime, fixedTime}}
	}
	st, err := NewPGSettingsStore(fake).GetByUserID(context.Background(), "user-id-1")
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}
	if st.FontSize != "" || st.ChatBackground != "" || st.Language != "zh-Hant" {
		t.Fatalf("settings = %+v", st)
	}

	fake2 := &postgrestest.FakeQuerier{}
	fake2.ExecFunc = func(query string, args []any) (sql.Result, error) {
		if !strings.Contains(query, "UPDATE user_settings") {
			t.Errorf("unexpected query: %s", query)
		}
		if args[3] != nil || args[4] != nil {
			t.Errorf("nullable args = %v, %v, want nil, nil", args[3], args[4])
		}
		return postgrestest.FakeResult{Affected: 1}, nil
	}
	if err := NewPGSettingsStore(fake2).Save(context.Background(), st); err != nil {
		t.Fatalf("Save: %v", err)
	}

	fake3 := &postgrestest.FakeQuerier{}
	fake3.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	if _, err := NewPGSettingsStore(fake3).GetByUserID(context.Background(), "missing"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}
