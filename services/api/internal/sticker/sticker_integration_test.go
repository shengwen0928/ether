//go:build integration

package sticker

import (
	"context"
	"testing"
	"time"

	"ether.local/api/internal/media"
	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
	"ether.local/api/internal/user"
)

// TestPG_StickerFlowPersists 驗證貼圖完整流程經 PostgreSQL 持久化：
// 建包（草稿）→ 上架 → 加貼圖（經 Media 掃描閘門）→ 授權發放 →
// 收藏／取消 → 最近使用 → 搜尋 → 刪除 → 第二個 Service 實例
// （模擬重啟）讀取已封存貼圖包為 404。
// 無隔離資料庫時跳過並標示「環境待驗證」。
func TestPG_StickerFlowPersists(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)
	q := postgres.WrapDB(db)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	users := user.NewService(user.NewPGUserStore(q), user.NewPGSettingsStore(q), clock)
	ua, err := users.Provision(ctx, "kc-subject-sa", "sa@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision a: %v", err)
	}
	ub, err := users.Provision(ctx, "kc-subject-sb", "sb@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision b: %v", err)
	}

	// 媒體檔案：上傳會話 → 完成 → 掃描通過，方能關聯為貼圖。
	mediaSvc := media.NewService(media.NewPGStore(q), users, media.NewMemoryObjectStore(), clock)
	upload, err := mediaSvc.InitiateUpload(ctx, ua.ID, media.InitiateRequest{
		MIMEType: "image/png", TotalBytes: 100, OriginalFilename: "sticker.png", Width: 512, Height: 512,
	})
	if err != nil {
		t.Fatalf("InitiateUpload: %v", err)
	}
	if _, err := mediaSvc.ReportProgress(ctx, ua.ID, upload.ID, 100, 1); err != nil {
		t.Fatalf("ReportProgress: %v", err)
	}
	if _, err := mediaSvc.StoreRelayBytes(ctx, ua.ID, upload.ID, make([]byte, 100)); err != nil {
		t.Fatalf("StoreRelayBytes: %v", err)
	}
	f, err := mediaSvc.CompleteUpload(ctx, ua.ID, upload.ID, media.CompleteRequest{
		MIMEType: "image/png", OriginalFilename: "sticker.png", Width: 512, Height: 512,
	})
	if err != nil {
		t.Fatalf("CompleteUpload: %v", err)
	}
	if _, err := mediaSvc.ReportScanResult(ctx, f.ID, true); err != nil {
		t.Fatalf("ReportScanResult: %v", err)
	}

	newSvc := func() *Service {
		return NewService(NewPGStore(q), users, mediaSvc, clock)
	}
	svc := newSvc()

	entry, err := svc.CreatePack(ctx, ua.ID, "測試包", "描述", CategoryAnimals, f.ID, false)
	if err != nil {
		t.Fatalf("CreatePack: %v", err)
	}
	packID := entry.Pack.ID
	if entry.Pack.Status != PackStatusDraft {
		t.Fatalf("新包狀態 = %s, want draft", entry.Pack.Status)
	}

	// 陌生人讀取草稿為 404；上架後付費包仍須授權。
	if _, err := svc.GetPack(ctx, ub.ID, packID); err == nil {
		t.Fatal("陌生人讀取草稿應為 404")
	}
	published := PackStatusPublished
	if _, err := svc.UpdatePack(ctx, ua.ID, packID, PackPatch{Status: &published}); err != nil {
		t.Fatalf("UpdatePack: %v", err)
	}
	if _, err := svc.GetPack(ctx, ub.ID, packID); err == nil {
		t.Fatal("無授權者讀取付費包應為 404")
	}

	st, err := svc.AddSticker(ctx, ua.ID, packID, f.ID, "第一張", false)
	if err != nil {
		t.Fatalf("AddSticker: %v", err)
	}
	if st.Position != 0 {
		t.Fatalf("序號 = %d, want 0", st.Position)
	}

	if err := svc.GrantPack(ctx, ub.ID, packID); err != nil {
		t.Fatalf("GrantPack: %v", err)
	}
	if _, err := svc.FavoritePack(ctx, ub.ID, packID); err != nil {
		t.Fatalf("FavoritePack: %v", err)
	}
	if _, err := svc.RecordStickerUse(ctx, ub.ID, st.ID); err != nil {
		t.Fatalf("RecordStickerUse: %v", err)
	}
	results, _, err := svc.SearchStickers(ctx, ub.ID, "第一張", "", "", 10)
	if err != nil {
		t.Fatalf("SearchStickers: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("搜尋 = %d 筆, want 1", len(results))
	}

	// 模擬重啟：新 Service 實例讀取同一 PG 儲存。
	svc2 := newSvc()
	favs, _, err := svc2.ListFavorites(ctx, ub.ID, "", 10)
	if err != nil {
		t.Fatalf("ListFavorites after restart: %v", err)
	}
	if len(favs) != 1 {
		t.Fatalf("重啟後收藏 = %d 筆, want 1", len(favs))
	}
	recent, _, err := svc2.ListRecentStickers(ctx, ub.ID, "", 10)
	if err != nil {
		t.Fatalf("ListRecentStickers after restart: %v", err)
	}
	if len(recent) != 1 {
		t.Fatalf("重啟後最近使用 = %d 筆, want 1", len(recent))
	}
	if _, err := svc2.UnfavoritePack(ctx, ub.ID, packID); err != nil {
		t.Fatalf("UnfavoritePack: %v", err)
	}
	if err := svc2.DeletePack(ctx, ua.ID, packID); err != nil {
		t.Fatalf("DeletePack: %v", err)
	}
	if _, err := svc2.GetPack(ctx, ua.ID, packID); err == nil {
		t.Fatal("刪除後讀取應為 404")
	}
}
