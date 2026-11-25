package message

import (
	"context"
	"errors"
	"testing"
	"time"

	"ether.local/api/internal/media"
	"ether.local/api/internal/outbox"
	"ether.local/api/internal/transport"
)

// fakeMediaAccess 是 MediaAccess 的測試替身：只認事先登記的檔案；
// 擁有權不符／查無檔案回 media.ErrMediaNotFound；掃描未通過回
// 欄位錯誤（語義同 media.CheckAttachable，精確對映見 media 套件測試）。
type fakeMediaAccess struct {
	files map[string]*media.MediaFile
}

func (f *fakeMediaAccess) CheckAttachable(_ context.Context, accessorID, fileID string) (*media.MediaFile, error) {
	file := f.files[fileID]
	if file == nil || file.OwnerID != accessorID {
		return nil, media.ErrMediaNotFound
	}
	if file.ScanStatus != media.ScanClean {
		return nil, &ValidationError{Fields: []transport.FieldError{{
			Field: "media_file_id", Issue: "INVALID_VALUE", Message: "檔案尚未通過掃描。",
		}}}
	}
	return file, nil
}

func testServiceWithMedia() (*Service, *fakeChats, *fakeMediaAccess) {
	fixed := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	chats := newFakeChats()
	acc := &fakeMediaAccess{files: map[string]*media.MediaFile{}}
	svc := NewServiceWithDeps(NewMemoryStore(), chats, nil, acc, func() time.Time { return fixed })
	return svc, chats, acc
}

func addTestFile(acc *fakeMediaAccess, id, owner string, status media.ScanStatus) {
	acc.files[id] = &media.MediaFile{
		ID: id, OwnerID: owner, MIMEType: "image/jpeg",
		SizeBytes: 100, OriginalFilename: "photo.jpg", ScanStatus: status,
	}
}

func TestSendWithAttachments_WritesSnapshotAndLists(t *testing.T) {
	svc, chats, acc := testServiceWithMedia()
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-1", "u-b", false)
	addTestFile(acc, "file-1", "u-a", media.ScanClean)
	ctx := context.Background()

	res, err := svc.SendWithAttachments(ctx, "u-a", "conv-1", "text", "附圖", "c-1", "", []string{"file-1"})
	if err != nil {
		t.Fatalf("SendWithAttachments: %v", err)
	}
	if !res.IsNew {
		t.Fatal("IsNew = false")
	}
	// 讀取路徑不以 Message 結構攜帶附件（須另查）。
	got, err := svc.Get(ctx, "u-b", res.Message.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Attachments != nil {
		t.Fatal("Get 回傳的訊息不得攜帶附件列")
	}
	atts, err := svc.ListAttachments(ctx, "u-b", res.Message.ID)
	if err != nil {
		t.Fatalf("ListAttachments: %v", err)
	}
	if len(atts) != 1 || atts[0].MediaFileID != "file-1" || atts[0].MIMEType != "image/jpeg" ||
		atts[0].SizeBytes != 100 || atts[0].FileName != "photo.jpg" || atts[0].MessageID != res.Message.ID {
		t.Fatalf("attachments = %+v", atts)
	}
}

func TestSendWithAttachments_NonOwnedOrMissingFile404(t *testing.T) {
	svc, chats, acc := testServiceWithMedia()
	chats.addMember("conv-1", "u-a", false)
	addTestFile(acc, "file-1", "u-b", media.ScanClean)
	ctx := context.Background()

	if _, err := svc.SendWithAttachments(ctx, "u-a", "conv-1", "text", "x", "", "", []string{"file-1"}); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("他人檔案 err = %v, want ErrMessageNotFound", err)
	}
	if _, err := svc.SendWithAttachments(ctx, "u-a", "conv-1", "text", "x", "", "", []string{"no-such-file"}); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("查無檔案 err = %v, want ErrMessageNotFound", err)
	}
}

func TestSendWithAttachments_ScanPendingIsValidationError(t *testing.T) {
	svc, chats, acc := testServiceWithMedia()
	chats.addMember("conv-1", "u-a", false)
	addTestFile(acc, "file-1", "u-a", media.ScanPending)
	ctx := context.Background()

	_, err := svc.SendWithAttachments(ctx, "u-a", "conv-1", "text", "x", "", "", []string{"file-1"})
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Fields[0].Field != "media_file_id" {
		t.Fatalf("err = %v, want media_file_id 欄位錯誤", err)
	}
}

func TestSendWithAttachments_MediaUnconfiguredFailClosed(t *testing.T) {
	svc, chats := testService()
	chats.addMember("conv-1", "u-a", false)
	ctx := context.Background()

	_, err := svc.SendWithAttachments(ctx, "u-a", "conv-1", "text", "x", "", "", []string{"file-1"})
	if !errors.Is(err, ErrMediaUnavailable) {
		t.Fatalf("err = %v, want ErrMediaUnavailable", err)
	}
	// 無附件的既有路徑不受影響。
	if _, err := svc.Send(ctx, "u-a", "conv-1", "text", "x", "", ""); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestSendWithAttachments_DedupeAndLimit(t *testing.T) {
	svc, chats, acc := testServiceWithMedia()
	chats.addMember("conv-1", "u-a", false)
	addTestFile(acc, "file-1", "u-a", media.ScanClean)
	ctx := context.Background()

	res, err := svc.SendWithAttachments(ctx, "u-a", "conv-1", "text", "x", "", "", []string{"file-1", "file-1"})
	if err != nil {
		t.Fatalf("重複識別碼應去重：%v", err)
	}
	atts, _ := svc.ListAttachments(ctx, "u-a", res.Message.ID)
	if len(atts) != 1 {
		t.Fatalf("去重後附件 = %d 筆，want 1", len(atts))
	}

	var many []string
	for i := 0; i < MaxAttachmentsPerMessage+1; i++ {
		id := "file-many-" + string(rune('a'+i))
		addTestFile(acc, id, "u-a", media.ScanClean)
		many = append(many, id)
	}
	_, err = svc.SendWithAttachments(ctx, "u-a", "conv-1", "text", "x", "", "", many)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("超上限 err = %v, want ValidationError", err)
	}
	if _, err := svc.SendWithAttachments(ctx, "u-a", "conv-1", "text", "x", "", "", []string{""}); !errors.As(err, &ve) {
		t.Fatalf("空識別碼 err = %v, want ValidationError", err)
	}
}

func TestAddAttachments_OnlyAuthorAndIdempotent(t *testing.T) {
	svc, chats, acc := testServiceWithMedia()
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-1", "u-b", false)
	addTestFile(acc, "file-1", "u-a", media.ScanClean)
	addTestFile(acc, "file-2", "u-a", media.ScanClean)
	ctx := context.Background()

	res, err := svc.Send(ctx, "u-a", "conv-1", "text", "原文", "", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	// 非作者不得事後附加。
	if _, err := svc.AddAttachments(ctx, "u-b", res.Message.ID, []string{"file-1"}); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("非作者 err = %v, want ErrMessageNotFound", err)
	}
	atts, err := svc.AddAttachments(ctx, "u-a", res.Message.ID, []string{"file-1"})
	if err != nil || len(atts) != 1 {
		t.Fatalf("atts = %+v, %v", atts, err)
	}
	atts, err = svc.AddAttachments(ctx, "u-a", res.Message.ID, []string{"file-1", "file-2"})
	if err != nil || len(atts) != 2 {
		t.Fatalf("重複關聯後 atts = %+v, %v", atts, err)
	}
}

func TestAddAttachments_RecalledMessageRejected(t *testing.T) {
	svc, chats, acc := testServiceWithMedia()
	chats.addMember("conv-1", "u-a", false)
	addTestFile(acc, "file-1", "u-a", media.ScanClean)
	ctx := context.Background()

	res, err := svc.Send(ctx, "u-a", "conv-1", "text", "原文", "", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if _, err := svc.Recall(ctx, "u-a", res.Message.ID); err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if _, err := svc.AddAttachments(ctx, "u-a", res.Message.ID, []string{"file-1"}); err == nil {
		t.Fatal("收回訊息新增附件應失敗")
	}
}

func TestForward_CreatesMessageAndRecordWithoutAttachments(t *testing.T) {
	svc, chats, acc := testServiceWithMedia()
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-1", "u-b", false)
	chats.addMember("conv-2", "u-a", false)
	chats.addMember("conv-2", "u-b", false)
	addTestFile(acc, "file-1", "u-a", media.ScanClean)
	ctx := context.Background()

	res, err := svc.SendWithAttachments(ctx, "u-a", "conv-1", "text", "原文", "c-1", "", []string{"file-1"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	fwd, err := svc.Forward(ctx, "u-a", res.Message.ID, "conv-2", "f-1")
	if err != nil {
		t.Fatalf("Forward: %v", err)
	}
	if !fwd.IsNew || fwd.Message.ConversationID != "conv-2" || fwd.Message.Content != "原文" ||
		fwd.Message.SequenceNumber != 1 || fwd.Message.SenderID != "u-a" {
		t.Fatalf("forwarded = %+v", fwd.Message)
	}
	if fwd.Forward == nil || fwd.Forward.SourceMessageID != res.Message.ID ||
		fwd.Forward.SourceConversationID != "conv-1" ||
		fwd.Forward.TargetConversationID != "conv-2" || fwd.Forward.ForwardedBy != "u-a" {
		t.Fatalf("forward record = %+v", fwd.Forward)
	}
	// 轉傳不帶附件。
	atts, err := svc.ListAttachments(ctx, "u-b", fwd.Message.ID)
	if err != nil || len(atts) != 0 {
		t.Fatalf("轉傳訊息附件 = %+v, %v", atts, err)
	}
	// 轉傳紀錄可另查；原始訊息並非轉傳產物，查紀錄為 404。
	got, err := svc.GetForward(ctx, "u-b", fwd.Message.ID)
	if err != nil || got.ID != fwd.Forward.ID {
		t.Fatalf("GetForward = %+v, %v", got, err)
	}
	if _, err := svc.GetForward(ctx, "u-b", res.Message.ID); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("原始訊息 GetForward err = %v, want ErrMessageNotFound", err)
	}
}

func TestForward_NonMemberAndRecalledGuards(t *testing.T) {
	svc, chats, _ := testServiceWithMedia()
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-2", "u-a", false)
	ctx := context.Background()

	res, err := svc.Send(ctx, "u-a", "conv-1", "text", "原文", "", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	// 目標對話非成員。
	if _, err := svc.Forward(ctx, "u-b", res.Message.ID, "conv-2", ""); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("非成員 err = %v, want ErrMessageNotFound", err)
	}
	// 來源對話非成員。
	chats.addMember("conv-2", "u-b", false)
	if _, err := svc.Forward(ctx, "u-b", res.Message.ID, "conv-2", ""); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("來源非成員 err = %v, want ErrMessageNotFound", err)
	}
	// 收回的來源訊息不得轉傳。
	if _, err := svc.Recall(ctx, "u-a", res.Message.ID); err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if _, err := svc.Forward(ctx, "u-a", res.Message.ID, "conv-2", ""); err == nil {
		t.Fatal("收回訊息轉傳應失敗")
	}
}

func TestForward_IdempotentByClientMessageID(t *testing.T) {
	svc, chats, _ := testServiceWithMedia()
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-2", "u-a", false)
	ctx := context.Background()

	res, err := svc.Send(ctx, "u-a", "conv-1", "text", "原文", "", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	first, err := svc.Forward(ctx, "u-a", res.Message.ID, "conv-2", "f-1")
	if err != nil || !first.IsNew {
		t.Fatalf("first = %+v, %v", first, err)
	}
	second, err := svc.Forward(ctx, "u-a", res.Message.ID, "conv-2", "f-1")
	if err != nil || second.IsNew || second.Message.ID != first.Message.ID {
		t.Fatalf("second = %+v, %v", second, err)
	}
	if len(chats.noted) != 2 {
		t.Fatalf("NoteMessage 呼叫 %d 次，want 2（發送＋轉傳各一）", len(chats.noted))
	}
}

func TestForward_DepthLimit(t *testing.T) {
	svc, chats, _ := testServiceWithMedia()
	convs := []string{"conv-1", "conv-2", "conv-3", "conv-4", "conv-5", "conv-6"}
	for _, c := range convs {
		chats.addMember(c, "u-a", false)
	}
	ctx := context.Background()

	res, err := svc.Send(ctx, "u-a", "conv-1", "text", "原文", "", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	current := res.Message.ID
	for i := 1; i < MaxForwardDepth; i++ {
		fwd, err := svc.Forward(ctx, "u-a", current, convs[i], "")
		if err != nil {
			t.Fatalf("第 %d 次轉傳：%v", i, err)
		}
		current = fwd.Message.ID
	}
	// 深度已達上限（5），再轉傳須拒絕。
	_, err = svc.Forward(ctx, "u-a", current, convs[5], "")
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("超深度 err = %v, want ValidationError", err)
	}
}

func TestForward_BlockedTargetDM(t *testing.T) {
	svc, chats, _ := testServiceWithMedia()
	chats.addMember("conv-1", "u-a", true)
	chats.addMember("conv-2", "u-a", true)
	ctx := context.Background()

	res, err := svc.Send(ctx, "u-a", "conv-1", "text", "原文", "", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	chats.blocked = true
	if _, err := svc.Forward(ctx, "u-a", res.Message.ID, "conv-2", ""); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("封鎖目標 err = %v, want ErrMessageNotFound", err)
	}
}

func TestForward_WritesReceivedEvent(t *testing.T) {
	fixed := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	chats := newFakeChats()
	obs := outbox.NewMemoryStore()
	svc := NewServiceWithDeps(NewMemoryStoreWithOutbox(obs), chats, obs, &fakeMediaAccess{files: map[string]*media.MediaFile{}}, func() time.Time { return fixed })
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-2", "u-a", false)
	ctx := context.Background()

	res, err := svc.Send(ctx, "u-a", "conv-1", "text", "原文", "", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	fwd, err := svc.Forward(ctx, "u-a", res.Message.ID, "conv-2", "")
	if err != nil {
		t.Fatalf("Forward: %v", err)
	}
	events, err := obs.Claim(ctx, 10)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	var found bool
	for _, ev := range events {
		if ev.EventName == EventMessageReceived && ev.MessageID == fwd.Message.ID && ev.ConversationID == "conv-2" {
			found = true
		}
	}
	if !found {
		t.Fatalf("轉傳未寫入新訊息的 message.received 事件：%+v", events)
	}
}
