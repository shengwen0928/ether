package media

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestInitiateUpload_IssuesRelayUploadURL：記憶體模式建立會話時
// 核發指向中繼端點的相對路徑（非 MinIO 簽名 URL，不得冒充）。
func TestInitiateUpload_IssuesRelayUploadURL(t *testing.T) {
	svc, _, _ := newTestService()
	u := initiate(t, svc, 100)
	want := "/api/v1/media/relay/uploads/" + u.ID
	if u.UploadURL != want {
		t.Fatalf("UploadURL = %q, want %q", u.UploadURL, want)
	}
	if u.UploadURLExpiresAt == nil || !u.UploadURLExpiresAt.Equal(fixedNow.Add(UploadURLTTL)) {
		t.Fatalf("UploadURLExpiresAt = %v", u.UploadURLExpiresAt)
	}
}

// TestIssueUploadURL_ReissueAndGuards：重新核發、已完成／他人
// 會話的守衛。
func TestIssueUploadURL_ReissueAndGuards(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()
	u := initiate(t, svc, 10)
	got, err := svc.IssueUploadURL(ctx, "u-owner", u.ID)
	if err != nil {
		t.Fatalf("IssueUploadURL: %v", err)
	}
	if got.URL != "/api/v1/media/relay/uploads/"+u.ID {
		t.Fatalf("URL = %q", got.URL)
	}
	if _, err := svc.IssueUploadURL(ctx, "u-other", u.ID); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("他人核發 err = %v, want ErrMediaNotFound", err)
	}
	complete(t, svc, u)
	if _, err := svc.IssueUploadURL(ctx, "u-owner", u.ID); err == nil {
		t.Fatal("已完成會話須拒絕重新核發")
	}
}

// TestCompleteUpload_RequiresObjectBytes：無位元組、大小不符皆
// 不得完成；位元組齊備時完成。
func TestCompleteUpload_RequiresObjectBytes(t *testing.T) {
	svc, _, objects := newTestService()
	ctx := context.Background()
	u := initiate(t, svc, 100)
	if _, err := svc.ReportProgress(ctx, "u-owner", u.ID, 100, 1); err != nil {
		t.Fatalf("ReportProgress: %v", err)
	}
	if _, err := svc.CompleteUpload(ctx, "u-owner", u.ID, CompleteRequest{MIMEType: "image/jpeg"}); err == nil {
		t.Fatal("無位元組完成須拒絕")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("err = %v, want ValidationError", err)
		}
	}
	if err := objects.PutObject(ctx, u.StorageKey, make([]byte, 50)); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	if _, err := svc.CompleteUpload(ctx, "u-owner", u.ID, CompleteRequest{MIMEType: "image/jpeg"}); err == nil {
		t.Fatal("大小不符完成須拒絕")
	}
	if err := objects.PutObject(ctx, u.StorageKey, make([]byte, 100)); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	f, err := svc.CompleteUpload(ctx, "u-owner", u.ID, CompleteRequest{MIMEType: "image/jpeg", OriginalFilename: "photo.jpg"})
	if err != nil {
		t.Fatalf("CompleteUpload: %v", err)
	}
	if f.SizeBytes != 100 {
		t.Fatalf("SizeBytes = %d, want 100", f.SizeBytes)
	}
}

// TestCompleteUpload_NilObjectStore：物件儲存未接線時完成回 503。
func TestCompleteUpload_NilObjectStore(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(store, stubUsers{ids: map[string]bool{"u-owner": true}}, nil, func() time.Time { return fixedNow })
	ctx := context.Background()
	now := fixedNow.UTC()
	u := &Upload{ID: "up-nil", UploaderID: "u-owner", StorageKey: "media/u-owner/f-nil",
		ReservedFileID: "f-nil", Status: UploadUploading, TotalBytes: 10, UploadedBytes: 10,
		CreatedAt: now, UpdatedAt: now}
	if err := store.CreateUpload(ctx, u); err != nil {
		t.Fatalf("CreateUpload: %v", err)
	}
	if _, err := svc.CompleteUpload(ctx, "u-owner", u.ID, CompleteRequest{MIMEType: "image/jpeg"}); !errors.Is(err, ErrObjectStoreUnavailable) {
		t.Fatalf("err = %v, want ErrObjectStoreUnavailable", err)
	}
}

// completeCleanFile 建立已掃描通過的檔案（供下載授權測試）。
func completeCleanFile(t *testing.T, svc *Service) *MediaFile {
	t.Helper()
	u := initiate(t, svc, 10)
	f := complete(t, svc, u)
	if _, err := svc.ReportScanResult(context.Background(), f.ID, true); err != nil {
		t.Fatalf("ReportScanResult: %v", err)
	}
	return f
}

// TestIssueDownloadURL_Gates：掃描閘門、擁有者限定、無 Presigner
// 時 URL 為空但 allowed 仍為真。
func TestIssueDownloadURL_Gates(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	// pending：擁有者亦不得分發、無 URL。
	u := initiate(t, svc, 10)
	pending := complete(t, svc, u)
	authz, err := svc.IssueDownloadURL(ctx, "u-owner", pending.ID, AccessDownload, false)
	if err != nil || authz.Allowed || authz.DownloadURL != "" || authz.ScanStatus != ScanPending {
		t.Fatalf("pending authz = %+v, %v", authz, err)
	}

	// clean：允許＋中繼 URL；他人拒絕且掃描狀態以 pending 佔位。
	f := completeCleanFile(t, svc)
	authz, err = svc.IssueDownloadURL(ctx, "u-owner", f.ID, AccessDownload, false)
	if err != nil || !authz.Allowed || authz.DownloadURL != "/api/v1/media/relay/files/"+f.ID || authz.ScanStatus != ScanClean {
		t.Fatalf("clean authz = %+v, %v", authz, err)
	}
	if authz.URLExpiresAt == nil || !authz.URLExpiresAt.Equal(fixedNow.Add(DownloadURLTTL)) {
		t.Fatalf("URLExpiresAt = %v", authz.URLExpiresAt)
	}
	authz, err = svc.IssueDownloadURL(ctx, "u-other", f.ID, AccessDownload, false)
	if err != nil || authz.Allowed || authz.ScanStatus != ScanPending {
		t.Fatalf("他人 authz = %+v, %v", authz, err)
	}
	if _, err := svc.IssueDownloadURL(ctx, "u-owner", f.ID, "stream", false); err == nil {
		t.Fatal("非法存取類型須拒絕")
	}

	// rejected：不得分發。
	u2 := initiate(t, svc, 10)
	rejected := complete(t, svc, u2)
	if _, err := svc.ReportScanResult(ctx, rejected.ID, false); err != nil {
		t.Fatalf("ReportScanResult: %v", err)
	}
	authz, err = svc.IssueDownloadURL(ctx, "u-owner", rejected.ID, AccessDownload, false)
	if err != nil || authz.Allowed || authz.DownloadURL != "" {
		t.Fatalf("rejected authz = %+v, %v", authz, err)
	}
}

// TestIssueDownloadURL_NilObjectStore：允許但無可分發 URL 時
// URL 為空（不得假裝可下載）。
func TestIssueDownloadURL_NilObjectStore(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(store, stubUsers{ids: map[string]bool{"u-owner": true}}, nil, func() time.Time { return fixedNow })
	ctx := context.Background()
	now := fixedNow.UTC()
	f := &MediaFile{ID: "f-bare", OwnerID: "u-owner", StorageKey: "media/u-owner/f-bare",
		MIMEType: "image/jpeg", SizeBytes: 10, ScanStatus: ScanClean, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateFile(ctx, f); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	authz, err := svc.IssueDownloadURL(ctx, "u-owner", f.ID, AccessDownload, false)
	if err != nil || !authz.Allowed || authz.DownloadURL != "" {
		t.Fatalf("authz = %+v, %v", authz, err)
	}
}

// TestStoreRelayBytes_Guards：大小不符、超過上限、非進行中會話。
func TestStoreRelayBytes_Guards(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()
	u := initiate(t, svc, 10)
	if _, err := svc.StoreRelayBytes(ctx, "u-owner", u.ID, make([]byte, 9)); err == nil {
		t.Fatal("與宣告總量不符須拒絕")
	}
	if _, err := svc.StoreRelayBytes(ctx, "u-other", u.ID, make([]byte, 10)); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("他人寫入 err = %v, want ErrMediaNotFound", err)
	}
	got, err := svc.StoreRelayBytes(ctx, "u-owner", u.ID, make([]byte, 10))
	if err != nil || got.UploadedBytes != 10 {
		t.Fatalf("StoreRelayBytes = %+v, %v", got, err)
	}
	// 總量未知時僅受上限約束。
	u2 := initiate(t, svc, 0)
	if _, err := svc.StoreRelayBytes(ctx, "u-owner", u2.ID, make([]byte, MaxFileSizeBytes+1)); err == nil {
		t.Fatal("超過上限須拒絕")
	}
}

// TestLoadRelayBytes_RoundTrip：掃描通過後位元組可原樣讀回；
// pending 時不得讀出。
func TestLoadRelayBytes_RoundTrip(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()
	u := initiate(t, svc, 10)
	want := []byte("0123456789")
	if _, err := svc.StoreRelayBytes(ctx, "u-owner", u.ID, want); err != nil {
		t.Fatalf("StoreRelayBytes: %v", err)
	}
	// 直接完成（不經 complete 輔助，避免輔助重寫位元組）。
	f, err := svc.CompleteUpload(ctx, "u-owner", u.ID, CompleteRequest{MIMEType: "image/jpeg", OriginalFilename: "photo.jpg"})
	if err != nil {
		t.Fatalf("CompleteUpload: %v", err)
	}
	allowed, _, _, err := svc.LoadRelayBytes(ctx, "u-owner", f.ID, AccessDownload, false)
	if err != nil || allowed {
		t.Fatalf("pending 讀出 = %v, %v, want false", allowed, err)
	}
	if _, err := svc.ReportScanResult(ctx, f.ID, true); err != nil {
		t.Fatalf("ReportScanResult: %v", err)
	}
	allowed, gotFile, data, err := svc.LoadRelayBytes(ctx, "u-owner", f.ID, AccessDownload, false)
	if err != nil || !allowed || gotFile.ID != f.ID || string(data) != string(want) {
		t.Fatalf("讀出 = %v, %+v, %q, %v", allowed, gotFile, data, err)
	}
	allowed, _, _, err = svc.LoadRelayBytes(ctx, "u-other", f.ID, AccessDownload, false)
	if err != nil || allowed {
		t.Fatalf("他人讀出 = %v, %v, want false", allowed, err)
	}
}

// TestMemoryObjectStore_StatAndAbort：位元組存在查詢與中止清理。
func TestMemoryObjectStore_StatAndAbort(t *testing.T) {
	objects := NewMemoryObjectStore()
	ctx := context.Background()
	if _, exists, err := objects.Stat(ctx, "media/u-1/f-1"); err != nil || exists {
		t.Fatalf("Stat = %v, %v", exists, err)
	}
	if err := objects.PutObject(ctx, "media/u-1/f-1", []byte("abc")); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	size, exists, err := objects.Stat(ctx, "media/u-1/f-1")
	if err != nil || !exists || size != 3 {
		t.Fatalf("Stat = %d, %v, %v", size, exists, err)
	}
	if err := objects.PutObject(ctx, "media/u-1/f-1", make([]byte, MaxFileSizeBytes+1)); err == nil {
		t.Fatal("超過上限寫入須拒絕")
	}
	if err := objects.AbortUpload(ctx, "media/u-1/f-1", "mem-upload-1"); err != nil {
		t.Fatalf("AbortUpload: %v", err)
	}
	if _, exists, _ := objects.Stat(ctx, "media/u-1/f-1"); exists {
		t.Fatal("中止後位元組須清除")
	}
}
