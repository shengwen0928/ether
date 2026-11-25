// Package media 的 HTTP handler：/api/v1/media 前綴下的媒體端點
// （API_STANDARDS.md §2）。全部端點皆須經 RequireAuth；非擁有者
// 的一切查詢皆以 404 回應，不得洩漏檔案存在性（ERROR_CODES.md
// §6.3）。存取授權檢查端點只做檔案側檢查：擁有者身分由認證身分
// 判定，業務權限不得由客戶端傳入（見 handler 的 authorize 說明）。
package media

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"ether.local/api/internal/transport"
)

// Handler 持有 media 端點所需的服務。
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

// writeMediaError 將領域錯誤對應為 HTTP 回應：欄位驗證錯誤 →
// 400＋details；ErrMediaNotFound → 404（訊息不可區分）；
// ErrObjectStoreUnavailable → 503；其餘 → 500。
func writeMediaError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrMediaNotFound):
		transport.WriteError(w, http.StatusNotFound, "ETHER_COMMON_NOT_FOUND", "找不到資源。")
	case errors.Is(err, ErrObjectStoreUnavailable):
		transport.WriteError(w, http.StatusServiceUnavailable, "ETHER_COMMON_UNAVAILABLE", "物件儲存未就緒。")
	case errors.Is(err, ErrScanFinalized):
		transport.WriteError(w, http.StatusConflict, "ETHER_COMMON_CONFLICT", "掃描狀態已終結。")
	default:
		transport.WriteServiceError(w, err, nil)
	}
}

func utcTime(t time.Time) time.Time { return t.UTC() }

func utcTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

type uploadResponse struct {
	UploadID           string     `json:"upload_id"`
	MediaFileID        string     `json:"media_file_id,omitempty"`
	Status             string     `json:"status"`
	StorageKey         string     `json:"storage_key"`
	UploadIDExternal   string     `json:"upload_id_external"`
	UploadedBytes      int64      `json:"uploaded_bytes"`
	TotalBytes         int64      `json:"total_bytes"`
	PartCount          int        `json:"part_count"`
	UploadURL          string     `json:"upload_url,omitempty"`
	UploadURLExpiresAt *time.Time `json:"upload_url_expires_at,omitempty"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func toUploadResponse(u *Upload) uploadResponse {
	return uploadResponse{
		UploadID:           u.ID,
		MediaFileID:        u.MediaFileID,
		Status:             string(u.Status),
		StorageKey:         u.StorageKey,
		UploadIDExternal:   u.ExternalUploadID,
		UploadedBytes:      u.UploadedBytes,
		TotalBytes:         u.TotalBytes,
		PartCount:          u.PartCount,
		UploadURL:          u.UploadURL,
		UploadURLExpiresAt: utcTimePtr(u.UploadURLExpiresAt),
		CompletedAt:        utcTimePtr(u.CompletedAt),
		CreatedAt:          utcTime(u.CreatedAt),
		UpdatedAt:          utcTime(u.UpdatedAt),
	}
}

type initiateUploadBody struct {
	MIMEType         string `json:"mime_type"`
	TotalBytes       int64  `json:"total_bytes"`
	OriginalFilename string `json:"original_filename"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	DurationSeconds  int64  `json:"duration_seconds"`
	Checksum         string `json:"checksum"`
}

// InitiateUpload 建立上傳會話（POST /api/v1/media/uploads）。
// 客戶端不得指定物件鍵：storage_key 由服務端核發後回傳。
func (h *Handler) InitiateUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req initiateUploadBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	u, err := h.Service.InitiateUpload(r.Context(), userID, InitiateRequest{
		MIMEType:         req.MIMEType,
		TotalBytes:       req.TotalBytes,
		OriginalFilename: req.OriginalFilename,
		Width:            req.Width,
		Height:           req.Height,
		DurationSeconds:  req.DurationSeconds,
		Checksum:         req.Checksum,
	})
	if err != nil {
		writeMediaError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusCreated, toUploadResponse(u))
}

// ListUploads 列出呼叫者的上傳會話（GET /api/v1/media/uploads）。
func (h *Handler) ListUploads(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	rows, err := h.Service.ListUploads(r.Context(), userID)
	if err != nil {
		writeMediaError(w, err)
		return
	}
	out := make([]uploadResponse, 0, len(rows))
	for _, u := range rows {
		out = append(out, toUploadResponse(u))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

// GetUpload 讀取單一上傳會話（GET /api/v1/media/uploads/{upload_id}）。
func (h *Handler) GetUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	u, err := h.Service.GetUpload(r.Context(), userID, r.PathValue("upload_id"))
	if err != nil {
		writeMediaError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toUploadResponse(u))
}

type progressBody struct {
	UploadedBytes int64 `json:"uploaded_bytes"`
	PartCount     int   `json:"part_count"`
}

// ReportProgress 回報上傳進度（POST /api/v1/media/uploads/{upload_id}/progress）。
func (h *Handler) ReportProgress(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req progressBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	u, err := h.Service.ReportProgress(r.Context(), userID, r.PathValue("upload_id"), req.UploadedBytes, req.PartCount)
	if err != nil {
		writeMediaError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toUploadResponse(u))
}

type completeUploadBody struct {
	MIMEType         string `json:"mime_type"`
	OriginalFilename string `json:"original_filename"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	DurationSeconds  int64  `json:"duration_seconds"`
	Checksum         string `json:"checksum"`
}

// CompleteUpload 完成上傳（POST /api/v1/media/uploads/{upload_id}/complete）。
func (h *Handler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req completeUploadBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	f, err := h.Service.CompleteUpload(r.Context(), userID, r.PathValue("upload_id"), CompleteRequest{
		MIMEType:         req.MIMEType,
		OriginalFilename: req.OriginalFilename,
		Width:            req.Width,
		Height:           req.Height,
		DurationSeconds:  req.DurationSeconds,
		Checksum:         req.Checksum,
	})
	if err != nil {
		writeMediaError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toFileResponse(f))
}

// CancelUpload 取消上傳（POST /api/v1/media/uploads/{upload_id}/cancel）。
func (h *Handler) CancelUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	u, err := h.Service.CancelUpload(r.Context(), userID, r.PathValue("upload_id"))
	if err != nil {
		writeMediaError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toUploadResponse(u))
}

type uploadURLResponse struct {
	UploadURL string    `json:"upload_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IssueUploadURL 為進行中的上傳會話重新核發上傳 URL（POST
// /api/v1/media/uploads/{upload_id}/upload-url）。MinIO 模式回傳
// presigned PUT URL；記憶體模式回傳中繼端點相對路徑（須經認證）。
func (h *Handler) IssueUploadURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	presigned, err := h.Service.IssueUploadURL(r.Context(), userID, r.PathValue("upload_id"))
	if err != nil {
		writeMediaError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, uploadURLResponse{
		UploadURL: presigned.URL,
		ExpiresAt: presigned.ExpiresAt.UTC(),
	})
}

// RelayUpload 經 api 中繼端點接收上傳位元組（PUT
// /api/v1/media/relay/uploads/{upload_id}，僅記憶體 stub 模式）：
// 請求體即檔案位元組（非 JSON），超過上限回 413；寫入後呼叫方
// 仍須經 complete 端點完成（Stat 校驗）。MinIO 模式回 503（位元
// 組走 presigned URL 直傳，不得經 api 中繼）。
func (h *Handler) RelayUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, MaxFileSizeBytes+1))
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	if int64(len(data)) > MaxFileSizeBytes {
		transport.WriteError(w, http.StatusRequestEntityTooLarge,
			"ETHER_COMMON_PAYLOAD_TOO_LARGE", "檔案超過大小上限。")
		return
	}
	u, err := h.Service.StoreRelayBytes(r.Context(), userID, r.PathValue("upload_id"), data)
	if err != nil {
		writeMediaError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toUploadResponse(u))
}

// RelayDownload 經 api 中繼端點分發檔案位元組（GET
// /api/v1/media/relay/files/{file_id}，僅記憶體 stub 模式）：先經
// 授權檢查並寫入稽核紀錄（掃描閘門一律適用）；未通過回 403，
// 查無回 404。MinIO 模式回 503（位元組走 presigned GET URL 經
// CDN／物件儲存分發，不得經 api 中繼）。
func (h *Handler) RelayDownload(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	accessType := AccessType(r.URL.Query().Get("access_type"))
	if accessType == "" {
		accessType = AccessDownload
	}
	allowed, f, data, err := h.Service.LoadRelayBytes(r.Context(), userID, r.PathValue("file_id"), accessType, false)
	if err != nil {
		writeMediaError(w, err)
		return
	}
	if !allowed {
		transport.WriteError(w, http.StatusForbidden,
			"ETHER_AUTH_FORBIDDEN", "不允許存取此檔案。")
		return
	}
	w.Header().Set("Content-Type", f.MIMEType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Content-Disposition", "attachment")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

type fileResponse struct {
	FileID           string     `json:"file_id"`
	MIMEType         string     `json:"mime_type"`
	SizeBytes        int64      `json:"size_bytes"`
	OriginalFilename string     `json:"original_filename"`
	Width            int        `json:"width"`
	Height           int        `json:"height"`
	DurationSeconds  int64      `json:"duration_seconds"`
	Checksum         string     `json:"checksum"`
	ScanStatus       string     `json:"scan_status"`
	ScanCompletedAt  *time.Time `json:"scan_completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func toFileResponse(f *MediaFile) fileResponse {
	return fileResponse{
		FileID:           f.ID,
		MIMEType:         f.MIMEType,
		SizeBytes:        f.SizeBytes,
		OriginalFilename: f.OriginalFilename,
		Width:            f.Width,
		Height:           f.Height,
		DurationSeconds:  f.DurationSeconds,
		Checksum:         f.Checksum,
		ScanStatus:       string(f.ScanStatus),
		ScanCompletedAt:  utcTimePtr(f.ScanCompletedAt),
		CreatedAt:        utcTime(f.CreatedAt),
		UpdatedAt:        utcTime(f.UpdatedAt),
	}
}

// ListFiles 列出呼叫者的檔案（GET /api/v1/media/files）。
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	rows, err := h.Service.ListMyFiles(r.Context(), userID)
	if err != nil {
		writeMediaError(w, err)
		return
	}
	out := make([]fileResponse, 0, len(rows))
	for _, f := range rows {
		out = append(out, toFileResponse(f))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

// GetFile 讀取單一檔案詮釋資料（GET /api/v1/media/files/{file_id}）。
func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	f, err := h.Service.GetFile(r.Context(), userID, r.PathValue("file_id"))
	if err != nil {
		writeMediaError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toFileResponse(f))
}

// DeleteFile 軟刪除檔案（DELETE /api/v1/media/files/{file_id}）。
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if err := h.Service.DeleteFile(r.Context(), userID, r.PathValue("file_id")); err != nil {
		writeMediaError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type thumbnailResponse struct {
	ThumbnailID string    `json:"thumbnail_id"`
	FileID      string    `json:"file_id"`
	VariantType string    `json:"variant_type"`
	MIMEType    string    `json:"mime_type"`
	SizeBytes   int64     `json:"size_bytes"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	CreatedAt   time.Time `json:"created_at"`
}

func toThumbnailResponse(t *Thumbnail) thumbnailResponse {
	return thumbnailResponse{
		ThumbnailID: t.ID,
		FileID:      t.MediaFileID,
		VariantType: string(t.Variant),
		MIMEType:    t.MIMEType,
		SizeBytes:   t.SizeBytes,
		Width:       t.Width,
		Height:      t.Height,
		CreatedAt:   utcTime(t.CreatedAt),
	}
}

// ListThumbnails 列出檔案的縮圖（GET /api/v1/media/files/{file_id}/thumbnails）。
func (h *Handler) ListThumbnails(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	rows, err := h.Service.ListThumbnails(r.Context(), userID, r.PathValue("file_id"))
	if err != nil {
		writeMediaError(w, err)
		return
	}
	out := make([]thumbnailResponse, 0, len(rows))
	for _, t := range rows {
		out = append(out, toThumbnailResponse(t))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type registerThumbnailBody struct {
	VariantType string `json:"variant_type"`
	MIMEType    string `json:"mime_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

// RegisterThumbnail 登記縮圖詮釋資料（POST /api/v1/media/files/{file_id}/thumbnails）。
// 物件鍵由服務端核發，請求中不得指定。
func (h *Handler) RegisterThumbnail(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req registerThumbnailBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	t, err := h.Service.RegisterThumbnail(r.Context(), userID, r.PathValue("file_id"), RegisterThumbnailRequest{
		Variant:   ThumbnailVariant(req.VariantType),
		MIMEType:  req.MIMEType,
		SizeBytes: req.SizeBytes,
		Width:     req.Width,
		Height:    req.Height,
	})
	if err != nil {
		writeMediaError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusCreated, toThumbnailResponse(t))
}

type authorizeBody struct {
	AccessType string `json:"access_type"`
}

type authorizeResponse struct {
	Allowed              bool       `json:"allowed"`
	ScanStatus           string     `json:"scan_status"`
	DownloadURL          string     `json:"download_url,omitempty"`
	DownloadURLExpiresAt *time.Time `json:"download_url_expires_at,omitempty"`
}

// AuthorizeAccess 檢查呼叫者本人對檔案的存取授權並核發下載 URL
// （POST /api/v1/media/files/{file_id}/authorize）。只做檔案側
// 檢查：擁有者身分由認證身分判定，業務權限不得由請求傳入——
// 非擁有者經此端點一律拒絕（業務權限由呼叫方模組經 Go 介面裁決
// 後傳入 IssueDownloadURL 的 businessAllowed，不得經 HTTP 傳入）。
// 掃描閘門一律適用；未通過者不核發 URL。檢查結果寫入稽核紀錄
// （服務層落實）。記憶體模式回傳中繼端點相對路徑；無可分發 URL
// 的部署回傳空 URL（allowed 仍為真，呼叫方不得假裝可下載）。
func (h *Handler) AuthorizeAccess(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req authorizeBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	authz, err := h.Service.IssueDownloadURL(r.Context(), userID, r.PathValue("file_id"), AccessType(req.AccessType), false)
	if err != nil {
		writeMediaError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, authorizeResponse{
		Allowed:              authz.Allowed,
		ScanStatus:           string(authz.ScanStatus),
		DownloadURL:          authz.DownloadURL,
		DownloadURLExpiresAt: utcTimePtr(authz.URLExpiresAt),
	})
}
