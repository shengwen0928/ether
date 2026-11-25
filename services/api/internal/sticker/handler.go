// Package sticker 的 HTTP handler：/api/v1/sticker-packs・
// /api/v1/stickers 前綴下的貼圖端點（API_STANDARDS.md §5.3，
// Sticker 擁有 /api/v1/stickers 前綴）。全部端點皆須經
// RequireAuth；不可見的貼圖包／貼圖一律以 404 回應，不得洩漏
// 存在性（ERROR_CODES.md §6.3）。貼圖無對應 WebSocket 事件名
// （§21 的 15 個事件皆為訊息事件），本里程碑不寫 Outbox 事件，
// 客戶端以 REST 查詢同步。購買交易端點屬 Payment Integration，
// 本模組不提供。
package sticker

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"ether.local/api/internal/transport"
)

// Handler 持有貼圖端點所需的服務。
type Handler struct {
	Service     *Service
	CurrentUser func(r *http.Request) (string, bool)
}

func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := h.CurrentUser(r)
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return "", false
	}
	return userID, true
}

// writeStickerError 將領域錯誤對應為 HTTP 回應：欄位驗證錯誤 →
// 400＋details；ErrPackNotFound／ErrStickerNotFound → 404（訊息
// 不可區分）；ErrCheckerUnavailable → 503 fail-closed；其餘 → 500。
func writeStickerError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrCheckerUnavailable) {
		transport.WriteError(w, http.StatusServiceUnavailable,
			"ETHER_COMMON_UNAVAILABLE", "服務暫時無法使用。")
		return
	}
	if errors.Is(err, ErrStickerNotFound) {
		transport.WriteError(w, http.StatusNotFound, "ETHER_COMMON_NOT_FOUND", "找不到資源。")
		return
	}
	transport.WriteServiceError(w, err, ErrPackNotFound)
}

func utcTime(t time.Time) time.Time { return t.UTC() }

type packResponse struct {
	PackID        string    `json:"pack_id"`
	CreatorID     string    `json:"creator_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Category      string    `json:"category"`
	CoverMediaID  string    `json:"cover_media_id"`
	IsFree        bool      `json:"is_free"`
	Status        string    `json:"status"`
	StickerCount  int       `json:"sticker_count"`
	FavoritedByMe bool      `json:"favorited_by_me"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toPackResponse(entry *PackEntry) packResponse {
	return packResponse{
		PackID:        entry.Pack.ID,
		CreatorID:     entry.Pack.CreatorID,
		Name:          entry.Pack.Name,
		Description:   entry.Pack.Description,
		Category:      string(entry.Pack.Category),
		CoverMediaID:  entry.Pack.CoverMediaID,
		IsFree:        entry.Pack.IsFree,
		Status:        string(entry.Pack.Status),
		StickerCount:  entry.StickerCount,
		FavoritedByMe: entry.FavoritedByMe,
		CreatedAt:     utcTime(entry.Pack.CreatedAt),
		UpdatedAt:     utcTime(entry.Pack.UpdatedAt),
	}
}

type stickerResponse struct {
	StickerID   string    `json:"sticker_id"`
	PackID      string    `json:"pack_id"`
	PackName    string    `json:"pack_name"`
	MediaFileID string    `json:"media_file_id"`
	Name        string    `json:"name"`
	IsAnimated  bool      `json:"is_animated"`
	Position    int       `json:"position"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toStickerResponse(entry StickerEntry) stickerResponse {
	return stickerResponse{
		StickerID:   entry.Sticker.ID,
		PackID:      entry.Sticker.PackID,
		PackName:    entry.PackName,
		MediaFileID: entry.Sticker.MediaFileID,
		Name:        entry.Sticker.Name,
		IsAnimated:  entry.Sticker.IsAnimated,
		Position:    entry.Sticker.Position,
		Status:      string(entry.Sticker.Status),
		CreatedAt:   utcTime(entry.Sticker.CreatedAt),
		UpdatedAt:   utcTime(entry.Sticker.UpdatedAt),
	}
}

type createPackBody struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	CoverMediaID string `json:"cover_media_id"`
	IsFree       bool   `json:"is_free"`
}

// CreatePack 建立貼圖包（POST /api/v1/sticker-packs）。新包一律為
// 草稿，須經 PATCH 上架。
func (h *Handler) CreatePack(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req createPackBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	entry, err := h.Service.CreatePack(r.Context(), userID, req.Name, req.Description, PackCategory(req.Category), req.CoverMediaID, req.IsFree)
	if err != nil {
		writeStickerError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusCreated, toPackResponse(entry))
}

// GetPack 讀取單一貼圖包（GET /api/v1/sticker-packs/{pack_id}）。
func (h *Handler) GetPack(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entry, err := h.Service.GetPack(r.Context(), userID, r.PathValue("pack_id"))
	if err != nil {
		writeStickerError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toPackResponse(entry))
}

type updatePackBody struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	Category     *string `json:"category"`
	CoverMediaID *string `json:"cover_media_id"`
	Status       *string `json:"status"`
	IsFree       *bool   `json:"is_free"`
}

// UpdatePack 編輯貼圖包（PATCH /api/v1/sticker-packs/{pack_id}）。
func (h *Handler) UpdatePack(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req updatePackBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	patch := PackPatch{Name: req.Name, Description: req.Description, IsFree: req.IsFree}
	if req.Category != nil {
		c := PackCategory(*req.Category)
		patch.Category = &c
	}
	if req.CoverMediaID != nil {
		patch.CoverMediaID = req.CoverMediaID
	}
	if req.Status != nil {
		s := PackStatus(*req.Status)
		patch.Status = &s
	}
	entry, err := h.Service.UpdatePack(r.Context(), userID, r.PathValue("pack_id"), patch)
	if err != nil {
		writeStickerError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toPackResponse(entry))
}

// DeletePack 刪除貼圖包（POST /api/v1/sticker-packs/{pack_id}/delete）。
// 刪除為 archived 狀態標記，刪除後一切讀取皆以 404 回應。
func (h *Handler) DeletePack(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if err := h.Service.DeletePack(r.Context(), userID, r.PathValue("pack_id")); err != nil {
		writeStickerError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

// parseLimit 解析 limit 查詢參數：缺省為 0（由服務層取預設值）；
// 非數字或負數為欄位錯誤。
func parseLimit(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, errors.New("limit 須為非負整數")
	}
	return n, nil
}

type packListResponse struct {
	Packs      []packResponse `json:"packs"`
	NextCursor string         `json:"next_cursor"`
}

// ListPacks 列出呼叫者可見的貼圖包（GET /api/v1/sticker-packs）。
// creator_id 有值時只列該作者的包；category／q 為選填篩選。
func (h *Handler) ListPacks(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	limit, err := parseLimit(r)
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "limit 參數錯誤。")
		return
	}
	q := r.URL.Query()
	creatorID := q.Get("creator_id")
	category := PackCategory(q.Get("category"))
	keyword := q.Get("q")
	cursor := q.Get("cursor")
	var entries []*PackEntry
	var next string
	if creatorID != "" {
		entries, next, err = h.Service.ListPacksByCreator(r.Context(), userID, creatorID, cursor, limit)
	} else {
		entries, next, err = h.Service.ListPacks(r.Context(), userID, category, keyword, cursor, limit)
	}
	if err != nil {
		writeStickerError(w, err)
		return
	}
	out := make([]packResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toPackResponse(e))
	}
	transport.WriteJSON(w, http.StatusOK, packListResponse{Packs: out, NextCursor: next})
}

type addStickerBody struct {
	MediaFileID string `json:"media_file_id"`
	Name        string `json:"name"`
	IsAnimated  bool   `json:"is_animated"`
}

// AddSticker 新增貼圖至貼圖包（POST /api/v1/sticker-packs/{pack_id}/stickers）。
func (h *Handler) AddSticker(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req addStickerBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	st, err := h.Service.AddSticker(r.Context(), userID, r.PathValue("pack_id"), req.MediaFileID, req.Name, req.IsAnimated)
	if err != nil {
		writeStickerError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusCreated, toStickerResponse(StickerEntry{Sticker: *st}))
}

// ListStickers 列出某包的生效貼圖（GET /api/v1/sticker-packs/{pack_id}/stickers）。
func (h *Handler) ListStickers(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entries, err := h.Service.ListStickers(r.Context(), userID, r.PathValue("pack_id"))
	if err != nil {
		writeStickerError(w, err)
		return
	}
	out := make([]stickerResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toStickerResponse(e))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

// RemoveSticker 移除貼圖（POST /api/v1/stickers/{sticker_id}/delete）。
// 移除為狀態標記，移除後讀取一律 404。
func (h *Handler) RemoveSticker(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if err := h.Service.RemoveSticker(r.Context(), userID, r.PathValue("sticker_id")); err != nil {
		writeStickerError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// FavoritePack 收藏貼圖包（POST /api/v1/sticker-packs/{pack_id}/favorite）。
// 重複收藏冪等成功。
func (h *Handler) FavoritePack(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entry, err := h.Service.FavoritePack(r.Context(), userID, r.PathValue("pack_id"))
	if err != nil {
		writeStickerError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toPackResponse(entry))
}

// UnfavoritePack 取消收藏（DELETE /api/v1/sticker-packs/{pack_id}/favorite）。
// 未收藏時冪等成功。
func (h *Handler) UnfavoritePack(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entry, err := h.Service.UnfavoritePack(r.Context(), userID, r.PathValue("pack_id"))
	if err != nil {
		writeStickerError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toPackResponse(entry))
}

// ListFavorites 列出使用者的收藏貼圖包（GET /api/v1/stickers/favorites）。
func (h *Handler) ListFavorites(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	limit, err := parseLimit(r)
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "limit 參數錯誤。")
		return
	}
	entries, next, err := h.Service.ListFavorites(r.Context(), userID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeStickerError(w, err)
		return
	}
	out := make([]packResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toPackResponse(e))
	}
	transport.WriteJSON(w, http.StatusOK, packListResponse{Packs: out, NextCursor: next})
}

// RecordStickerUse 記錄最近使用（POST /api/v1/stickers/{sticker_id}/use）。
func (h *Handler) RecordStickerUse(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entry, err := h.Service.RecordStickerUse(r.Context(), userID, r.PathValue("sticker_id"))
	if err != nil {
		writeStickerError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toStickerResponse(*entry))
}

// ListRecentStickers 列出使用者的最近使用貼圖（GET /api/v1/stickers/recent）。
func (h *Handler) ListRecentStickers(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	limit, err := parseLimit(r)
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "limit 參數錯誤。")
		return
	}
	entries, next, err := h.Service.ListRecentStickers(r.Context(), userID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeStickerError(w, err)
		return
	}
	out := make([]stickerResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toStickerResponse(*e))
	}
	transport.WriteJSON(w, http.StatusOK, struct {
		Stickers   []stickerResponse `json:"stickers"`
		NextCursor string            `json:"next_cursor"`
	}{Stickers: out, NextCursor: next})
}

type stickerListResponse struct {
	Stickers   []stickerResponse `json:"stickers"`
	NextCursor string            `json:"next_cursor"`
}

// SearchStickers 搜尋貼圖（GET /api/v1/stickers/search?q=…）。
func (h *Handler) SearchStickers(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	limit, err := parseLimit(r)
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "limit 參數錯誤。")
		return
	}
	q := r.URL.Query()
	entries, next, err := h.Service.SearchStickers(r.Context(), userID, q.Get("q"), PackCategory(q.Get("category")), q.Get("cursor"), limit)
	if err != nil {
		writeStickerError(w, err)
		return
	}
	out := make([]stickerResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toStickerResponse(*e))
	}
	transport.WriteJSON(w, http.StatusOK, stickerListResponse{Stickers: out, NextCursor: next})
}
