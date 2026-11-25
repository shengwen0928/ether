package message

import (
	"context"
	"errors"
	"testing"
)

func sendTestMessage(t *testing.T, svc *Service, chats *fakeChats, sender, conv, content string) string {
	t.Helper()
	res, err := svc.Send(context.Background(), sender, conv, "text", content, "", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	return res.Message.ID
}

func TestReactions_SetReplaceRemoveAndSummary(t *testing.T) {
	svc, chats := testService()
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-1", "u-b", false)
	ctx := context.Background()
	id := sendTestMessage(t, svc, chats, "u-a", "conv-1", "你好")

	// u-b 按讚，u-a 愛心：彙總兩種表情、各一人。
	if _, err := svc.SetReaction(ctx, "u-b", id, "👍"); err != nil {
		t.Fatalf("SetReaction: %v", err)
	}
	summary, err := svc.SetReaction(ctx, "u-a", id, "❤️")
	if err != nil {
		t.Fatalf("SetReaction: %v", err)
	}
	if len(summary.Reactions) != 2 {
		t.Fatalf("彙總 = %+v, want 2 種表情", summary.Reactions)
	}
	for _, c := range summary.Reactions {
		if c.Count != 1 {
			t.Fatalf("彙總 = %+v, want 各 1 人", summary.Reactions)
		}
	}
	if summary.Reactions[0].Emoji == "❤️" && !summary.Reactions[0].ReactedByMe {
		t.Fatalf("u-a 的愛心應標示 reacted_by_me：%+v", summary.Reactions)
	}

	// u-b 更換為大笑：讚消失、大笑出現，u-b 同時僅一則生效反應。
	summary, err = svc.SetReaction(ctx, "u-b", id, "😂")
	if err != nil {
		t.Fatalf("更換反應: %v", err)
	}
	if len(summary.Reactions) != 2 {
		t.Fatalf("更換後彙總 = %+v, want 2 種表情", summary.Reactions)
	}
	for _, c := range summary.Reactions {
		if c.Emoji == "👍" {
			t.Fatalf("更換後不應殘留讚：%+v", summary.Reactions)
		}
	}

	// 重送相同表情為冪等：彙總不變。
	again, err := svc.SetReaction(ctx, "u-b", id, "😂")
	if err != nil {
		t.Fatalf("重送相同表情: %v", err)
	}
	if len(again.Reactions) != 2 {
		t.Fatalf("重送後彙總 = %+v", again.Reactions)
	}

	// u-a 取消反應：只剩大笑。
	summary, err = svc.RemoveReaction(ctx, "u-a", id)
	if err != nil {
		t.Fatalf("RemoveReaction: %v", err)
	}
	if len(summary.Reactions) != 1 || summary.Reactions[0].Emoji != "😂" || summary.Reactions[0].Count != 1 {
		t.Fatalf("取消後彙總 = %+v", summary.Reactions)
	}

	// 再次取消為冪等成功，彙總不變。
	summary, err = svc.RemoveReaction(ctx, "u-a", id)
	if err != nil || len(summary.Reactions) != 1 {
		t.Fatalf("重複取消：%v, %+v", err, summary.Reactions)
	}
}

func TestReactions_ValidationAndGuards(t *testing.T) {
	svc, chats := testService()
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-1", "u-b", false)
	ctx := context.Background()
	id := sendTestMessage(t, svc, chats, "u-a", "conv-1", "你好")

	if _, err := svc.SetReaction(ctx, "u-b", id, ""); err == nil {
		t.Fatal("空反應應失敗")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("err = %v, want ValidationError", err)
		}
	}
	if _, err := svc.SetReaction(ctx, "u-b", id, "🔥"); err == nil {
		t.Fatal("不在允許清單的表情應失敗")
	}
	if _, err := svc.SetReaction(ctx, "u-ghost", id, "👍"); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("非成員 err = %v, want ErrMessageNotFound", err)
	}
	if _, err := svc.SetReaction(ctx, "u-b", "ghost-msg", "👍"); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("不存在的訊息 err = %v, want ErrMessageNotFound", err)
	}

	// 收回的訊息不得新增反應，但可取消與列出。
	if _, err := svc.Recall(ctx, "u-a", id); err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if _, err := svc.SetReaction(ctx, "u-b", id, "👍"); err == nil {
		t.Fatal("收回的訊息新增反應應失敗")
	}
	if _, err := svc.RemoveReaction(ctx, "u-b", id); err != nil {
		t.Fatalf("收回訊息取消反應應成功：%v", err)
	}
	if _, err := svc.ListReactions(ctx, "u-b", id); err != nil {
		t.Fatalf("收回訊息列出反應應成功：%v", err)
	}
	if _, err := svc.ListReactions(ctx, "u-ghost", id); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("非成員列出 err = %v, want ErrMessageNotFound", err)
	}
}

func TestReactions_DeletedForSelfGuard(t *testing.T) {
	svc, chats := testService()
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-1", "u-b", false)
	ctx := context.Background()
	id := sendTestMessage(t, svc, chats, "u-a", "conv-1", "你好")
	if _, err := svc.DeleteForSelf(ctx, "u-a", id); err != nil {
		t.Fatalf("DeleteForSelf: %v", err)
	}
	// 作者本人不得對僅自己刪除的訊息新增反應；他人不受影響。
	if _, err := svc.SetReaction(ctx, "u-a", id, "👍"); err == nil {
		t.Fatal("作者對已刪除訊息新增反應應失敗")
	}
	if _, err := svc.SetReaction(ctx, "u-b", id, "👍"); err != nil {
		t.Fatalf("他人反應應成功：%v", err)
	}
}

func TestPins_PinUnpinListAndCap(t *testing.T) {
	svc, chats := testService()
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-1", "u-b", false)
	ctx := context.Background()
	ids := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		ids = append(ids, sendTestMessage(t, svc, chats, "u-a", "conv-1", "訊息"))
	}

	// 置頂第一則：新建；重複置頂回傳既有列（冪等）。
	pin, isNew, err := svc.PinMessage(ctx, "u-b", "conv-1", ids[0])
	if err != nil || !isNew || pin.MessageID != ids[0] || pin.PinnedBy != "u-b" {
		t.Fatalf("PinMessage = %+v, isNew=%v, %v", pin, isNew, err)
	}
	pin2, isNew, err := svc.PinMessage(ctx, "u-a", "conv-1", ids[0])
	if err != nil || isNew || pin2.ID != pin.ID {
		t.Fatalf("重複置頂應回傳既有列：%+v, isNew=%v, %v", pin2, isNew, err)
	}

	// 補滿上限 5 則，第六則拒絕。
	for _, id := range ids[1:5] {
		if _, _, err := svc.PinMessage(ctx, "u-a", "conv-1", id); err != nil {
			t.Fatalf("PinMessage: %v", err)
		}
	}
	if _, _, err := svc.PinMessage(ctx, "u-a", "conv-1", ids[5]); err == nil {
		t.Fatal("超過置頂上限應失敗")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("err = %v, want ValidationError", err)
		}
	}

	// 達上限時重送已置頂訊息為冪等成功，不受上限影響。
	if _, isNew, err := svc.PinMessage(ctx, "u-a", "conv-1", ids[1]); err != nil || isNew {
		t.Fatalf("達上限時重送已置頂訊息：isNew=%v, %v", isNew, err)
	}

	// 清單 5 則，集合為前五則訊息（置頂時間相同時不保證順序）。
	pins, err := svc.ListPins(ctx, "u-a", "conv-1")
	if err != nil || len(pins) != 5 {
		t.Fatalf("ListPins = %d 筆，%v", len(pins), err)
	}
	seen := map[string]bool{}
	for _, p := range pins {
		seen[p.MessageID] = true
	}
	for _, id := range ids[:5] {
		if !seen[id] {
			t.Fatalf("ListPins 缺 %s：%+v", id, pins)
		}
	}

	// 取消第一則後可再置頂第六則；取消為冪等。
	unpinned, err := svc.UnpinMessage(ctx, "u-a", "conv-1", ids[0])
	if err != nil || unpinned == nil || unpinned.UnpinnedAt == nil {
		t.Fatalf("UnpinMessage = %+v, %v", unpinned, err)
	}
	if _, _, err := svc.PinMessage(ctx, "u-a", "conv-1", ids[5]); err != nil {
		t.Fatalf("取消後再置頂應成功：%v", err)
	}
	again, err := svc.UnpinMessage(ctx, "u-a", "conv-1", ids[0])
	if err != nil || again == nil || again.UnpinnedAt == nil {
		t.Fatalf("重複取消應回傳歷史列：%+v, %v", again, err)
	}
	never, err := svc.UnpinMessage(ctx, "u-a", "conv-1", ids[1])
	if err != nil {
		t.Fatalf("取消生效置頂應成功：%v", err)
	}
	if never == nil || never.UnpinnedAt == nil {
		t.Fatalf("取消生效置頂應回傳被取消的列：%+v", never)
	}
}

func TestPins_Guards(t *testing.T) {
	svc, chats := testService()
	chats.addMember("conv-1", "u-a", false)
	chats.addMember("conv-1", "u-b", false)
	chats.addMember("conv-2", "u-a", false)
	ctx := context.Background()
	id1 := sendTestMessage(t, svc, chats, "u-a", "conv-1", "一")
	id2 := sendTestMessage(t, svc, chats, "u-a", "conv-2", "二")

	if _, _, err := svc.PinMessage(ctx, "u-ghost", "conv-1", id1); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("非成員置頂 err = %v, want ErrMessageNotFound", err)
	}
	if _, _, err := svc.PinMessage(ctx, "u-b", "conv-1", id2); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("置頂他對話的訊息 err = %v, want ErrMessageNotFound", err)
	}
	if _, _, err := svc.PinMessage(ctx, "u-b", "conv-1", "ghost-msg"); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("置頂不存在的訊息 err = %v, want ErrMessageNotFound", err)
	}
	if _, err := svc.ListPins(ctx, "u-ghost", "conv-1"); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("非成員列出置頂 err = %v, want ErrMessageNotFound", err)
	}
	if _, err := svc.UnpinMessage(ctx, "u-b", "conv-1", id2); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("取消他對話訊息的置頂 err = %v, want ErrMessageNotFound", err)
	}

	// 收回的訊息不得置頂。
	if _, err := svc.Recall(ctx, "u-a", id1); err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if _, _, err := svc.PinMessage(ctx, "u-b", "conv-1", id1); err == nil {
		t.Fatal("收回的訊息置頂應失敗")
	}
}
