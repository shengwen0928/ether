// Package profile 的 HTTP handler：/api/v1/profiles 前綴下的
// 公開檔案端點（API_STANDARDS.md §2）。全部端點皆須經 RequireAuth；
// 回應不得含私密欄位（電話、Email 不在此回傳）。
package profile

import (
	"net/http"
	"time"

	"ether.local/api/internal/transport"
)

// Handler 持有 profiles 端點所需的服務。
type Handler struct {
	Service     *Service
	CurrentUser func(r *http.Request) (string, bool)
}

type profileResponse struct {
	UserID        string    `json:"user_id"`
	DisplayName   string    `json:"display_name"`
	PublicID      string    `json:"public_id"`
	AvatarMediaID string    `json:"avatar_media_id"`
	Bio           string    `json:"bio"`
	StatusMessage string    `json:"status_message"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toProfileResponse(p *Profile) profileResponse {
	return profileResponse{
		UserID:        p.UserID,
		DisplayName:   p.DisplayName,
		PublicID:      p.PublicID,
		AvatarMediaID: p.AvatarMediaID,
		Bio:           p.Bio,
		StatusMessage: p.StatusMessage,
		UpdatedAt:     p.UpdatedAt.UTC(),
	}
}

// GetMe 回傳目前登入者的公開檔案（GET /api/v1/profiles/me）。
// 檔案不存在時以預設顯示名稱建立（顯示名稱取 subject 縮寫，
// 不得為空）。
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.CurrentUser(r)
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return
	}
	p, err := h.Service.GetOrCreate(r.Context(), userID, "")
	if err != nil {
		transport.WriteServiceError(w, err, ErrProfileNotFound)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toProfileResponse(p))
}

type profileRequest struct {
	DisplayName   *string `json:"display_name"`
	PublicID      *string `json:"public_id"`
	AvatarMediaID *string `json:"avatar_media_id"`
	Bio           *string `json:"bio"`
	StatusMessage *string `json:"status_message"`
}

// UpdateMe 部分更新公開檔案（PATCH /api/v1/profiles/me）。
// 未提供欄位不更動；未知欄位拒絕（嚴格模式）。
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.CurrentUser(r)
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return
	}
	var req profileRequest
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	updated, err := h.Service.Update(r.Context(), userID, Patch{
		DisplayName:   req.DisplayName,
		PublicID:      req.PublicID,
		AvatarMediaID: req.AvatarMediaID,
		Bio:           req.Bio,
		StatusMessage: req.StatusMessage,
	})
	if err != nil {
		transport.WriteServiceError(w, err, ErrProfileNotFound)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toProfileResponse(updated))
}
