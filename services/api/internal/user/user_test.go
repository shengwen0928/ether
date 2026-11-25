package user

import (
	"context"
	"errors"
	"testing"
	"time"
)

func fixedNow() time.Time { return time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC) }

func newService() *Service {
	return NewService(NewMemoryUserStore(), NewMemorySettingsStore(), fixedNow)
}

func TestProvision_IdempotentAndEmailUniqueness(t *testing.T) {
	ctx := context.Background()
	svc := newService()

	u1, err := svc.Provision(ctx, "sub-1", "a@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if u1.Status != StatusActive || u1.RegisteredAt.IsZero() {
		t.Fatalf("provisioned = %+v", u1)
	}
	// 預設設定已建立。
	settings, err := svc.GetSettings(ctx, u1.ID)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings.Language != "zh-Hant" || settings.Theme != "system" {
		t.Fatalf("default settings = %+v", settings)
	}
	// 重複建置：同一使用者。
	u2, err := svc.Provision(ctx, "sub-1", "a@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision again: %v", err)
	}
	if u2.ID != u1.ID {
		t.Fatalf("second provision id = %q, want %q", u2.ID, u1.ID)
	}
	// 不同 subject、同一 Email：拒絕。
	if _, err := svc.Provision(ctx, "sub-2", "a@example.invalid", true); err == nil {
		t.Fatal("want error for duplicate email")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("err = %T %v, want *ValidationError", err, err)
		}
	}
}

func TestSetStatus_Transitions(t *testing.T) {
	ctx := context.Background()
	svc := newService()
	u, _ := svc.Provision(ctx, "sub-1", "", false)

	disabled, err := svc.SetStatus(ctx, u.ID, StatusDisabled)
	if err != nil {
		t.Fatalf("SetStatus disabled: %v", err)
	}
	if disabled.Status != StatusDisabled {
		t.Fatalf("status = %q", disabled.Status)
	}
	if _, err := svc.SetStatus(ctx, u.ID, StatusActive); err != nil {
		t.Fatalf("SetStatus active: %v", err)
	}
	if _, err := svc.SetStatus(ctx, u.ID, StatusPendingDeletion); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("pending_deletion: err = %v, want ErrInvalidStatusTransition", err)
	}
	if _, err := svc.SetStatus(ctx, u.ID, "bogus"); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("bogus status: err = %v, want ErrInvalidStatusTransition", err)
	}
}

func TestUpdateSettings_PatchSemantics(t *testing.T) {
	ctx := context.Background()
	svc := newService()
	u, _ := svc.Provision(ctx, "sub-1", "", false)

	theme := "dark"
	updated, err := svc.UpdateSettings(ctx, u.ID, SettingsPatch{Theme: &theme})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if updated.Theme != "dark" || updated.Language != "zh-Hant" {
		t.Fatalf("patched = %+v (language must be unchanged)", updated)
	}
	bad := "neon"
	if _, err := svc.UpdateSettings(ctx, u.ID, SettingsPatch{Theme: &bad}); err == nil {
		t.Fatal("want validation error for bad theme")
	}
	lang := "en!"
	if _, err := svc.UpdateSettings(ctx, u.ID, SettingsPatch{Language: &lang}); err == nil {
		t.Fatal("want validation error for bad language tag")
	}
}
