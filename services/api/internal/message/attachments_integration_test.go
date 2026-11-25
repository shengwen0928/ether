//go:build integration

package message

// 整合測試：附件關聯與轉傳的 PostgreSQL 持久化。
// 無 ETHER_TEST_DATABASE_URL 時跳過，標示「環境待驗證」。
// 須事先對測試資料庫套用 migration 000001–000011（Track A 步驟 7）。

import (
	"context"
	"testing"
	"time"

	"ether.local/api/internal/chat"
	"ether.local/api/internal/friend"
	"ether.local/api/internal/media"
	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
	"ether.local/api/internal/profile"
	"ether.local/api/internal/user"
)

func TestMessageAttachmentsForward_Integration(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)
	q := postgres.WrapDB(db)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	users := user.NewService(user.NewPGUserStore(q), user.NewPGSettingsStore(q), func() time.Time { return now })
	ua, err := users.Provision(ctx, "kc-subject-a", "a@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision a: %v", err)
	}
	ub, err := users.Provision(ctx, "kc-subject-b", "b@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision b: %v", err)
	}
	uc, err := users.Provision(ctx, "kc-subject-c", "c@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision c: %v", err)
	}
	profiles := profile.NewService(profile.NewPGStore(q), users, func() time.Time { return now })
	friends := friend.NewService(friend.NewPGStore(q), users, profiles, func() time.Time { return now })
	if _, err := friends.SendRequest(ctx, ua.ID, ub.ID, ""); err != nil {
		t.Fatalf("SendRequest u-a→u-b: %v", err)
	}
	reqs, err := friends.ListRequests(ctx, ub.ID, true)
	if err != nil || len(reqs) != 1 {
		t.Fatalf("ListRequests u-b = %d, %v", len(reqs), err)
	}
	if _, err := friends.RespondRequest(ctx, ub.ID, reqs[0].Request.ID, true); err != nil {
		t.Fatalf("RespondRequest u-b: %v", err)
	}
	if _, err := friends.SendRequest(ctx, ua.ID, uc.ID, ""); err != nil {
		t.Fatalf("SendRequest u-a→u-c: %v", err)
	}
	reqs, err = friends.ListRequests(ctx, uc.ID, true)
	if err != nil || len(reqs) != 1 {
		t.Fatalf("ListRequests u-c = %d, %v", len(reqs), err)
	}
	if _, err := friends.RespondRequest(ctx, uc.ID, reqs[0].Request.ID, true); err != nil {
		t.Fatalf("RespondRequest u-c: %v", err)
	}

	chats := chat.NewService(chat.NewPGStore(q), users, friends, func() time.Time { return now })
	convAB, _, err := chats.CreateDirect(ctx, ua.ID, ub.ID)
	if err != nil {
		t.Fatalf("CreateDirect u-a/u-b: %v", err)
	}
	convAC, _, err := chats.CreateDirect(ctx, ua.ID, uc.ID)
	if err != nil {
		t.Fatalf("CreateDirect u-a/u-c: %v", err)
	}

	// 媒體檔案：上傳→完成→掃描通過。
	mediaSvc := media.NewService(media.NewPGStore(q), users, media.NewMemoryObjectStore(), func() time.Time { return now })
	up, err := mediaSvc.InitiateUpload(ctx, ua.ID, media.InitiateRequest{
		MIMEType: "image/jpeg", TotalBytes: 100, OriginalFilename: "photo.jpg",
	})
	if err != nil {
		t.Fatalf("InitiateUpload: %v", err)
	}
	if _, err := mediaSvc.ReportProgress(ctx, ua.ID, up.ID, 100, 1); err != nil {
		t.Fatalf("ReportProgress: %v", err)
	}
	if _, err := mediaSvc.StoreRelayBytes(ctx, ua.ID, up.ID, make([]byte, 100)); err != nil {
		t.Fatalf("StoreRelayBytes: %v", err)
	}
	file, err := mediaSvc.CompleteUpload(ctx, ua.ID, up.ID, media.CompleteRequest{
		MIMEType: "image/jpeg", OriginalFilename: "photo.jpg",
	})
	if err != nil {
		t.Fatalf("CompleteUpload: %v", err)
	}
	if _, err := mediaSvc.ReportScanResult(ctx, file.ID, true); err != nil {
		t.Fatalf("ReportScanResult: %v", err)
	}

	svc := NewServiceWithDeps(NewPGStore(q), chats, nil, mediaSvc, func() time.Time { return now })

	// 發送時關聯附件；重啟後的第二個實例可讀出附件快照。
	sent, err := svc.SendWithAttachments(ctx, ua.ID, convAB.ID, "text", "附圖", "c-int-1", "", []string{file.ID})
	if err != nil {
		t.Fatalf("SendWithAttachments: %v", err)
	}
	if !sent.IsNew {
		t.Fatal("IsNew = false")
	}
	svc2 := NewServiceWithDeps(NewPGStore(q), chats, nil, mediaSvc, func() time.Time { return now })
	atts, err := svc2.ListAttachments(ctx, ub.ID, sent.Message.ID)
	if err != nil || len(atts) != 1 || atts[0].MediaFileID != file.ID || atts[0].FileName != "photo.jpg" {
		t.Fatalf("重啟後附件 = %+v, %v", atts, err)
	}

	// 轉傳至另一對話；轉傳紀錄持久化，轉傳訊息不帶附件。
	fwd, err := svc.Forward(ctx, ua.ID, sent.Message.ID, convAC.ID, "f-int-1")
	if err != nil {
		t.Fatalf("Forward: %v", err)
	}
	got, err := svc2.GetForward(ctx, uc.ID, fwd.Message.ID)
	if err != nil || got.SourceMessageID != sent.Message.ID || got.ForwardedBy != ua.ID {
		t.Fatalf("重啟後轉傳紀錄 = %+v, %v", got, err)
	}
	atts2, err := svc2.ListAttachments(ctx, uc.ID, fwd.Message.ID)
	if err != nil || len(atts2) != 0 {
		t.Fatalf("轉傳訊息附件 = %+v, %v", atts2, err)
	}
	// 掃描未通過的檔案不得關聯：再建一筆 pending 檔案。
	up2, err := mediaSvc.InitiateUpload(ctx, ua.ID, media.InitiateRequest{
		MIMEType: "image/jpeg", TotalBytes: 50, OriginalFilename: "pending.jpg",
	})
	if err != nil {
		t.Fatalf("InitiateUpload 2: %v", err)
	}
	if _, err := mediaSvc.ReportProgress(ctx, ua.ID, up2.ID, 50, 1); err != nil {
		t.Fatalf("ReportProgress 2: %v", err)
	}
	if _, err := mediaSvc.StoreRelayBytes(ctx, ua.ID, up2.ID, make([]byte, 50)); err != nil {
		t.Fatalf("StoreRelayBytes 2: %v", err)
	}
	file2, err := mediaSvc.CompleteUpload(ctx, ua.ID, up2.ID, media.CompleteRequest{
		MIMEType: "image/jpeg", OriginalFilename: "pending.jpg",
	})
	if err != nil {
		t.Fatalf("CompleteUpload 2: %v", err)
	}
	if _, err := svc.SendWithAttachments(ctx, ua.ID, convAB.ID, "text", "x", "", "", []string{file2.ID}); err == nil {
		t.Fatal("pending 檔案關聯應失敗")
	}
}
