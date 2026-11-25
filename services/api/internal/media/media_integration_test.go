//go:build integration

package media

import (
	"context"
	"testing"
	"time"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
	"ether.local/api/internal/user"
)

// TestPG_MediaFlowPersists 驗證媒體完整流程經 PostgreSQL 持久化：
// 上傳會話建立→進度→完成（檔案列建立）→掃描通過→授權檢查寫入
// 稽核紀錄→縮圖登記→第二個 Service 實例（模擬重啟）讀取。
// 無隔離資料庫時跳過並標示「環境待驗證」。
func TestPG_MediaFlowPersists(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)
	q := postgres.WrapDB(db)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	users := user.NewService(user.NewPGUserStore(q), user.NewPGSettingsStore(q), func() time.Time { return now })
	u, err := users.Provision(ctx, "kc-subject-media", "media@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}

	svc := NewService(NewPGStore(q), users, NewMemoryObjectStore(), func() time.Time { return now })
	upload, err := svc.InitiateUpload(ctx, u.ID, InitiateRequest{
		MIMEType: "image/jpeg", TotalBytes: 100, OriginalFilename: "photo.jpg",
		Width: 800, Height: 600,
	})
	if err != nil {
		t.Fatalf("InitiateUpload: %v", err)
	}
	if _, err := svc.ReportProgress(ctx, u.ID, upload.ID, 100, 1); err != nil {
		t.Fatalf("ReportProgress: %v", err)
	}
	if _, err := svc.StoreRelayBytes(ctx, u.ID, upload.ID, make([]byte, 100)); err != nil {
		t.Fatalf("StoreRelayBytes: %v", err)
	}
	f, err := svc.CompleteUpload(ctx, u.ID, upload.ID, CompleteRequest{
		MIMEType: "image/jpeg", OriginalFilename: "photo.jpg", Width: 800, Height: 600,
	})
	if err != nil {
		t.Fatalf("CompleteUpload: %v", err)
	}
	if f.ScanStatus != ScanPending {
		t.Fatalf("scan_status = %s, want pending", f.ScanStatus)
	}
	if _, err := svc.ReportScanResult(ctx, f.ID, true); err != nil {
		t.Fatalf("ReportScanResult: %v", err)
	}
	if _, err := svc.RegisterThumbnail(ctx, u.ID, f.ID, RegisterThumbnailRequest{
		Variant: VariantThumbnail, MIMEType: "image/jpeg", SizeBytes: 1200, Width: 160, Height: 120,
	}); err != nil {
		t.Fatalf("RegisterThumbnail: %v", err)
	}
	allowed, err := svc.AuthorizeAccess(ctx, u.ID, f.ID, AccessDownload, false)
	if err != nil || !allowed {
		t.Fatalf("AuthorizeAccess = %v, %v, want true", allowed, err)
	}

	// 第二個 Service 實例（模擬重啟）讀取同一資料庫。
	svc2 := NewService(NewPGStore(q), users, NewMemoryObjectStore(), func() time.Time { return now })
	got, err := svc2.GetFile(ctx, u.ID, f.ID)
	if err != nil {
		t.Fatalf("重啟後 GetFile: %v", err)
	}
	if got.ScanStatus != ScanClean || got.SizeBytes != 100 {
		t.Fatalf("重啟後 file = %+v", got)
	}
	thumbs, err := svc2.ListThumbnails(ctx, u.ID, f.ID)
	if err != nil || len(thumbs) != 1 || thumbs[0].Variant != VariantThumbnail {
		t.Fatalf("重啟後縮圖 = %+v, %v", thumbs, err)
	}
	uploads, err := svc2.ListUploads(ctx, u.ID)
	if err != nil || len(uploads) != 1 || uploads[0].Status != UploadCompleted {
		t.Fatalf("重啟後會話 = %+v, %v", uploads, err)
	}
	if uploads[0].MediaFileID != f.ID {
		t.Fatalf("會話關聯檔案 = %q, want %q", uploads[0].MediaFileID, f.ID)
	}
}
