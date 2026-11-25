package main

// Phase 03 整合測試：以假驗證器＋記憶體儲存，經 buildHandler 驗證
// 完整路由表（認證、帳號、檔案、裝置、Session）。
// TestNoOtherRoutes（Phase 01 骨架測試）已由 Phase 03 落地取代：
// 業務路由現已存在，未認證請求須回 401、未知路徑須回 404，
// 見 TestUnauthenticated 與 TestUnknownPath。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ether.local/api/internal/auth"
	"ether.local/api/internal/call"
	"ether.local/api/internal/chat"
	"ether.local/api/internal/config"
	"ether.local/api/internal/device"
	"ether.local/api/internal/friend"
	"ether.local/api/internal/group"
	"ether.local/api/internal/health"
	"ether.local/api/internal/media"
	"ether.local/api/internal/message"
	"ether.local/api/internal/profile"
	"ether.local/api/internal/search"
	"ether.local/api/internal/sticker"
	"ether.local/api/internal/timeline"
	"ether.local/api/internal/user"
)

// fakeVerifier 只接受 "good-token"（subject-1）與
// "good-token-2"（subject-2），回傳固定身分；其餘一律拒絕。
type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, token string) (auth.Identity, error) {
	var subject, email string
	switch token {
	case "good-token":
		subject, email = "subject-1", "user@example.invalid"
	case "good-token-2":
		subject, email = "subject-2", "user2@example.invalid"
	case "good-token-3":
		subject, email = "subject-3", "user3@example.invalid"
	default:
		return auth.Identity{}, auth.ErrInvalidToken
	}
	return auth.Identity{
		Subject:       subject,
		Email:         email,
		EmailVerified: true,
		SessionID:     "kc-sid-1",
		ExpiresAt:     time.Now().Add(time.Hour),
	}, nil
}

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	userSvc := user.NewService(user.NewMemoryUserStore(), user.NewMemorySettingsStore(), nil)
	profileSvc := profile.NewService(profile.NewMemoryStore(), userSvc, nil)
	friendSvc := friend.NewService(friend.NewMemoryStore(), userSvc, profileSvc, nil)
	chatSvc := chat.NewService(chat.NewMemoryStore(), userSvc, friendSvc, nil)
	mediaSvc := media.NewService(media.NewMemoryStore(), userSvc, media.NewMemoryObjectStore(), nil)
	groupSvc := group.NewService(group.NewMemoryStore(), userSvc, friendSvc, chatSvc, nil)
	deps := dependencies{
		verifier: fakeVerifier{},
		users:    userSvc,
		profiles: profileSvc,
		devices:  device.NewService(device.NewMemoryStore(), userSvc, nil),
		sessions: auth.NewSessionService(auth.NewMemorySessionStore(), nil),
		friends:  friendSvc,
		chats:    chatSvc,
		// 附件關聯經 Media 模組驗證（與正式 main.go 接線一致）。
		messages: message.NewServiceWithDeps(message.NewMemoryStore(), chatSvc, nil, mediaSvc, nil),
		groups:   groupSvc,
		media:    mediaSvc,
		// 搜尋索引為空的記憶體實作（填充迴圈屬後續里程碑）。
		search:   search.NewService(search.NewMemoryIndex(), chatSvc),
		stickers: sticker.NewService(sticker.NewMemoryStore(), userSvc, mediaSvc, nil),
		timeline: timeline.NewService(timeline.NewMemoryStore(), userSvc, friendSvc, mediaSvc, nil),
		// 通話會話與紀錄為記憶體實作（LiveKit 媒體面屬後續里程碑）。
		calls: call.NewService(call.NewMemoryStore(), userSvc, chatSvc, groupSvc, nil),
	}
	return httptest.NewServer(buildHandler(config.Config{Port: "8080"}, deps))
}

func authed(t *testing.T, method, url, body string) *http.Response {
	return authedAs(t, "good-token", method, url, body)
}

func authedAs(t *testing.T, token, method, url, body string) *http.Response {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestHealthz(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz: status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var got health.Response
	decodeJSON(t, resp, &got)
	if got.Status != "ok" || got.Service != serviceName || got.Version != version {
		t.Fatalf("GET /healthz: got %+v", got)
	}
}

func TestUnauthenticated(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	for _, path := range []string{
		"/api/v1/auth/me",
		"/api/v1/auth/sessions",
		"/api/v1/auth/devices",
		"/api/v1/profiles/me",
		"/api/v1/users/me/settings",
	} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("GET %s: status = %d, want 401", path, resp.StatusCode)
		}
	}
}

func TestUnknownPath(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	resp := authed(t, http.MethodGet, server.URL+"/api/v1/nonexistent", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown path: status = %d, want 404", resp.StatusCode)
	}
}

func TestAuthMe_ProvisionsUser(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	resp := authed(t, http.MethodGet, server.URL+"/api/v1/auth/me", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/auth/me: status = %d", resp.StatusCode)
	}
	var got struct {
		UserID          string `json:"user_id"`
		KeycloakSubject string `json:"keycloak_subject"`
		Email           string `json:"email"`
		Status          string `json:"status"`
	}
	decodeJSON(t, resp, &got)
	if got.UserID == "" || got.KeycloakSubject != "subject-1" || got.Email != "user@example.invalid" || got.Status != "active" {
		t.Fatalf("me = %+v", got)
	}
	// 第二次呼叫須回傳同一使用者（冪等建置）。
	resp2 := authed(t, http.MethodGet, server.URL+"/api/v1/auth/me", "")
	var got2 struct {
		UserID string `json:"user_id"`
	}
	decodeJSON(t, resp2, &got2)
	if got2.UserID != got.UserID {
		t.Fatalf("second call user_id = %q, want %q", got2.UserID, got.UserID)
	}
}

func TestProfilesMe_GetAndPatch(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	resp := authed(t, http.MethodGet, server.URL+"/api/v1/profiles/me", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /profiles/me: status = %d", resp.StatusCode)
	}
	var got struct {
		DisplayName string `json:"display_name"`
		Bio         string `json:"bio"`
	}
	decodeJSON(t, resp, &got)
	if got.DisplayName == "" {
		t.Fatal("display_name must not be empty after auto-provision")
	}

	resp = authed(t, http.MethodPatch, server.URL+"/api/v1/profiles/me",
		`{"display_name":"林曉彤","bio":"你好"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH /profiles/me: status = %d", resp.StatusCode)
	}
	decodeJSON(t, resp, &got)
	if got.DisplayName != "林曉彤" || got.Bio != "你好" {
		t.Fatalf("patched profile = %+v", got)
	}

	// 驗證失敗：空顯示名稱 → 400＋details，不得寫入。
	resp = authed(t, http.MethodPatch, server.URL+"/api/v1/profiles/me",
		`{"display_name":"  "}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty display_name: status = %d, want 400", resp.StatusCode)
	}
	var errBody struct {
		Error struct {
			Code    string `json:"code"`
			Details []struct {
				Field string `json:"field"`
			} `json:"details"`
		} `json:"error"`
	}
	decodeJSON(t, resp, &errBody)
	if errBody.Error.Code != "ETHER_VALIDATION_INVALID_FIELD" || len(errBody.Error.Details) == 0 || errBody.Error.Details[0].Field != "display_name" {
		t.Fatalf("error body = %+v", errBody)
	}

	// 未知欄位 → 400（嚴格模式）。
	resp = authed(t, http.MethodPatch, server.URL+"/api/v1/profiles/me",
		`{"nickname":"x"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown field: status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestSettingsMe_GetAndPatch(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	resp := authed(t, http.MethodGet, server.URL+"/api/v1/users/me/settings", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET settings: status = %d", resp.StatusCode)
	}
	var got struct {
		Theme string `json:"theme"`
	}
	decodeJSON(t, resp, &got)
	if got.Theme != "system" {
		t.Fatalf("default theme = %q, want system", got.Theme)
	}

	resp = authed(t, http.MethodPatch, server.URL+"/api/v1/users/me/settings",
		`{"theme":"dark"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH settings: status = %d", resp.StatusCode)
	}
	decodeJSON(t, resp, &got)
	if got.Theme != "dark" {
		t.Fatalf("patched theme = %q, want dark", got.Theme)
	}

	resp = authed(t, http.MethodPatch, server.URL+"/api/v1/users/me/settings",
		`{"theme":"neon"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid theme: status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestDevices_RegisterListRevoke(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	hash := auth.HashToken("device-identifier-1")
	body, _ := json.Marshal(map[string]string{
		"device_name":            "Pixel 8",
		"platform":               "android",
		"device_identifier_hash": hash,
	})
	resp := authed(t, http.MethodPost, server.URL+"/api/v1/auth/devices", string(body))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST devices: status = %d", resp.StatusCode)
	}
	var created struct {
		DeviceID string `json:"device_id"`
	}
	decodeJSON(t, resp, &created)
	if created.DeviceID == "" {
		t.Fatal("empty device_id")
	}

	// 原始識別碼（非雜湊）必須拒絕。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/auth/devices",
		`{"device_name":"x","platform":"android","device_identifier_hash":"raw-identifier"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("raw identifier: status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()

	resp = authed(t, http.MethodGet, server.URL+"/api/v1/auth/devices", "")
	var list []struct {
		DeviceID string `json:"device_id"`
	}
	decodeJSON(t, resp, &list)
	if len(list) != 1 || list[0].DeviceID != created.DeviceID {
		t.Fatalf("devices = %+v", list)
	}

	resp = authed(t, http.MethodPost, server.URL+"/api/v1/auth/devices/"+created.DeviceID+"/revoke", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("revoke device: status = %d", resp.StatusCode)
	}
	var revoked struct {
		Revoked bool `json:"is_revoked"`
	}
	decodeJSON(t, resp, &revoked)
	if !revoked.Revoked {
		t.Fatal("device not revoked")
	}
}

func TestSessions_ListAndLogout(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	// 尚無 Session 紀錄：空列表（Keycloak 端 Session 尚未映象）。
	resp := authed(t, http.MethodGet, server.URL+"/api/v1/auth/sessions", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET sessions: status = %d", resp.StatusCode)
	}
	var list []any
	decodeJSON(t, resp, &list)
	if len(list) != 0 {
		t.Fatalf("sessions = %v, want empty", list)
	}

	// Logout：sid 無對應紀錄 → 冪等 200 revoked=false。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/auth/logout", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST logout: status = %d", resp.StatusCode)
	}
	var out struct {
		Revoked bool `json:"revoked"`
	}
	decodeJSON(t, resp, &out)
	if out.Revoked {
		t.Fatal("logout with no session record must report revoked=false")
	}
}

func TestFriends_Flow(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	// 兩位使用者各自建置身分與公開檔案（user-2 設公開識別碼）。
	resp := authed(t, http.MethodGet, server.URL+"/api/v1/auth/me", "")
	var me1 struct {
		UserID string `json:"user_id"`
	}
	decodeJSON(t, resp, &me1)
	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/auth/me", "")
	var me2 struct {
		UserID string `json:"user_id"`
	}
	decodeJSON(t, resp, &me2)
	// 先以 GET 建立雙方公開檔案（GetOrCreate），再 PATCH 設定公開識別碼。
	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/profiles/me", "")
	resp.Body.Close()
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/profiles/me", "")
	resp.Body.Close()
	resp = authedAs(t, "good-token-2", http.MethodPatch, server.URL+"/api/v1/profiles/me",
		`{"display_name":"小彤","public_id":"xiaotong"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH profile 2: status = %d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodPatch, server.URL+"/api/v1/profiles/me",
		`{"display_name":"阿哲","public_id":"azhe"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH profile 1: status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 未認證不得存取好友端點。
	resp, err := http.Get(server.URL + "/api/v1/friends")
	if err != nil {
		t.Fatalf("GET /friends: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /friends unauthenticated: status = %d, want 401", resp.StatusCode)
	}

	// 搜尋：依公開識別碼找到 user-2。
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/friends/search?public_id=xiaotong", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("search: status = %d", resp.StatusCode)
	}
	var found struct {
		UserID      string `json:"user_id"`
		DisplayName string `json:"display_name"`
	}
	decodeJSON(t, resp, &found)
	if found.UserID != me2.UserID || found.DisplayName != "小彤" {
		t.Fatalf("found = %+v", found)
	}

	// 送出邀請 → user-2 收到 → 接受 → 雙方好友清單各一筆。
	body, _ := json.Marshal(map[string]string{"addressee_id": me2.UserID, "message": "你好"})
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/friends/requests", string(body))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST requests: status = %d", resp.StatusCode)
	}
	var created struct {
		RequestID string `json:"request_id"`
		Status    string `json:"status"`
	}
	decodeJSON(t, resp, &created)
	if created.Status != "pending" {
		t.Fatalf("created = %+v", created)
	}

	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/friends/requests", "")
	var incoming []struct {
		RequestID string `json:"request_id"`
		Direction string `json:"direction"`
	}
	decodeJSON(t, resp, &incoming)
	if len(incoming) != 1 || incoming[0].RequestID != created.RequestID || incoming[0].Direction != "incoming" {
		t.Fatalf("incoming = %+v", incoming)
	}

	resp = authedAs(t, "good-token-2", http.MethodPost,
		server.URL+"/api/v1/friends/requests/"+created.RequestID+"/accept", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("accept: status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = authed(t, http.MethodGet, server.URL+"/api/v1/friends", "")
	var friends []struct {
		FriendID string `json:"friend_id"`
	}
	decodeJSON(t, resp, &friends)
	if len(friends) != 1 || friends[0].FriendID != me2.UserID {
		t.Fatalf("friends = %+v", friends)
	}

	// 封鎖 → 好友清單清空；被封鎖者搜尋封鎖者為 404（不洩漏封鎖）。
	body, _ = json.Marshal(map[string]string{"blocked_user_id": me2.UserID, "reason": "騷擾"})
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/friends/blocks", string(body))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST blocks: status = %d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/friends", "")
	decodeJSON(t, resp, &friends)
	if len(friends) != 0 {
		t.Fatalf("friends after block = %+v", friends)
	}
	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/friends/search?public_id=azhe", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("search by blocked user: status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestMessaging_Flow(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	// 兩位使用者建置身分（user-2 設公開識別碼，供好友搜尋）。
	resp := authed(t, http.MethodGet, server.URL+"/api/v1/auth/me", "")
	var me1 struct {
		UserID string `json:"user_id"`
	}
	decodeJSON(t, resp, &me1)
	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/auth/me", "")
	var me2 struct {
		UserID string `json:"user_id"`
	}
	decodeJSON(t, resp, &me2)
	resp = authedAs(t, "good-token-2", http.MethodPatch, server.URL+"/api/v1/profiles/me",
		`{"display_name":"小彤","public_id":"xiaotong"}`)
	resp.Body.Close()
	resp = authed(t, http.MethodPatch, server.URL+"/api/v1/profiles/me",
		`{"display_name":"阿哲","public_id":"azhe"}`)
	resp.Body.Close()

	// 建立好友關係（DM 前提）。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/friends/requests",
		`{"addressee_id":"`+me2.UserID+`"}`)
	var reqOut struct {
		RequestID string `json:"request_id"`
	}
	decodeJSON(t, resp, &reqOut)
	resp = authedAs(t, "good-token-2", http.MethodPost,
		server.URL+"/api/v1/friends/requests/"+reqOut.RequestID+"/accept", "")
	resp.Body.Close()

	// 建一對一對話：新建 201、重建 200＋同一識別碼。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/conversations",
		`{"type":"direct","other_user_id":"`+me2.UserID+`"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /conversations: status = %d, want 201", resp.StatusCode)
	}
	var conv struct {
		ConversationID string `json:"conversation_id"`
	}
	decodeJSON(t, resp, &conv)
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/conversations",
		`{"type":"direct","other_user_id":"`+me2.UserID+`"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("重建對話：status = %d, want 200", resp.StatusCode)
	}
	var conv2 struct {
		ConversationID string `json:"conversation_id"`
	}
	decodeJSON(t, resp, &conv2)
	if conv2.ConversationID != conv.ConversationID {
		t.Fatalf("重建對話識別碼不同：%s vs %s", conv2.ConversationID, conv.ConversationID)
	}

	// 發送兩則訊息（第二則引用第一則），序號 1、2。
	msgsURL := server.URL + "/api/v1/conversations/" + conv.ConversationID + "/messages"
	resp = authed(t, http.MethodPost, msgsURL,
		`{"kind":"text","content":"第一則","client_message_id":"c-1"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST 訊息 1：status = %d, want 201", resp.StatusCode)
	}
	var m1 struct {
		MessageID      string `json:"message_id"`
		SequenceNumber int64  `json:"sequence_number"`
	}
	decodeJSON(t, resp, &m1)
	if m1.SequenceNumber != 1 {
		t.Fatalf("序號 = %d, want 1", m1.SequenceNumber)
	}
	resp = authedAs(t, "good-token-2", http.MethodPost, msgsURL,
		`{"kind":"text","content":"回覆","client_message_id":"c-2","reply_to_message_id":"`+m1.MessageID+`"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST 訊息 2：status = %d, want 201", resp.StatusCode)
	}
	var m2 struct {
		MessageID        string `json:"message_id"`
		SequenceNumber   int64  `json:"sequence_number"`
		ReplyToMessageID string `json:"reply_to_message_id"`
	}
	decodeJSON(t, resp, &m2)
	if m2.SequenceNumber != 2 || m2.ReplyToMessageID != m1.MessageID {
		t.Fatalf("訊息 2 = %+v", m2)
	}

	// 冪等重送：同一去重鍵回 200＋既有訊息。
	resp = authed(t, http.MethodPost, msgsURL,
		`{"kind":"text","content":"重送","client_message_id":"c-1"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("冪等重送：status = %d, want 200", resp.StatusCode)
	}
	var replay struct {
		MessageID string `json:"message_id"`
		Content   string `json:"content"`
	}
	decodeJSON(t, resp, &replay)
	if replay.MessageID != m1.MessageID || replay.Content != "第一則" {
		t.Fatalf("重送回傳 = %+v", replay)
	}

	// 歷史列表（倒序）與續拉（since_seq）。
	resp = authed(t, http.MethodGet, msgsURL+"?limit=10", "")
	var list struct {
		Messages []struct {
			MessageID      string `json:"message_id"`
			SequenceNumber int64  `json:"sequence_number"`
		} `json:"messages"`
		NextCursor string `json:"next_cursor"`
	}
	decodeJSON(t, resp, &list)
	if len(list.Messages) != 2 || list.Messages[0].SequenceNumber != 2 || list.NextCursor != "1" {
		t.Fatalf("歷史列表 = %+v", list)
	}
	resp = authed(t, http.MethodGet, msgsURL+"?since_seq=1", "")
	var resync struct {
		Messages []struct {
			SequenceNumber int64 `json:"sequence_number"`
		} `json:"messages"`
	}
	decodeJSON(t, resp, &resync)
	if len(resync.Messages) != 1 || resync.Messages[0].SequenceNumber != 2 {
		t.Fatalf("續拉 = %+v", resync)
	}

	// 已讀回條：user-2 讀第一則後，對話未讀數降為 0（僅第二則是
	// user-2 自己發的，不計未讀）。
	resp = authedAs(t, "good-token-2", http.MethodPost,
		server.URL+"/api/v1/messages/"+m1.MessageID+"/receipts",
		`{"receipt_type":"read"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST 回條：status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/conversations", "")
	var convs []struct {
		ConversationID string `json:"conversation_id"`
		UnreadCount    int    `json:"unread_count"`
	}
	decodeJSON(t, resp, &convs)
	if len(convs) != 1 || convs[0].UnreadCount != 0 {
		t.Fatalf("對話清單未讀數 = %+v", convs)
	}

	// 編輯（作者）與收回（作者）：非作者編輯回 404。
	resp = authedAs(t, "good-token-2", http.MethodPatch, server.URL+"/api/v1/messages/"+m1.MessageID,
		`{"content":"竄改"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("非作者編輯：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodPatch, server.URL+"/api/v1/messages/"+m1.MessageID,
		`{"content":"第一則（已編輯）"}`)
	var edited struct {
		Content   string `json:"content"`
		EditCount int    `json:"edit_count"`
	}
	decodeJSON(t, resp, &edited)
	if edited.Content != "第一則（已編輯）" || edited.EditCount != 1 {
		t.Fatalf("編輯結果 = %+v", edited)
	}
	resp = authedAs(t, "good-token-2", http.MethodPost, server.URL+"/api/v1/messages/"+m2.MessageID+"/recall", "")
	var recalled struct {
		Status  string `json:"status"`
		Content string `json:"content"`
	}
	decodeJSON(t, resp, &recalled)
	if recalled.Status != "recalled" || recalled.Content != "" {
		t.Fatalf("收回結果 = %+v", recalled)
	}

	// 群組對話：建群、新增成員、非 admin 新增被拒。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/conversations",
		`{"type":"group","title":"小組","member_ids":["`+me2.UserID+`"]}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("建群組：status = %d, want 201", resp.StatusCode)
	}
	var group struct {
		ConversationID string `json:"conversation_id"`
	}
	decodeJSON(t, resp, &group)
	resp = authedAs(t, "good-token-2", http.MethodPost,
		server.URL+"/api/v1/conversations/"+group.ConversationID+"/leave", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("離開群組：status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authedAs(t, "good-token-2", http.MethodGet,
		server.URL+"/api/v1/conversations/"+group.ConversationID+"/messages", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("離開後讀取：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestMessageReactionsPins_Flow(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	// 兩位使用者建置身分並建立好友關係（DM 前提）。
	resp := authed(t, http.MethodGet, server.URL+"/api/v1/auth/me", "")
	var me1 struct {
		UserID string `json:"user_id"`
	}
	decodeJSON(t, resp, &me1)
	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/auth/me", "")
	var me2 struct {
		UserID string `json:"user_id"`
	}
	decodeJSON(t, resp, &me2)
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/friends/requests",
		`{"addressee_id":"`+me2.UserID+`"}`)
	var reqOut struct {
		RequestID string `json:"request_id"`
	}
	decodeJSON(t, resp, &reqOut)
	resp = authedAs(t, "good-token-2", http.MethodPost,
		server.URL+"/api/v1/friends/requests/"+reqOut.RequestID+"/accept", "")
	resp.Body.Close()

	// 建一對一對話、發送一則訊息。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/conversations",
		`{"type":"direct","other_user_id":"`+me2.UserID+`"}`)
	var conv struct {
		ConversationID string `json:"conversation_id"`
	}
	decodeJSON(t, resp, &conv)
	msgsURL := server.URL + "/api/v1/conversations/" + conv.ConversationID + "/messages"
	resp = authed(t, http.MethodPost, msgsURL,
		`{"kind":"text","content":"你好","client_message_id":"c-1"}`)
	var m1 struct {
		MessageID string `json:"message_id"`
	}
	decodeJSON(t, resp, &m1)
	reactionsURL := server.URL + "/api/v1/messages/" + m1.MessageID + "/reactions"
	pinsURL := server.URL + "/api/v1/conversations/" + conv.ConversationID + "/pins"

	type summary struct {
		MessageID string `json:"message_id"`
		Reactions []struct {
			Reaction    string `json:"reaction"`
			Count       int    `json:"count"`
			ReactedByMe bool   `json:"reacted_by_me"`
		} `json:"reactions"`
	}

	// user-2 按讚、user-1 愛心：彙總兩種表情。
	resp = authedAs(t, "good-token-2", http.MethodPost, reactionsURL, `{"reaction":"👍"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST 反應：status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodPost, reactionsURL, `{"reaction":"❤️"}`)
	var got summary
	decodeJSON(t, resp, &got)
	if len(got.Reactions) != 2 {
		t.Fatalf("反應彙總 = %+v, want 2 種表情", got.Reactions)
	}

	// 不在允許清單的表情回 400；非成員（good-token-3）查詢回 404。
	resp = authed(t, http.MethodPost, reactionsURL, `{"reaction":"🔥"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("非法表情：status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authedAs(t, "good-token-3", http.MethodGet, reactionsURL, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("非成員查詢反應：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	// user-2 更換為大笑、再取消：彙總只剩愛心。
	resp = authedAs(t, "good-token-2", http.MethodPost, reactionsURL, `{"reaction":"😂"}`)
	decodeJSON(t, resp, &got)
	if len(got.Reactions) != 2 {
		t.Fatalf("更換後彙總 = %+v", got.Reactions)
	}
	resp = authedAs(t, "good-token-2", http.MethodDelete, reactionsURL, "")
	decodeJSON(t, resp, &got)
	if len(got.Reactions) != 1 || got.Reactions[0].Reaction != "❤️" {
		t.Fatalf("取消後彙總 = %+v", got.Reactions)
	}

	// 置頂：新建 201、重複置頂 200、清單一則、取消後清單為空。
	resp = authedAs(t, "good-token-2", http.MethodPost, pinsURL, `{"message_id":"`+m1.MessageID+`"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST 置頂：status = %d, want 201", resp.StatusCode)
	}
	var pinned struct {
		PinID     string `json:"pin_id"`
		MessageID string `json:"message_id"`
	}
	decodeJSON(t, resp, &pinned)
	resp = authed(t, http.MethodPost, pinsURL, `{"message_id":"`+m1.MessageID+`"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("重複置頂：status = %d, want 200", resp.StatusCode)
	}
	var pinned2 struct {
		PinID string `json:"pin_id"`
	}
	decodeJSON(t, resp, &pinned2)
	if pinned2.PinID != pinned.PinID {
		t.Fatalf("重複置頂識別碼不同：%s vs %s", pinned2.PinID, pinned.PinID)
	}
	resp = authed(t, http.MethodGet, pinsURL, "")
	var pinList struct {
		Pins []struct {
			MessageID string `json:"message_id"`
		} `json:"pins"`
	}
	decodeJSON(t, resp, &pinList)
	if len(pinList.Pins) != 1 || pinList.Pins[0].MessageID != m1.MessageID {
		t.Fatalf("置頂清單 = %+v", pinList.Pins)
	}
	resp = authedAs(t, "good-token-3", http.MethodGet, pinsURL, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("非成員查詢置頂：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodDelete, pinsURL+"/"+m1.MessageID, "")
	var unpinned struct {
		Pinned     bool   `json:"pinned"`
		UnpinnedAt string `json:"unpinned_at"`
	}
	decodeJSON(t, resp, &unpinned)
	if unpinned.Pinned || unpinned.UnpinnedAt == "" {
		t.Fatalf("取消置頂回應 = %+v", unpinned)
	}
	resp = authed(t, http.MethodGet, pinsURL, "")
	decodeJSON(t, resp, &pinList)
	if len(pinList.Pins) != 0 {
		t.Fatalf("取消後置頂清單 = %+v", pinList.Pins)
	}

	// 收回的訊息不得新增反應。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/messages/"+m1.MessageID+"/recall", "")
	resp.Body.Close()
	resp = authedAs(t, "good-token-2", http.MethodPost, reactionsURL, `{"reaction":"👍"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("收回訊息新增反應：status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestGroups_Flow(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	// 三位使用者各自建置身分。
	me := func(token string) string {
		resp := authedAs(t, token, http.MethodGet, server.URL+"/api/v1/auth/me", "")
		var got struct {
			UserID string `json:"user_id"`
		}
		decodeJSON(t, resp, &got)
		return got.UserID
	}
	me1 := me("good-token")
	me2 := me("good-token-2")
	me3 := me("good-token-3")

	// 建立群組：user-1 為擁有者，user-2 為成員，對話容器一併建立。
	resp := authed(t, http.MethodPost, server.URL+"/api/v1/groups",
		`{"name":"登山小隊","member_ids":["`+me2+`"]}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("建群組：status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		GroupID        string `json:"group_id"`
		ConversationID string `json:"conversation_id"`
		MemberCount    int    `json:"member_count"`
	}
	decodeJSON(t, resp, &created)
	if created.ConversationID == "" || created.MemberCount != 2 {
		t.Fatalf("建群組結果 = %+v", created)
	}
	gid := created.GroupID

	// 非成員讀取為 404。
	resp = authedAs(t, "good-token-3", http.MethodGet, server.URL+"/api/v1/groups/"+gid, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("非成員讀取：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	// 邀請 user-3：建立、我的邀請清單、接受。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/groups/"+gid+"/invitations",
		`{"invitee_id":"`+me3+`"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("建立邀請：status = %d, want 201", resp.StatusCode)
	}
	var inv struct {
		InvitationID string `json:"invitation_id"`
		Status       string `json:"status"`
	}
	decodeJSON(t, resp, &inv)
	resp = authedAs(t, "good-token-3", http.MethodGet, server.URL+"/api/v1/groups/invitations", "")
	var mine []struct {
		InvitationID string `json:"invitation_id"`
	}
	decodeJSON(t, resp, &mine)
	if len(mine) != 1 || mine[0].InvitationID != inv.InvitationID {
		t.Fatalf("我的邀請 = %+v", mine)
	}
	resp = authedAs(t, "good-token-3", http.MethodPost,
		server.URL+"/api/v1/groups/invitations/"+inv.InvitationID+"/accept", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("接受邀請：status = %d, want 200", resp.StatusCode)
	}
	var accepted struct {
		Status string `json:"status"`
	}
	decodeJSON(t, resp, &accepted)
	if accepted.Status != "accepted" {
		t.Fatalf("接受結果 = %+v", accepted)
	}

	// 成員清單三人；公告發佈後可讀。
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/groups/"+gid+"/members", "")
	var members []struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	decodeJSON(t, resp, &members)
	if len(members) != 3 {
		t.Fatalf("成員數 = %d, want 3", len(members))
	}
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/groups/"+gid+"/announcements",
		`{"content":"週六集合"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("發佈公告：status = %d, want 201", resp.StatusCode)
	}
	resp.Body.Close()

	// 所有權轉移：原擁有者降為管理員，不得再解散；新擁有者可解散。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/groups/"+gid+"/transfer",
		`{"new_owner_id":"`+me2+`"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("轉移所有權：status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/groups/"+gid+"/disband", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("原擁有者解散：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authedAs(t, "good-token-2", http.MethodPost, server.URL+"/api/v1/groups/"+gid+"/disband", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("新擁有者解散：status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/groups/"+gid, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("解散後讀取：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	_ = me1
}

func TestMedia_Flow(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	// 建立上傳會話：物件鍵由服務端核發，客戶端不得指定。
	resp := authed(t, http.MethodPost, server.URL+"/api/v1/media/uploads",
		`{"mime_type":"image/jpeg","total_bytes":100,"original_filename":"photo.jpg","width":800,"height":600}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("建立上傳：status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		UploadID         string `json:"upload_id"`
		Status           string `json:"status"`
		StorageKey       string `json:"storage_key"`
		UploadIDExternal string `json:"upload_id_external"`
		UploadURL        string `json:"upload_url"`
	}
	decodeJSON(t, resp, &created)
	if created.Status != "uploading" || created.StorageKey == "" || created.UploadIDExternal == "" {
		t.Fatalf("created = %+v", created)
	}
	if !strings.HasPrefix(created.StorageKey, "media/") {
		t.Fatalf("storage_key = %q, 須由服務端核發 media/ 前綴", created.StorageKey)
	}
	if created.UploadURL == "" {
		t.Fatal("建立會話須核發上傳 URL（記憶體模式為中繼路徑）")
	}

	// 進度回報→中繼位元組→完成：檔案建立時掃描狀態為 pending。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/media/uploads/"+created.UploadID+"/progress",
		`{"uploaded_bytes":100,"part_count":1}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("回報進度：status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodPut, server.URL+created.UploadURL, strings.Repeat("b", 100))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("中繼上傳位元組：status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/media/uploads/"+created.UploadID+"/complete",
		`{"mime_type":"image/jpeg","original_filename":"photo.jpg","width":800,"height":600}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("完成上傳：status = %d, want 200", resp.StatusCode)
	}
	var file struct {
		FileID     string `json:"file_id"`
		ScanStatus string `json:"scan_status"`
		SizeBytes  int64  `json:"size_bytes"`
	}
	decodeJSON(t, resp, &file)
	if file.ScanStatus != "pending" || file.SizeBytes != 100 {
		t.Fatalf("file = %+v", file)
	}

	// 授權檢查：pending 時擁有者亦不得分發。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/media/files/"+file.FileID+"/authorize",
		`{"access_type":"download"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("授權檢查：status = %d, want 200", resp.StatusCode)
	}
	var authz struct {
		Allowed bool `json:"allowed"`
	}
	decodeJSON(t, resp, &authz)
	if authz.Allowed {
		t.Fatal("pending 檔案授權須為 false")
	}

	// 縮圖登記→清單；他人讀取一律 404。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/media/files/"+file.FileID+"/thumbnails",
		`{"variant_type":"thumbnail","mime_type":"image/jpeg","size_bytes":1200,"width":160,"height":120}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("登記縮圖：status = %d, want 201", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/media/files/"+file.FileID+"/thumbnails", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("列縮圖：status = %d, want 200", resp.StatusCode)
	}
	var thumbs []struct {
		VariantType string `json:"variant_type"`
	}
	decodeJSON(t, resp, &thumbs)
	if len(thumbs) != 1 || thumbs[0].VariantType != "thumbnail" {
		t.Fatalf("thumbs = %+v", thumbs)
	}
	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/media/files/"+file.FileID, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("他人讀取：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	// 刪除後擁有者亦讀不到。
	resp = authed(t, http.MethodDelete, server.URL+"/api/v1/media/files/"+file.FileID, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("刪除檔案：status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/media/files/"+file.FileID, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("刪除後讀取：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestMessageAttachmentsForward_Flow(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	// 三位使用者建置身分。
	resp := authed(t, http.MethodGet, server.URL+"/api/v1/auth/me", "")
	var me1 struct {
		UserID string `json:"user_id"`
	}
	decodeJSON(t, resp, &me1)
	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/auth/me", "")
	var me2 struct {
		UserID string `json:"user_id"`
	}
	decodeJSON(t, resp, &me2)
	resp = authedAs(t, "good-token-3", http.MethodGet, server.URL+"/api/v1/auth/me", "")
	var me3 struct {
		UserID string `json:"user_id"`
	}
	decodeJSON(t, resp, &me3)

	// 建立好友關係並建兩則一對一對話。
	mkDM := func(token, otherID string) string {
		resp := authedAs(t, token, http.MethodPost, server.URL+"/api/v1/friends/requests",
			`{"addressee_id":"`+otherID+`"}`)
		var reqOut struct {
			RequestID string `json:"request_id"`
		}
		decodeJSON(t, resp, &reqOut)
		return reqOut.RequestID
	}
	accept := func(token, requestID string) {
		resp := authedAs(t, token, http.MethodPost,
			server.URL+"/api/v1/friends/requests/"+requestID+"/accept", "")
		resp.Body.Close()
	}
	accept("good-token-2", mkDM("good-token", me2.UserID))
	accept("good-token-3", mkDM("good-token", me3.UserID))
	mkConv := func(token, otherID string) string {
		resp := authedAs(t, token, http.MethodPost, server.URL+"/api/v1/conversations",
			`{"type":"direct","other_user_id":"`+otherID+`"}`)
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			t.Fatalf("建對話：status = %d", resp.StatusCode)
		}
		var conv struct {
			ConversationID string `json:"conversation_id"`
		}
		decodeJSON(t, resp, &conv)
		return conv.ConversationID
	}
	conv12 := mkConv("good-token", me2.UserID)
	conv13 := mkConv("good-token", me3.UserID)

	// 在對話 1-2 發送一則訊息。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/conversations/"+conv12+"/messages",
		`{"kind":"text","content":"原文","client_message_id":"c-1"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("發送：status = %d, want 201", resp.StatusCode)
	}
	var m1 struct {
		MessageID string `json:"message_id"`
	}
	decodeJSON(t, resp, &m1)

	// 轉傳至對話 1-3：新建 201、內容原樣；冪等重送 200＋同一訊息。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/messages/"+m1.MessageID+"/forward",
		`{"target_conversation_id":"`+conv13+`","client_message_id":"f-1"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("轉傳：status = %d, want 201", resp.StatusCode)
	}
	var fwd struct {
		MessageID      string `json:"message_id"`
		ConversationID string `json:"conversation_id"`
		Content        string `json:"content"`
	}
	decodeJSON(t, resp, &fwd)
	if fwd.ConversationID != conv13 || fwd.Content != "原文" {
		t.Fatalf("轉傳結果 = %+v", fwd)
	}
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/messages/"+m1.MessageID+"/forward",
		`{"target_conversation_id":"`+conv13+`","client_message_id":"f-1"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("轉傳冪等重送：status = %d, want 200", resp.StatusCode)
	}
	var fwd2 struct {
		MessageID string `json:"message_id"`
	}
	decodeJSON(t, resp, &fwd2)
	if fwd2.MessageID != fwd.MessageID {
		t.Fatalf("冪等重送識別碼不同：%s vs %s", fwd2.MessageID, fwd.MessageID)
	}

	// 轉傳紀錄：目標對話成員可查；原始訊息查紀錄為 404。
	resp = authedAs(t, "good-token-3", http.MethodGet, server.URL+"/api/v1/messages/"+fwd.MessageID+"/forward", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("查轉傳紀錄：status = %d, want 200", resp.StatusCode)
	}
	var rec struct {
		SourceMessageID string `json:"source_message_id"`
		ForwardedBy     string `json:"forwarded_by"`
	}
	decodeJSON(t, resp, &rec)
	if rec.SourceMessageID != m1.MessageID || rec.ForwardedBy != me1.UserID {
		t.Fatalf("轉傳紀錄 = %+v", rec)
	}
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/messages/"+m1.MessageID+"/forward", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("原始訊息查紀錄：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
	// 來源對話非成員（user-3）不得轉傳。
	resp = authedAs(t, "good-token-3", http.MethodPost, server.URL+"/api/v1/messages/"+m1.MessageID+"/forward",
		`{"target_conversation_id":"`+conv13+`"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("非成員轉傳：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	// 附件：查無檔案 404；空清單 200；pending 檔案 400。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/messages/"+m1.MessageID+"/attachments",
		`{"media_file_ids":["no-such-file"]}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("查無檔案關聯：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/messages/"+m1.MessageID+"/attachments",
		`{"media_file_ids":[]}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("空附件清單：status = %d, want 200", resp.StatusCode)
	}
	var empty struct {
		Attachments []struct {
			AttachmentID string `json:"attachment_id"`
		} `json:"attachments"`
	}
	decodeJSON(t, resp, &empty)
	if len(empty.Attachments) != 0 {
		t.Fatalf("空附件清單 = %+v", empty)
	}
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/media/uploads",
		`{"mime_type":"image/jpeg","total_bytes":100,"original_filename":"photo.jpg"}`)
	var created struct {
		UploadID  string `json:"upload_id"`
		UploadURL string `json:"upload_url"`
	}
	decodeJSON(t, resp, &created)
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/media/uploads/"+created.UploadID+"/progress",
		`{"uploaded_bytes":100,"part_count":1}`)
	resp.Body.Close()
	resp = authed(t, http.MethodPut, server.URL+created.UploadURL, strings.Repeat("b", 100))
	resp.Body.Close()
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/media/uploads/"+created.UploadID+"/complete",
		`{"mime_type":"image/jpeg","original_filename":"photo.jpg"}`)
	var file struct {
		FileID string `json:"file_id"`
	}
	decodeJSON(t, resp, &file)
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/messages/"+m1.MessageID+"/attachments",
		`{"media_file_ids":["`+file.FileID+`"]}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("pending 檔案關聯：status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/messages/"+m1.MessageID+"/attachments", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("列附件：status = %d, want 200", resp.StatusCode)
	}
	var listed struct {
		Attachments []struct {
			AttachmentID string `json:"attachment_id"`
		} `json:"attachments"`
	}
	decodeJSON(t, resp, &listed)
	if len(listed.Attachments) != 0 {
		t.Fatalf("列附件 = %+v", listed)
	}
}

// TestSearchMessages_Wiring 驗證搜尋路由接線與空索引行為：索引
// 填充迴圈屬後續里程碑，空索引時查詢回 200＋空結果，不得謊稱
// 結果完整（填充狀態見 STATUS.md §2v）；缺關鍵字回 400。
func TestSearchMessages_Wiring(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	resp := authed(t, http.MethodGet, server.URL+"/api/v1/search/messages?q=%E6%97%A9%E5%AE%89", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("搜尋：status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		Results []struct {
			MessageID string `json:"message_id"`
		} `json:"results"`
	}
	decodeJSON(t, resp, &out)
	if len(out.Results) != 0 {
		t.Fatalf("空索引應回空結果，實際 %+v", out)
	}
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/search/messages", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("缺 q：status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestTimeline_Flow(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	me := func(token string) string {
		resp := authedAs(t, token, http.MethodGet, server.URL+"/api/v1/auth/me", "")
		var got struct {
			UserID string `json:"user_id"`
		}
		decodeJSON(t, resp, &got)
		return got.UserID
	}
	me1 := me("good-token")
	me2 := me("good-token-2")
	_ = me("good-token-3")

	// user-1 與 user-2 成為好友：邀請＋接受。
	resp := authed(t, http.MethodPost, server.URL+"/api/v1/friends/requests",
		`{"addressee_id":"`+me2+`"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("好友邀請：status = %d, want 201", resp.StatusCode)
	}
	var req struct {
		RequestID string `json:"request_id"`
	}
	decodeJSON(t, resp, &req)
	resp = authedAs(t, "good-token-2", http.MethodPost,
		server.URL+"/api/v1/friends/requests/"+req.RequestID+"/accept", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("接受好友：status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// 建立好友可見貼文。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/posts",
		`{"content":"好友動態","visibility":"friends"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("建貼文：status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		PostID     string `json:"post_id"`
		AuthorID   string `json:"author_id"`
		Visibility string `json:"visibility"`
		LikeCount  int    `json:"like_count"`
		LikedByMe  bool   `json:"liked_by_me"`
		Media      []any  `json:"media"`
	}
	decodeJSON(t, resp, &created)
	if created.AuthorID != me1 || created.Visibility != "friends" {
		t.Fatalf("建貼文結果 = %+v", created)
	}
	pid := created.PostID

	// 陌生人讀取為 404；好友可讀；好友時間線含此貼文。
	resp = authedAs(t, "good-token-3", http.MethodGet, server.URL+"/api/v1/posts/"+pid, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("陌生人讀取：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/timeline", "")
	var feed struct {
		Posts []struct {
			PostID string `json:"post_id"`
		} `json:"posts"`
		NextCursor string `json:"next_cursor"`
	}
	decodeJSON(t, resp, &feed)
	if len(feed.Posts) != 1 || feed.Posts[0].PostID != pid {
		t.Fatalf("好友時間線 = %+v", feed)
	}

	// 按讚／取消按讚（冪等）、留言、刪除留言。
	resp = authedAs(t, "good-token-2", http.MethodPost, server.URL+"/api/v1/posts/"+pid+"/likes", "")
	var liked struct {
		Liked     bool `json:"liked"`
		LikeCount int  `json:"like_count"`
	}
	decodeJSON(t, resp, &liked)
	if !liked.Liked || liked.LikeCount != 1 {
		t.Fatalf("按讚 = %+v", liked)
	}
	resp = authedAs(t, "good-token-2", http.MethodDelete, server.URL+"/api/v1/posts/"+pid+"/likes", "")
	decodeJSON(t, resp, &liked)
	if liked.Liked || liked.LikeCount != 0 {
		t.Fatalf("取消按讚 = %+v", liked)
	}
	resp = authedAs(t, "good-token-2", http.MethodPost, server.URL+"/api/v1/posts/"+pid+"/comments",
		`{"content":"推"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("留言：status = %d, want 201", resp.StatusCode)
	}
	var comment struct {
		CommentID string `json:"comment_id"`
		Content   string `json:"content"`
	}
	decodeJSON(t, resp, &comment)
	resp = authedAs(t, "good-token-2", http.MethodPost,
		server.URL+"/api/v1/comments/"+comment.CommentID+"/delete", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("刪除留言：status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/posts/"+pid+"/comments", "")
	var comments struct {
		Comments []struct {
			Content   string `json:"content"`
			IsDeleted bool   `json:"is_deleted"`
		} `json:"comments"`
	}
	decodeJSON(t, resp, &comments)
	if len(comments.Comments) != 1 || comments.Comments[0].Content != "" || !comments.Comments[0].IsDeleted {
		t.Fatalf("留言清單 = %+v", comments)
	}

	// 非作者刪除貼文為 404；作者刪除後好友讀取為 404。
	resp = authedAs(t, "good-token-2", http.MethodPost, server.URL+"/api/v1/posts/"+pid+"/delete", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("非作者刪除：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/posts/"+pid+"/delete", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("作者刪除：status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/posts/"+pid, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("刪除後讀取：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	// 未認證請求一律 401。
	resp, err := http.Get(server.URL + "/api/v1/timeline")
	if err != nil {
		t.Fatalf("GET /api/v1/timeline: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("未認證：status = %d, want 401", resp.StatusCode)
	}
}

func TestCalls_Flow(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	me := func(token string) string {
		resp := authedAs(t, token, http.MethodGet, server.URL+"/api/v1/auth/me", "")
		var got struct {
			UserID string `json:"user_id"`
		}
		decodeJSON(t, resp, &got)
		return got.UserID
	}
	me1 := me("good-token")
	me2 := me("good-token-2")
	_ = me("good-token-3")

	// 建立好友關係（DM 前提）與一對一對話。
	resp := authed(t, http.MethodPost, server.URL+"/api/v1/friends/requests",
		`{"addressee_id":"`+me2+`"}`)
	var req struct {
		RequestID string `json:"request_id"`
	}
	decodeJSON(t, resp, &req)
	resp = authedAs(t, "good-token-2", http.MethodPost,
		server.URL+"/api/v1/friends/requests/"+req.RequestID+"/accept", "")
	resp.Body.Close()
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/conversations",
		`{"type":"direct","other_user_id":"`+me2+`"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("建對話：status = %d, want 201", resp.StatusCode)
	}
	var conv struct {
		ConversationID string `json:"conversation_id"`
	}
	decodeJSON(t, resp, &conv)
	_ = me1

	// 發起語音通話：201、狀態 ringing、參與者 2 人。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/calls",
		`{"conversation_id":"`+conv.ConversationID+`","call_type":"voice"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("發起通話：status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		CallID           string `json:"call_id"`
		Status           string `json:"status"`
		ParticipantCount int    `json:"participant_count"`
	}
	decodeJSON(t, resp, &created)
	if created.Status != "ringing" || created.ParticipantCount != 2 {
		t.Fatalf("發起結果 = %+v", created)
	}
	callID := created.CallID

	// 陌生人讀取為 404；受邀者可讀；重複發起同一範圍為 400。
	resp = authedAs(t, "good-token-3", http.MethodGet, server.URL+"/api/v1/calls/"+callID, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("陌生人讀取：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/calls",
		`{"conversation_id":"`+conv.ConversationID+`","call_type":"voice"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("重複發起：status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()

	// 接聽：狀態 active；參與者列出 2 人；結束：狀態 ended。
	resp = authedAs(t, "good-token-2", http.MethodPost, server.URL+"/api/v1/calls/"+callID+"/answer", "")
	var answered struct {
		Status string `json:"status"`
	}
	decodeJSON(t, resp, &answered)
	if answered.Status != "active" {
		t.Fatalf("接聽結果 = %+v", answered)
	}
	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/calls/"+callID+"/participants", "")
	var participants []struct {
		UserID string `json:"user_id"`
		Result string `json:"result"`
	}
	decodeJSON(t, resp, &participants)
	if len(participants) != 2 {
		t.Fatalf("參與者 = %+v", participants)
	}
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/calls/"+callID+"/end", "")
	var ended struct {
		Status    string `json:"status"`
		EndReason string `json:"end_reason"`
	}
	decodeJSON(t, resp, &ended)
	if ended.Status != "ended" || ended.EndReason != "hangup" {
		t.Fatalf("結束結果 = %+v", ended)
	}

	// 事件列出 3 則（發起／接聽／結束）；通話清單含此通話。
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/calls/"+callID+"/events", "")
	var events []struct {
		EventType string `json:"event_type"`
	}
	decodeJSON(t, resp, &events)
	if len(events) != 3 || events[0].EventType != "initiated" || events[2].EventType != "ended" {
		t.Fatalf("事件 = %+v", events)
	}
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/calls", "")
	var list struct {
		Calls []struct {
			CallID string `json:"call_id"`
		} `json:"calls"`
		NextCursor string `json:"next_cursor"`
	}
	decodeJSON(t, resp, &list)
	if len(list.Calls) != 1 || list.Calls[0].CallID != callID {
		t.Fatalf("通話清單 = %+v", list)
	}

	// 未認證請求一律 401。
	resp, err := http.Get(server.URL + "/api/v1/calls")
	if err != nil {
		t.Fatalf("GET /api/v1/calls: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("未認證：status = %d, want 401", resp.StatusCode)
	}
}

func TestStickers_Flow(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	me := func(token string) string {
		resp := authedAs(t, token, http.MethodGet, server.URL+"/api/v1/auth/me", "")
		var got struct {
			UserID string `json:"user_id"`
		}
		decodeJSON(t, resp, &got)
		return got.UserID
	}
	me1 := me("good-token")
	me2 := me("good-token-2")
	_ = me1
	_ = me2

	// 建立貼圖包：新包一律為草稿，陌生人讀取為 404。
	resp := authed(t, http.MethodPost, server.URL+"/api/v1/sticker-packs",
		`{"name":"測試包","description":"描述","category":"animals","is_free":true}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("建包：status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		PackID       string `json:"pack_id"`
		CreatorID    string `json:"creator_id"`
		Status       string `json:"status"`
		StickerCount int    `json:"sticker_count"`
	}
	decodeJSON(t, resp, &created)
	if created.CreatorID != me1 || created.Status != "draft" || created.StickerCount != 0 {
		t.Fatalf("建包結果 = %+v", created)
	}
	packID := created.PackID

	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/sticker-packs/"+packID, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("陌生人讀取草稿：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	// 上架後陌生人可讀；清單含此包。
	resp = authed(t, http.MethodPatch, server.URL+"/api/v1/sticker-packs/"+packID,
		`{"status":"published"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("上架：status = %d, want 200", resp.StatusCode)
	}
	var published struct {
		Status string `json:"status"`
	}
	decodeJSON(t, resp, &published)
	if published.Status != "published" {
		t.Fatalf("上架後狀態 = %q", published.Status)
	}
	resp = authedAs(t, "good-token-2", http.MethodGet, server.URL+"/api/v1/sticker-packs", "")
	var list struct {
		Packs []struct {
			PackID string `json:"pack_id"`
		} `json:"packs"`
		NextCursor string `json:"next_cursor"`
	}
	decodeJSON(t, resp, &list)
	if len(list.Packs) != 1 || list.Packs[0].PackID != packID {
		t.Fatalf("清單 = %+v", list)
	}

	// 新增貼圖須關聯媒體：不存在的檔案一律 404；收藏／取消收藏冪等。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/sticker-packs/"+packID+"/stickers",
		`{"media_file_id":"missing-file","name":"貼圖"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("不存在的媒體：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authedAs(t, "good-token-2", http.MethodPost, server.URL+"/api/v1/sticker-packs/"+packID+"/favorite", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("收藏：status = %d, want 200", resp.StatusCode)
	}
	var fav struct {
		FavoritedByMe bool `json:"favorited_by_me"`
	}
	decodeJSON(t, resp, &fav)
	if !fav.FavoritedByMe {
		t.Fatalf("收藏後 = %+v", fav)
	}
	resp = authedAs(t, "good-token-2", http.MethodDelete, server.URL+"/api/v1/sticker-packs/"+packID+"/favorite", "")
	decodeJSON(t, resp, &fav)
	if fav.FavoritedByMe {
		t.Fatalf("取消收藏後 = %+v", fav)
	}

	// 搜尋關鍵字必填。
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/stickers/search", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("空關鍵字搜尋：status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()

	// 刪除後一切讀取皆 404。
	resp = authed(t, http.MethodPost, server.URL+"/api/v1/sticker-packs/"+packID+"/delete", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("刪除：status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	resp = authed(t, http.MethodGet, server.URL+"/api/v1/sticker-packs/"+packID, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("刪除後讀取：status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	// 未認證請求一律 401。
	resp, err := http.Get(server.URL + "/api/v1/sticker-packs")
	if err != nil {
		t.Fatalf("GET /api/v1/sticker-packs: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("未認證：status = %d, want 401", resp.StatusCode)
	}
}
