// Package device 的 HTTP handler：/api/v1/auth 前綴下的裝置端點
// （API_STANDARDS.md §5.3：裝置端點屬 /auth 或 /users 二選一，
// 本階段選 /auth，待定稿確認；不得兩處並存）。
// 全部端點皆須經 RequireAuth；回應不得含裝置識別碼雜湊。
package device

import (
	"net/http"
	"time"

	"ether.local/api/internal/transport"
)

// Handler 持有 devices 端點所需的服務。
type Handler struct {
	Service     *Service
	CurrentUser func(r *http.Request) (string, bool)
}

type deviceResponse struct {
	DeviceID     string     `json:"device_id"`
	DeviceName   string     `json:"device_name"`
	Platform     string     `json:"platform"`
	Trusted      bool       `json:"is_trusted"`
	Revoked      bool       `json:"is_revoked"`
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func toDeviceResponse(d *Device) deviceResponse {
	var last *time.Time
	if d.LastActiveAt != nil {
		u := d.LastActiveAt.UTC()
		last = &u
	}
	return deviceResponse{
		DeviceID:     d.ID,
		DeviceName:   d.DeviceName,
		Platform:     d.Platform,
		Trusted:      d.Trusted,
		Revoked:      d.Revoked,
		LastActiveAt: last,
		CreatedAt:    d.CreatedAt.UTC(),
	}
}

// ListDevices 列出目前登入者的裝置（GET /api/v1/auth/devices）。
func (h *Handler) ListDevices(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.CurrentUser(r)
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return
	}
	devices, err := h.Service.List(r.Context(), userID)
	if err != nil {
		transport.WriteServiceError(w, err, ErrDeviceNotFound)
		return
	}
	out := make([]deviceResponse, 0, len(devices))
	for _, d := range devices {
		out = append(out, toDeviceResponse(d))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type registerRequest struct {
	DeviceName           string `json:"device_name"`
	Platform             string `json:"platform"`
	DeviceIdentifierHash string `json:"device_identifier_hash"`
}

// RegisterDevice 登記裝置（POST /api/v1/auth/devices）。
// device_identifier_hash 須為客戶端雜湊後的 SHA-256 hex；
// 伺服器拒收原始識別碼。
func (h *Handler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.CurrentUser(r)
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return
	}
	var req registerRequest
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	d, err := h.Service.Register(r.Context(), userID, req.DeviceName, req.Platform, req.DeviceIdentifierHash)
	if err != nil {
		transport.WriteServiceError(w, err, ErrDeviceNotFound)
		return
	}
	transport.WriteJSON(w, http.StatusCreated, toDeviceResponse(d))
}

// RevokeDevice 撤銷裝置（POST /api/v1/auth/devices/{device_id}/revoke，
// 遠端登出單一裝置）。非擁有者一律 404。
func (h *Handler) RevokeDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.CurrentUser(r)
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return
	}
	revoked, err := h.Service.Revoke(r.Context(), userID, r.PathValue("device_id"))
	if err != nil {
		transport.WriteServiceError(w, err, ErrDeviceNotFound)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toDeviceResponse(revoked))
}
