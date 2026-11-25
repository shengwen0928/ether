package media

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// 測試用簽發金鑰一律為明顯假值，不得使用真實憑證。
const (
	testAccessKey = "test-only-access-key"
	testSecretKey = "test-only-secret-key"
)

var fixedSignTime = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

func newTestMinIOStore(t *testing.T, endpoint string) *MinIOStore {
	t.Helper()
	s, err := NewMinIOStore(endpoint, "ether-media", testAccessKey, testSecretKey, func() time.Time { return fixedSignTime })
	if err != nil {
		t.Fatalf("NewMinIOStore: %v", err)
	}
	return s
}

func TestNewMinIOStore_Validation(t *testing.T) {
	for _, tc := range []struct {
		endpoint, bucket, ak, sk string
	}{
		{"", "b", "ak", "sk"},
		{"minio:9000", "", "ak", "sk"},
		{"minio:9000", "b", "", "sk"},
		{"minio:9000", "b", "ak", ""},
		{"ftp://minio:9000", "b", "ak", "sk"},
		{"://bad", "b", "ak", "sk"},
	} {
		if _, err := NewMinIOStore(tc.endpoint, tc.bucket, tc.ak, tc.sk, nil); err == nil {
			t.Fatalf("endpoint=%q bucket=%q 須拒絕", tc.endpoint, tc.bucket)
		}
	}
	s, err := NewMinIOStore("minio:9000", "b", "ak", "sk", nil)
	if err != nil {
		t.Fatalf("host:port 須接受：%v", err)
	}
	if !strings.HasPrefix(s.endpoint, "http://minio:9000") {
		t.Fatalf("endpoint = %q, 須補 http scheme", s.endpoint)
	}
}

func TestMinIOPresign_DeterministicAndShaped(t *testing.T) {
	s := newTestMinIOStore(t, "http://minio:9000")
	ctx := context.Background()
	put1, err := s.PresignPut(ctx, "media/u-1/f-1", UploadURLTTL)
	if err != nil {
		t.Fatalf("PresignPut: %v", err)
	}
	put2, err := s.PresignPut(ctx, "media/u-1/f-1", UploadURLTTL)
	if err != nil {
		t.Fatalf("PresignPut 2: %v", err)
	}
	if put1.URL != put2.URL {
		t.Fatal("固定時間與金鑰下簽名 URL 須確定")
	}
	if !put1.ExpiresAt.Equal(fixedSignTime.Add(UploadURLTTL)) {
		t.Fatalf("ExpiresAt = %v", put1.ExpiresAt)
	}
	u, err := url.Parse(put1.URL)
	if err != nil {
		t.Fatalf("URL 解析：%v", err)
	}
	q := u.Query()
	if q.Get("X-Amz-Algorithm") != "AWS4-HMAC-SHA256" || q.Get("X-Amz-Expires") != "900" {
		t.Fatalf("簽名參數 = %v", q)
	}
	if !strings.Contains(q.Get("X-Amz-Credential"), testAccessKey+"/20261125/us-east-1/s3/aws4_request") {
		t.Fatalf("Credential = %q", q.Get("X-Amz-Credential"))
	}
	sig := q.Get("X-Amz-Signature")
	if len(sig) != 64 {
		t.Fatalf("簽章長度 = %d, want 64 hex", len(sig))
	}
	if !strings.HasPrefix(u.Path, "/ether-media/media/u-1/f-1") {
		t.Fatalf("path = %q", u.Path)
	}
	get, err := s.PresignGet(ctx, "media/u-1/f-1", DownloadURLTTL)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}
	if get.URL == put1.URL {
		t.Fatal("PUT 與 GET 簽名須不同（方法計入簽名）")
	}
	other, err := NewMinIOStore("http://minio:9000", "ether-media", testAccessKey, "test-only-other-secret", func() time.Time { return fixedSignTime })
	if err != nil {
		t.Fatalf("NewMinIOStore: %v", err)
	}
	putOther, err := other.PresignPut(ctx, "media/u-1/f-1", UploadURLTTL)
	if err != nil {
		t.Fatalf("PresignPut: %v", err)
	}
	if putOther.URL == put1.URL {
		t.Fatal("不同金鑰的簽章須不同")
	}
}

func TestMinIOPresign_ExpiryClampedAndKeyEscaped(t *testing.T) {
	s := newTestMinIOStore(t, "http://minio:9000")
	ctx := context.Background()
	long, err := s.PresignPut(ctx, "media/u-1/f-1", 8*24*time.Hour)
	if err != nil {
		t.Fatalf("PresignPut: %v", err)
	}
	u, _ := url.Parse(long.URL)
	if u.Query().Get("X-Amz-Expires") != "604800" {
		t.Fatalf("有效期須夾取至 7 天：%q", u.Query().Get("X-Amz-Expires"))
	}
	escaped, err := s.PresignGet(ctx, "media/u-1/my photo.jpg", time.Minute)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}
	if !strings.Contains(escaped.URL, "my%20photo.jpg") {
		t.Fatalf("鍵中的空白須以 %%20 編碼：%q", escaped.URL)
	}
	if _, err := s.PresignPut(ctx, "media/u-1/f-1", 0); err == nil {
		t.Fatal("零有效期須拒絕")
	}
}

func TestMinIOStore_StatAndAbortAgainstFakeServer(t *testing.T) {
	var gotAuth string
	var gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		switch r.URL.Path {
		case "/ether-media/media/u-1/exists.bin":
			w.Header().Set("Content-Length", "42")
			w.WriteHeader(http.StatusOK)
		case "/ether-media/media/u-1/missing.bin":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	s := newTestMinIOStore(t, server.URL)
	ctx := context.Background()
	size, exists, err := s.Stat(ctx, "media/u-1/exists.bin")
	if err != nil || !exists || size != 42 {
		t.Fatalf("Stat = %d, %v, %v, want 42 true nil", size, exists, err)
	}
	if gotMethod != http.MethodHead || !strings.HasPrefix(gotAuth, "AWS4-HMAC-SHA256 Credential=test-only-access-key/") {
		t.Fatalf("HEAD 須帶 SigV4 標頭：method=%q auth=%q", gotMethod, gotAuth)
	}
	if _, exists, err := s.Stat(ctx, "media/u-1/missing.bin"); err != nil || exists {
		t.Fatalf("缺失物件 Stat = %v, %v, want false nil", exists, err)
	}
	if _, _, err := s.Stat(ctx, "media/u-1/boom.bin"); err == nil {
		t.Fatal("伺服器錯誤時 Stat 須回錯誤（fail-closed）")
	}
	if err := s.AbortUpload(ctx, "media/u-1/missing.bin", "single-put"); err != nil {
		t.Fatalf("Abort 不存在物件須成功：%v", err)
	}
	if id, err := s.CreateUpload(ctx, "media/u-1/f-1"); err != nil || id == "" {
		t.Fatalf("CreateUpload = %q, %v", id, err)
	}
}

func TestMinIOStore_SignedRequestShape(t *testing.T) {
	s := newTestMinIOStore(t, "http://minio:9000")
	req, err := s.signedRequest(context.Background(), http.MethodHead, "media/u-1/f-1", nil)
	if err != nil {
		t.Fatalf("signedRequest: %v", err)
	}
	auth := req.Header.Get("Authorization")
	if !strings.Contains(auth, "SignedHeaders=host;x-amz-content-sha256;x-amz-date") {
		t.Fatalf("Authorization = %q", auth)
	}
	if req.Header.Get("X-Amz-Date") != "20261125T120000Z" {
		t.Fatalf("X-Amz-Date = %q", req.Header.Get("X-Amz-Date"))
	}
	_ = io.Discard
}
