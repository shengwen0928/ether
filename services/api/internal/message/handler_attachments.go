// 訊息附件關聯與轉傳的 HTTP handler（見 handler.go 的檔頭慣例：
// 全部端點皆須經 RequireAuth；非成員的一切查詢皆以 404 回應，
// 不得洩漏存在性）。
package message

import (
	"errors"
	"net/http"
	"time"

	"ether.local/api/internal/transport"
)

func writeAttachmentError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrMediaUnavailable) {
		transport.WriteError(w, http.StatusServiceUnavailable,
			"ETHER_COMMON_UNAVAILABLE", "附件服務未就緒。")
		return
	}
	writeMessageError(w, err)
}

type attachmentResponse struct {
	AttachmentID string    `json:"attachment_id"`
	MessageID    string    `json:"message_id"`
	MediaFileID  string    `json:"media_file_id"`
	MIMEType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	FileName     string    `json:"file_name"`
	CreatedAt    time.Time `json:"created_at"`
}

func toAttachmentResponse(a *Attachment) attachmentResponse {
	return attachmentResponse{
		AttachmentID: a.ID,
		MessageID:    a.MessageID,
		MediaFileID:  a.MediaFileID,
		MIMEType:     a.MIMEType,
		SizeBytes:    a.SizeBytes,
		FileName:     a.FileName,
		CreatedAt:    a.CreatedAt.UTC(),
	}
}

type attachmentListResponse struct {
	Attachments []attachmentResponse `json:"attachments"`
}

func toAttachmentListResponse(atts []*Attachment) attachmentListResponse {
	out := attachmentListResponse{Attachments: []attachmentResponse{}}
	for _, a := range atts {
		out.Attachments = append(out.Attachments, toAttachmentResponse(a))
	}
	return out
}

type addAttachmentsBody struct {
	MediaFileIDs []string `json:"media_file_ids"`
}

// AddAttachments 為既有訊息新增附件（POST /api/v1/messages/{message_id}/attachments）。
// Upsert 語義：一律回傳 200＋完整附件清單（新建、重複關聯皆同）。
func (h *Handler) AddAttachments(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req addAttachmentsBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	atts, err := h.Service.AddAttachments(r.Context(), userID, r.PathValue("message_id"), req.MediaFileIDs)
	if err != nil {
		writeAttachmentError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toAttachmentListResponse(atts))
}

// ListAttachments 列出訊息的附件（GET /api/v1/messages/{message_id}/attachments）。
func (h *Handler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	atts, err := h.Service.ListAttachments(r.Context(), userID, r.PathValue("message_id"))
	if err != nil {
		writeAttachmentError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toAttachmentListResponse(atts))
}

type forwardMessageBody struct {
	TargetConversationID string `json:"target_conversation_id"`
	ClientMessageID      string `json:"client_message_id"`
}

// ForwardMessage 轉傳訊息（POST /api/v1/messages/{message_id}/forward）。
// 冪等命中（同一去重鍵）回傳 200＋既有訊息，新建回傳 201。
// 回應為新訊息（message.received 事件承載同一內容）；轉傳紀錄
// 以 GET /api/v1/messages/{message_id}/forward 另查。
func (h *Handler) ForwardMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req forwardMessageBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	result, err := h.Service.Forward(r.Context(), userID, r.PathValue("message_id"),
		req.TargetConversationID, req.ClientMessageID)
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

type forwardResponse struct {
	ForwardID            string    `json:"forward_id"`
	SourceMessageID      string    `json:"source_message_id"`
	ForwardedMessageID   string    `json:"forwarded_message_id"`
	SourceConversationID string    `json:"source_conversation_id"`
	TargetConversationID string    `json:"target_conversation_id"`
	ForwardedBy          string    `json:"forwarded_by"`
	CreatedAt            time.Time `json:"created_at"`
}

// GetForward 讀取訊息的轉傳紀錄（GET /api/v1/messages/{message_id}/forward）。
// 訊息並非轉傳產物時回傳 404（與查無訊息／非成員同一回應）。
func (h *Handler) GetForward(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	fwd, err := h.Service.GetForward(r.Context(), userID, r.PathValue("message_id"))
	if err != nil {
		writeAttachmentError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, forwardResponse{
		ForwardID:            fwd.ID,
		SourceMessageID:      fwd.SourceMessageID,
		ForwardedMessageID:   fwd.ForwardedMessageID,
		SourceConversationID: fwd.SourceConversationID,
		TargetConversationID: fwd.TargetConversationID,
		ForwardedBy:          fwd.ForwardedBy,
		CreatedAt:            fwd.CreatedAt.UTC(),
	})
}
