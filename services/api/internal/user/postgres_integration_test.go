//go:build integration

package user

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
)

func newPGService(db *sql.DB, now func() time.Time) *Service {
	q := postgres.WrapDB(db)
	return NewService(NewPGUserStore(q), NewPGSettingsStore(q), now)
}

func TestPG_ProvisionPersistsAndReprovisions(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	svc := newPGService(db, func() time.Time { return now })

	u, err := svc.Provision(ctx, "kc-subject-1", "a@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if u.KeycloakSubject != "kc-subject-1" || !u.EmailVerified || u.Status != StatusActive {
		t.Fatalf("provisioned = %+v", u)
	}
	st, err := svc.GetSettings(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if st.Language != "zh-Hant" || st.Theme != "system" {
		t.Fatalf("default settings = %+v", st)
	}

	// 第二個 Service 實例（模擬重啟）讀到同一列，並更新登入資訊。
	later := now.Add(time.Hour)
	svc2 := newPGService(db, func() time.Time { return later })
	u2, err := svc2.Provision(ctx, "kc-subject-1", "b@example.invalid", false)
	if err != nil {
		t.Fatalf("re-Provision: %v", err)
	}
	if u2.ID != u.ID || u2.Email != "b@example.invalid" || u2.EmailVerified || !u2.LastLoginAt.Equal(later) {
		t.Fatalf("re-provisioned = %+v, want same id, new email, unverified, later login", u2)
	}

	// 不同 subject 佔用同一 Email：儲存層唯一約束 → ValidationError。
	other := &User{ID: "00000000-0000-4000-8000-000000000001", KeycloakSubject: "kc-subject-2", Email: "b@example.invalid", Status: StatusActive, RegisteredAt: now, CreatedAt: now, UpdatedAt: now}
	err = NewPGUserStore(postgres.WrapDB(db)).Create(ctx, other)
	var ve *ValidationError
	if !errors.As(err, &ve) || len(ve.Fields) != 1 || ve.Fields[0].Field != "email" {
		t.Fatalf("duplicate email err = %v (%T), want ValidationError email", err, err)
	}

	// 設定更新持久化。
	theme := "dark"
	updated, err := svc2.UpdateSettings(ctx, u.ID, SettingsPatch{Theme: &theme})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if updated.Theme != "dark" || updated.Language != "zh-Hant" {
		t.Fatalf("updated settings = %+v", updated)
	}
	if _, err := svc2.GetSettings(ctx, "00000000-0000-4000-8000-000000000099"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("missing settings err = %v, want ErrUserNotFound", err)
	}
}
