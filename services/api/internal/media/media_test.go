package media

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"ether.local/api/internal/transport"
)

// stubUsers 是 UserResolver 的測試替身：只認事先登記的使用者。
type stubUsers struct{ ids map[string]bool }

func (s stubUsers) Exists(_ context.Context, userID string) (bool, error) {
	return s.ids[userID], nil
}

var fixedNow = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

func newTestService() (*Service, *MemoryStore, *MemoryObjectStore) {
	store := NewMemoryStore()
	objects := NewMemoryObjectStore()
	svc := NewService(store, stubUsers{ids: map[string]bool{"u-owner": true, "u-other": true}}, objects, func() time.Time { return fixedNow })
	return svc, store, objects
}

func initiate(t *testing.T, svc *Service, total int64) *Upload {
	t.Helper()
	u, err := svc.InitiateUpload(context.Background(), "u-owner", InitiateRequest{
		MIMEType: "image/jpeg", TotalBytes: total, OriginalFilename: "photo.jpg",
		Width: 800, Height: 600,
	})
	if err != nil {
		t.Fatalf("InitiateUpload: %v", err)
	}
	return u
}

func complete(t *testing.T, svc *Service, u *Upload) *MediaFile {
	t.Helper()
	// 完成前須先有位元組（§2z：Stat 校驗）：經中繼寫入宣告總量
	// 的位元組，再完成。
	if _, err := svc.StoreRelayBytes(context.Background(), "u-owner", u.ID, make([]byte, u.TotalBytes)); err != nil {
		t.Fatalf("StoreRelayBytes: %v", err)
	}
	f, err := svc.CompleteUpload(context.Background(), "u-owner", u.ID, CompleteRequest{
		MIMEType: "image/jpeg", OriginalFilename: "photo.jpg", Width: 800, Height: 600,
	})
	if err != nil {
		t.Fatalf("CompleteUpload: %v", err)
	}
	return f
}

func TestIssueObjectKey_Format(t *testing.T) {
	key := IssueObjectKey("u-owner", "f-1")
	if key != "media/u-owner/f-1" {
		t.Fatalf("key = %q", key)
	}
	if IssueObjectKey("u-owner", "f-2") == key {
		t.Fatal("不同檔案的鍵不得相同")
	}
	if IssueObjectKey("", "f-1") != "" || IssueObjectKey("u-owner", "") != "" {
		t.Fatal("空識別碼須回空鍵")
	}
	thumb := IssueThumbnailKey("u-owner", "f-1", "thumbnail")
	if thumb != "media/u-owner/f-1/thumb/thumbnail" {
		t.Fatalf("thumb key = %q", thumb)
	}
	if !strings.HasPrefix(thumb, key+"/") {
		t.Fatal("縮圖鍵須以原檔鍵為前綴")
	}
}

func TestInitiateUpload_Validation(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.InitiateUpload(ctx, "u-owner", InitiateRequest{MIMEType: "application/x-sh", TotalBytes: 10}); err == nil {
		t.Fatal("白名單外 MIME 須拒絕")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) || ve.Fields[0].Field != "mime_type" {
			t.Fatalf("err = %v, want mime_type 欄位錯誤", err)
		}
	}
	if _, err := svc.InitiateUpload(ctx, "u-owner", InitiateRequest{MIMEType: "image/png", TotalBytes: MaxFileSizeBytes + 1}); err == nil {
		t.Fatal("超過上限須拒絕")
	}
	if _, err := svc.InitiateUpload(ctx, "u-ghost", InitiateRequest{MIMEType: "image/png", TotalBytes: 10}); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("未知使用者 err = %v, want ErrMediaNotFound", err)
	}
	// 物件儲存未接線時 fail-closed。
	bare := NewService(NewMemoryStore(), stubUsers{ids: map[string]bool{"u-owner": true}}, nil, nil)
	if _, err := bare.InitiateUpload(ctx, "u-owner", InitiateRequest{MIMEType: "image/png", TotalBytes: 10}); !errors.Is(err, ErrObjectStoreUnavailable) {
		t.Fatalf("未接線 err = %v, want ErrObjectStoreUnavailable", err)
	}
}

func TestUploadFlow_ProgressCompleteAndScanGate(t *testing.T) {
	svc, store, objects := newTestService()
	ctx := context.Background()
	u := initiate(t, svc, 1000)
	if u.StorageKey == "" || u.ExternalUploadID == "" {
		t.Fatal("會話須攜帶核發的物件鍵與物件儲存識別碼")
	}
	if got := objects.CreatedKeys(); len(got) != 1 || got[0] != u.StorageKey {
		t.Fatalf("物件儲存建立鍵 = %v, want [%s]", got, u.StorageKey)
	}
	if u.StorageKey != IssueObjectKey("u-owner", u.ReservedFileID) {
		t.Fatalf("鍵須由預留檔案識別碼核發：%q", u.StorageKey)
	}

	// 未傳完不得完成。
	if _, err := svc.ReportProgress(ctx, "u-owner", u.ID, 400, 1); err != nil {
		t.Fatalf("ReportProgress: %v", err)
	}
	if _, err := svc.CompleteUpload(ctx, "u-owner", u.ID, CompleteRequest{MIMEType: "image/jpeg"}); err == nil {
		t.Fatal("未傳完完成須拒絕")
	}
	// 進度不得倒退、不得超過總量。
	if _, err := svc.ReportProgress(ctx, "u-owner", u.ID, 300, 1); err == nil {
		t.Fatal("進度倒退須拒絕")
	}
	if _, err := svc.ReportProgress(ctx, "u-owner", u.ID, 1200, 2); err == nil {
		t.Fatal("超過總量須拒絕")
	}
	if _, err := svc.ReportProgress(ctx, "u-owner", u.ID, 1000, 2); err != nil {
		t.Fatalf("ReportProgress: %v", err)
	}
	// 非擁有者不得操作他人會話。
	if _, err := svc.ReportProgress(ctx, "u-other", u.ID, 1000, 2); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("他人會話 err = %v, want ErrMediaNotFound", err)
	}

	f := complete(t, svc, u)
	if f.ID != u.ReservedFileID || f.StorageKey != u.StorageKey {
		t.Fatal("檔案列須沿用會話核發的識別碼與鍵")
	}
	if f.ScanStatus != ScanPending || f.SizeBytes != 1000 {
		t.Fatalf("file = %+v", f)
	}
	// 掃描閘門：pending 時擁有者亦不得分發。
	allowed, err := svc.AuthorizeAccess(ctx, "u-owner", f.ID, AccessDownload, false)
	if err != nil || allowed {
		t.Fatalf("pending 授權 = %v, %v, want false", allowed, err)
	}
	// 掃描通過後：擁有者可存取，他人須業務權限。
	if _, err := svc.ReportScanResult(ctx, f.ID, true); err != nil {
		t.Fatalf("ReportScanResult: %v", err)
	}
	allowed, err = svc.AuthorizeAccess(ctx, "u-owner", f.ID, AccessDownload, false)
	if err != nil || !allowed {
		t.Fatalf("擁有者授權 = %v, %v, want true", allowed, err)
	}
	allowed, err = svc.AuthorizeAccess(ctx, "u-other", f.ID, AccessDownload, false)
	if err != nil || allowed {
		t.Fatalf("他人無業務權限授權 = %v, %v, want false", allowed, err)
	}
	allowed, err = svc.AuthorizeAccess(ctx, "u-other", f.ID, AccessDownload, true)
	if err != nil || !allowed {
		t.Fatalf("他人有業務權限授權 = %v, %v, want true", allowed, err)
	}
	// 稽核紀錄：四次檢查皆寫入（1 拒絕＋3 次，含拒絕）。
	logs := store.AccessLogs()
	if len(logs) != 4 {
		t.Fatalf("存取紀錄 %d 筆，want 4", len(logs))
	}
	if logs[0].AccessResult != AccessDenied || logs[1].AccessResult != AccessAllowed {
		t.Fatalf("紀錄結果順序錯誤：%+v", logs)
	}
}

func TestCompleteUpload_Idempotent(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()
	u := initiate(t, svc, 10)
	if _, err := svc.ReportProgress(ctx, "u-owner", u.ID, 10, 1); err != nil {
		t.Fatalf("ReportProgress: %v", err)
	}
	first := complete(t, svc, u)
	second, err := svc.CompleteUpload(ctx, "u-owner", u.ID, CompleteRequest{MIMEType: "image/jpeg"})
	if err != nil {
		t.Fatalf("重複完成：%v", err)
	}
	if second.ID != first.ID {
		t.Fatal("重複完成須回傳同一檔案，不得重建")
	}
}

func TestUploadFlow_CancelAndExpire(t *testing.T) {
	svc, store, objects := newTestService()
	ctx := context.Background()
	u := initiate(t, svc, 100)
	cancelled, err := svc.CancelUpload(ctx, "u-owner", u.ID)
	if err != nil {
		t.Fatalf("CancelUpload: %v", err)
	}
	if cancelled.Status != UploadCancelled {
		t.Fatalf("status = %s", cancelled.Status)
	}
	if got := objects.AbortedKeys(); len(got) != 1 || got[0] != u.StorageKey {
		t.Fatalf("中止鍵 = %v", got)
	}
	if _, err := svc.CancelUpload(ctx, "u-owner", u.ID); err == nil {
		t.Fatal("已取消者再取消須拒絕")
	}

	// 逾期：舊會話被標記，近期會話不受影響。
	old := initiate(t, svc, 100)
	// 以較晚的時鐘建立第二個服務，共享同一儲存，建一個近期會話。
	later := NewService(store, stubUsers{ids: map[string]bool{"u-owner": true}}, objects, func() time.Time { return fixedNow.Add(3 * time.Hour) })
	fresh, err := later.InitiateUpload(ctx, "u-owner", InitiateRequest{MIMEType: "image/png", TotalBytes: 100})
	if err != nil {
		t.Fatalf("InitiateUpload fresh: %v", err)
	}
	n, err := later.ExpireStaleUploads(ctx, fixedNow.Add(time.Hour))
	if err != nil {
		t.Fatalf("ExpireStaleUploads: %v", err)
	}
	if n != 1 {
		t.Fatalf("逾期 %d 筆，want 1（僅 old 早於截止）", n)
	}
	got, err := svc.GetUpload(ctx, "u-owner", old.ID)
	if err != nil || got.Status != UploadExpired {
		t.Fatalf("逾期狀態 = %+v, %v", got, err)
	}
	got, err = svc.GetUpload(ctx, "u-owner", fresh.ID)
	if err != nil || got.Status != UploadUploading {
		t.Fatalf("近期狀態 = %+v, %v", got, err)
	}
	if _, err := svc.ReportProgress(ctx, "u-owner", old.ID, 10, 1); err == nil {
		t.Fatal("逾期會話不得再回報進度")
	}
}

func TestThumbnail_DuplicateAndOwnership(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()
	u := initiate(t, svc, 10)
	if _, err := svc.ReportProgress(ctx, "u-owner", u.ID, 10, 1); err != nil {
		t.Fatalf("ReportProgress: %v", err)
	}
	f := complete(t, svc, u)

	thumb, err := svc.RegisterThumbnail(ctx, "u-owner", f.ID, RegisterThumbnailRequest{
		Variant: VariantThumbnail, MIMEType: "image/jpeg", SizeBytes: 1200, Width: 160, Height: 120,
	})
	if err != nil {
		t.Fatalf("RegisterThumbnail: %v", err)
	}
	if thumb.StorageKey != IssueThumbnailKey("u-owner", f.ID, "thumbnail") {
		t.Fatalf("縮圖鍵 = %q", thumb.StorageKey)
	}
	if _, err := svc.RegisterThumbnail(ctx, "u-owner", f.ID, RegisterThumbnailRequest{
		Variant: VariantThumbnail, MIMEType: "image/jpeg", SizeBytes: 900,
	}); err == nil {
		t.Fatal("同一變體重複登記須拒絕")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) || ve.Fields[0].Field != "variant_type" {
			t.Fatalf("err = %v, want variant_type 欄位錯誤", err)
		}
	}
	if _, err := svc.RegisterThumbnail(ctx, "u-owner", f.ID, RegisterThumbnailRequest{
		Variant: VariantPreview, MIMEType: "image/png", SizeBytes: 900,
	}); err != nil {
		t.Fatalf("不同變體登記：%v", err)
	}
	if _, err := svc.RegisterThumbnail(ctx, "u-other", f.ID, RegisterThumbnailRequest{
		Variant: VariantCompressed, MIMEType: "image/png", SizeBytes: 900,
	}); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("他人登記 err = %v, want ErrMediaNotFound", err)
	}
	rows, err := svc.ListThumbnails(ctx, "u-owner", f.ID)
	if err != nil || len(rows) != 2 {
		t.Fatalf("縮圖 %d 筆，%v", len(rows), err)
	}
	if _, err := svc.ListThumbnails(ctx, "u-other", f.ID); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("他人列縮圖 err = %v, want ErrMediaNotFound", err)
	}
}

func TestScanResult_RejectedAndFinalized(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()
	u := initiate(t, svc, 10)
	if _, err := svc.ReportProgress(ctx, "u-owner", u.ID, 10, 1); err != nil {
		t.Fatalf("ReportProgress: %v", err)
	}
	f := complete(t, svc, u)
	if _, err := svc.ReportScanResult(ctx, f.ID, false); err != nil {
		t.Fatalf("ReportScanResult: %v", err)
	}
	// 掃描未通過：擁有者＋業務權限皆不得分發。
	allowed, err := svc.AuthorizeAccess(ctx, "u-owner", f.ID, AccessPreview, true)
	if err != nil || allowed {
		t.Fatalf("rejected 授權 = %v, %v, want false", allowed, err)
	}
	if _, err := svc.ReportScanResult(ctx, f.ID, true); !errors.Is(err, ErrScanFinalized) {
		t.Fatalf("重複回報 err = %v, want ErrScanFinalized", err)
	}
}

func TestDeleteFile_HidesEverywhere(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()
	u := initiate(t, svc, 10)
	if _, err := svc.ReportProgress(ctx, "u-owner", u.ID, 10, 1); err != nil {
		t.Fatalf("ReportProgress: %v", err)
	}
	f := complete(t, svc, u)
	if err := svc.DeleteFile(ctx, "u-other", f.ID); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("他人刪除 err = %v, want ErrMediaNotFound", err)
	}
	if err := svc.DeleteFile(ctx, "u-owner", f.ID); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if _, err := svc.GetFile(ctx, "u-owner", f.ID); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("已刪除讀取 err = %v, want ErrMediaNotFound", err)
	}
	if _, err := svc.AuthorizeAccess(ctx, "u-owner", f.ID, AccessDownload, true); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("已刪除授權 err = %v, want ErrMediaNotFound", err)
	}
	files, err := svc.ListMyFiles(ctx, "u-owner")
	if err != nil || len(files) != 0 {
		t.Fatalf("清單 %d 筆，%v", len(files), err)
	}
}

func TestAuthorizeAccess_InvalidType(t *testing.T) {
	svc, store, _ := newTestService()
	ctx := context.Background()
	u := initiate(t, svc, 10)
	if _, err := svc.ReportProgress(ctx, "u-owner", u.ID, 10, 1); err != nil {
		t.Fatalf("ReportProgress: %v", err)
	}
	f := complete(t, svc, u)
	if _, err := svc.AuthorizeAccess(ctx, "u-owner", f.ID, "stream", true); err == nil {
		t.Fatal("非法存取類型須拒絕")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("err = %v, want ValidationError", err)
		}
	}
	if len(store.AccessLogs()) != 0 {
		t.Fatal("非法請求不得寫入稽核紀錄")
	}
	if _, err := svc.AuthorizeAccess(ctx, "u-owner", "missing", AccessDownload, true); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("查無檔案 err = %v, want ErrMediaNotFound", err)
	}
}

func TestValidationError_FieldErrors(t *testing.T) {
	ve := validationError("mime_type", "INVALID_VALUE", "不支援的檔案類型。")
	fields := ve.FieldErrors()
	if len(fields) != 1 || fields[0].Field != "mime_type" {
		t.Fatalf("fields = %+v", fields)
	}
	var _ transport.FieldError = fields[0]
}
