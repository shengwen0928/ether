//go:build integration

package media

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLiveMinIO_PresignedRoundTrip 以真實 MinIO 驗證 presigned
// 直傳往返：PUT 位元組→Stat→GET 位元組比對→Abort 清理。須以
// 環境變數提供測試用 MinIO（MINIO_TEST_ENDPOINT、
// MINIO_TEST_BUCKET、MINIO_TEST_ACCESS_KEY、
// MINIO_TEST_SECRET_KEY）；未設定時跳過並標示「環境待驗證」。
// 憑證一律取自環境，不得寫入程式碼。
func TestLiveMinIO_PresignedRoundTrip(t *testing.T) {
	endpoint := os.Getenv("MINIO_TEST_ENDPOINT")
	bucket := os.Getenv("MINIO_TEST_BUCKET")
	accessKey := os.Getenv("MINIO_TEST_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_TEST_SECRET_KEY")
	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		t.Skip("未設定 MINIO_TEST_*，MinIO 實機驗證待隔離環境")
	}
	store, err := NewMinIOStore(endpoint, bucket, accessKey, secretKey, nil)
	if err != nil {
		t.Fatalf("NewMinIOStore: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	key := "media/test-only/presign-roundtrip.bin"
	want := []byte("ether-presign-roundtrip-test-only")

	put, err := store.PresignPut(ctx, key, 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignPut: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, put.URL, strings.NewReader(string(want)))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT status = %d", resp.StatusCode)
	}
	size, exists, err := store.Stat(ctx, key)
	if err != nil || !exists || size != int64(len(want)) {
		t.Fatalf("Stat = %d, %v, %v", size, exists, err)
	}
	get, err := store.PresignGet(ctx, key, 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}
	resp, err = http.Get(get.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	got, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("GET 位元組 = %q, want %q", got, want)
	}
	if err := store.AbortUpload(ctx, key, "single-put"); err != nil {
		t.Fatalf("AbortUpload: %v", err)
	}
	if _, exists, err := store.Stat(ctx, key); err != nil || exists {
		t.Fatalf("清理後 Stat = %v, %v", exists, err)
	}
}
