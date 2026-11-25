// Package media 的領域模型與服務：上傳會話、檔案詮釋資料、
// 縮圖紀錄、存取授權的檔案側檢查與存取紀錄寫入。位元組通道
// 為 MinIO presigned 直傳（§2z，DATA_FLOW.md §5）：會話建立核發
// 短效 PUT URL、完成時以 Stat 校驗物件存在與大小、授權時核發
// 短效 GET URL；記憶體 stub 模式以 api 內中繼端點承載位元組
// （開發／測試用）。
//
// 責任邊界（MODULE_BOUNDARIES.md §5.10）：本模組只做檔案側檢查
// （檔案是否存在、是否刪除、掃描是否通過）；訊息／貼文／群組等
// 業務權限一律由呼叫方模組裁決，以 businessAllowed 傳入，不得
// 在本模組內判斷。物件鍵一律由本模組核發，客戶端不得指定，
// 他模組不得自拼鍵或直連 MinIO。未掃描完成前不得開放檔案分發
// （掃描閘門見 AuthorizeAccess）。
//
// 實作預設（皆待定稿覆核，見 STATUS.md §3 待決定 18）：檔案大小
// 上限 100 MiB、MIME 白名單、掃描／上傳／存取值域詞彙、整數秒
// 的 duration。掃描實作（供應商／方式）不在範圍，僅留狀態欄位
// 與 ReportScanResult 介面供後續掃描器／Worker 呼叫；縮圖產生
// 本身不在範圍，僅留詮釋資料紀錄。
package media

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	"unicode/utf8"

	"ether.local/api/internal/ids"
	"ether.local/api/internal/transport"
)

// ErrMediaNotFound 表示查無檔案／上傳會話／縮圖，或呼叫者非擁有
// 者而以「不存在」回應的情形。對外一律 404，不得區分原因，不得
// 洩漏存在性（ERROR_CODES.md §6.3）。
var ErrMediaNotFound = errors.New("media: media not found")

// ErrObjectStoreUnavailable 表示物件儲存未接線（本里程碑僅記憶體
// stub；MinIO 轉接器待落地）。對外一律 503，不得放行上傳。
var ErrObjectStoreUnavailable = errors.New("media: object store unavailable")

// ErrScanFinalized 表示檔案掃描狀態已終結（clean／rejected），
// 不得重複回報。
var ErrScanFinalized = errors.New("media: scan already finalized")

// ScanStatus 是檔案掃描狀態。值域為本里程碑實作詞彙（ERD §6.6.1
// 稱值域待定稿）：pending 為初始狀態；clean／rejected 為終結狀態。
type ScanStatus string

const (
	// ScanPending 掃描尚未完成：不得開放檔案分發。
	ScanPending ScanStatus = "pending"
	// ScanClean 掃描通過：可依業務權限分發。
	ScanClean ScanStatus = "clean"
	// ScanRejected 掃描未通過：不得開放存取。
	ScanRejected ScanStatus = "rejected"
)

// UploadStatus 是上傳會話狀態。值域為本里程碑實作詞彙（ERD
// §6.6.2 稱值域待定稿）：僅 uploading 可回報進度／完成／取消。
type UploadStatus string

const (
	// UploadUploading 上傳進行中。
	UploadUploading UploadStatus = "uploading"
	// UploadCompleted 上傳已完成，檔案詮釋資料已建立。
	UploadCompleted UploadStatus = "completed"
	// UploadFailed 上傳失敗（由後續 Worker／掃描流程寫入，本里程碑
	// 無轉入此狀態的服務方法，僅保留詞彙）。
	UploadFailed UploadStatus = "failed"
	// UploadCancelled 上傳已取消。
	UploadCancelled UploadStatus = "cancelled"
	// UploadExpired 上傳逾期（逾 ExpireStaleUploads 的截止時間）。
	UploadExpired UploadStatus = "expired"
)

// AccessType 是存取類型。值域為本里程碑實作詞彙（ERD §6.6.4 稱
// 值域待定稿）。
type AccessType string

const (
	// AccessDownload 下載原檔。
	AccessDownload AccessType = "download"
	// AccessPreview 預覽原檔。
	AccessPreview AccessType = "preview"
	// AccessThumbnail 存取縮圖／衍生變體。
	AccessThumbnail AccessType = "thumbnail"
)

// AccessResult 是存取檢查結果，寫入稽核紀錄，不得修改。
type AccessResult string

const (
	// AccessAllowed 存取被允許。
	AccessAllowed AccessResult = "allowed"
	// AccessDenied 存取被拒絕。
	AccessDenied AccessResult = "denied"
)

// UploadURLTTL 與 DownloadURLTTL 是 presigned URL 有效期實作預設
// （短效，待定稿覆核）；記憶體中繼路徑的有效性另受會話狀態約束。
const (
	// UploadURLTTL 是上傳 URL 有效期。
	UploadURLTTL = 15 * time.Minute
	// DownloadURLTTL 是下載 URL 有效期。
	DownloadURLTTL = 15 * time.Minute
)

// MaxFileSizeBytes 是單檔大小上限（本里程碑實作預設 100 MiB，待
// API／ERD 定稿覆核，不得解讀為已定合約）。
const MaxFileSizeBytes = 100 << 20

// allowedMIMETypes 是 MIME 白名單（本里程碑實作預設，待定稿覆核）。
// 白名單外的類型一律拒絕，不得以副檔名代替 MIME 判斷。
var allowedMIMETypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/webp": true, "image/gif": true,
	"video/mp4": true, "video/quicktime": true, "video/webm": true,
	"audio/mpeg": true, "audio/mp4": true, "audio/ogg": true, "audio/webm": true,
	"application/pdf": true,
}

// allowedThumbnailMIMETypes 是縮圖 MIME 白名單（縮圖皆為圖片）。
var allowedThumbnailMIMETypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/webp": true,
}

// ThumbnailVariant 是縮圖變體類型。值域為本里程碑實作詞彙（ERD
// §6.6.3 稱值域待定稿）。
type ThumbnailVariant string

const (
	// VariantThumbnail 小縮圖。
	VariantThumbnail ThumbnailVariant = "thumbnail"
	// VariantPreview 預覽圖。
	VariantPreview ThumbnailVariant = "preview"
	// VariantCompressed 壓縮圖。
	VariantCompressed ThumbnailVariant = "compressed"
)

func (v ThumbnailVariant) valid() bool {
	switch v {
	case VariantThumbnail, VariantPreview, VariantCompressed:
		return true
	}
	return false
}

// MediaFile 是 media_files 的一列。Width／Height／DurationSeconds
// 為 0 表示未知（migration 以 NOT NULL DEFAULT 0 承載，見
// 000008 檔頭）。Checksum 為客戶端提供的完整性校驗值（演算法
// 待定稿，本里程碑只驗長度，不驗格式）。IsDeleted 為軟刪除標記，
// 不得物理刪除檔案列（保留期與孤兒清理待定稿）。
type MediaFile struct {
	ID               string
	OwnerID          string
	StorageKey       string
	MIMEType         string
	SizeBytes        int64
	OriginalFilename string
	Width            int
	Height           int
	DurationSeconds  int64
	Checksum         string
	ScanStatus       ScanStatus
	ScanCompletedAt  *time.Time
	IsDeleted        bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Upload 是 media_uploads 的一列。StorageKey 與 ReservedFileID
// 由本模組在建立會話時核發並保留（ERD 草案無此二欄，見 000008
// 檔頭）；MediaFileID 在完成前為空，完成時寫入。TotalBytes 為 0
// 表示總量未知。ErrorMessage 不得存敏感資訊（本里程碑不寫入）。
// UploadURL／UploadURLExpiresAt 為暫存欄位（不持久化）：僅建立
// 會話當次核發，重新查詢須經 IssueUploadURL 重新核發。
type Upload struct {
	ID                 string
	MediaFileID        string
	UploaderID         string
	StorageKey         string
	ReservedFileID     string
	ExternalUploadID   string
	Status             UploadStatus
	UploadedBytes      int64
	TotalBytes         int64
	PartCount          int
	CompletedAt        *time.Time
	ErrorMessage       string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	UploadURL          string
	UploadURLExpiresAt *time.Time
}

// Thumbnail 是 media_thumbnails 的一列。StorageKey 由本模組核發；
// 縮圖產生本身不在範圍，本列只記錄已產生的衍生變體詮釋資料。
type Thumbnail struct {
	ID          string
	MediaFileID string
	Variant     ThumbnailVariant
	StorageKey  string
	MIMEType    string
	SizeBytes   int64
	Width       int
	Height      int
	CreatedAt   time.Time
}

// AccessLog 是 media_access_logs 的一列。Append-only，不得修改、
// 不得提前刪除。IP／User-Agent 雜湊欄位保留但本里程碑不寫入
// （雜湊方式待隱私文件定稿，不得存原始值）。
type AccessLog struct {
	ID           string
	MediaFileID  string
	AccessorID   string
	AccessType   AccessType
	AccessResult AccessResult
	CreatedAt    time.Time
}

// ValidationError 承載欄位驗證失敗清單。
type ValidationError struct {
	Fields []transport.FieldError
}

func (e *ValidationError) Error() string { return "media: validation failed" }

// FieldErrors 供 transport.WriteServiceError 寫出 details。
func (e *ValidationError) FieldErrors() []transport.FieldError { return e.Fields }

func validationError(field, issue, message string) *ValidationError {
	return &ValidationError{Fields: []transport.FieldError{{Field: field, Issue: issue, Message: message}}}
}

// UserResolver 是 Media 對 User 模組的最小依賴（依賴 DAG：
// Media --> User）：僅確認使用者存在。
type UserResolver interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// Store 是 Media 四張表的儲存介面。
type Store interface {
	GetFile(ctx context.Context, id string) (*MediaFile, error)
	ListFilesByOwner(ctx context.Context, ownerID string) ([]*MediaFile, error)
	CreateFile(ctx context.Context, f *MediaFile) error
	SaveFile(ctx context.Context, f *MediaFile) error

	GetUpload(ctx context.Context, id string) (*Upload, error)
	ListUploadsByUploader(ctx context.Context, uploaderID string) ([]*Upload, error)
	ListStaleUploads(ctx context.Context, cutoff time.Time) ([]*Upload, error)
	CreateUpload(ctx context.Context, u *Upload) error
	SaveUpload(ctx context.Context, u *Upload) error

	GetThumbnail(ctx context.Context, id string) (*Thumbnail, error)
	ListThumbnails(ctx context.Context, mediaFileID string) ([]*Thumbnail, error)
	CreateThumbnail(ctx context.Context, t *Thumbnail) error

	CreateAccessLog(ctx context.Context, l *AccessLog) error
}

// Service 是 Media 領域服務。
type Service struct {
	store   Store
	users   UserResolver
	objects ObjectStore // 可為空：未接線時上傳建立回 503，不得放行。
	now     func() time.Time
}

// NewService 建立 Service。now 為空時用 time.Now；objects 為空時
// 上傳建立一律回 ErrObjectStoreUnavailable（fail-closed），僅供測試
// 驗證未接線行為。
func NewService(store Store, users UserResolver, objects ObjectStore, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, users: users, objects: objects, now: now}
}

func validateMIMEType(mimeType string) *ValidationError {
	if mimeType == "" {
		return validationError("mime_type", "REQUIRED", "須提供 MIME 類型。")
	}
	if !allowedMIMETypes[mimeType] {
		return validationError("mime_type", "INVALID_VALUE", "不支援的檔案類型。")
	}
	return nil
}

func validateFileMeta(mimeType string, sizeBytes int64, filename string, width, height int, durationSeconds int64, checksum string) *ValidationError {
	if err := validateMIMEType(mimeType); err != nil {
		return err
	}
	if sizeBytes <= 0 {
		return validationError("size_bytes", "INVALID_VALUE", "檔案大小須大於 0。")
	}
	if sizeBytes > MaxFileSizeBytes {
		return validationError("size_bytes", "TOO_LONG", "檔案大小超過上限。")
	}
	if utf8.RuneCountInString(filename) > 255 {
		return validationError("original_filename", "TOO_LONG", "原始檔名過長。")
	}
	if width < 0 || height < 0 {
		return validationError("width", "INVALID_VALUE", "寬高不得為負數。")
	}
	if durationSeconds < 0 {
		return validationError("duration_seconds", "INVALID_VALUE", "長度不得為負數。")
	}
	if len(checksum) > 128 {
		return validationError("checksum", "TOO_LONG", "校驗值過長。")
	}
	return nil
}

// loadOwnedFile 讀取檔案列：不存在、已刪除、非擁有者皆回
// ErrMediaNotFound（不得區分原因，不得洩漏存在性）。
func (s *Service) loadOwnedFile(ctx context.Context, callerID, fileID string) (*MediaFile, error) {
	f, err := s.store.GetFile(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if f == nil || f.IsDeleted || f.OwnerID != callerID {
		return nil, ErrMediaNotFound
	}
	return f, nil
}

// issueUploadURLFor 為指定會話核發上傳 URL：Presigner 實作
// （MinIO）核發 presigned PUT URL；ByteStore 實作（記憶體 stub）
// 核發指向 api 中繼端點的相對路徑（須經認證、受會話狀態約束）；
// 其餘情形回 ErrObjectStoreUnavailable（fail-closed）。
func (s *Service) issueUploadURLFor(ctx context.Context, u *Upload) (PresignedURL, error) {
	if s.objects == nil {
		return PresignedURL{}, ErrObjectStoreUnavailable
	}
	if p, ok := s.objects.(Presigner); ok {
		return p.PresignPut(ctx, u.StorageKey, UploadURLTTL)
	}
	if _, ok := s.objects.(ByteStore); ok {
		expiresAt := s.now().UTC().Add(UploadURLTTL)
		return PresignedURL{
			URL:       "/api/v1/media/relay/uploads/" + u.ID,
			ExpiresAt: expiresAt,
		}, nil
	}
	return PresignedURL{}, ErrObjectStoreUnavailable
}

// IssueUploadURL 為進行中的上傳會話重新核發上傳 URL（會話建立
// 時核發的 URL 逾期後經此端點重新取得；僅上傳者可呼叫，僅
// uploading 可核發）。
func (s *Service) IssueUploadURL(ctx context.Context, uploaderID, uploadID string) (PresignedURL, error) {
	u, err := s.loadOwnedUpload(ctx, uploaderID, uploadID)
	if err != nil {
		return PresignedURL{}, err
	}
	if u.Status != UploadUploading {
		return PresignedURL{}, validationError("status", "INVALID_VALUE", "上傳會話已不在進行中。")
	}
	return s.issueUploadURLFor(ctx, u)
}

// InitiateRequest 是建立上傳會話的輸入。
type InitiateRequest struct {
	MIMEType         string
	TotalBytes       int64 // 0 表示總量未知。
	OriginalFilename string
	Width            int
	Height           int
	DurationSeconds  int64
	Checksum         string
}

// InitiateUpload 建立上傳會話並核發物件鍵與上傳 URL：客戶端
// 不得指定 storage_key（請求中無此欄位）；物件鍵與預留檔案識別
// 碼由本模組產生並寫入會話列，完成時據此建立檔案列。物件儲存
// 未接線或無法核發上傳 URL 時回 503，不得建立僅有詮釋資料而
// 無位元組去向的會話。
func (s *Service) InitiateUpload(ctx context.Context, uploaderID string, req InitiateRequest) (*Upload, error) {
	exists, err := s.users.Exists(ctx, uploaderID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrMediaNotFound
	}
	// 驗證時以 TotalBytes 作為宣告大小；總量未知時以 1 驗證下限，
	// 實際大小以完成時的 Stat 校驗為準。
	declaredSize := req.TotalBytes
	if declaredSize == 0 {
		declaredSize = 1
	}
	if err := validateFileMeta(req.MIMEType, declaredSize, req.OriginalFilename, req.Width, req.Height, req.DurationSeconds, req.Checksum); err != nil {
		return nil, err
	}
	if s.objects == nil {
		return nil, ErrObjectStoreUnavailable
	}
	reservedFileID := ids.New()
	key := IssueObjectKey(uploaderID, reservedFileID)
	now := s.now().UTC()
	u := &Upload{
		ID:             ids.New(),
		UploaderID:     uploaderID,
		StorageKey:     key,
		ReservedFileID: reservedFileID,
		Status:         UploadUploading,
		UploadedBytes:  0,
		TotalBytes:     req.TotalBytes,
		PartCount:      0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	// 先核發上傳 URL（失敗時不建立會話，fail-closed）；再建立
	// 物件儲存側會話（MinIO 單次 PUT 模式僅回傳標記）。
	presigned, err := s.issueUploadURLFor(ctx, u)
	if err != nil {
		return nil, err
	}
	u.UploadURL = presigned.URL
	expiresAt := presigned.ExpiresAt
	u.UploadURLExpiresAt = &expiresAt
	externalID, err := s.objects.CreateUpload(ctx, key)
	if err != nil {
		return nil, err
	}
	u.ExternalUploadID = externalID
	// 會話列只保留鍵與預留識別碼，不存檔案詮釋資料：完成時由
	// 呼叫方再次提供詮釋資料（見 CompleteUpload），大小以 Stat
	// 校驗的實際物件大小為準。建立時先驗宣告值，避免無效上傳
	// 佔用物件儲存會話。
	if err := s.store.CreateUpload(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// loadOwnedUpload 讀取上傳會話：不存在或非上傳者皆回
// ErrMediaNotFound。
func (s *Service) loadOwnedUpload(ctx context.Context, uploaderID, uploadID string) (*Upload, error) {
	u, err := s.store.GetUpload(ctx, uploadID)
	if err != nil {
		return nil, err
	}
	if u == nil || u.UploaderID != uploaderID {
		return nil, ErrMediaNotFound
	}
	return u, nil
}

// ReportProgress 回報上傳進度：僅 uploading 可回報；已上傳位元組
// 不得倒退、不得超過已知總量；分片數不得為負。
func (s *Service) ReportProgress(ctx context.Context, uploaderID, uploadID string, uploadedBytes int64, partCount int) (*Upload, error) {
	u, err := s.loadOwnedUpload(ctx, uploaderID, uploadID)
	if err != nil {
		return nil, err
	}
	if u.Status != UploadUploading {
		return nil, validationError("status", "INVALID_VALUE", "上傳會話已不在進行中。")
	}
	if uploadedBytes < u.UploadedBytes {
		return nil, validationError("uploaded_bytes", "INVALID_VALUE", "已上傳位元組不得倒退。")
	}
	if u.TotalBytes > 0 && uploadedBytes > u.TotalBytes {
		return nil, validationError("uploaded_bytes", "INVALID_VALUE", "已上傳位元組不得超過總量。")
	}
	if partCount < 0 {
		return nil, validationError("part_count", "INVALID_VALUE", "分片數不得為負數。")
	}
	u.UploadedBytes = uploadedBytes
	u.PartCount = partCount
	u.UpdatedAt = s.now().UTC()
	if err := s.store.SaveUpload(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// CompleteRequest 是完成上傳時提供的檔案詮釋資料（會話建立時
// 只核發鍵，未存詮釋資料；完成時由呼叫方提供，與建立時一致為
// 呼叫方責任——大小以實際上傳位元組為準，不得與回報進度矛盾）。
type CompleteRequest struct {
	MIMEType         string
	OriginalFilename string
	Width            int
	Height           int
	DurationSeconds  int64
	Checksum         string
}

// CompleteUpload 完成上傳：僅 uploading 可完成；總量已知時實際
// 位元組須等於總量，否則為未傳完；以 Stat 校驗物件確實存在且
// 大小與回報進度一致（無位元組、大小不符皆不得完成，fail-closed，
// 不得以僅有詮釋資料的會話建立檔案列）；建立檔案列（掃描狀態
// pending，未掃描完成前不得分發），會話標記完成並關聯檔案。
// 已完成的會話重複完成時回傳既有檔案（冪等，不得重建檔案列）。
func (s *Service) CompleteUpload(ctx context.Context, uploaderID, uploadID string, req CompleteRequest) (*MediaFile, error) {
	u, err := s.loadOwnedUpload(ctx, uploaderID, uploadID)
	if err != nil {
		return nil, err
	}
	if u.Status == UploadCompleted {
		if u.MediaFileID == "" {
			return nil, ErrMediaNotFound
		}
		f, err := s.store.GetFile(ctx, u.MediaFileID)
		if err != nil {
			return nil, err
		}
		if f == nil || f.IsDeleted {
			return nil, ErrMediaNotFound
		}
		return f, nil
	}
	if u.Status != UploadUploading {
		return nil, validationError("status", "INVALID_VALUE", "上傳會話已不在進行中。")
	}
	if u.TotalBytes > 0 && u.UploadedBytes != u.TotalBytes {
		return nil, validationError("uploaded_bytes", "INVALID_VALUE", "上傳尚未傳完，不得完成。")
	}
	if s.objects == nil {
		return nil, ErrObjectStoreUnavailable
	}
	size, exists, err := s.objects.Stat(ctx, u.StorageKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, validationError("uploaded_bytes", "INVALID_VALUE", "物件尚未上傳，不得完成。")
	}
	if size != u.UploadedBytes {
		return nil, validationError("uploaded_bytes", "INVALID_VALUE", "物件大小與回報進度不一致，不得完成。")
	}
	if err := validateFileMeta(req.MIMEType, size, req.OriginalFilename, req.Width, req.Height, req.DurationSeconds, req.Checksum); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	completedAt := now
	f := &MediaFile{
		ID:               u.ReservedFileID,
		OwnerID:          uploaderID,
		StorageKey:       u.StorageKey,
		MIMEType:         req.MIMEType,
		SizeBytes:        size,
		OriginalFilename: req.OriginalFilename,
		Width:            req.Width,
		Height:           req.Height,
		DurationSeconds:  req.DurationSeconds,
		Checksum:         req.Checksum,
		ScanStatus:       ScanPending,
		IsDeleted:        false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.store.CreateFile(ctx, f); err != nil {
		return nil, err
	}
	u.Status = UploadCompleted
	u.MediaFileID = f.ID
	u.CompletedAt = &completedAt
	u.UpdatedAt = now
	if err := s.store.SaveUpload(ctx, u); err != nil {
		return nil, err
	}
	return f, nil
}

// CancelUpload 取消上傳：僅 uploading 可取消；先中止物件儲存的
// 分段上傳（失敗時狀態不變，fail-closed），再標記取消。
func (s *Service) CancelUpload(ctx context.Context, uploaderID, uploadID string) (*Upload, error) {
	u, err := s.loadOwnedUpload(ctx, uploaderID, uploadID)
	if err != nil {
		return nil, err
	}
	if u.Status != UploadUploading {
		return nil, validationError("status", "INVALID_VALUE", "上傳會話已不在進行中。")
	}
	if s.objects != nil {
		if err := s.objects.AbortUpload(ctx, u.StorageKey, u.ExternalUploadID); err != nil {
			return nil, err
		}
	}
	u.Status = UploadCancelled
	u.UpdatedAt = s.now().UTC()
	if err := s.store.SaveUpload(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// ExpireStaleUploads 將最後更新早於 cutoff 且仍在進行中的會話
// 標記逾期（供後續排程／Worker 呼叫，本里程碑無 HTTP 端點）；
// 物件儲存中止採 best-effort（失敗不影響逾期標記，孤兒分段上傳
// 由保留期清理）。回傳逾期筆數。
func (s *Service) ExpireStaleUploads(ctx context.Context, cutoff time.Time) (int, error) {
	stale, err := s.store.ListStaleUploads(ctx, cutoff)
	if err != nil {
		return 0, err
	}
	now := s.now().UTC()
	count := 0
	for _, u := range stale {
		if s.objects != nil {
			_ = s.objects.AbortUpload(ctx, u.StorageKey, u.ExternalUploadID)
		}
		u.Status = UploadExpired
		u.UpdatedAt = now
		if err := s.store.SaveUpload(ctx, u); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// GetUpload 讀取單一上傳會話（擁有者限定）。
func (s *Service) GetUpload(ctx context.Context, uploaderID, uploadID string) (*Upload, error) {
	return s.loadOwnedUpload(ctx, uploaderID, uploadID)
}

// ListUploads 列出呼叫者的上傳會話（依建立時間倒序）。
func (s *Service) ListUploads(ctx context.Context, uploaderID string) ([]*Upload, error) {
	return s.store.ListUploadsByUploader(ctx, uploaderID)
}

// GetFile 讀取單一檔案詮釋資料（擁有者限定，非擁有者／已刪除
// 一律 404）。
func (s *Service) GetFile(ctx context.Context, callerID, fileID string) (*MediaFile, error) {
	return s.loadOwnedFile(ctx, callerID, fileID)
}

// ListMyFiles 列出呼叫者未刪除的檔案（依建立時間倒序）。
func (s *Service) ListMyFiles(ctx context.Context, ownerID string) ([]*MediaFile, error) {
	return s.store.ListFilesByOwner(ctx, ownerID)
}

// DeleteFile 軟刪除檔案（擁有者限定）：標記 is_deleted，檔案列
// 與物件保留（物理刪除與孤兒清理待定稿）。已刪除者再刪回 404。
func (s *Service) DeleteFile(ctx context.Context, ownerID, fileID string) error {
	f, err := s.loadOwnedFile(ctx, ownerID, fileID)
	if err != nil {
		return err
	}
	f.IsDeleted = true
	f.UpdatedAt = s.now().UTC()
	return s.store.SaveFile(ctx, f)
}

// RegisterThumbnailRequest 是登記縮圖的輸入（詮釋資料のみ；
// 產生本身不在範圍）。
type RegisterThumbnailRequest struct {
	Variant   ThumbnailVariant
	MIMEType  string
	SizeBytes int64
	Width     int
	Height    int
}

// RegisterThumbnail 登記縮圖詮釋資料：父檔案須存在、未刪除且為
// 呼叫者所有；同一檔案同一變體僅一筆（重複為欄位錯誤）；物件鍵
// 由本模組核發，不得由客戶端指定。
func (s *Service) RegisterThumbnail(ctx context.Context, ownerID, fileID string, req RegisterThumbnailRequest) (*Thumbnail, error) {
	f, err := s.loadOwnedFile(ctx, ownerID, fileID)
	if err != nil {
		return nil, err
	}
	if !req.Variant.valid() {
		return nil, validationError("variant_type", "INVALID_VALUE", "不支援的縮圖變體。")
	}
	if req.MIMEType == "" {
		return nil, validationError("mime_type", "REQUIRED", "須提供 MIME 類型。")
	}
	if !allowedThumbnailMIMETypes[req.MIMEType] {
		return nil, validationError("mime_type", "INVALID_VALUE", "不支援的縮圖類型。")
	}
	if req.SizeBytes <= 0 {
		return nil, validationError("size_bytes", "INVALID_VALUE", "檔案大小須大於 0。")
	}
	if req.Width < 0 || req.Height < 0 {
		return nil, validationError("width", "INVALID_VALUE", "寬高不得為負數。")
	}
	existing, err := s.store.ListThumbnails(ctx, fileID)
	if err != nil {
		return nil, err
	}
	for _, t := range existing {
		if t.Variant == req.Variant {
			return nil, validationError("variant_type", "DUPLICATE", "此變體的縮圖已存在。")
		}
	}
	t := &Thumbnail{
		ID:          ids.New(),
		MediaFileID: f.ID,
		Variant:     req.Variant,
		StorageKey:  IssueThumbnailKey(ownerID, f.ID, string(req.Variant)),
		MIMEType:    req.MIMEType,
		SizeBytes:   req.SizeBytes,
		Width:       req.Width,
		Height:      req.Height,
		CreatedAt:   s.now().UTC(),
	}
	if err := s.store.CreateThumbnail(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// ListThumbnails 列出檔案的縮圖（擁有者限定）。
func (s *Service) ListThumbnails(ctx context.Context, callerID, fileID string) ([]*Thumbnail, error) {
	if _, err := s.loadOwnedFile(ctx, callerID, fileID); err != nil {
		return nil, err
	}
	return s.store.ListThumbnails(ctx, fileID)
}

// ReportScanResult 回報掃描結果（供後續掃描器／Worker 呼叫，
// 本里程碑無 HTTP 端點，不得由客戶端直接呼叫）：僅 pending 可
// 轉為終結狀態；已終結者重複回報回錯誤，不得覆寫。
func (s *Service) ReportScanResult(ctx context.Context, fileID string, clean bool) (*MediaFile, error) {
	f, err := s.store.GetFile(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, ErrMediaNotFound
	}
	if f.ScanStatus != ScanPending {
		return nil, ErrScanFinalized
	}
	now := s.now().UTC()
	if clean {
		f.ScanStatus = ScanClean
	} else {
		f.ScanStatus = ScanRejected
	}
	f.ScanCompletedAt = &now
	f.UpdatedAt = now
	if err := s.store.SaveFile(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

// CheckAttachable 驗證檔案可被 accessor 關聯為訊息附件（供 Message
// 模組經 MediaAccess 介面呼叫，依賴 DAG：Message --> Media）：
// 檔案須存在、未刪除、為 accessor 所有、掃描已通過，四項缺一
// 不可。查無檔案／已刪除／非擁有者一律回 ErrMediaNotFound（不得
// 區分原因，不得洩漏存在性）；掃描 pending／rejected 回欄位錯誤
// （fail-closed，不得關聯未通過掃描的檔案）。本方法唯讀，不寫
// 入存取稽核紀錄（關聯檢查非檔案分發，不得以稽核列冒充授權）。
func (s *Service) CheckAttachable(ctx context.Context, accessorID, fileID string) (*MediaFile, error) {
	f, err := s.store.GetFile(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if f == nil || f.IsDeleted || f.OwnerID != accessorID {
		return nil, ErrMediaNotFound
	}
	switch f.ScanStatus {
	case ScanClean:
		return f, nil
	case ScanPending:
		return nil, validationError("media_file_id", "INVALID_VALUE", "檔案尚未完成掃描，不得關聯為附件。")
	case ScanRejected:
		return nil, validationError("media_file_id", "INVALID_VALUE", "檔案未通過掃描，不得關聯為附件。")
	default:
		return nil, validationError("media_file_id", "INVALID_VALUE", "檔案掃描狀態不明，不得關聯為附件。")
	}
}

// AuthorizeAccess 執行檔案側存取授權檢查，並寫入稽核紀錄。
// businessAllowed 由呼叫方模組裁決業務權限後傳入（本模組不得
// 判斷業務權限）；擁有者不受業務權限限制。掃描閘門一律適用：
// pending／rejected 不得分發，擁有者亦同。檔案不存在或已刪除
// 時回 ErrMediaNotFound（不寫紀錄：無檔案列可供 FK 參照）。
// 檢查通過與否皆寫入紀錄；紀錄寫入失敗時回錯誤（fail-closed，
// 稽核完整性優先）。accessType 非法為欄位錯誤，不寫紀錄。
func (s *Service) AuthorizeAccess(ctx context.Context, accessorID, fileID string, accessType AccessType, businessAllowed bool) (bool, error) {
	switch accessType {
	case AccessDownload, AccessPreview, AccessThumbnail:
	default:
		return false, validationError("access_type", "INVALID_VALUE", "不支援的存取類型。")
	}
	f, err := s.store.GetFile(ctx, fileID)
	if err != nil {
		return false, err
	}
	if f == nil || f.IsDeleted {
		return false, ErrMediaNotFound
	}
	allowed := f.ScanStatus == ScanClean && (f.OwnerID == accessorID || businessAllowed)
	result := AccessDenied
	if allowed {
		result = AccessAllowed
	}
	l := &AccessLog{
		ID:           ids.New(),
		MediaFileID:  f.ID,
		AccessorID:   accessorID,
		AccessType:   accessType,
		AccessResult: result,
		CreatedAt:    s.now().UTC(),
	}
	if err := s.store.CreateAccessLog(ctx, l); err != nil {
		return false, err
	}
	return allowed, nil
}

// DownloadAuthorization 是下載授權核發結果：Allowed 為檔案側
// 檢查結果（稽核紀錄已由 AuthorizeAccess 寫入）；允許時
// DownloadURL 為短效下載位址（MinIO presigned GET URL，或記憶體
// 模式的中繼端點相對路徑）；不允許或無可分發 URL 時 URL 為空。
// ScanStatus 僅擁有者可見真實值，非擁有者以 pending 佔位（不得
// 以本方法探測他人檔案的掃描狀態，沿 authorize 端點既有語義）。
type DownloadAuthorization struct {
	Allowed      bool
	ScanStatus   ScanStatus
	DownloadURL  string
	URLExpiresAt *time.Time
}

// IssueDownloadURL 執行檔案側授權檢查並核發下載 URL：掃描閘門
// 一律適用（pending／rejected 不得分發，擁有者亦同）；檢查通過
// 與否皆寫入稽核紀錄（AuthorizeAccess 落實）；未通過者不核發 URL。
func (s *Service) IssueDownloadURL(ctx context.Context, accessorID, fileID string, accessType AccessType, businessAllowed bool) (DownloadAuthorization, error) {
	allowed, err := s.AuthorizeAccess(ctx, accessorID, fileID, accessType, businessAllowed)
	if err != nil {
		return DownloadAuthorization{}, err
	}
	scanStatus := ScanPending
	f, ferr := s.store.GetFile(ctx, fileID)
	if ferr != nil {
		return DownloadAuthorization{}, ferr
	}
	if f != nil && f.OwnerID == accessorID {
		scanStatus = f.ScanStatus
	}
	out := DownloadAuthorization{Allowed: allowed, ScanStatus: scanStatus}
	if !allowed {
		return out, nil
	}
	if f == nil || f.IsDeleted {
		return DownloadAuthorization{}, ErrMediaNotFound
	}
	if s.objects == nil {
		return out, nil
	}
	if p, ok := s.objects.(Presigner); ok {
		presigned, err := p.PresignGet(ctx, f.StorageKey, DownloadURLTTL)
		if err != nil {
			return DownloadAuthorization{}, err
		}
		out.DownloadURL = presigned.URL
		expiresAt := presigned.ExpiresAt
		out.URLExpiresAt = &expiresAt
		return out, nil
	}
	if _, ok := s.objects.(ByteStore); ok {
		expiresAt := s.now().UTC().Add(DownloadURLTTL)
		out.DownloadURL = "/api/v1/media/relay/files/" + f.ID
		out.URLExpiresAt = &expiresAt
		return out, nil
	}
	return out, nil
}

// StoreRelayBytes 經 api 中繼端點寫入上傳位元組（僅記憶體 stub
// 模式）：會話須為進行中；位元組超過上限或與宣告總量不符時拒絕
// （fail-closed，不寫入、不推進進度）。寫入後更新已上傳位元組。
func (s *Service) StoreRelayBytes(ctx context.Context, uploaderID, uploadID string, data []byte) (*Upload, error) {
	u, err := s.loadOwnedUpload(ctx, uploaderID, uploadID)
	if err != nil {
		return nil, err
	}
	if u.Status != UploadUploading {
		return nil, validationError("status", "INVALID_VALUE", "上傳會話已不在進行中。")
	}
	store, ok := s.objects.(ByteStore)
	if !ok || s.objects == nil {
		return nil, ErrObjectStoreUnavailable
	}
	if int64(len(data)) > MaxFileSizeBytes {
		return nil, validationError("size_bytes", "TOO_LONG", "檔案大小超過上限。")
	}
	if u.TotalBytes > 0 && int64(len(data)) != u.TotalBytes {
		return nil, validationError("uploaded_bytes", "INVALID_VALUE", "位元組大小須等於宣告總量。")
	}
	if err := store.PutObject(ctx, u.StorageKey, data); err != nil {
		return nil, err
	}
	u.UploadedBytes = int64(len(data))
	u.UpdatedAt = s.now().UTC()
	if err := s.store.SaveUpload(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// LoadRelayBytes 經 api 中繼端點讀出檔案位元組（僅記憶體 stub
// 模式）：先經 AuthorizeAccess 檢查並寫入稽核紀錄；未通過時
// allowed 為 false（不洩漏位元組）；通過但物件已遺失時回 404。
func (s *Service) LoadRelayBytes(ctx context.Context, accessorID, fileID string, accessType AccessType, businessAllowed bool) (allowed bool, f *MediaFile, data []byte, err error) {
	allowed, err = s.AuthorizeAccess(ctx, accessorID, fileID, accessType, businessAllowed)
	if err != nil || !allowed {
		return allowed, nil, nil, err
	}
	f, err = s.store.GetFile(ctx, fileID)
	if err != nil {
		return false, nil, nil, err
	}
	if f == nil || f.IsDeleted {
		return false, nil, nil, ErrMediaNotFound
	}
	store, ok := s.objects.(ByteStore)
	if !ok || s.objects == nil {
		return false, nil, nil, ErrObjectStoreUnavailable
	}
	data, exists, err := store.GetObject(ctx, f.StorageKey)
	if err != nil {
		return false, nil, nil, err
	}
	if !exists {
		return false, nil, nil, ErrMediaNotFound
	}
	return true, f, data, nil
}

// MemoryStore 是 Store 的記憶體實作（開發與測試用）。
type MemoryStore struct {
	mu          sync.Mutex
	files       map[string]*MediaFile
	uploads     map[string]*Upload
	thumbnails  map[string]*Thumbnail
	thumbByFile map[string][]*Thumbnail
	logs        []*AccessLog
}

// NewMemoryStore 建立空的記憶體媒體儲存。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		files:       map[string]*MediaFile{},
		uploads:     map[string]*Upload{},
		thumbnails:  map[string]*Thumbnail{},
		thumbByFile: map[string][]*Thumbnail{},
	}
}

func copyFile(f *MediaFile) *MediaFile {
	if f == nil {
		return nil
	}
	c := *f
	if f.ScanCompletedAt != nil {
		t := *f.ScanCompletedAt
		c.ScanCompletedAt = &t
	}
	return &c
}

func copyUpload(u *Upload) *Upload {
	if u == nil {
		return nil
	}
	c := *u
	if u.CompletedAt != nil {
		t := *u.CompletedAt
		c.CompletedAt = &t
	}
	return &c
}

func (s *MemoryStore) GetFile(_ context.Context, id string) (*MediaFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return copyFile(s.files[id]), nil
}

func (s *MemoryStore) ListFilesByOwner(_ context.Context, ownerID string) ([]*MediaFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*MediaFile
	for _, f := range s.files {
		if f.OwnerID == ownerID && !f.IsDeleted {
			out = append(out, copyFile(f))
		}
	}
	sortFilesByCreatedDesc(out)
	return out, nil
}

func sortFilesByCreatedDesc(files []*MediaFile) {
	for i := 1; i < len(files); i++ {
		for j := i; j > 0 && files[j].CreatedAt.After(files[j-1].CreatedAt); j-- {
			files[j], files[j-1] = files[j-1], files[j]
		}
	}
}

func (s *MemoryStore) CreateFile(_ context.Context, f *MediaFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.files[f.ID]; exists {
		return fmt.Errorf("media: 檔案識別碼重複")
	}
	for _, other := range s.files {
		if other.StorageKey == f.StorageKey {
			return fmt.Errorf("media: 物件鍵重複")
		}
	}
	s.files[f.ID] = copyFile(f)
	return nil
}

func (s *MemoryStore) SaveFile(_ context.Context, f *MediaFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.files[f.ID]; !exists {
		return ErrMediaNotFound
	}
	s.files[f.ID] = copyFile(f)
	return nil
}

func (s *MemoryStore) GetUpload(_ context.Context, id string) (*Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return copyUpload(s.uploads[id]), nil
}

func (s *MemoryStore) ListUploadsByUploader(_ context.Context, uploaderID string) ([]*Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*Upload
	for _, u := range s.uploads {
		if u.UploaderID == uploaderID {
			out = append(out, copyUpload(u))
		}
	}
	sortUploadsByCreatedDesc(out)
	return out, nil
}

func sortUploadsByCreatedDesc(uploads []*Upload) {
	for i := 1; i < len(uploads); i++ {
		for j := i; j > 0 && uploads[j].CreatedAt.After(uploads[j-1].CreatedAt); j-- {
			uploads[j], uploads[j-1] = uploads[j-1], uploads[j]
		}
	}
}

func (s *MemoryStore) ListStaleUploads(_ context.Context, cutoff time.Time) ([]*Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*Upload
	for _, u := range s.uploads {
		if u.Status == UploadUploading && u.UpdatedAt.Before(cutoff) {
			out = append(out, copyUpload(u))
		}
	}
	return out, nil
}

func (s *MemoryStore) CreateUpload(_ context.Context, u *Upload) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.uploads[u.ID]; exists {
		return fmt.Errorf("media: 上傳會話識別碼重複")
	}
	s.uploads[u.ID] = copyUpload(u)
	return nil
}

func (s *MemoryStore) SaveUpload(_ context.Context, u *Upload) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.uploads[u.ID]; !exists {
		return ErrMediaNotFound
	}
	s.uploads[u.ID] = copyUpload(u)
	return nil
}

func (s *MemoryStore) GetThumbnail(_ context.Context, id string) (*Thumbnail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.thumbnails[id]
	if t == nil {
		return nil, nil
	}
	c := *t
	return &c, nil
}

func (s *MemoryStore) ListThumbnails(_ context.Context, mediaFileID string) ([]*Thumbnail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*Thumbnail
	for _, t := range s.thumbByFile[mediaFileID] {
		c := *t
		out = append(out, &c)
	}
	return out, nil
}

func (s *MemoryStore) CreateThumbnail(_ context.Context, t *Thumbnail) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.thumbnails[t.ID]; exists {
		return fmt.Errorf("media: 縮圖識別碼重複")
	}
	for _, other := range s.thumbnails {
		if other.StorageKey == t.StorageKey {
			return fmt.Errorf("media: 縮圖物件鍵重複")
		}
	}
	c := *t
	s.thumbnails[t.ID] = &c
	s.thumbByFile[t.MediaFileID] = append(s.thumbByFile[t.MediaFileID], &c)
	return nil
}

func (s *MemoryStore) CreateAccessLog(_ context.Context, l *AccessLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := *l
	s.logs = append(s.logs, &c)
	return nil
}

// AccessLogs 僅供測試讀取稽核紀錄（正式程式碼不得依賴）。
func (s *MemoryStore) AccessLogs() []*AccessLog {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*AccessLog, 0, len(s.logs))
	for _, l := range s.logs {
		c := *l
		out = append(out, &c)
	}
	return out
}
