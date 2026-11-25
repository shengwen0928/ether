// Package friend 的 HTTP handler：/api/v1/friends 前綴下的好友端點
// （API_STANDARDS.md §2）。全部端點皆須經 RequireAuth；回應不得含
// 私密欄位（電話、Email 不在此回傳）；被封鎖者的一切查詢皆以
// 404 回應，不得洩漏封鎖狀態（ERROR_CODES.md §6.3）。
package friend

import (
	"errors"
	"net/http"
	"time"

	"ether.local/api/internal/transport"
)

// Handler 持有 friends 端點所需的服務。
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

// writeFriendError 將領域錯誤對應為 HTTP 回應：欄位驗證錯誤 →
// 400＋details；三種「不存在」哨兵 → 404（訊息不可區分）；其餘 → 500。
func writeFriendError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrRequestNotFound) || errors.Is(err, ErrFriendshipNotFound) || errors.Is(err, ErrUserNotFound) {
		transport.WriteError(w, http.StatusNotFound, "ETHER_COMMON_NOT_FOUND", "找不到資源。")
		return
	}
	transport.WriteServiceError(w, err, nil)
}

type friendResponse struct {
	FriendID      string    `json:"friend_id"`
	DisplayName   string    `json:"display_name"`
	PublicID      string    `json:"public_id"`
	AvatarMediaID string    `json:"avatar_media_id"`
	StatusMessage string    `json:"status_message"`
	IsHidden      bool      `json:"is_hidden"`
	EstablishedAt time.Time `json:"established_at"`
}

func toFriendResponse(entry FriendEntry) friendResponse {
	return friendResponse{
		FriendID:      entry.Friend.UserID,
		DisplayName:   entry.Friend.DisplayName,
		PublicID:      entry.Friend.PublicID,
		AvatarMediaID: entry.Friend.AvatarMediaID,
		StatusMessage: entry.Friend.StatusMessage,
		IsHidden:      entry.Hidden,
		EstablishedAt: entry.AddedAt.UTC(),
	}
}

// ListFriends 列出好友（GET /api/v1/friends）。預設排除隱藏者；
// ?include_hidden=true 一併列出並標示 is_hidden。
func (h *Handler) ListFriends(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entries, err := h.Service.ListFriends(r.Context(), userID, r.URL.Query().Get("include_hidden") == "true")
	if err != nil {
		writeFriendError(w, err)
		return
	}
	out := make([]friendResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toFriendResponse(e))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type searchResponse struct {
	UserID        string `json:"user_id"`
	DisplayName   string `json:"display_name"`
	PublicID      string `json:"public_id"`
	AvatarMediaID string `json:"avatar_media_id"`
	StatusMessage string `json:"status_message"`
}

// SearchByPublicID 依公開識別碼搜尋使用者（GET
// /api/v1/friends/search?public_id=...）。查無、搜尋自己、或
// 任一方向封鎖中，皆回傳 404，不得區分原因。
func (h *Handler) SearchByPublicID(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	publicID := r.URL.Query().Get("public_id")
	if publicID == "" {
		transport.WriteValidationError(w, []transport.FieldError{{
			Field: "public_id", Issue: "REQUIRED", Message: "請提供公開識別碼。",
		}})
		return
	}
	summary, err := h.Service.SearchByPublicID(r.Context(), userID, publicID)
	if err != nil {
		writeFriendError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, searchResponse{
		UserID:        summary.UserID,
		DisplayName:   summary.DisplayName,
		PublicID:      summary.PublicID,
		AvatarMediaID: summary.AvatarMediaID,
		StatusMessage: summary.StatusMessage,
	})
}

type requestResponse struct {
	RequestID   string     `json:"request_id"`
	RequesterID string     `json:"requester_id"`
	AddresseeID string     `json:"addressee_id"`
	Status      string     `json:"status"`
	Message     string     `json:"message"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	// 以下僅清單回應填寫（對端摘要與方向，依呼叫者視角）。
	OtherUserID      string `json:"other_user_id,omitempty"`
	OtherDisplayName string `json:"other_display_name,omitempty"`
	OtherPublicID    string `json:"other_public_id,omitempty"`
	Direction        string `json:"direction,omitempty"`
}

func toRequestResponse(req FriendRequest) requestResponse {
	var responded *time.Time
	if req.RespondedAt != nil {
		u := req.RespondedAt.UTC()
		responded = &u
	}
	return requestResponse{
		RequestID:   req.ID,
		RequesterID: req.RequesterID,
		AddresseeID: req.AddresseeID,
		Status:      string(req.Status),
		Message:     req.Message,
		RespondedAt: responded,
		CreatedAt:   req.CreatedAt.UTC(),
		UpdatedAt:   req.UpdatedAt.UTC(),
	}
}

func toRequestEntryResponse(entry RequestEntry) requestResponse {
	resp := toRequestResponse(entry.Request)
	resp.OtherUserID = entry.Other.UserID
	resp.OtherDisplayName = entry.Other.DisplayName
	resp.OtherPublicID = entry.Other.PublicID
	if entry.Incoming {
		resp.Direction = "incoming"
	} else {
		resp.Direction = "outgoing"
	}
	return resp
}

// ListRequests 列出待處理邀請（GET /api/v1/friends/requests）。
// ?direction=incoming（預設，收到的）或 outgoing（送出的）。
func (h *Handler) ListRequests(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	direction := r.URL.Query().Get("direction")
	incoming := true
	switch direction {
	case "", "incoming":
	case "outgoing":
		incoming = false
	default:
		transport.WriteValidationError(w, []transport.FieldError{{
			Field: "direction", Issue: "INVALID_VALUE", Message: "direction 須為 incoming 或 outgoing。",
		}})
		return
	}
	entries, err := h.Service.ListRequests(r.Context(), userID, incoming)
	if err != nil {
		writeFriendError(w, err)
		return
	}
	out := make([]requestResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toRequestEntryResponse(e))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type sendRequestBody struct {
	AddresseeID string `json:"addressee_id"`
	Message     string `json:"message"`
}

// SendRequest 發送好友邀請（POST /api/v1/friends/requests）。
func (h *Handler) SendRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req sendRequestBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	created, err := h.Service.SendRequest(r.Context(), userID, req.AddresseeID, req.Message)
	if err != nil {
		writeFriendError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusCreated, toRequestResponse(*created))
}

// RespondRequest 接受邀請（POST /api/v1/friends/requests/{request_id}/accept）。
func (h *Handler) AcceptRequest(w http.ResponseWriter, r *http.Request) {
	h.respond(w, r, true)
}

// DeclineRequest 拒絕邀請（POST /api/v1/friends/requests/{request_id}/decline）。
func (h *Handler) DeclineRequest(w http.ResponseWriter, r *http.Request) {
	h.respond(w, r, false)
}

func (h *Handler) respond(w http.ResponseWriter, r *http.Request, accept bool) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	updated, err := h.Service.RespondRequest(r.Context(), userID, r.PathValue("request_id"), accept)
	if err != nil {
		writeFriendError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toRequestResponse(*updated))
}

// CancelRequest 取消自己送出的邀請（POST /api/v1/friends/requests/{request_id}/cancel）。
func (h *Handler) CancelRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	updated, err := h.Service.CancelRequest(r.Context(), userID, r.PathValue("request_id"))
	if err != nil {
		writeFriendError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toRequestResponse(*updated))
}

type friendActionResponse struct {
	FriendID string `json:"friend_id"`
	Status   string `json:"status"`
	IsHidden bool   `json:"is_hidden,omitempty"`
}

// RemoveFriend 解除好友關係（POST /api/v1/friends/{friend_id}/remove）。
// 非好友一律 404。
func (h *Handler) RemoveFriend(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	friendID := r.PathValue("friend_id")
	if err := h.Service.RemoveFriend(r.Context(), userID, friendID); err != nil {
		writeFriendError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, friendActionResponse{FriendID: friendID, Status: "ended"})
}

// HideFriend 隱藏好友（POST /api/v1/friends/{friend_id}/hide）。
func (h *Handler) HideFriend(w http.ResponseWriter, r *http.Request) {
	h.setHidden(w, r, true)
}

// UnhideFriend 取消隱藏（POST /api/v1/friends/{friend_id}/unhide）。
func (h *Handler) UnhideFriend(w http.ResponseWriter, r *http.Request) {
	h.setHidden(w, r, false)
}

func (h *Handler) setHidden(w http.ResponseWriter, r *http.Request, hidden bool) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	friendID := r.PathValue("friend_id")
	if err := h.Service.SetHidden(r.Context(), userID, friendID, hidden); err != nil {
		writeFriendError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, friendActionResponse{FriendID: friendID, Status: "active", IsHidden: hidden})
}

type blockResponse struct {
	BlockedUserID string     `json:"blocked_user_id"`
	DisplayName   string     `json:"display_name,omitempty"`
	PublicID      string     `json:"public_id,omitempty"`
	Reason        string     `json:"reason"`
	BlockedAt     time.Time  `json:"blocked_at"`
	UnblockedAt   *time.Time `json:"unblocked_at,omitempty"`
}

func toBlockResponse(b Block, summary ProfileSummary) blockResponse {
	var unblocked *time.Time
	if b.UnblockedAt != nil {
		u := b.UnblockedAt.UTC()
		unblocked = &u
	}
	return blockResponse{
		BlockedUserID: b.BlockedUserID,
		DisplayName:   summary.DisplayName,
		PublicID:      summary.PublicID,
		Reason:        b.Reason,
		BlockedAt:     b.BlockedAt.UTC(),
		UnblockedAt:   unblocked,
	}
}

// ListBlocks 列出生效封鎖（GET /api/v1/friends/blocks）。
func (h *Handler) ListBlocks(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entries, err := h.Service.ListBlocks(r.Context(), userID)
	if err != nil {
		writeFriendError(w, err)
		return
	}
	out := make([]blockResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toBlockResponse(e.Block, e.Blocked))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type blockRequestBody struct {
	BlockedUserID string `json:"blocked_user_id"`
	Reason        string `json:"reason"`
}

// Block 封鎖使用者（POST /api/v1/friends/blocks）。已封鎖中則
// 冪等回傳既有封鎖（200）；新建封鎖回傳 201。
func (h *Handler) Block(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req blockRequestBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	created, isNew, err := h.Service.Block(r.Context(), userID, req.BlockedUserID, req.Reason)
	if err != nil {
		writeFriendError(w, err)
		return
	}
	status := http.StatusOK
	if isNew {
		status = http.StatusCreated
	}
	transport.WriteJSON(w, status, toBlockResponse(*created, ProfileSummary{}))
}

// Unblock 解除封鎖（POST /api/v1/friends/blocks/{blocked_user_id}/unblock）。
func (h *Handler) Unblock(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	unblocked, err := h.Service.Unblock(r.Context(), userID, r.PathValue("blocked_user_id"))
	if err != nil {
		writeFriendError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toBlockResponse(*unblocked, ProfileSummary{}))
}
