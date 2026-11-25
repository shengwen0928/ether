// Package message 的 HTTP handler：對話內訊息端點（巢狀於
// /api/v1/conversations）與訊息端點（/api/v1/messages 前綴，
// API_STANDARDS.md §2）。全部端點皆須經 RequireAuth；非成員的
// 一切查詢皆以 404 回應，不得洩漏存在性（ERROR_CODES.md §6.3）。
// 收回／僅自己刪除的訊息，內容一律以空字串回傳，不得回傳原值。
package message

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"ether.local/api/internal/transport"
)

// Handler 持有訊息端點所需的服務。
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

func writeMessageError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrMessageNotFound) {
		transport.WriteError(w, http.StatusNotFound, "ETHER_COMMON_NOT_FOUND", "找不到資源。")
		return
	}
	transport.WriteServiceError(w, err, nil)
}

type messageResponse struct {
	MessageID        string     `json:"message_id"`
	ConversationID   string     `json:"conversation_id"`
	SenderID         string     `json:"sender_id"`
	Kind             string     `json:"kind"`
	Content          string     `json:"content"`
	Status           string     `json:"status"`
	ClientMessageID  string     `json:"client_message_id,omitempty"`
	ReplyToMessageID string     `json:"reply_to_message_id,omitempty"`
	SequenceNumber   int64      `json:"sequence_number"`
	EditCount        int        `json:"edit_count"`
	EditedAt         *time.Time `json:"edited_at,omitempty"`
	RecalledAt       *time.Time `json:"recalled_at,omitempty"`
	DeletedForSelf   bool       `json:"deleted_for_self"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// toMessageResponse 將領域訊息轉為回應：收回訊息內容為空；僅
// 自己刪除的訊息，只對作者本人顯示為已刪除（內容為空）。
func toMessageResponse(m *Message, viewerID string) messageResponse {
	content := m.Content
	deletedForSelf := false
	if m.Status == StatusRecalled {
		content = ""
	}
	if m.DeletedForSenderAt != nil && m.SenderID == viewerID {
		content = ""
		deletedForSelf = true
	}
	var edited, recalled *time.Time
	if m.EditedAt != nil {
		u := m.EditedAt.UTC()
		edited = &u
	}
	if m.RecalledAt != nil {
		u := m.RecalledAt.UTC()
		recalled = &u
	}
	return messageResponse{
		MessageID:        m.ID,
		ConversationID:   m.ConversationID,
		SenderID:         m.SenderID,
		Kind:             string(m.Kind),
		Content:          content,
		Status:           string(m.Status),
		ClientMessageID:  m.ClientMessageID,
		ReplyToMessageID: m.ReplyToMessageID,
		SequenceNumber:   m.SequenceNumber,
		EditCount:        m.EditCount,
		EditedAt:         edited,
		RecalledAt:       recalled,
		DeletedForSelf:   deletedForSelf,
		CreatedAt:        m.CreatedAt.UTC(),
		UpdatedAt:        m.UpdatedAt.UTC(),
	}
}

type sendMessageBody struct {
	Kind              string   `json:"kind"`
	Content           string   `json:"content"`
	ClientMessageID   string   `json:"client_message_id"`
	ReplyToMessageID  string   `json:"reply_to_message_id"`
	AttachmentFileIDs []string `json:"attachment_file_ids"`
}

// SendMessage 送出訊息（POST /api/v1/conversations/{conversation_id}/messages）。
// 冪等命中（同一去重鍵）回傳 200＋既有訊息，新建回傳 201。
// attachment_file_ids 非空時，附件列與訊息列同一寫入路徑提交
// （檔案須為發送者所有且掃描已通過，見 attachments.go）。
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req sendMessageBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	kind := req.Kind
	if kind == "" {
		kind = string(KindText)
	}
	result, err := h.Service.SendWithAttachments(r.Context(), userID, r.PathValue("conversation_id"),
		kind, req.Content, req.ClientMessageID, req.ReplyToMessageID, req.AttachmentFileIDs)
	if err != nil {
		writeAttachmentError(w, err)
		return
	}
	status := http.StatusOK
	if result.IsNew {
		status = http.StatusCreated
	}
	transport.WriteJSON(w, status, toMessageResponse(result.Message, userID))
}

type messageListResponse struct {
	Messages   []messageResponse `json:"messages"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

func parseSeqParam(r *http.Request, name string) (*int64, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return nil, errors.New("invalid cursor")
	}
	return &n, nil
}

func parseLimitParam(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, errors.New("invalid limit")
	}
	return n, nil
}

// ListMessages 列出對話訊息（GET /api/v1/conversations/{conversation_id}/messages）。
// 游標分頁（游標為序號，待 API 定稿）：?before_seq=（較舊歷史，倒序）、
// ?since_seq=（續拉，順序，二選一）、?limit=（預設 30、上限 100）。
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	beforeSeq, err := parseSeqParam(r, "before_seq")
	if err != nil {
		transport.WriteValidationError(w, []transport.FieldError{{
			Field: "before_seq", Issue: "INVALID_VALUE", Message: "before_seq 須為非負整數。",
		}})
		return
	}
	sinceSeq, err := parseSeqParam(r, "since_seq")
	if err != nil {
		transport.WriteValidationError(w, []transport.FieldError{{
			Field: "since_seq", Issue: "INVALID_VALUE", Message: "since_seq 須為非負整數。",
		}})
		return
	}
	limit, err := parseLimitParam(r)
	if err != nil {
		transport.WriteValidationError(w, []transport.FieldError{{
			Field: "limit", Issue: "INVALID_VALUE", Message: "limit 須為正整數。",
		}})
		return
	}
	page, err := h.Service.List(r.Context(), userID, r.PathValue("conversation_id"), beforeSeq, sinceSeq, limit)
	if err != nil {
		writeMessageError(w, err)
		return
	}
	out := make([]messageResponse, 0, len(page.Messages))
	for _, m := range page.Messages {
		out = append(out, toMessageResponse(m, userID))
	}
	transport.WriteJSON(w, http.StatusOK, messageListResponse{Messages: out, NextCursor: page.NextCursor})
}

type editMessageBody struct {
	Content string `json:"content"`
}

// EditMessage 編輯訊息（PATCH /api/v1/messages/{message_id}）。
func (h *Handler) EditMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req editMessageBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	updated, err := h.Service.Edit(r.Context(), userID, r.PathValue("message_id"), req.Content)
	if err != nil {
		writeMessageError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toMessageResponse(updated, userID))
}

// RecallMessage 收回訊息（POST /api/v1/messages/{message_id}/recall）。
func (h *Handler) RecallMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	updated, err := h.Service.Recall(r.Context(), userID, r.PathValue("message_id"))
	if err != nil {
		writeMessageError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toMessageResponse(updated, userID))
}

// DeleteMessage 刪除僅自己檢視的訊息（POST /api/v1/messages/{message_id}/delete）。
func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	updated, err := h.Service.DeleteForSelf(r.Context(), userID, r.PathValue("message_id"))
	if err != nil {
		writeMessageError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toMessageResponse(updated, userID))
}

type receiptBody struct {
	ReceiptType string `json:"receipt_type"`
	DeviceID    string `json:"device_id"`
}

type receiptResponse struct {
	MessageID   string    `json:"message_id"`
	UserID      string    `json:"user_id"`
	ReceiptType string    `json:"receipt_type"`
	DeviceID    string    `json:"device_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func toReceiptResponse(r *Receipt) receiptResponse {
	return receiptResponse{
		MessageID:   r.MessageID,
		UserID:      r.UserID,
		ReceiptType: string(r.Type),
		DeviceID:    r.DeviceID,
		CreatedAt:   r.CreatedAt.UTC(),
	}
}

// PostReceipt 寫入回條（POST /api/v1/messages/{message_id}/receipts）。
// 重複寫入同一回條為冪等成功（200＋既有列）。
func (h *Handler) PostReceipt(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req receiptBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	receipt, err := h.Service.PostReceipt(r.Context(), userID, r.PathValue("message_id"),
		ReceiptType(req.ReceiptType), req.DeviceID)
	if err != nil {
		writeMessageError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toReceiptResponse(receipt))
}

// ListReceipts 列出訊息的回條（GET /api/v1/messages/{message_id}/receipts）。
func (h *Handler) ListReceipts(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	receipts, err := h.Service.ListReceipts(r.Context(), userID, r.PathValue("message_id"))
	if err != nil {
		writeMessageError(w, err)
		return
	}
	out := make([]receiptResponse, 0, len(receipts))
	for _, rc := range receipts {
		out = append(out, toReceiptResponse(rc))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}
