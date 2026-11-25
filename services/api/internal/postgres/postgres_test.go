package postgres

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/lib/pq"
)

func TestIsUniqueViolation(t *testing.T) {
	if !IsUniqueViolation(&pq.Error{Code: "23505", Constraint: "users_email_key"}) {
		t.Fatal("23505 must be a unique violation")
	}
	if IsUniqueViolation(&pq.Error{Code: "23503"}) {
		t.Fatal("23503 must not be a unique violation")
	}
	if IsUniqueViolation(sql.ErrNoRows) {
		t.Fatal("ErrNoRows must not be a unique violation")
	}
	if IsUniqueViolation(errors.New("plain error")) {
		t.Fatal("plain error must not be a unique violation")
	}
}

func TestUniqueConstraintName(t *testing.T) {
	if got := UniqueConstraintName(&pq.Error{Code: "23505", Constraint: "users_email_key"}); got != "users_email_key" {
		t.Fatalf("constraint = %q, want users_email_key", got)
	}
	if got := UniqueConstraintName(&pq.Error{Code: "23503", Constraint: "users_id_fkey"}); got != "" {
		t.Fatalf("constraint = %q, want empty for non-23505", got)
	}
	if got := UniqueConstraintName(sql.ErrNoRows); got != "" {
		t.Fatalf("constraint = %q, want empty", got)
	}
}

func TestIsForeignKeyViolationAndNoRows(t *testing.T) {
	if !IsForeignKeyViolation(&pq.Error{Code: "23503"}) {
		t.Fatal("23503 must be a foreign key violation")
	}
	if !IsNoRows(sql.ErrNoRows) {
		t.Fatal("ErrNoRows must be detected")
	}
	if IsNoRows(errors.New("other")) {
		t.Fatal("other error must not be NoRows")
	}
}

func TestNullHelpers(t *testing.T) {
	if NullString("") != nil || NullString("x") != "x" {
		t.Fatal("NullString mapping wrong")
	}
	if NullTime(nil) != nil {
		t.Fatal("NullTime(nil) must be nil")
	}
	if StringFromNull(sql.NullString{String: "a", Valid: true}) != "a" || StringFromNull(sql.NullString{}) != "" {
		t.Fatal("StringFromNull mapping wrong")
	}
	if TimePtrFromNull(sql.NullTime{}) != nil {
		t.Fatal("TimePtrFromNull invalid must be nil")
	}
}
