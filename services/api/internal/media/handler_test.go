package media

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakePresignStore 僅提供 ObjectStore＋Presigner（刻意不實作
// ByteStore）：驗證中繼端點在 MinIO 模式回 503。
type fakePresignStore struct{}

func (s *fakePresignStore) CreateUpload(_ context.Context, _ string) (string, error) {
	return "single-put", nil
}

func (s *fakePresignStore) AbortUpload(_ context.Context, _, _ string) error { return nil }

func (s *fakePresignStore) Stat(_ context.Context, _ string) (int64, bool, error) {
	return 0, false, nil
}

func (s *fakePresignStore) PresignPut(_ context.Context, key string, _ time.Duration) (PresignedURL, error) {
	return PresignedURL{URL: "https://minio.test/bucket/" + key + "?sig=test-only", ExpiresAt: fixedNow.Add(UploadURLTTL)}, nil
}

func (s *fakePresignStore) PresignGet(_ context.Context, key string, _ time.Duration) (PresignedURL, error) {
	return PresignedURL{URL: "https://minio.test/bucket/" + key + "?sig=test-only", ExpiresAt: fixedNow.Add(DownloadURLTTL)}, nil
}

func newTestHandler(objects ObjectStore) (*Handler, *Service) {
	store := NewMemoryStore()
	svc := NewService(store, stubUsers{ids: map[string]bool{"u-owner": true, "u-other": true}}, objects, func() time.Time { return fixedNow })
	h := &Handler{Service: svc, CurrentUser: func(r *http.Request) (string, bool) { return "u-owner", true }}
	return h, svc
}

func serveHandler(t *testing.T, h *Handler) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/media/uploads", h.InitiateUpload)
	mux.HandleFunc("POST /api/v1/media/uploads/{upload_id}/upload-url", h.IssueUploadURL)
	mux.HandleFunc("POST /api/v1/media/uploads/{upload_id}/complete", h.CompleteUpload)
	mux.HandleFunc("POST /api/v1/media/files/{file_id}/authorize", h.AuthorizeAccess)
	mux.HandleFunc("PUT /api/v1/media/relay/uploads/{upload_id}", h.RelayUpload)
	mux.HandleFunc("GET /api/v1/media/relay/files/{file_id}", h.RelayDownload)
	return httptest.NewServer(mux)
}

func doReq(t *testing.T, server *httptest.Server, method, path, body string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, server.URL+path, reader)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

// TestHandler_RelayFlow：中繼上傳→完成→授權核發下載 URL→中繼
// 下載，位元組原樣往返；pending 時下載回 403。
func TestHandler_RelayFlow(t *testing.T) {
	h, svc := newTestHandler(NewMemoryObjectStore())
	server := serveHandler(t, h)
	defer server.Close()

	resp := doReq(t, server, http.MethodPost, "/api/v1/media/uploads",
		`{"mime_type":"image/jpeg","total_bytes":4,"original_filename":"a.jpg"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("建立會話：status = %d", resp.StatusCode)
	}
	var created struct {
		UploadID  string `json:"upload_id"`
		UploadURL string `json:"upload_url"`
	}
	decodeBody(t, resp, &created)
	if created.UploadURL == "" {
		t.Fatal("建立回應須含上傳 URL")
	}

	// 重新核發。
	resp = doReq(t, server, http.MethodPost, "/api/v1/media/uploads/"+created.UploadID+"/upload-url", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("重新核發：status = %d", resp.StatusCode)
	}
	var reissued struct {
		UploadURL string `json:"upload_url"`
	}
	decodeBody(t, resp, &reissued)
	if reissued.UploadURL != created.UploadURL {
		t.Fatalf("重新核發 URL = %q, want %q", reissued.UploadURL, created.UploadURL)
	}

	// 中繼上傳位元組→完成。
	resp = doReq(t, server, http.MethodPut, created.UploadURL, "ABCD")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("中繼上傳：status = %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
	resp = doReq(t, server, http.MethodPost, "/api/v1/media/uploads/"+created.UploadID+"/complete",
		`{"mime_type":"image/jpeg","original_filename":"a.jpg"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("完成：status = %d", resp.StatusCode)
	}
	var file struct {
		FileID string `json:"file_id"`
	}
	decodeBody(t, resp, &file)

	// pending：授權拒絕且無 URL；中繼下載回 403。
	resp = doReq(t, server, http.MethodPost, "/api/v1/media/files/"+file.FileID+"/authorize", `{"access_type":"download"}`)
	var authz struct {
		Allowed     bool   `json:"allowed"`
		DownloadURL string `json:"download_url"`
	}
	decodeBody(t, resp, &authz)
	if authz.Allowed || authz.DownloadURL != "" {
		t.Fatalf("pending 授權 = %+v", authz)
	}
	resp = doReq(t, server, http.MethodGet, "/api/v1/media/relay/files/"+file.FileID, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("pending 下載：status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 掃描通過後：授權核發下載 URL，中繼下載回傳原位元組。
	if _, err := svc.ReportScanResult(context.Background(), file.FileID, true); err != nil {
		t.Fatalf("ReportScanResult: %v", err)
	}
	resp = doReq(t, server, http.MethodPost, "/api/v1/media/files/"+file.FileID+"/authorize", `{"access_type":"download"}`)
	decodeBody(t, resp, &authz)
	if !authz.Allowed || authz.DownloadURL == "" {
		t.Fatalf("clean 授權 = %+v", authz)
	}
	resp = doReq(t, server, http.MethodGet, authz.DownloadURL, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("中繼下載：status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if string(body) != "ABCD" {
		t.Fatalf("下載位元組 = %q, want ABCD", body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Fatalf("Content-Type = %q", ct)
	}
}

// TestHandler_RelayGuards：不存在會話／檔案、超過上限。
func TestHandler_RelayGuards(t *testing.T) {
	h, _ := newTestHandler(NewMemoryObjectStore())
	server := serveHandler(t, h)
	defer server.Close()

	resp := doReq(t, server, http.MethodPut, "/api/v1/media/relay/uploads/no-such", "x")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("不存在會話：status = %d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()
	resp = doReq(t, server, http.MethodGet, "/api/v1/media/relay/files/no-such", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("不存在檔案：status = %d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()

	resp = doReq(t, server, http.MethodPost, "/api/v1/media/uploads",
		`{"mime_type":"image/jpeg","total_bytes":0,"original_filename":"big.bin"}`)
	var created struct {
		UploadID  string `json:"upload_id"`
		UploadURL string `json:"upload_url"`
	}
	decodeBody(t, resp, &created)
	resp = doReq(t, server, http.MethodPut, created.UploadURL, strings.Repeat("b", int(MaxFileSizeBytes)+1))
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("超過上限：status = %d, want 413", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// TestHandler_RelayUnavailableInPresignMode：Presigner 模式（MinIO）
// 的中繼端點回 503，位元組不得經 api 中繼。
func TestHandler_RelayUnavailableInPresignMode(t *testing.T) {
	objects := &fakePresignStore{}
	h, _ := newTestHandler(objects)
	server := serveHandler(t, h)
	defer server.Close()

	resp := doReq(t, server, http.MethodPost, "/api/v1/media/uploads",
		`{"mime_type":"image/jpeg","total_bytes":4,"original_filename":"a.jpg"}`)
	var created struct {
		UploadID  string `json:"upload_id"`
		UploadURL string `json:"upload_url"`
	}
	decodeBody(t, resp, &created)
	if !strings.HasPrefix(created.UploadURL, "https://minio.test/") {
		t.Fatalf("Presigner 模式須核發簽名 URL：%q", created.UploadURL)
	}
	resp = doReq(t, server, http.MethodPut, "/api/v1/media/relay/uploads/"+created.UploadID, "ABCD")
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("中繼上傳：status = %d, want 503", resp.StatusCode)
	}
	_ = resp.Body.Close()
	resp = doReq(t, server, http.MethodGet, "/api/v1/media/relay/files/no-such", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("查無檔案須先於模式檢查回 404：status = %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}
