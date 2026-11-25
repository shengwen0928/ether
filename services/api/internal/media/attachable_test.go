package media

import (
	"context"
	"errors"
	"testing"
)

// TestCheckAttachable_Gate 驗證附件關聯的檔案側檢查：僅擁有者
// 所有、掃描已通過的檔案可關聯；其餘情形一律拒絕（不存在／
// 非擁有者／已刪除為 ErrMediaNotFound，掃描未通過為欄位錯誤）。
func TestCheckAttachable_Gate(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()
	u := initiate(t, svc, 100)
	if _, err := svc.ReportProgress(ctx, "u-owner", u.ID, 100, 1); err != nil {
		t.Fatalf("ReportProgress: %v", err)
	}
	f := complete(t, svc, u)

	// pending：欄位錯誤，不得關聯。
	if _, err := svc.CheckAttachable(ctx, "u-owner", f.ID); err == nil {
		t.Fatal("pending 檔案應拒絕關聯")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) || ve.Fields[0].Field != "media_file_id" {
			t.Fatalf("pending err = %v, want media_file_id 欄位錯誤", err)
		}
	}
	// 非擁有者與查無檔案：一律 ErrMediaNotFound。
	if _, err := svc.CheckAttachable(ctx, "u-other", f.ID); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("非擁有者 err = %v, want ErrMediaNotFound", err)
	}
	if _, err := svc.CheckAttachable(ctx, "u-owner", "no-such-file"); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("查無檔案 err = %v, want ErrMediaNotFound", err)
	}
	// 掃描通過後可關聯；拒絕的檔案不得關聯。
	if _, err := svc.ReportScanResult(ctx, f.ID, true); err != nil {
		t.Fatalf("ReportScanResult: %v", err)
	}
	got, err := svc.CheckAttachable(ctx, "u-owner", f.ID)
	if err != nil || got.ID != f.ID || got.MIMEType != "image/jpeg" || got.SizeBytes != 100 {
		t.Fatalf("CheckAttachable = %+v, %v", got, err)
	}
	// 第二個檔案：掃描拒絕。
	u2 := initiate(t, svc, 50)
	if _, err := svc.ReportProgress(ctx, "u-owner", u2.ID, 50, 1); err != nil {
		t.Fatalf("ReportProgress 2: %v", err)
	}
	f2 := complete(t, svc, u2)
	if _, err := svc.ReportScanResult(ctx, f2.ID, false); err != nil {
		t.Fatalf("ReportScanResult 2: %v", err)
	}
	if _, err := svc.CheckAttachable(ctx, "u-owner", f2.ID); err == nil {
		t.Fatal("rejected 檔案應拒絕關聯")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("rejected err = %v, want ValidationError", err)
		}
	}
	// 已刪除的檔案不得關聯。
	if err := svc.DeleteFile(ctx, "u-owner", f.ID); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if _, err := svc.CheckAttachable(ctx, "u-owner", f.ID); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("已刪除 err = %v, want ErrMediaNotFound", err)
	}
}
