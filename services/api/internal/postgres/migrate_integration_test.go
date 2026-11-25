//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
)

func TestMigrateUpAndStatus(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)

	applied, pending, err := postgres.Status(ctx, db)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(applied) != 4 || len(pending) != 0 {
		t.Fatalf("applied = %v, pending = %v, want 4 applied, 0 pending", applied, pending)
	}

	// 冪等：再次 Up 不套用任何版本。
	again, err := postgres.MigrateUp(ctx, db)
	if err != nil {
		t.Fatalf("MigrateUp again: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second MigrateUp applied %v, want none", again)
	}

	// 回滾一步：000004 退出；再 Up 回來。
	rolledBack, err := postgres.MigrateDownOne(ctx, db)
	if err != nil {
		t.Fatalf("MigrateDownOne: %v", err)
	}
	if rolledBack != "000004" {
		t.Fatalf("rolled back = %q, want 000004", rolledBack)
	}
	applied, pending, err = postgres.Status(ctx, db)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(applied) != 3 || len(pending) != 1 || pending[0] != "000004" {
		t.Fatalf("after down: applied = %v, pending = %v", applied, pending)
	}
	if _, err := postgres.MigrateUp(ctx, db); err != nil {
		t.Fatalf("MigrateUp restore: %v", err)
	}
}

func TestMigrateDownTo(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)

	rolledBack, err := postgres.MigrateDownTo(ctx, db, "000001")
	if err != nil {
		t.Fatalf("MigrateDownTo: %v", err)
	}
	if len(rolledBack) != 3 || rolledBack[0] != "000004" || rolledBack[1] != "000003" || rolledBack[2] != "000002" {
		t.Fatalf("rolled back = %v, want [000004 000003 000002]", rolledBack)
	}
	applied, pending, err := postgres.Status(ctx, db)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(applied) != 1 || applied[0] != "000001" || len(pending) != 3 {
		t.Fatalf("after down-to: applied = %v, pending = %v", applied, pending)
	}

	// 全部回滾再全部套用：往返一致。
	if _, err := postgres.MigrateDownTo(ctx, db, "000000"); err != nil {
		t.Fatalf("MigrateDownTo 000000: %v", err)
	}
	applied, _, err = postgres.Status(ctx, db)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("applied = %v, want none after full rollback", applied)
	}
	restored, err := postgres.MigrateUp(ctx, db)
	if err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}
	if len(restored) != 4 {
		t.Fatalf("restored = %v, want 4 versions", restored)
	}
}
