// Command api 是 ETHER API 服務。
//
// Phase 03（身分帳號）起，業務 handler 經 transport.Router 註冊，
// 不得直接依賴框架。框架已決議：Echo v5（≥v5.4.7），見 ADR-016；
// 依賴尚未引入，本程式只用標準庫 net/http（見 internal/transport）。
// 依賴引入後以 adapter 替換 StdRouter，不得改動 handler。
//
// 執行期設定見 internal/config：依賴位址由 docker-compose.yml
// 組裝，缺值時明確報錯退出，不得以預設值啟動。
//
// 身分儲存現況（M2b）：依 IDENTITY_STORE 選擇後端，預設為
// PostgreSQL 儲存庫（internal/postgres＋各領域 PG 實作），啟動
// 時 Ping 失敗即明確報錯退出；IDENTITY_STORE=memory 時以記憶體
// 儲存承載，資料不持久、僅供開發與測試，不得用於正式環境。
// 啟動時以日誌明示所選後端。Migration 不在啟動時自動套用，
// 須以 cmd/migrate 明確執行（見 migrations/MIGRATIONS.md §5）。
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

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
	"ether.local/api/internal/outbox"
	"ether.local/api/internal/postgres"
	"ether.local/api/internal/profile"
	"ether.local/api/internal/relay"
	"ether.local/api/internal/search"
	"ether.local/api/internal/sticker"
	"ether.local/api/internal/timeline"
	"ether.local/api/internal/transport"
	"ether.local/api/internal/user"
)

// version 是服務版本佔位符。版號規則待定稿（ROADMAP.md §4.1，
// 語義化版本為草案方向），本值不得解讀為已發布的版本號。
const version = "0.1.0-placeholder"

// serviceName 是 /healthz 回應中的服務名，與 compose 服務名一致。
const serviceName = "api"

// dependencies 是 buildHandler 的顯式依賴：驗證器與領域服務。
// 驗證器為 nil 時認證中介層一律回 503，不得放行（fail-closed）。
type dependencies struct {
	verifier auth.TokenVerifier
	users    *user.Service
	profiles *profile.Service
	devices  *device.Service
	sessions *auth.SessionService
	friends  *friend.Service
	chats    *chat.Service
	messages *message.Service
	groups   *group.Service
	media    *media.Service
	search   *search.Service
	stickers *sticker.Service
	timeline *timeline.Service
	calls    *call.Service
}

// buildHandler 組裝 HTTP handler：/healthz（liveness）與 Phase 03
// 身分端點。全部業務端點皆經 RequireAuth，不得註冊未驗證的
// 業務路由。
func buildHandler(cfg config.Config, deps dependencies) http.Handler {
	_ = cfg
	currentUser := func(r *http.Request) (string, bool) {
		identity, ok := auth.IdentityFromContext(r.Context())
		if !ok {
			return "", false
		}
		u, err := deps.users.Provision(r.Context(), identity.Subject, identity.Email, identity.EmailVerified)
		if err != nil {
			return "", false
		}
		return u.ID, true
	}

	authHandler := &auth.Handler{
		Users:       deps.users,
		Sessions:    deps.sessions,
		CurrentUser: currentUser,
	}
	profileHandler := &profile.Handler{Service: deps.profiles, CurrentUser: currentUser}
	deviceHandler := &device.Handler{Service: deps.devices, CurrentUser: currentUser}
	userHandler := &user.Handler{Service: deps.users, CurrentUser: currentUser}
	friendHandler := &friend.Handler{Service: deps.friends, CurrentUser: currentUser}
	chatHandler := &chat.Handler{Service: deps.chats, UnreadCounter: deps.messages, CurrentUser: currentUser}
	messageHandler := &message.Handler{Service: deps.messages, CurrentUser: currentUser}
	groupHandler := &group.Handler{Service: deps.groups, CurrentUser: currentUser}
	mediaHandler := &media.Handler{Service: deps.media, CurrentUser: currentUser}
	searchHandler := &search.Handler{Service: deps.search, CurrentUser: currentUser}
	stickerHandler := &sticker.Handler{Service: deps.stickers, CurrentUser: currentUser}
	timelineHandler := &timeline.Handler{Service: deps.timeline, CurrentUser: currentUser}
	callHandler := &call.Handler{Service: deps.calls, CurrentUser: currentUser}

	router := transport.NewStdRouter()
	router.Handle(http.MethodGet, "/healthz", health.Handler(serviceName, version))

	requireAuth := func(h transport.Handler) transport.Handler {
		return auth.RequireAuth(deps.verifier, h)
	}
	router.Handle(http.MethodGet, "/api/v1/auth/me", requireAuth(authHandler.Me))
	router.Handle(http.MethodGet, "/api/v1/auth/sessions", requireAuth(authHandler.ListSessions))
	router.Handle(http.MethodPost, "/api/v1/auth/sessions/{session_id}/revoke", requireAuth(authHandler.RevokeSession))
	router.Handle(http.MethodPost, "/api/v1/auth/logout", requireAuth(authHandler.Logout))
	router.Handle(http.MethodGet, "/api/v1/auth/devices", requireAuth(deviceHandler.ListDevices))
	router.Handle(http.MethodPost, "/api/v1/auth/devices", requireAuth(deviceHandler.RegisterDevice))
	router.Handle(http.MethodPost, "/api/v1/auth/devices/{device_id}/revoke", requireAuth(deviceHandler.RevokeDevice))
	router.Handle(http.MethodGet, "/api/v1/profiles/me", requireAuth(profileHandler.GetMe))
	router.Handle(http.MethodPatch, "/api/v1/profiles/me", requireAuth(profileHandler.UpdateMe))
	router.Handle(http.MethodGet, "/api/v1/users/me/settings", requireAuth(userHandler.GetSettings))
	router.Handle(http.MethodPatch, "/api/v1/users/me/settings", requireAuth(userHandler.UpdateSettings))
	router.Handle(http.MethodGet, "/api/v1/friends", requireAuth(friendHandler.ListFriends))
	router.Handle(http.MethodGet, "/api/v1/friends/search", requireAuth(friendHandler.SearchByPublicID))
	router.Handle(http.MethodGet, "/api/v1/friends/requests", requireAuth(friendHandler.ListRequests))
	router.Handle(http.MethodPost, "/api/v1/friends/requests", requireAuth(friendHandler.SendRequest))
	router.Handle(http.MethodPost, "/api/v1/friends/requests/{request_id}/accept", requireAuth(friendHandler.AcceptRequest))
	router.Handle(http.MethodPost, "/api/v1/friends/requests/{request_id}/decline", requireAuth(friendHandler.DeclineRequest))
	router.Handle(http.MethodPost, "/api/v1/friends/requests/{request_id}/cancel", requireAuth(friendHandler.CancelRequest))
	router.Handle(http.MethodPost, "/api/v1/friends/{friend_id}/remove", requireAuth(friendHandler.RemoveFriend))
	router.Handle(http.MethodPost, "/api/v1/friends/{friend_id}/hide", requireAuth(friendHandler.HideFriend))
	router.Handle(http.MethodPost, "/api/v1/friends/{friend_id}/unhide", requireAuth(friendHandler.UnhideFriend))
	router.Handle(http.MethodGet, "/api/v1/friends/blocks", requireAuth(friendHandler.ListBlocks))
	router.Handle(http.MethodPost, "/api/v1/friends/blocks", requireAuth(friendHandler.Block))
	router.Handle(http.MethodPost, "/api/v1/friends/blocks/{blocked_user_id}/unblock", requireAuth(friendHandler.Unblock))
	router.Handle(http.MethodGet, "/api/v1/conversations", requireAuth(chatHandler.ListConversations))
	router.Handle(http.MethodPost, "/api/v1/conversations", requireAuth(chatHandler.CreateConversation))
	router.Handle(http.MethodGet, "/api/v1/conversations/{conversation_id}", requireAuth(chatHandler.GetConversation))
	router.Handle(http.MethodGet, "/api/v1/conversations/{conversation_id}/members", requireAuth(chatHandler.ListMembers))
	router.Handle(http.MethodPost, "/api/v1/conversations/{conversation_id}/members", requireAuth(chatHandler.AddMembers))
	router.Handle(http.MethodPost, "/api/v1/conversations/{conversation_id}/leave", requireAuth(chatHandler.Leave))
	router.Handle(http.MethodGet, "/api/v1/conversations/{conversation_id}/messages", requireAuth(messageHandler.ListMessages))
	router.Handle(http.MethodPost, "/api/v1/conversations/{conversation_id}/messages", requireAuth(messageHandler.SendMessage))
	router.Handle(http.MethodPatch, "/api/v1/messages/{message_id}", requireAuth(messageHandler.EditMessage))
	router.Handle(http.MethodPost, "/api/v1/messages/{message_id}/recall", requireAuth(messageHandler.RecallMessage))
	router.Handle(http.MethodPost, "/api/v1/messages/{message_id}/delete", requireAuth(messageHandler.DeleteMessage))
	router.Handle(http.MethodGet, "/api/v1/messages/{message_id}/receipts", requireAuth(messageHandler.ListReceipts))
	router.Handle(http.MethodPost, "/api/v1/messages/{message_id}/receipts", requireAuth(messageHandler.PostReceipt))
	router.Handle(http.MethodGet, "/api/v1/messages/{message_id}/reactions", requireAuth(messageHandler.ListReactions))
	router.Handle(http.MethodPost, "/api/v1/messages/{message_id}/reactions", requireAuth(messageHandler.SetReaction))
	router.Handle(http.MethodDelete, "/api/v1/messages/{message_id}/reactions", requireAuth(messageHandler.RemoveReaction))
	router.Handle(http.MethodGet, "/api/v1/messages/{message_id}/attachments", requireAuth(messageHandler.ListAttachments))
	router.Handle(http.MethodPost, "/api/v1/messages/{message_id}/attachments", requireAuth(messageHandler.AddAttachments))
	router.Handle(http.MethodGet, "/api/v1/messages/{message_id}/forward", requireAuth(messageHandler.GetForward))
	router.Handle(http.MethodPost, "/api/v1/messages/{message_id}/forward", requireAuth(messageHandler.ForwardMessage))
	router.Handle(http.MethodGet, "/api/v1/conversations/{conversation_id}/pins", requireAuth(messageHandler.ListPins))
	router.Handle(http.MethodPost, "/api/v1/conversations/{conversation_id}/pins", requireAuth(messageHandler.PinConversationMessage))
	router.Handle(http.MethodDelete, "/api/v1/conversations/{conversation_id}/pins/{message_id}", requireAuth(messageHandler.UnpinConversationMessage))
	router.Handle(http.MethodGet, "/api/v1/groups", requireAuth(groupHandler.ListGroups))
	router.Handle(http.MethodPost, "/api/v1/groups", requireAuth(groupHandler.CreateGroup))
	router.Handle(http.MethodGet, "/api/v1/groups/invitations", requireAuth(groupHandler.ListMyInvitations))
	router.Handle(http.MethodPost, "/api/v1/groups/invitations/{invitation_id}/accept", requireAuth(groupHandler.AcceptInvitation))
	router.Handle(http.MethodPost, "/api/v1/groups/invitations/{invitation_id}/decline", requireAuth(groupHandler.DeclineInvitation))
	router.Handle(http.MethodPost, "/api/v1/groups/invitations/{invitation_id}/cancel", requireAuth(groupHandler.CancelInvitation))
	router.Handle(http.MethodGet, "/api/v1/groups/{group_id}", requireAuth(groupHandler.GetGroup))
	router.Handle(http.MethodPatch, "/api/v1/groups/{group_id}", requireAuth(groupHandler.UpdateGroup))
	router.Handle(http.MethodPost, "/api/v1/groups/{group_id}/disband", requireAuth(groupHandler.Disband))
	router.Handle(http.MethodPost, "/api/v1/groups/{group_id}/transfer", requireAuth(groupHandler.TransferOwnership))
	router.Handle(http.MethodGet, "/api/v1/groups/{group_id}/members", requireAuth(groupHandler.ListMembers))
	router.Handle(http.MethodPost, "/api/v1/groups/{group_id}/members", requireAuth(groupHandler.AddMembers))
	router.Handle(http.MethodPatch, "/api/v1/groups/{group_id}/members/{user_id}", requireAuth(groupHandler.SetRole))
	router.Handle(http.MethodPost, "/api/v1/groups/{group_id}/members/{user_id}/remove", requireAuth(groupHandler.RemoveMember))
	router.Handle(http.MethodPost, "/api/v1/groups/{group_id}/leave", requireAuth(groupHandler.Leave))
	router.Handle(http.MethodGet, "/api/v1/groups/{group_id}/invitations", requireAuth(groupHandler.ListGroupInvitations))
	router.Handle(http.MethodPost, "/api/v1/groups/{group_id}/invitations", requireAuth(groupHandler.CreateInvitation))
	router.Handle(http.MethodGet, "/api/v1/groups/{group_id}/announcements", requireAuth(groupHandler.ListAnnouncements))
	router.Handle(http.MethodPost, "/api/v1/groups/{group_id}/announcements", requireAuth(groupHandler.CreateAnnouncement))
	router.Handle(http.MethodPatch, "/api/v1/groups/{group_id}/announcements/{announcement_id}", requireAuth(groupHandler.UpdateAnnouncement))
	router.Handle(http.MethodPost, "/api/v1/groups/{group_id}/announcements/{announcement_id}/retract", requireAuth(groupHandler.RetractAnnouncement))
	router.Handle(http.MethodGet, "/api/v1/groups/{group_id}/settings", requireAuth(groupHandler.GetSettings))
	router.Handle(http.MethodPatch, "/api/v1/groups/{group_id}/settings", requireAuth(groupHandler.UpdateSettings))
	router.Handle(http.MethodPost, "/api/v1/media/uploads", requireAuth(mediaHandler.InitiateUpload))
	router.Handle(http.MethodGet, "/api/v1/media/uploads", requireAuth(mediaHandler.ListUploads))
	router.Handle(http.MethodGet, "/api/v1/media/uploads/{upload_id}", requireAuth(mediaHandler.GetUpload))
	router.Handle(http.MethodPost, "/api/v1/media/uploads/{upload_id}/progress", requireAuth(mediaHandler.ReportProgress))
	router.Handle(http.MethodPost, "/api/v1/media/uploads/{upload_id}/complete", requireAuth(mediaHandler.CompleteUpload))
	router.Handle(http.MethodPost, "/api/v1/media/uploads/{upload_id}/cancel", requireAuth(mediaHandler.CancelUpload))
	router.Handle(http.MethodPost, "/api/v1/media/uploads/{upload_id}/upload-url", requireAuth(mediaHandler.IssueUploadURL))
	router.Handle(http.MethodPut, "/api/v1/media/relay/uploads/{upload_id}", requireAuth(mediaHandler.RelayUpload))
	router.Handle(http.MethodGet, "/api/v1/media/relay/files/{file_id}", requireAuth(mediaHandler.RelayDownload))
	router.Handle(http.MethodGet, "/api/v1/media/files", requireAuth(mediaHandler.ListFiles))
	router.Handle(http.MethodGet, "/api/v1/media/files/{file_id}", requireAuth(mediaHandler.GetFile))
	router.Handle(http.MethodDelete, "/api/v1/media/files/{file_id}", requireAuth(mediaHandler.DeleteFile))
	router.Handle(http.MethodGet, "/api/v1/media/files/{file_id}/thumbnails", requireAuth(mediaHandler.ListThumbnails))
	router.Handle(http.MethodPost, "/api/v1/media/files/{file_id}/thumbnails", requireAuth(mediaHandler.RegisterThumbnail))
	router.Handle(http.MethodPost, "/api/v1/media/files/{file_id}/authorize", requireAuth(mediaHandler.AuthorizeAccess))
	router.Handle(http.MethodGet, "/api/v1/search/messages", requireAuth(searchHandler.SearchMessages))
	router.Handle(http.MethodPost, "/api/v1/sticker-packs", requireAuth(stickerHandler.CreatePack))
	router.Handle(http.MethodGet, "/api/v1/sticker-packs", requireAuth(stickerHandler.ListPacks))
	router.Handle(http.MethodGet, "/api/v1/sticker-packs/{pack_id}", requireAuth(stickerHandler.GetPack))
	router.Handle(http.MethodPatch, "/api/v1/sticker-packs/{pack_id}", requireAuth(stickerHandler.UpdatePack))
	router.Handle(http.MethodPost, "/api/v1/sticker-packs/{pack_id}/delete", requireAuth(stickerHandler.DeletePack))
	router.Handle(http.MethodPost, "/api/v1/sticker-packs/{pack_id}/stickers", requireAuth(stickerHandler.AddSticker))
	router.Handle(http.MethodGet, "/api/v1/sticker-packs/{pack_id}/stickers", requireAuth(stickerHandler.ListStickers))
	router.Handle(http.MethodPost, "/api/v1/sticker-packs/{pack_id}/favorite", requireAuth(stickerHandler.FavoritePack))
	router.Handle(http.MethodDelete, "/api/v1/sticker-packs/{pack_id}/favorite", requireAuth(stickerHandler.UnfavoritePack))
	router.Handle(http.MethodPost, "/api/v1/stickers/{sticker_id}/delete", requireAuth(stickerHandler.RemoveSticker))
	router.Handle(http.MethodGet, "/api/v1/stickers/favorites", requireAuth(stickerHandler.ListFavorites))
	router.Handle(http.MethodPost, "/api/v1/stickers/{sticker_id}/use", requireAuth(stickerHandler.RecordStickerUse))
	router.Handle(http.MethodGet, "/api/v1/stickers/recent", requireAuth(stickerHandler.ListRecentStickers))
	router.Handle(http.MethodGet, "/api/v1/stickers/search", requireAuth(stickerHandler.SearchStickers))
	router.Handle(http.MethodPost, "/api/v1/posts", requireAuth(timelineHandler.CreatePost))
	router.Handle(http.MethodGet, "/api/v1/posts", requireAuth(timelineHandler.ListAuthorPosts))
	router.Handle(http.MethodGet, "/api/v1/timeline", requireAuth(timelineHandler.ListFeed))
	router.Handle(http.MethodGet, "/api/v1/posts/{post_id}", requireAuth(timelineHandler.GetPost))
	router.Handle(http.MethodPatch, "/api/v1/posts/{post_id}", requireAuth(timelineHandler.UpdatePost))
	router.Handle(http.MethodPost, "/api/v1/posts/{post_id}/delete", requireAuth(timelineHandler.DeletePost))
	router.Handle(http.MethodGet, "/api/v1/posts/{post_id}/media", requireAuth(timelineHandler.ListPostMedia))
	router.Handle(http.MethodPost, "/api/v1/posts/{post_id}/likes", requireAuth(timelineHandler.LikePost))
	router.Handle(http.MethodDelete, "/api/v1/posts/{post_id}/likes", requireAuth(timelineHandler.UnlikePost))
	router.Handle(http.MethodPost, "/api/v1/posts/{post_id}/comments", requireAuth(timelineHandler.CreateComment))
	router.Handle(http.MethodGet, "/api/v1/posts/{post_id}/comments", requireAuth(timelineHandler.ListComments))
	router.Handle(http.MethodPost, "/api/v1/comments/{comment_id}/delete", requireAuth(timelineHandler.DeleteComment))
	router.Handle(http.MethodPost, "/api/v1/calls", requireAuth(callHandler.InitiateCall))
	router.Handle(http.MethodGet, "/api/v1/calls", requireAuth(callHandler.ListCalls))
	router.Handle(http.MethodGet, "/api/v1/calls/{call_id}", requireAuth(callHandler.GetCall))
	router.Handle(http.MethodPost, "/api/v1/calls/{call_id}/answer", requireAuth(callHandler.AnswerCall))
	router.Handle(http.MethodPost, "/api/v1/calls/{call_id}/decline", requireAuth(callHandler.DeclineCall))
	router.Handle(http.MethodPost, "/api/v1/calls/{call_id}/busy", requireAuth(callHandler.BusyCall))
	router.Handle(http.MethodPost, "/api/v1/calls/{call_id}/end", requireAuth(callHandler.EndCall))
	router.Handle(http.MethodGet, "/api/v1/calls/{call_id}/participants", requireAuth(callHandler.ListParticipants))
	router.Handle(http.MethodGet, "/api/v1/calls/{call_id}/events", requireAuth(callHandler.ListEvents))
	return router.Handler()
}

// buildVerifier 由 OIDC discovery 建立 Token 驗證器：
// Keycloak 位址＋Realm → discovery 文件 → jwks_uri。
// 探測失敗時明確報錯，不得以無驗證器狀態啟動業務路由。
func buildVerifier(ctx context.Context, cfg config.Config) (auth.TokenVerifier, error) {
	base := strings.TrimSuffix(cfg.KeycloakURL, "/")
	issuer := base + "/realms/" + cfg.KeycloakRealm
	discoveryURL := issuer + "/.well-known/openid-configuration"
	discoveredIssuer, jwksURL, err := auth.DiscoverJWKSURL(ctx, discoveryURL, nil)
	if err != nil {
		return nil, err
	}
	if discoveredIssuer != issuer {
		return nil, errors.New("api: discovery issuer 與設定不一致")
	}
	// Audience：KEYCLOAK_AUDIENCE 有設定時強制（Token 的 aud 須
	// 包含該值，字串或陣列形態皆可）；空值時不強制（M1 預設行為）。
	// 正式部署應設定為 API Client 的識別碼，見
	// docs/development/KEYCLOAK_INTEGRATION.md。
	return auth.NewVerifier(issuer, jwksURL, cfg.KeycloakAudience)
}

// buildObjectStore 依環境選擇物件儲存：MINIO_BUCKET／
// MINIO_ACCESS_KEY／MINIO_SECRET_KEY 三者齊備時以 MinIO 轉接器
// 承載（endpoint 取 config.MinIOEndpoint，憑證不得寫入程式碼）；
// 否則以記憶體 stub 承載（位元組不持久，僅供開發與測試），並以
// 日誌明示。憑證缺失時不得以空憑證建構 MinIO 轉接器。
func buildObjectStore(cfg config.Config) media.ObjectStore {
	bucket := strings.TrimSpace(os.Getenv("MINIO_BUCKET"))
	accessKey := strings.TrimSpace(os.Getenv("MINIO_ACCESS_KEY"))
	secretKey := strings.TrimSpace(os.Getenv("MINIO_SECRET_KEY"))
	if bucket != "" && accessKey != "" && secretKey != "" {
		store, err := media.NewMinIOStore(cfg.MinIOEndpoint, bucket, accessKey, secretKey, nil)
		if err != nil {
			log.Fatalf("api: 建立 MinIO 物件儲存失敗: %v", err)
		}
		log.Printf("api: 物件儲存為 MinIO（endpoint %s，bucket %s），位元組走 presigned 直傳", cfg.MinIOEndpoint, bucket)
		return store
	}
	// 物件儲存：記憶體 stub（MinIO 憑證未設定）；上傳物件不持久，
	// 僅供開發與測試，位元組經 api 內中繼端點承載。
	log.Printf("api: 物件儲存為記憶體 stub（MinIO 憑證未設定），上傳物件不持久")
	return media.NewMemoryObjectStore()
}

// envFlag 讀取布林環境開關：true／1／yes（大小寫不限）為開，
// 其餘（含未設定）為關。
func envFlag(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

// buildSearchIndex 依 SEARCH_INDEX 選擇搜尋索引後端：
//   - 空值或 memory：記憶體索引（不持久、重啟後為空，須由消費
//     迴圈或重建填充）。
//   - meilisearch：Meilisearch 轉接器（位址取 config.MeilisearchURL，
//     API 金鑰取 MEILISEARCH_API_KEY 環境變數；任一缺失或連線
//     建構失敗時明確報錯退出，不得靜默退回記憶體索引）。
//   - 其他值：明確報錯退出，不得靜默降級。
func buildSearchIndex(cfg config.Config) search.Index {
	switch mode := strings.ToLower(strings.TrimSpace(os.Getenv("SEARCH_INDEX"))); mode {
	case "", "memory":
		log.Printf("api: 搜尋索引為記憶體實作（SEARCH_INDEX 未設定或 memory），重啟後索引為空")
		return search.NewMemoryIndex()
	case "meilisearch":
		apiKey := strings.TrimSpace(os.Getenv("MEILISEARCH_API_KEY"))
		idx, err := search.DialMeilisearch(cfg.MeilisearchURL, apiKey, "")
		if err != nil {
			log.Fatalf("api: 建立 Meilisearch 索引失敗: %v", err)
		}
		log.Printf("api: 搜尋索引為 Meilisearch（%s，索引 %s）", cfg.MeilisearchURL, search.MessagesIndexUID)
		return idx
	default:
		log.Fatalf("api: SEARCH_INDEX 須為 memory 或 meilisearch，現為 %q", mode)
		return nil
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		// 設定缺值是明確的啟動失敗，不得以預設真憑證或空位址啟動。
		log.Fatalf("api: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	verifier, err := buildVerifier(ctx, cfg)
	if err != nil {
		log.Fatalf("api: 建立身分驗證器失敗: %v", err)
	}

	// 身分儲存後端依 IDENTITY_STORE 選擇（預設 postgres）。PostgreSQL
	// 後端在啟動時 Ping，失敗即明確報錯退出，不得靜默降級為記憶體。
	var userSvc *user.Service
	var profileSvc *profile.Service
	var deviceSvc *device.Service
	var sessionSvc *auth.SessionService
	var friendSvc *friend.Service
	var chatSvc *chat.Service
	var messageSvc *message.Service
	var groupSvc *group.Service
	var mediaSvc *media.Service
	var stickerSvc *sticker.Service
	var timelineSvc *timeline.Service
	var callSvc *call.Service
	// Outbox 儲存（供 Relay 發佈迴圈共用）：PostgreSQL 模式以
	// PG 儲存、記憶體模式以記憶體儲存，皆與訊息寫入同一後端。
	var outboxStore outbox.Store
	// 搜尋索引後端：依 SEARCH_INDEX 選擇（見 buildSearchIndex），
	// 預設記憶體索引；Meilisearch 模式須備妥位址與 API 金鑰。
	searchIndex := buildSearchIndex(cfg)
	var pgDB *postgres.DB
	if cfg.IdentityStore == "postgres" {
		db, err := postgres.Open(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("api: 開啟 PostgreSQL 連線池失敗: %v", err)
		}
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
		pingErr := db.Ping(pingCtx)
		pingCancel()
		if pingErr != nil {
			_ = db.Close()
			log.Fatalf("api: PostgreSQL 連線失敗（IDENTITY_STORE=postgres）: %v", pingErr)
		}
		pgDB = db
		defer func() { _ = pgDB.Close() }()
		log.Printf("api: 身分儲存為 PostgreSQL（IDENTITY_STORE=postgres）")
		q := pgDB.Querier()
		userSvc = user.NewService(user.NewPGUserStore(q), user.NewPGSettingsStore(q), nil)
		profileSvc = profile.NewService(profile.NewPGStore(q), userSvc, nil)
		deviceSvc = device.NewService(device.NewPGStore(q), userSvc, nil)
		sessionSvc = auth.NewSessionService(auth.NewPGSessionStore(q), nil)
		friendSvc = friend.NewService(friend.NewPGStore(q), userSvc, profileSvc, nil)
		chatSvc = chat.NewService(chat.NewPGStore(q), userSvc, friendSvc, nil)
		// 物件儲存：依環境選擇 MinIO 轉接器或記憶體 stub（見
		// buildObjectStore）；上傳物件在 stub 模式不持久，僅供開發
		// 與測試。
		mediaSvc = media.NewService(media.NewPGStore(q), userSvc, buildObjectStore(cfg), nil)
		// Outbox 接線（訊息推播中繼）：訊息列與事件列同一交易寫入
		// （PGStoreWithDB）。Relay 發佈迴圈依 RELAY_ENABLED 環境開關
		// 在本行程啟動（見 main 末段的背景迴圈接線）。附件關聯經
		// Media 模組驗證檔案可關聯性（見 internal/message）。
		outboxStore = outbox.NewPGStore(q)
		messageSvc = message.NewServiceWithDeps(message.NewPGStoreWithDB(q, pgDB), chatSvc, outboxStore, mediaSvc, nil)
		groupSvc = group.NewService(group.NewPGStore(q), userSvc, friendSvc, chatSvc, nil)
		stickerSvc = sticker.NewService(sticker.NewPGStore(q), userSvc, mediaSvc, nil)
		timelineSvc = timeline.NewService(timeline.NewPGStore(q), userSvc, friendSvc, mediaSvc, nil)
		callSvc = call.NewService(call.NewPGStore(q), userSvc, chatSvc, groupSvc, nil)
	} else {
		// IDENTITY_STORE=memory：開發／測試用，資料不持久。
		log.Printf("api: 身分儲存為記憶體實作（IDENTITY_STORE=memory），資料不持久，不得用於正式環境")
		userSvc = user.NewService(user.NewMemoryUserStore(), user.NewMemorySettingsStore(), nil)
		profileSvc = profile.NewService(profile.NewMemoryStore(), userSvc, nil)
		deviceSvc = device.NewService(device.NewMemoryStore(), userSvc, nil)
		sessionSvc = auth.NewSessionService(auth.NewMemorySessionStore(), nil)
		friendSvc = friend.NewService(friend.NewMemoryStore(), userSvc, profileSvc, nil)
		chatSvc = chat.NewService(chat.NewMemoryStore(), userSvc, friendSvc, nil)
		mediaSvc = media.NewService(media.NewMemoryStore(), userSvc, buildObjectStore(cfg), nil)
		// Outbox 接線（記憶體模式）：訊息列與事件列同一互斥鎖內寫入；
		// 附件關聯經 Media 模組驗證檔案可關聯性。
		memoryOutbox := outbox.NewMemoryStore()
		outboxStore = memoryOutbox
		messageSvc = message.NewServiceWithDeps(message.NewMemoryStoreWithOutbox(memoryOutbox), chatSvc, memoryOutbox, mediaSvc, nil)
		groupSvc = group.NewService(group.NewMemoryStore(), userSvc, friendSvc, chatSvc, nil)
		stickerSvc = sticker.NewService(sticker.NewMemoryStore(), userSvc, mediaSvc, nil)
		timelineSvc = timeline.NewService(timeline.NewMemoryStore(), userSvc, friendSvc, mediaSvc, nil)
		callSvc = call.NewService(call.NewMemoryStore(), userSvc, chatSvc, groupSvc, nil)
	}

	// 搜尋查詢服務：索引後端由 buildSearchIndex 決定（記憶體或
	// Meilisearch）；權限過濾經 chatSvc。索引為空時查詢回傳空結果，
	// 不得謊稱結果完整。
	searchSvc := search.NewService(searchIndex, chatSvc)

	deps := dependencies{
		verifier: verifier,
		users:    userSvc,
		profiles: profileSvc,
		devices:  deviceSvc,
		sessions: sessionSvc,
		friends:  friendSvc,
		chats:    chatSvc,
		messages: messageSvc,
		groups:   groupSvc,
		media:    mediaSvc,
		search:   searchSvc,
		stickers: stickerSvc,
		timeline: timelineSvc,
		calls:    callSvc,
	}

	// 背景迴圈（Track A，預設關閉、依環境開關啟動）：
	//  - RELAY_ENABLED=true：Relay 輪詢 Outbox、經 NATS JetStream
	//    發佈事件（發佈者見 internal/relay/nats.go）。Stream 由部署
	//    方預先建立；連線失敗時明確報錯退出（fail-closed）。
	//  - SEARCH_CONSUMER_ENABLED=true：訂閱 NATS 事件、經 Projector
	//    寫入搜尋索引（消費迴圈見 internal/search）。Worker 行程
	//    接線待模組重構（見 STATUS.md），本行程先行承載。
	// 兩迴圈共用一條 NATS 連線（需要時才建立）。關停時先停 HTTP、
	// 再取消背景迴圈、最後關閉 NATS 連線。
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()
	var bgWG sync.WaitGroup
	var natsConn *nats.Conn
	var jetStream nats.JetStreamContext
	dialJetStream := func() (nats.JetStreamContext, error) {
		if jetStream != nil {
			return jetStream, nil
		}
		nc, js, err := relay.DialNATS(cfg.NATSURL)
		if err != nil {
			return nil, err
		}
		natsConn = nc
		jetStream = js
		return js, nil
	}
	if envFlag("RELAY_ENABLED") {
		js, err := dialJetStream()
		if err != nil {
			log.Fatalf("api: 建立 NATS 連線失敗（RELAY_ENABLED）: %v", err)
		}
		publisher, err := relay.NewJetStreamPublisher(js, "")
		if err != nil {
			log.Fatalf("api: 建立 NATS 發佈者失敗: %v", err)
		}
		relayInstance := relay.NewRelay(outboxStore, publisher, 0, 0, 0)
		relayInstance.ErrHandler = func(err error) {
			log.Printf("api: relay 錯誤: %v", err)
		}
		bgWG.Add(1)
		go func() {
			defer bgWG.Done()
			if err := relayInstance.Run(bgCtx); err != nil {
				log.Printf("api: relay 迴圈結束: %v", err)
			}
		}()
		log.Printf("api: Relay 發佈迴圈已啟動（NATS JetStream，stream %s）", relay.DefaultStreamName)
	}
	if envFlag("SEARCH_CONSUMER_ENABLED") {
		if _, err := dialJetStream(); err != nil {
			log.Fatalf("api: 建立 NATS 連線失敗（SEARCH_CONSUMER_ENABLED）: %v", err)
		}
		sub, err := search.NewNATSSubscription(natsConn, "", "", "")
		if err != nil {
			log.Fatalf("api: 建立搜尋索引訂閱失敗: %v", err)
		}
		consumer, err := search.NewConsumer(search.NewProjector(searchIndex), func(err error) {
			log.Printf("api: 搜尋索引消費錯誤: %v", err)
		})
		if err != nil {
			log.Fatalf("api: 建立搜尋索引消費迴圈失敗: %v", err)
		}
		bgWG.Add(1)
		go func() {
			defer bgWG.Done()
			defer func() { _ = sub.Close() }()
			if err := consumer.Run(bgCtx, sub.Channel()); err != nil {
				log.Printf("api: 搜尋索引消費迴圈結束: %v", err)
			}
		}()
		log.Printf("api: 搜尋索引消費迴圈已啟動（NATS 訂閱 %s）", search.DefaultConsumerSubject)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           buildHandler(cfg, deps),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// 優雅關停：收到 SIGINT／SIGTERM 時停止接受新連線，等待
	// 進行中的請求完成（逾時後強制關閉）。TODO：Phase 05 起與
	// Gateway／Worker 的關停順序協調（NATS 排空等）待定稿。
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("api: listen: %v", err)
		}
	}()
	log.Printf("api: listening on %s", server.Addr)

	<-sigCtx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("api: shutdown: %v", err)
	}
	// 背景迴圈關停：取消背景 ctx、等待迴圈結束、關閉 NATS 連線。
	bgCancel()
	bgWG.Wait()
	if natsConn != nil {
		natsConn.Close()
	}
}
