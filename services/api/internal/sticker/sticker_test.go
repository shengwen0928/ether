package sticker

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"ether.local/api/internal/media"
)

// fakeUsers 是 UserResolver 的測試替身。
type fakeUsers struct {
	existing map[string]bool
}

func (f *fakeUsers) Exists(_ context.Context, userID string) (bool, error) {
	return f.existing[userID], nil
}

// fakeMedia 是 MediaAccess 的測試替身：allowed 為可關聯的檔案
// 識別碼集合；其餘一律回 ErrMediaNotFound。
type fakeMedia struct {
	allowed map[string]bool
}

func (f *fakeMedia) CheckAttachable(_ context.Context, _, fileID string) (*media.MediaFile, error) {
	if f.allowed[fileID] {
		return &media.MediaFile{ID: fileID}, nil
	}
	return nil, media.ErrMediaNotFound
}

func testClock() func() time.Time {
	base := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	n := 0
	return func() time.Time {
		n++
		return base.Add(time.Duration(n) * time.Second)
	}
}

func testService(m *fakeMedia) *Service {
	if m == nil {
		m = &fakeMedia{allowed: map[string]bool{}}
	}
	return NewService(NewMemoryStore(), &fakeUsers{existing: map[string]bool{
		"u-a": true, "u-b": true, "u-c": true, "u-d": true,
	}}, m, testClock())
}

func mustPack(t *testing.T, svc *Service, creator, name string, isFree bool) *PackEntry {
	t.Helper()
	entry, err := svc.CreatePack(context.Background(), creator, name, "", CategoryGeneral, "", isFree)
	if err != nil {
		t.Fatalf("CreatePack: %v", err)
	}
	return entry
}

func mustPublish(t *testing.T, svc *Service, creator, packID string) {
	t.Helper()
	status := PackStatusPublished
	if _, err := svc.UpdatePack(context.Background(), creator, packID, PackPatch{Status: &status}); err != nil {
		t.Fatalf("publish: %v", err)
	}
}

func TestCreatePack_Validation(t *testing.T) {
	ctx := context.Background()
	svc := testService(&fakeMedia{allowed: map[string]bool{"f-1": true}})

	if _, err := svc.CreatePack(ctx, "u-a", "   ", "", CategoryGeneral, "", true); err == nil {
		t.Fatal("空名稱應為欄位錯誤")
	}
	if _, err := svc.CreatePack(ctx, "u-a", strings.Repeat("名", 65), "", CategoryGeneral, "", true); err == nil {
		t.Fatal("過長名稱應為欄位錯誤")
	}
	if _, err := svc.CreatePack(ctx, "u-a", "包", "", PackCategory("bogus"), "", true); err == nil {
		t.Fatal("非法分類應為欄位錯誤")
	}
	if _, err := svc.CreatePack(ctx, "u-a", "包", strings.Repeat("述", 501), CategoryGeneral, "", true); err == nil {
		t.Fatal("過長描述應為欄位錯誤")
	}
	// 封面媒體不存在 → 404，不得洩漏檔案存在性。
	if _, err := svc.CreatePack(ctx, "u-a", "包", "", CategoryGeneral, "missing", true); !errors.Is(err, ErrPackNotFound) {
		t.Fatalf("封面媒體不存在應回 ErrPackNotFound，現為 %v", err)
	}
	// Media 未接線時 fail-closed。
	svcNoMedia := NewService(NewMemoryStore(), &fakeUsers{existing: map[string]bool{"u-a": true}}, nil, testClock())
	if _, err := svcNoMedia.CreatePack(ctx, "u-a", "包", "", CategoryGeneral, "f-1", true); !errors.Is(err, ErrCheckerUnavailable) {
		t.Fatalf("Media 未接線應回 ErrCheckerUnavailable，現為 %v", err)
	}
	// 新包一律為草稿。
	entry, err := svc.CreatePack(ctx, "u-a", "包", "描述", CategoryAnimals, "f-1", true)
	if err != nil {
		t.Fatalf("CreatePack: %v", err)
	}
	if entry.Pack.Status != PackStatusDraft || entry.Pack.CoverMediaID != "f-1" || entry.StickerCount != 0 {
		t.Fatalf("新包 = %+v", entry)
	}
}

func TestPackVisibility_Matrix(t *testing.T) {
	ctx := context.Background()
	svc := testService(nil)

	draft := mustPack(t, svc, "u-a", "草稿包", true)
	unlisted := mustPack(t, svc, "u-a", "未上架包", true)
	status := PackStatusUnlisted
	if _, err := svc.UpdatePack(ctx, "u-a", unlisted.Pack.ID, PackPatch{Status: &status}); err != nil {
		t.Fatalf("UpdatePack: %v", err)
	}
	free := mustPack(t, svc, "u-a", "免費包", true)
	mustPublish(t, svc, "u-a", free.Pack.ID)
	paid := mustPack(t, svc, "u-a", "付費包", false)
	mustPublish(t, svc, "u-a", paid.Pack.ID)

	// 作者本人全可見。
	for _, e := range []*PackEntry{draft, unlisted, free, paid} {
		if _, err := svc.GetPack(ctx, "u-a", e.Pack.ID); err != nil {
			t.Fatalf("作者讀取 %s：%v", e.Pack.Name, err)
		}
	}
	// 陌生人：僅免費上架包可見。
	if _, err := svc.GetPack(ctx, "u-b", free.Pack.ID); err != nil {
		t.Fatalf("陌生人讀取免費包：%v", err)
	}
	for _, e := range []*PackEntry{draft, unlisted, paid} {
		if _, err := svc.GetPack(ctx, "u-b", e.Pack.ID); !errors.Is(err, ErrPackNotFound) {
			t.Fatalf("陌生人讀取 %s 應為 404，現為 %v", e.Pack.Name, err)
		}
	}
	// 授權後付費包可見；撤回後再次不可見（fail-closed）。
	if err := svc.GrantPack(ctx, "u-b", paid.Pack.ID); err != nil {
		t.Fatalf("GrantPack: %v", err)
	}
	if _, err := svc.GetPack(ctx, "u-b", paid.Pack.ID); err != nil {
		t.Fatalf("授權後讀取付費包：%v", err)
	}
	if err := svc.RevokeGrant(ctx, "u-b", paid.Pack.ID); err != nil {
		t.Fatalf("RevokeGrant: %v", err)
	}
	if _, err := svc.GetPack(ctx, "u-b", paid.Pack.ID); !errors.Is(err, ErrPackNotFound) {
		t.Fatalf("撤回後讀取付費包應為 404，現為 %v", err)
	}
}

func TestUpdateAndDeletePack(t *testing.T) {
	ctx := context.Background()
	svc := testService(nil)
	entry := mustPack(t, svc, "u-a", "舊名", true)

	// 非作者編輯／刪除一律 404。
	name := "新名"
	if _, err := svc.UpdatePack(ctx, "u-b", entry.Pack.ID, PackPatch{Name: &name}); !errors.Is(err, ErrPackNotFound) {
		t.Fatalf("非作者編輯應為 404，現為 %v", err)
	}
	if err := svc.DeletePack(ctx, "u-b", entry.Pack.ID); !errors.Is(err, ErrPackNotFound) {
		t.Fatalf("非作者刪除應為 404，現為 %v", err)
	}
	// 狀態 archived 不接受 PATCH（須走刪除端點）；非法狀態拒絕。
	archived := PackStatusArchived
	if _, err := svc.UpdatePack(ctx, "u-a", entry.Pack.ID, PackPatch{Status: &archived}); err == nil {
		t.Fatal("PATCH 為 archived 應為欄位錯誤")
	}
	bogus := PackStatus("bogus")
	if _, err := svc.UpdatePack(ctx, "u-a", entry.Pack.ID, PackPatch{Status: &bogus}); err == nil {
		t.Fatal("非法狀態應為欄位錯誤")
	}
	// 作者編輯成功。
	updated, err := svc.UpdatePack(ctx, "u-a", entry.Pack.ID, PackPatch{Name: &name})
	if err != nil {
		t.Fatalf("UpdatePack: %v", err)
	}
	if updated.Pack.Name != "新名" {
		t.Fatalf("編輯後名稱 = %q", updated.Pack.Name)
	}
	// 刪除後一切讀取皆 404，列保留（儲存層仍可讀到 archived 列）。
	if err := svc.DeletePack(ctx, "u-a", entry.Pack.ID); err != nil {
		t.Fatalf("DeletePack: %v", err)
	}
	if _, err := svc.GetPack(ctx, "u-a", entry.Pack.ID); !errors.Is(err, ErrPackNotFound) {
		t.Fatalf("刪除後作者讀取應為 404，現為 %v", err)
	}
}

func TestAddAndRemoveSticker(t *testing.T) {
	ctx := context.Background()
	svc := testService(&fakeMedia{allowed: map[string]bool{"f-1": true, "f-2": true}})
	entry := mustPack(t, svc, "u-a", "包", true)
	packID := entry.Pack.ID

	// 非作者新增一律 404。
	if _, err := svc.AddSticker(ctx, "u-b", packID, "f-1", "貼圖", false); !errors.Is(err, ErrPackNotFound) {
		t.Fatalf("非作者新增應為 404，現為 %v", err)
	}
	// 媒體不存在 → 404；未關聯媒體 → 欄位錯誤。
	if _, err := svc.AddSticker(ctx, "u-a", packID, "missing", "貼圖", false); !errors.Is(err, ErrPackNotFound) {
		t.Fatalf("媒體不存在應為 404，現為 %v", err)
	}
	if _, err := svc.AddSticker(ctx, "u-a", packID, "", "貼圖", false); err == nil {
		t.Fatal("未關聯媒體應為欄位錯誤")
	}
	// 序號依序指派。
	st1, err := svc.AddSticker(ctx, "u-a", packID, "f-1", "第一張", false)
	if err != nil {
		t.Fatalf("AddSticker: %v", err)
	}
	st2, err := svc.AddSticker(ctx, "u-a", packID, "f-2", "第二張", true)
	if err != nil {
		t.Fatalf("AddSticker: %v", err)
	}
	if st1.Position != 0 || st2.Position != 1 || !st2.IsAnimated {
		t.Fatalf("序號／動態旗標：%+v %+v", st1, st2)
	}
	// 移除後序號不重用：新貼圖序號為 2。
	if err := svc.RemoveSticker(ctx, "u-a", st1.ID); err != nil {
		t.Fatalf("RemoveSticker: %v", err)
	}
	st3, err := svc.AddSticker(ctx, "u-a", packID, "f-1", "第三張", false)
	if err != nil {
		t.Fatalf("AddSticker: %v", err)
	}
	if st3.Position != 2 {
		t.Fatalf("移除後新序號 = %d, want 2", st3.Position)
	}
	// 清單只含生效貼圖，依序號升序。
	mustPublish(t, svc, "u-a", packID)
	listed, err := svc.ListStickers(ctx, "u-b", packID)
	if err != nil {
		t.Fatalf("ListStickers: %v", err)
	}
	if len(listed) != 2 || listed[0].Sticker.ID != st2.ID || listed[1].Sticker.ID != st3.ID {
		t.Fatalf("清單 = %+v", listed)
	}
	// 非作者移除一律 404；重複移除一律 404。
	if err := svc.RemoveSticker(ctx, "u-b", st2.ID); !errors.Is(err, ErrStickerNotFound) {
		t.Fatalf("非作者移除應為 404，現為 %v", err)
	}
	if err := svc.RemoveSticker(ctx, "u-a", st1.ID); !errors.Is(err, ErrStickerNotFound) {
		t.Fatalf("重複移除應為 404，現為 %v", err)
	}
}

func TestFavoriteGrantInterplay(t *testing.T) {
	ctx := context.Background()
	svc := testService(nil)
	paid := mustPack(t, svc, "u-a", "付費包", false)
	mustPublish(t, svc, "u-a", paid.Pack.ID)
	packID := paid.Pack.ID

	// 無授權者收藏付費包 → 404（fail-closed）。
	if _, err := svc.FavoritePack(ctx, "u-b", packID); !errors.Is(err, ErrPackNotFound) {
		t.Fatalf("無授權收藏付費包應為 404，現為 %v", err)
	}
	// 授權後收藏：重複收藏冪等；來源維持 granted、旗標立起。
	if err := svc.GrantPack(ctx, "u-b", packID); err != nil {
		t.Fatalf("GrantPack: %v", err)
	}
	entry, err := svc.FavoritePack(ctx, "u-b", packID)
	if err != nil {
		t.Fatalf("FavoritePack: %v", err)
	}
	if !entry.FavoritedByMe {
		t.Fatalf("收藏後 FavoritedByMe 應為真：%+v", entry)
	}
	if _, err := svc.FavoritePack(ctx, "u-b", packID); err != nil {
		t.Fatalf("重複收藏：%v", err)
	}
	// 取消收藏保留授權列：仍可見、旗標清除。
	entry, err = svc.UnfavoritePack(ctx, "u-b", packID)
	if err != nil {
		t.Fatalf("UnfavoritePack: %v", err)
	}
	if entry.FavoritedByMe {
		t.Fatalf("取消後 FavoritedByMe 應為假：%+v", entry)
	}
	if _, err := svc.GetPack(ctx, "u-b", packID); err != nil {
		t.Fatalf("取消收藏後授權仍應可見：%v", err)
	}
	// 撤回授權：未收藏者列刪除，再次不可見。
	if err := svc.RevokeGrant(ctx, "u-b", packID); err != nil {
		t.Fatalf("RevokeGrant: %v", err)
	}
	if _, err := svc.GetPack(ctx, "u-b", packID); !errors.Is(err, ErrPackNotFound) {
		t.Fatalf("撤回後應為 404，現為 %v", err)
	}

	// 免費包收藏／取消：取消後列硬刪除，收藏清單為空。
	free := mustPack(t, svc, "u-a", "免費包", true)
	mustPublish(t, svc, "u-a", free.Pack.ID)
	if _, err := svc.FavoritePack(ctx, "u-b", free.Pack.ID); err != nil {
		t.Fatalf("FavoritePack: %v", err)
	}
	favs, _, err := svc.ListFavorites(ctx, "u-b", "", 10)
	if err != nil || len(favs) != 1 {
		t.Fatalf("收藏清單 = %d, %v", len(favs), err)
	}
	if _, err := svc.UnfavoritePack(ctx, "u-b", free.Pack.ID); err != nil {
		t.Fatalf("UnfavoritePack: %v", err)
	}
	favs, _, err = svc.ListFavorites(ctx, "u-b", "", 10)
	if err != nil || len(favs) != 0 {
		t.Fatalf("取消後收藏清單 = %d, %v", len(favs), err)
	}
	// 未收藏時取消冪等成功。
	if _, err := svc.UnfavoritePack(ctx, "u-b", free.Pack.ID); err != nil {
		t.Fatalf("重複取消：%v", err)
	}
}

func TestGrantOnFavoriteKeepsFavorite(t *testing.T) {
	ctx := context.Background()
	svc := testService(nil)
	free := mustPack(t, svc, "u-a", "包", true)
	mustPublish(t, svc, "u-a", free.Pack.ID)
	// 先收藏（來源 favorite），後授權（來源改 granted、旗標保留），
	// 撤回授權後來源改回 favorite、旗標保留。
	if _, err := svc.FavoritePack(ctx, "u-b", free.Pack.ID); err != nil {
		t.Fatalf("FavoritePack: %v", err)
	}
	if err := svc.GrantPack(ctx, "u-b", free.Pack.ID); err != nil {
		t.Fatalf("GrantPack: %v", err)
	}
	if err := svc.RevokeGrant(ctx, "u-b", free.Pack.ID); err != nil {
		t.Fatalf("RevokeGrant: %v", err)
	}
	favs, _, err := svc.ListFavorites(ctx, "u-b", "", 10)
	if err != nil || len(favs) != 1 {
		t.Fatalf("撤回授權後收藏清單 = %d, %v", len(favs), err)
	}
}

func TestRecentUse_CapAndDedupe(t *testing.T) {
	ctx := context.Background()
	svc := testService(&fakeMedia{allowed: map[string]bool{"f-1": true}})
	entry := mustPack(t, svc, "u-a", "包", true)
	mustPublish(t, svc, "u-a", entry.Pack.ID)

	// 建立 35 張貼圖（同一檔案可重複關聯不同貼圖列）。
	var ids []string
	for i := 0; i < 35; i++ {
		st, err := svc.AddSticker(ctx, "u-a", entry.Pack.ID, "f-1", "貼圖", false)
		if err != nil {
			t.Fatalf("AddSticker %d: %v", i, err)
		}
		ids = append(ids, st.ID)
	}
	for _, id := range ids {
		if _, err := svc.RecordStickerUse(ctx, "u-b", id); err != nil {
			t.Fatalf("RecordStickerUse: %v", err)
		}
	}
	// 只保留最新 30 筆。
	recent, _, err := svc.ListRecentStickers(ctx, "u-b", "", 100)
	if err != nil {
		t.Fatalf("ListRecentStickers: %v", err)
	}
	if len(recent) != MaxRecentStickers {
		t.Fatalf("最近使用 = %d 筆, want %d", len(recent), MaxRecentStickers)
	}
	if recent[0].Sticker.ID != ids[34] {
		t.Fatalf("最新一筆 = %s, want %s", recent[0].Sticker.ID, ids[34])
	}
	// 重複記錄舊貼圖：去重（總數不變）、置頂。
	if _, err := svc.RecordStickerUse(ctx, "u-b", ids[10]); err != nil {
		t.Fatalf("RecordStickerUse: %v", err)
	}
	recent, _, err = svc.ListRecentStickers(ctx, "u-b", "", 100)
	if err != nil {
		t.Fatalf("ListRecentStickers: %v", err)
	}
	if len(recent) != MaxRecentStickers || recent[0].Sticker.ID != ids[10] {
		t.Fatalf("去重後首筆 = %s（共 %d 筆）, want %s", recent[0].Sticker.ID, len(recent), ids[10])
	}
	// 已移除貼圖的最近使用列靜默略過。
	if err := svc.RemoveSticker(ctx, "u-a", ids[10]); err != nil {
		t.Fatalf("RemoveSticker: %v", err)
	}
	recent, _, err = svc.ListRecentStickers(ctx, "u-b", "", 100)
	if err != nil {
		t.Fatalf("ListRecentStickers: %v", err)
	}
	for _, e := range recent {
		if e.Sticker.ID == ids[10] {
			t.Fatal("已移除貼圖不應出現在最近使用清單")
		}
	}
	// 不可見包的貼圖不得記錄。
	paid := mustPack(t, svc, "u-a", "付費包", false)
	mustPublish(t, svc, "u-a", paid.Pack.ID)
	paidSticker, err := svc.AddSticker(ctx, "u-a", paid.Pack.ID, "f-1", "付費貼圖", false)
	if err != nil {
		t.Fatalf("AddSticker: %v", err)
	}
	if _, err := svc.RecordStickerUse(ctx, "u-b", paidSticker.ID); !errors.Is(err, ErrStickerNotFound) {
		t.Fatalf("不可見包貼圖記錄應為 404，現為 %v", err)
	}
}

func TestSearchStickers(t *testing.T) {
	ctx := context.Background()
	svc := testService(&fakeMedia{allowed: map[string]bool{"f-1": true}})
	pack := mustPack(t, svc, "u-a", "貓咪日常", true)
	mustPublish(t, svc, "u-a", pack.Pack.ID)
	if _, err := svc.AddSticker(ctx, "u-a", pack.Pack.ID, "f-1", "開心貓", false); err != nil {
		t.Fatalf("AddSticker: %v", err)
	}
	if _, err := svc.AddSticker(ctx, "u-a", pack.Pack.ID, "f-1", "睡覺狗", false); err != nil {
		t.Fatalf("AddSticker: %v", err)
	}
	other := mustPack(t, svc, "u-a", "狗狗樂園", true)
	mustPublish(t, svc, "u-a", other.Pack.ID)
	if _, err := svc.AddSticker(ctx, "u-a", other.Pack.ID, "f-1", "奔跑狗", false); err != nil {
		t.Fatalf("AddSticker: %v", err)
	}

	// q 必填；AND 比對：兩 token 皆須命中（貼圖名或包名）。
	if _, _, err := svc.SearchStickers(ctx, "u-b", "   ", CategoryGeneral, "", 10); err == nil {
		t.Fatal("空關鍵字應為欄位錯誤")
	}
	results, _, err := svc.SearchStickers(ctx, "u-b", "貓", "", "", 10)
	if err != nil {
		t.Fatalf("SearchStickers: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("搜尋「貓」= %d 筆, want 2（包名命中兩張）", len(results))
	}
	results, _, err = svc.SearchStickers(ctx, "u-b", "貓 狗", "", "", 10)
	if err != nil {
		t.Fatalf("SearchStickers: %v", err)
	}
	// token 比對貼圖名與包名的聯集：「睡覺狗」（包名「貓咪日常」）
	// 同時含兩 token，故命中 1 筆。
	if len(results) != 1 || results[0].Sticker.Name != "睡覺狗" {
		t.Fatalf("搜尋「貓 狗」= %d 筆, want 1（睡覺狗）", len(results))
	}
	// 分類篩選：包分類須一致。
	results, _, err = svc.SearchStickers(ctx, "u-b", "狗", CategoryGeneral, "", 10)
	if err != nil {
		t.Fatalf("SearchStickers: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("搜尋「狗」= %d 筆, want 2", len(results))
	}
	// 不可見包的貼圖不回傳。
	paid := mustPack(t, svc, "u-a", "付費貓", false)
	mustPublish(t, svc, "u-a", paid.Pack.ID)
	if _, err := svc.AddSticker(ctx, "u-a", paid.Pack.ID, "f-1", "貴族貓", false); err != nil {
		t.Fatalf("AddSticker: %v", err)
	}
	results, _, err = svc.SearchStickers(ctx, "u-b", "貓", "", "", 10)
	if err != nil {
		t.Fatalf("SearchStickers: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("搜尋「貓」含付費包 = %d 筆, want 2", len(results))
	}
}

func TestListPacks_FilterAndPagination(t *testing.T) {
	ctx := context.Background()
	svc := testService(nil)
	for i := 0; i < 5; i++ {
		entry := mustPack(t, svc, "u-a", "包", true)
		mustPublish(t, svc, "u-a", entry.Pack.ID)
	}
	// 分頁：limit 2，游標續拉不重不漏。
	var seen []string
	cursor := ""
	for {
		page, next, err := svc.ListPacks(ctx, "u-b", "", "", cursor, 2)
		if err != nil {
			t.Fatalf("ListPacks: %v", err)
		}
		for _, e := range page {
			seen = append(seen, e.Pack.ID)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if len(seen) != 5 {
		t.Fatalf("分頁合計 = %d 筆, want 5", len(seen))
	}
	// 壞游標為欄位錯誤。
	if _, _, err := svc.ListPacks(ctx, "u-b", "", "", "!!!", 10); err == nil {
		t.Fatal("壞游標應為欄位錯誤")
	}
	// 作者清單：陌生人只見已上架包；作者本人見草稿。
	draft := mustPack(t, svc, "u-a", "草稿", true)
	mine, _, err := svc.ListPacksByCreator(ctx, "u-a", "u-a", "", 10)
	if err != nil {
		t.Fatalf("ListPacksByCreator: %v", err)
	}
	if len(mine) != 6 {
		t.Fatalf("作者本人清單 = %d 筆, want 6", len(mine))
	}
	others, _, err := svc.ListPacksByCreator(ctx, "u-b", "u-a", "", 10)
	if err != nil {
		t.Fatalf("ListPacksByCreator: %v", err)
	}
	if len(others) != 5 {
		t.Fatalf("陌生人作者清單 = %d 筆, want 5", len(others))
	}
	_ = draft
	// 作者不存在 → 404。
	if _, _, err := svc.ListPacksByCreator(ctx, "u-b", "u-x", "", 10); !errors.Is(err, ErrPackNotFound) {
		t.Fatalf("作者不存在應為 404，現為 %v", err)
	}
}

func TestStickerCap(t *testing.T) {
	ctx := context.Background()
	svc := testService(&fakeMedia{allowed: map[string]bool{"f-1": true}})
	entry := mustPack(t, svc, "u-a", "包", true)
	for i := 0; i < MaxStickersPerPack; i++ {
		if _, err := svc.AddSticker(ctx, "u-a", entry.Pack.ID, "f-1", "貼圖", false); err != nil {
			t.Fatalf("AddSticker %d: %v", i, err)
		}
	}
	if _, err := svc.AddSticker(ctx, "u-a", entry.Pack.ID, "f-1", "超量", false); err == nil {
		t.Fatal("超過上限應為欄位錯誤")
	}
}
