// ObjectStore 是 Media 對物件儲存的唯一介面：實體檔案存於物件
// 儲存，資料表只存詮釋資料與物件鍵。本模組不得直連 MinIO，
// 他模組不得自拼物件鍵或繞過本模組存取物件。
//
// 位元組通道（§2z，MinIO presigned 直傳）：物件儲存實作除分段上傳
// 建立／中止外，須提供 Stat（完成時校驗物件存在與大小）與
// Presigner（核發短效 presigned PUT／GET URL，客戶端直傳／下載
// 不經 api 中繼）。未提供 Presigner 的實作（如記憶體 stub）由
// 服務層核發指向 api 內中繼端點的相對路徑（僅開發／測試用，
// 須經 RequireAuth 且受會話狀態約束，不得冒充 MinIO 簽名 URL）。
package media

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PresignedURL 是核發的短效簽名 URL：URL 為客戶端可直接使用的
// 位址（MinIO 模式為絕對 URL，記憶體模式為 api 中繼端點的相對
// 路徑）；ExpiresAt 為到期時間（記憶體中繼路徑的有效性另受會話
// 狀態約束，見 IssueUploadURL）。
type PresignedURL struct {
	URL       string
	ExpiresAt time.Time
}

// Presigner 核發 presigned URL。expiry 為有效期（須大於 0，
// 實作可夾取上限）；URL 不得含憑證明文以外的簽名參數。
type Presigner interface {
	// PresignPut 核發單次 PUT 上傳 URL（指定物件鍵，短效、單檔）。
	PresignPut(ctx context.Context, key string, expiry time.Duration) (PresignedURL, error)
	// PresignGet 核發單次 GET 下載 URL（指定物件鍵，短效）。
	PresignGet(ctx context.Context, key string, expiry time.Duration) (PresignedURL, error)
}

// Statter 查詢物件是否存在與大小（完成上傳時的校驗依據）。
type Statter interface {
	// Stat 回傳物件大小與是否存在；物件儲存不可用時回錯誤
	// （fail-closed，不得以「不存在」冒充錯誤）。
	Stat(ctx context.Context, key string) (size int64, exists bool, err error)
}

// ByteStore 是行程內位元組儲存（僅記憶體 stub 實作，供開發／
// 測試中繼端點使用）：位元組經 api 中繼端點寫入／讀出，不持久。
type ByteStore interface {
	// PutObject 寫入物件位元組（覆寫同鍵既有內容）；超過
	// MaxFileSizeBytes 時回欄位錯誤。
	PutObject(ctx context.Context, key string, data []byte) error
	// GetObject 讀出物件位元組；不存在時 exists 為 false。
	GetObject(ctx context.Context, key string) (data []byte, exists bool, err error)
}

// ObjectStore 是物件儲存的最小操作面：分段上傳的建立與中止、
// 物件存在與大小查詢（Stat）、presigned URL 核發（Presigner，
// 僅部分實作提供）。完成回報由呼叫方經 CompleteUpload 在詮釋
// 資料層完成；物件本體的位元組傳輸走 presigned URL 直傳，
// 不得由客戶端直接拼 URL。
type ObjectStore interface {
	Statter
	// CreateUpload 在物件儲存建立分段上傳，回傳物件儲存側的上傳
	// 識別碼（externalUploadID，寫入 media_uploads.upload_id_external）。
	// 單次 PUT 直傳模式的實作可回傳固定標記而不建立伺服器端狀態
	// （見 MinIOStore）。
	CreateUpload(ctx context.Context, key string) (externalUploadID string, err error)
	// AbortUpload 中止分段上傳（取消／逾期時呼叫）。
	AbortUpload(ctx context.Context, key, externalUploadID string) error
}

// IssueObjectKey 核發原檔物件鍵。格式由本模組定：
// media/<ownerID>/<fileID>。客戶端不得指定鍵；呼叫方傳入空
// 識別碼時回空字串（防禦性，服務層一律傳入已產生的識別碼）。
func IssueObjectKey(ownerID, fileID string) string {
	if ownerID == "" || fileID == "" {
		return ""
	}
	return "media/" + ownerID + "/" + fileID
}

// IssueThumbnailKey 核發縮圖物件鍵。格式由本模組定：
// media/<ownerID>/<fileID>/thumb/<variant>。
func IssueThumbnailKey(ownerID, fileID, variant string) string {
	if ownerID == "" || fileID == "" || variant == "" {
		return ""
	}
	return "media/" + ownerID + "/" + fileID + "/thumb/" + variant
}

// MemoryObjectStore 是 ObjectStore 的行程內 stub：記錄建立與中止
// 呼叫，核發 mem-upload-<序號> 識別碼，並以行程內 map 承載位元
// 組（供中繼端點與 Stat 校驗）；不提供 Presigner（presigned URL
// 由服務層核發中繼路徑），上傳物件不持久，僅供開發與測試驗證
// 流程；正式環境接線前須以後端日誌明示。
type MemoryObjectStore struct {
	mu      sync.Mutex
	created []string
	aborted []string
	objects map[string][]byte
	seq     int
}

// NewMemoryObjectStore 建立空的記憶體物件儲存 stub。
func NewMemoryObjectStore() *MemoryObjectStore {
	return &MemoryObjectStore{objects: map[string][]byte{}}
}

func (s *MemoryObjectStore) CreateUpload(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	s.created = append(s.created, key)
	return fmt.Sprintf("mem-upload-%d", s.seq), nil
}

func (s *MemoryObjectStore) AbortUpload(_ context.Context, key, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.aborted = append(s.aborted, key)
	delete(s.objects, key)
	return nil
}

// Stat 回傳記憶體位元組的存在與大小。
func (s *MemoryObjectStore) Stat(_ context.Context, key string) (int64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.objects[key]
	if !ok {
		return 0, false, nil
	}
	return int64(len(data)), true, nil
}

// PutObject 寫入位元組（覆寫同鍵內容）；超過上限回欄位錯誤。
func (s *MemoryObjectStore) PutObject(_ context.Context, key string, data []byte) error {
	if int64(len(data)) > MaxFileSizeBytes {
		return validationError("size_bytes", "TOO_LONG", "檔案大小超過上限。")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	s.objects[key] = cp
	return nil
}

// GetObject 讀出位元組副本；不存在時 exists 為 false。
func (s *MemoryObjectStore) GetObject(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.objects[key]
	if !ok {
		return nil, false, nil
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, true, nil
}

// CreatedKeys 僅供測試讀取已建立的物件鍵。
func (s *MemoryObjectStore) CreatedKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.created))
	copy(out, s.created)
	return out
}

// AbortedKeys 僅供測試讀取已中止的物件鍵。
func (s *MemoryObjectStore) AbortedKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.aborted))
	copy(out, s.aborted)
	return out
}

// 確保 MemoryObjectStore 實作 ByteStore（Presigner 刻意不實作）。
var (
	_ ByteStore   = (*MemoryObjectStore)(nil)
	_ Statter     = (*MemoryObjectStore)(nil)
	_ ObjectStore = (*MemoryObjectStore)(nil)
)
