// MinIOStore 是 ObjectStore 的 MinIO／S3 相容實作：以標準庫實作
// SigV4 簽名，零新依賴。位元組通道為單次 PUT presigned 直傳
// （DATA_FLOW.md §5）：CreateUpload 不建立伺服器端分段狀態，
// 僅回傳固定標記；AbortUpload 以 DELETE 清理可能已寫入的物件；
// Stat 以 HEAD 校驗存在與大小；PresignPut／PresignGet 核發短效
// 簽名 URL。未設定憑證時不得建構本實作（由 main.go 依環境變數
// 選擇，缺憑證時退回記憶體 stub 並以日誌明示）。
package media

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// minIO 實作預設（待定稿覆核）：簽名區域固定 us-east-1（MinIO
// 預設區域）；presigned URL 有效期上限 7 天（SigV4 規範上限）。
const minIOSigningRegion = "us-east-1"

const maxPresignExpiry = 7 * 24 * time.Hour

// MinIOStore 是 S3 相容物件儲存的 HTTP 客戶端。endpoint 為
// MinIO 位址（可為 host:port，預設 http；或含 scheme 的完整 URL）；
// bucket 為儲存桶名；憑證以建構參數傳入，不得寫入程式碼。
type MinIOStore struct {
	endpoint   string // 正規化後：scheme://host，不得含尾端斜線。
	bucket     string
	accessKey  string
	secretKey  string
	httpClient *http.Client
	now        func() time.Time
}

// NewMinIOStore 建立 MinIOStore：endpoint／bucket／憑證缺一不可，
// endpoint 須可解析且含主機名；now 為空時用 time.Now（測試以
// 固定時間取得確定性簽名 URL）。
func NewMinIOStore(endpoint, bucket, accessKey, secretKey string, now func() time.Time) (*MinIOStore, error) {
	endpoint = strings.TrimSpace(endpoint)
	bucket = strings.TrimSpace(bucket)
	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		return nil, errors.New("media: MinIO 設定不完整（endpoint、bucket、憑證缺一不可）")
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("media: MinIO endpoint 須為 http(s) 位址")
	}
	if now == nil {
		now = time.Now
	}
	return &MinIOStore{
		endpoint:   strings.TrimSuffix(u.String(), "/"),
		bucket:     bucket,
		accessKey:  accessKey,
		secretKey:  secretKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		now:        now,
	}, nil
}

// objectURL 以 path-style 拼出物件 URL（MinIO 預設支援；避免
// virtual-hosted style 對自簽／IP 憑證的額外要求）。
func (s *MinIOStore) objectURL(key string) string {
	segments := strings.Split(key, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return s.endpoint + "/" + s.bucket + "/" + strings.Join(segments, "/")
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write([]byte(data))
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// signingKey 依 SigV4 衍生簽名金鑰。
func (s *MinIOStore) signingKey(dateStamp string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+s.secretKey), dateStamp)
	kRegion := hmacSHA256(kDate, minIOSigningRegion)
	kService := hmacSHA256(kRegion, "s3")
	return hmacSHA256(kService, "aws4_request")
}

func (s *MinIOStore) scope(dateStamp string) string {
	return dateStamp + "/" + minIOSigningRegion + "/s3/aws4_request"
}

// presign 簽發 presigned URL：canonical request 以
// UNSIGNED-PAYLOAD 計入（與 minio-go presigned URL 語義一致，
// 客戶端 PUT／GET 時不須帶內容雜湊標頭）。
func (s *MinIOStore) presign(method, key string, expiry time.Duration) (PresignedURL, error) {
	if expiry <= 0 {
		return PresignedURL{}, errors.New("media: presign 有效期須大於 0")
	}
	if expiry > maxPresignExpiry {
		expiry = maxPresignExpiry
	}
	now := s.now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	rawURL := s.objectURL(key)
	u, err := url.Parse(rawURL)
	if err != nil {
		return PresignedURL{}, err
	}
	host := u.Host
	credential := s.accessKey + "/" + s.scope(dateStamp)
	params := map[string]string{
		"X-Amz-Algorithm":     "AWS4-HMAC-SHA256",
		"X-Amz-Credential":    credential,
		"X-Amz-Date":          amzDate,
		"X-Amz-Expires":       strconv.Itoa(int(expiry / time.Second)),
		"X-Amz-SignedHeaders": "host",
	}
	names := make([]string, 0, len(params))
	for name := range params {
		names = append(names, name)
	}
	sort.Strings(names)
	var queryParts []string
	for _, name := range names {
		queryParts = append(queryParts, url.QueryEscape(name)+"="+url.QueryEscape(params[name]))
	}
	canonicalQuery := strings.Join(queryParts, "&")
	canonicalRequest := method + "\n" + u.EscapedPath() + "\n" + canonicalQuery + "\n" +
		"host:" + host + "\n\n" + "host" + "\n" + "UNSIGNED-PAYLOAD"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + s.scope(dateStamp) + "\n" + sha256Hex([]byte(canonicalRequest))
	signature := hex.EncodeToString(hmacSHA256(s.signingKey(dateStamp), stringToSign))
	u.RawQuery = canonicalQuery + "&X-Amz-Signature=" + signature
	return PresignedURL{URL: u.String(), ExpiresAt: now.Add(expiry)}, nil
}

// PresignPut 核發單次 PUT 上傳 URL。
func (s *MinIOStore) PresignPut(_ context.Context, key string, expiry time.Duration) (PresignedURL, error) {
	return s.presign(http.MethodPut, key, expiry)
}

// PresignGet 核發單次 GET 下載 URL。
func (s *MinIOStore) PresignGet(_ context.Context, key string, expiry time.Duration) (PresignedURL, error) {
	return s.presign(http.MethodGet, key, expiry)
}

// CreateUpload 在單次 PUT 模式下不建立伺服器端狀態：回傳固定
// 標記（寫入 media_uploads.upload_id_external，僅供追蹤），
// 真正的位元組傳輸走 PresignPut 核發的 URL。
func (s *MinIOStore) CreateUpload(_ context.Context, _ string) (string, error) {
	return "single-put", nil
}

// AbortUpload 以 DELETE 清理可能已寫入的物件；物件不存在
// （404）視為已清理成功。
func (s *MinIOStore) AbortUpload(ctx context.Context, key, _ string) error {
	req, err := s.signedRequest(ctx, http.MethodDelete, key, nil)
	if err != nil {
		return err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("media: minio abort: %w", ErrObjectStoreUnavailable)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK, http.StatusNotFound:
		return nil
	default:
		return fmt.Errorf("media: minio abort: status %d: %w", resp.StatusCode, ErrObjectStoreUnavailable)
	}
}

// Stat 以 HEAD 查詢物件存在與大小：404 為不存在（非錯誤）；
// 其餘非 200 一律回物件儲存不可用（fail-closed）。
func (s *MinIOStore) Stat(ctx context.Context, key string) (int64, bool, error) {
	req, err := s.signedRequest(ctx, http.MethodHead, key, nil)
	if err != nil {
		return 0, false, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("media: minio stat: %w", ErrObjectStoreUnavailable)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		return resp.ContentLength, true, nil
	case http.StatusNotFound:
		return 0, false, nil
	default:
		return 0, false, fmt.Errorf("media: minio stat: status %d: %w", resp.StatusCode, ErrObjectStoreUnavailable)
	}
}

// signedRequest 建立帶 SigV4 Authorization 標頭的請求（header
// 簽名，payload 雜湊為空字串雜湊；DELETE／HEAD 無 body）。
func (s *MinIOStore) signedRequest(ctx context.Context, method, key string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, s.objectURL(key), body)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	payloadHash := sha256Hex(nil)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	canonicalURI := req.URL.EscapedPath()
	canonicalRequest := method + "\n" + canonicalURI + "\n" + req.URL.RawQuery + "\n" +
		"host:" + req.URL.Host + "\n" +
		"x-amz-content-sha256:" + payloadHash + "\n" +
		"x-amz-date:" + amzDate + "\n\n" +
		"host;x-amz-content-sha256;x-amz-date" + "\n" + payloadHash
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + s.scope(dateStamp) + "\n" + sha256Hex([]byte(canonicalRequest))
	signature := hex.EncodeToString(hmacSHA256(s.signingKey(dateStamp), stringToSign))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+s.accessKey+"/"+s.scope(dateStamp)+
		", SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature="+signature)
	return req, nil
}

// 確保 MinIOStore 實作 ObjectStore 與 Presigner。
var (
	_ ObjectStore = (*MinIOStore)(nil)
	_ Presigner   = (*MinIOStore)(nil)
	_ Statter     = (*MinIOStore)(nil)
)
