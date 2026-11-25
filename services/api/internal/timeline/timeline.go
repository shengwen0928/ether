package timeline

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"ether.local/api/internal/ids"
	"ether.local/api/internal/media"
	"ether.local/api/internal/transport"
)

// ErrPostNotFound 表示查無貼文、呼叫者無權限、貼文已刪除，或
// 權限不足而以「不存在」回應的情形。對外一律 404，不得區分原因，
// 不得洩漏貼文存在性（ERROR_CODES.md §6.3）。
var ErrPostNotFound = errors.New("timeline: post not found")

// ErrCommentNotFound 表示查無留言、呼叫者無權限、留言所屬貼文
// 不可見，或權限不足而以「不存在」回應的情形。對外一律 404。
var ErrCommentNotFound = errors.New("timeline: comment not found")

// ErrCheckerUnavailable 表示權限裁決所需的唯讀介面未接線：
// fail-closed，不得放行（呼叫方須回 503）。
var ErrCheckerUnavailable = errors.New("timeline: checker unavailable")

// Visibility 是貼文能見度。值域為本里程碑實作詞彙 public／
// friends／only_me（ERD §6.8.1 稱值域待定稿，指定好友可見屬
// 後續里程碑），待端點 Schema 定稿覆核。
type Visibility string

const (
	// VisibilityPublic 全體可見（封鎖者除外）。
	VisibilityPublic Visibility = "public"
	// VisibilityFriends 僅作者好友可見。
	VisibilityFriends Visibility = "friends"
	// VisibilityOnlyMe 僅作者可見。
	VisibilityOnlyMe Visibility = "only_me"
)

// Post 是 posts 的一列。刪除為狀態標記，不得物理刪除；內容
// 保留於列中供稽核，讀取路徑一律以 404 回應已刪除貼文，不得
// 回傳其內容。
type Post struct {
	ID         string
	AuthorID   string
	Content    string
	Visibility Visibility
	EditedAt   *time.Time
	IsDeleted  bool
	DeletedAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// PostMedia 是 post_media 的一列：貼文與媒體檔案的關聯，
// 只存 media_file_id 直接參照（沿 M10 附件關聯先例，不存
// storage_path；詮釋資料於關聯時不快照，讀取時以檔案列為準
// 的差異見 STATUS.md 已知限制）。Caption 欄位保留，寫入時為空，
// 逐媒體說明文字屬後續里程碑。
type PostMedia struct {
	ID          string
	PostID      string
	MediaFileID string
	Position    int
	Caption     string
	CreatedAt   time.Time
}

// Like 是 post_likes 的一列：每人每則貼文僅一列；取消按讚為
// unliked_at 標記，不得物理刪除（ERD §6.8.3 草案）。重新按讚
// 清空標記、更新 liked_at。
type Like struct {
	PostID    string
	UserID    string
	LikedAt   time.Time
	UnlikedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Active 回傳該按讚是否生效。
func (l *Like) Active() bool { return l.UnlikedAt == nil }

// Comment 是 post_comments 的一列。ParentCommentID 為空表示
// 頂層留言，否則為回覆（父留言須屬同一貼文）。刪除為狀態標記，
// 讀取時內容回空字串，子留言保留（顯示規則待定稿）。
type Comment struct {
	ID              string
	PostID          string
	AuthorID        string
	ParentCommentID string
	Content         string
	EditedAt        *time.Time
	IsDeleted       bool
	DeletedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ValidationError 承載欄位驗證失敗清單。
type ValidationError struct {
	Fields []transport.FieldError
}

func (e *ValidationError) Error() string { return "timeline: validation failed" }

// FieldErrors 供 transport.WriteServiceError 寫出 details。
func (e *ValidationError) FieldErrors() []transport.FieldError { return e.Fields }

func validationError(field, issue, message string) *ValidationError {
	return &ValidationError{Fields: []transport.FieldError{{Field: field, Issue: issue, Message: message}}}
}

// UserResolver 是 Timeline 對 User 模組的最小依賴（依賴 DAG：
// Timeline --> User）：僅確認使用者存在。
type UserResolver interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// FriendChecker 是 Timeline 對 Friend 模組的最小依賴（依賴 DAG：
// Timeline --> Friend）：僅讀取好友與封鎖狀態，不得寫入好友表。
type FriendChecker interface {
	AreFriends(ctx context.Context, a, b string) (bool, error)
	BlockedBetween(ctx context.Context, a, b string) (bool, error)
}

// MediaAccess 是 Timeline 對 Media 模組的最小依賴（依賴 DAG：
// Timeline --> Media）：僅驗證檔案可被呼叫者關聯為貼文媒體。
// 檔案存在、未刪除、為呼叫者所有、掃描已通過四項條件缺一不可，
// 任一不符回傳的錯誤語義由 Media 模組決定（不存在／非擁有者／
// 未通過掃描皆不得關聯）。本模組不得直讀 media_files 表。
type MediaAccess interface {
	CheckAttachable(ctx context.Context, accessorID, fileID string) (*media.MediaFile, error)
}

// Store 是 Timeline 四張表的儲存介面。游標分頁一律以
// (created_at, id) 倒序鍵為準：ListRecentPosts 與
// ListPostsByAuthor 回傳 cursor 之後（更舊）的一頁，呼叫方以
// 回傳列的最後一筆組裝下一頁游標。
type Store interface {
	GetPost(ctx context.Context, id string) (*Post, error)
	CreatePost(ctx context.Context, p *Post) error
	SavePost(ctx context.Context, p *Post) error
	// ListRecentPosts 列出未刪除貼文（全體作者），依建立時間倒序。
	ListRecentPosts(ctx context.Context, cursor string, limit int) ([]*Post, error)
	// ListPostsByAuthor 列出某作者的未刪除貼文，依建立時間倒序。
	ListPostsByAuthor(ctx context.Context, authorID, cursor string, limit int) ([]*Post, error)

	GetPostMedia(ctx context.Context, postID string) ([]*PostMedia, error)
	CreatePostMedia(ctx context.Context, m *PostMedia) error

	GetLike(ctx context.Context, postID, userID string) (*Like, error)
	SaveLike(ctx context.Context, l *Like) error
	CountActiveLikes(ctx context.Context, postID string) (int, error)

	GetComment(ctx context.Context, id string) (*Comment, error)
	CreateComment(ctx context.Context, c *Comment) error
	SaveComment(ctx context.Context, c *Comment) error
	// ListComments 列出某貼文的留言（含已刪除標記者，內容由
	// 服務層遮蔽），依建立時間升序。
	ListComments(ctx context.Context, postID, cursor string, limit int) ([]*Comment, error)
	CountVisibleComments(ctx context.Context, postID string) (int, error)
}

// Service 是 Timeline 領域服務。
type Service struct {
	store   Store
	users   UserResolver
	friends FriendChecker
	media   MediaAccess // 可為空：未接線時媒體關聯一律拒絕。
	now     func() time.Time
}

// NewService 建立 Service。now 為空時用 time.Now；friends 為空時
// 一切權限裁決 fail-closed（回 ErrCheckerUnavailable）。
func NewService(store Store, users UserResolver, friends FriendChecker, media MediaAccess, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, users: users, friends: friends, media: media, now: now}
}

// MaxPostContentRunes 是貼文內容上限（本里程碑實作預設，待定稿覆核）。
const MaxPostContentRunes = 2000

// MaxCommentContentRunes 是留言內容上限（本里程碑實作預設，待定稿覆核）。
const MaxCommentContentRunes = 1000

// MaxPostMedia 是單則貼文關聯媒體上限（本里程碑實作預設，待定稿覆核）。
const MaxPostMedia = 9

// DefaultFeedLimit 是時間線與作者貼文清單的預設分頁大小。
const DefaultFeedLimit = 20

// MaxFeedLimit 是時間線與作者貼文清單的分頁上限。
const MaxFeedLimit = 50

// DefaultCommentLimit 是留言清單的預設分頁大小。
const DefaultCommentLimit = 30

// MaxCommentLimit 是留言清單的分頁上限。
const MaxCommentLimit = 100

func validateContent(content string, maxRunes int, field string) *ValidationError {
	if utf8.RuneCountInString(content) > maxRunes {
		return validationError(field, "TOO_LONG", "內容過長。")
	}
	return nil
}

func validateVisibility(v Visibility) *ValidationError {
	switch v {
	case VisibilityPublic, VisibilityFriends, VisibilityOnlyMe:
		return nil
	default:
		return validationError("visibility", "INVALID_VALUE", "能見度須為 public、friends 或 only_me。")
	}
}

// loadPost 讀取貼文列：不存在或已刪除皆回 ErrPostNotFound。
func (s *Service) loadPost(ctx context.Context, postID string) (*Post, error) {
	p, err := s.store.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if p == nil || p.IsDeleted {
		return nil, ErrPostNotFound
	}
	return p, nil
}

// canView 裁決呼叫者是否可見該貼文：作者本人恆可見；任一方向
// 封鎖一律不可見；public 全體可見；friends 僅好友可見；only_me
// 僅作者可見。friends 檢查器未接線時 fail-closed。
func (s *Service) canView(ctx context.Context, viewerID string, p *Post) (bool, error) {
	if viewerID == p.AuthorID {
		return true, nil
	}
	if s.friends == nil {
		return false, ErrCheckerUnavailable
	}
	blocked, err := s.friends.BlockedBetween(ctx, viewerID, p.AuthorID)
	if err != nil {
		return false, err
	}
	if blocked {
		return false, nil
	}
	switch p.Visibility {
	case VisibilityPublic:
		return true, nil
	case VisibilityFriends:
		return s.friends.AreFriends(ctx, viewerID, p.AuthorID)
	case VisibilityOnlyMe:
		return false, nil
	default:
		return false, nil
	}
}

// visiblePost 讀取貼文並裁決可見性：不可見一律回 ErrPostNotFound，
// 不得區分原因。
func (s *Service) visiblePost(ctx context.Context, viewerID, postID string) (*Post, error) {
	p, err := s.loadPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	ok, err := s.canView(ctx, viewerID, p)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrPostNotFound
	}
	return p, nil
}

// PostEntry 是貼文查詢的一項：貼文列＋媒體關聯＋計數＋呼叫者
// 按讚狀態。計數一律即時彙總，不得以快取代替（ERD §6.8.3）。
type PostEntry struct {
	Post         Post
	Media        []PostMedia
	LikeCount    int
	CommentCount int
	LikedByMe    bool
}

func (s *Service) entryOf(ctx context.Context, viewerID string, p *Post) (*PostEntry, error) {
	mediaRows, err := s.store.GetPostMedia(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	likes, err := s.store.CountActiveLikes(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	comments, err := s.store.CountVisibleComments(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	liked := false
	if viewerID != "" {
		l, err := s.store.GetLike(ctx, p.ID, viewerID)
		if err != nil {
			return nil, err
		}
		liked = l != nil && l.Active()
	}
	outMedia := make([]PostMedia, 0, len(mediaRows))
	for _, m := range mediaRows {
		outMedia = append(outMedia, *m)
	}
	return &PostEntry{Post: *p, Media: outMedia, LikeCount: likes, CommentCount: comments, LikedByMe: liked}, nil
}

// CreatePost 建立貼文：內容可為空，僅限附媒體時（純媒體貼文）；
// 內容與媒體皆無時為欄位錯誤。媒體關聯經 Media 模組驗證（存在、
// 未刪除、為呼叫者所有、掃描已通過，缺一不可）。能見度為空時
// 預設 friends（保守預設，待定稿覆核）。貼文列與媒體關聯列
// 分步寫入，跨模組無交易邊界（媒體列寫入失敗時殘留無媒體貼文列，
// 為已知限制）。
func (s *Service) CreatePost(ctx context.Context, authorID, content string, visibility Visibility, mediaFileIDs []string) (*PostEntry, error) {
	if ve := validateContent(content, MaxPostContentRunes, "content"); ve != nil {
		return nil, ve
	}
	if visibility == "" {
		visibility = VisibilityFriends
	}
	if ve := validateVisibility(visibility); ve != nil {
		return nil, ve
	}
	mediaIDs := dedupeMediaIDs(mediaFileIDs)
	if len(mediaIDs) > MaxPostMedia {
		return nil, validationError("media_file_ids", "TOO_LONG", "單則貼文最多關聯 9 個媒體檔案。")
	}
	if strings.TrimSpace(content) == "" && len(mediaIDs) == 0 {
		return nil, validationError("content", "REQUIRED", "貼文內容與媒體不得皆為空。")
	}
	if len(mediaIDs) > 0 {
		if s.media == nil {
			return nil, ErrCheckerUnavailable
		}
		for _, id := range mediaIDs {
			if _, err := s.media.CheckAttachable(ctx, authorID, id); err != nil {
				return nil, err
			}
		}
	}
	now := s.now().UTC()
	p := &Post{
		ID:         ids.New(),
		AuthorID:   authorID,
		Content:    content,
		Visibility: visibility,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if p.ID == "" {
		return nil, errors.New("timeline: 識別碼產生失敗")
	}
	if err := s.store.CreatePost(ctx, p); err != nil {
		return nil, err
	}
	for i, id := range mediaIDs {
		m := &PostMedia{
			ID:          ids.New(),
			PostID:      p.ID,
			MediaFileID: id,
			Position:    i,
			CreatedAt:   now,
		}
		if m.ID == "" {
			return nil, errors.New("timeline: 識別碼產生失敗")
		}
		if err := s.store.CreatePostMedia(ctx, m); err != nil {
			return nil, err
		}
	}
	return s.entryOf(ctx, authorID, p)
}

func dedupeMediaIDs(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, id := range in {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// GetPost 讀取單一貼文：僅可見者可讀，不可見一律 404。
func (s *Service) GetPost(ctx context.Context, viewerID, postID string) (*PostEntry, error) {
	p, err := s.visiblePost(ctx, viewerID, postID)
	if err != nil {
		return nil, err
	}
	return s.entryOf(ctx, viewerID, p)
}

// PostPatch 是貼文更新的指標語義：nil 表示不改。
type PostPatch struct {
	Content    *string
	Visibility *Visibility
}

// UpdatePost 編輯貼文：僅作者可編輯；已刪除貼文不可編輯。
// 編輯後記 edited_at（編輯歷史不保留，待定稿）。內容清空時
// 須仍附媒體，否則為欄位錯誤。
func (s *Service) UpdatePost(ctx context.Context, userID, postID string, patch PostPatch) (*PostEntry, error) {
	p, err := s.loadPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if p.AuthorID != userID {
		return nil, ErrPostNotFound
	}
	if patch.Content != nil {
		if ve := validateContent(*patch.Content, MaxPostContentRunes, "content"); ve != nil {
			return nil, ve
		}
		p.Content = *patch.Content
	}
	if patch.Visibility != nil {
		if ve := validateVisibility(*patch.Visibility); ve != nil {
			return nil, ve
		}
		p.Visibility = *patch.Visibility
	}
	if strings.TrimSpace(p.Content) == "" {
		mediaRows, err := s.store.GetPostMedia(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		if len(mediaRows) == 0 {
			return nil, validationError("content", "REQUIRED", "貼文內容與媒體不得皆為空。")
		}
	}
	now := s.now().UTC()
	p.EditedAt = &now
	p.UpdatedAt = now
	if err := s.store.SavePost(ctx, p); err != nil {
		return nil, err
	}
	return s.entryOf(ctx, userID, p)
}

// DeletePost 刪除貼文：僅作者可刪除。刪除為狀態標記，貼文列、
// 媒體關聯、按讚與留言歷史皆保留（append-only，不得連帶物理刪除）；
// 刪除後一切讀取端點皆以 404 回應。
func (s *Service) DeletePost(ctx context.Context, userID, postID string) error {
	p, err := s.loadPost(ctx, postID)
	if err != nil {
		return err
	}
	if p.AuthorID != userID {
		return ErrPostNotFound
	}
	now := s.now().UTC()
	p.IsDeleted = true
	p.DeletedAt = &now
	p.UpdatedAt = now
	return s.store.SavePost(ctx, p)
}

// encodeCursor 將 (created_at, id) 倒序鍵編為不透明游標。
func encodeCursor(t time.Time, id string) string {
	raw := t.UTC().Format(time.RFC3339Nano) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeCursor 解析游標：格式錯誤回欄位錯誤。
func decodeCursor(cursor string) (string, error) {
	if cursor == "" {
		return "", nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", validationError("cursor", "INVALID_VALUE", "游標格式錯誤。")
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", validationError("cursor", "INVALID_VALUE", "游標格式錯誤。")
	}
	if _, err := time.Parse(time.RFC3339Nano, parts[0]); err != nil {
		return "", validationError("cursor", "INVALID_VALUE", "游標格式錯誤。")
	}
	return cursor, nil
}

func normalizeFeedLimit(limit int) int {
	if limit <= 0 {
		return DefaultFeedLimit
	}
	if limit > MaxFeedLimit {
		return MaxFeedLimit
	}
	return limit
}

func normalizeCommentLimit(limit int) int {
	if limit <= 0 {
		return DefaultCommentLimit
	}
	if limit > MaxCommentLimit {
		return MaxCommentLimit
	}
	return limit
}

// feedPage 掃描候選貼文並以可見性過濾，累積至 limit 筆或候選
// 耗盡：搜尋模組同款「掃描後過濾」作法（Meilisearch 落地前
// 的已知限制，見 STATUS.md）。每輪多取一列以判斷是否還有下一頁
// 候選。掃描游標（scanCursor）每輪推進至最後一個候選列，避免
// 全數被濾除時原地重掃；回傳的 nextCursor 則取最後一筆回傳列
// 的排序鍵（與 Store 介面契約一致：呼叫方以回傳列的最後一筆
// 組裝下一頁游標）。兩者不得混用：以最後候選列為回傳游標會
// 使探測列在下一頁被跳過（2026-11-25 實證：五筆全可見、
// limit 2 時合計只取回四筆，第三筆遺失）。
func (s *Service) feedPage(ctx context.Context, viewerID, cursor string, limit int, list func(ctx context.Context, cursor string, limit int) ([]*Post, error)) ([]*Post, string, error) {
	var out []*Post
	scanCursor := cursor
	for len(out) < limit {
		want := limit - len(out) + 1
		candidates, err := list(ctx, scanCursor, want)
		if err != nil {
			return nil, "", err
		}
		if len(candidates) == 0 {
			return out, "", nil
		}
		for _, p := range candidates {
			ok, err := s.canView(ctx, viewerID, p)
			if err != nil {
				return nil, "", err
			}
			if ok && len(out) < limit {
				out = append(out, p)
			}
		}
		last := candidates[len(candidates)-1]
		scanCursor = encodeCursor(last.CreatedAt, last.ID)
		if len(candidates) < want {
			return out, "", nil
		}
		if len(out) >= limit {
			lastReturned := out[len(out)-1]
			return out, encodeCursor(lastReturned.CreatedAt, lastReturned.ID), nil
		}
	}
	return out, "", nil
}

// ListFeed 列出呼叫者的好友時間線：本人全部貼文＋好友的
// public／friends 貼文（封鎖者排除），依建立時間倒序、游標分頁。
// 回傳 nextCursor 為空表示已到盡頭。
func (s *Service) ListFeed(ctx context.Context, viewerID, cursor string, limit int) ([]*PostEntry, string, error) {
	if _, err := decodeCursor(cursor); err != nil {
		return nil, "", err
	}
	limit = normalizeFeedLimit(limit)
	posts, next, err := s.feedPage(ctx, viewerID, cursor, limit, s.store.ListRecentPosts)
	if err != nil {
		return nil, "", err
	}
	return s.entriesOf(ctx, viewerID, posts, next)
}

// ListAuthorPosts 列出某作者的貼文（個人時間線）：作者不存在
// 時 404；僅回傳呼叫者可見者，依建立時間倒序、游標分頁。
func (s *Service) ListAuthorPosts(ctx context.Context, viewerID, authorID, cursor string, limit int) ([]*PostEntry, string, error) {
	if _, err := decodeCursor(cursor); err != nil {
		return nil, "", err
	}
	if s.users == nil {
		return nil, "", ErrCheckerUnavailable
	}
	exists, err := s.users.Exists(ctx, authorID)
	if err != nil {
		return nil, "", err
	}
	if !exists {
		return nil, "", ErrPostNotFound
	}
	limit = normalizeFeedLimit(limit)
	posts, next, err := s.feedPage(ctx, viewerID, cursor, limit, func(ctx context.Context, c string, n int) ([]*Post, error) {
		return s.store.ListPostsByAuthor(ctx, authorID, c, n)
	})
	if err != nil {
		return nil, "", err
	}
	return s.entriesOf(ctx, viewerID, posts, next)
}

func (s *Service) entriesOf(ctx context.Context, viewerID string, posts []*Post, next string) ([]*PostEntry, string, error) {
	out := make([]*PostEntry, 0, len(posts))
	for _, p := range posts {
		e, err := s.entryOf(ctx, viewerID, p)
		if err != nil {
			return nil, "", err
		}
		out = append(out, e)
	}
	return out, next, nil
}

// ListPostMedia 列出貼文的媒體關聯：僅可見者可讀。
func (s *Service) ListPostMedia(ctx context.Context, viewerID, postID string) ([]PostMedia, error) {
	if _, err := s.visiblePost(ctx, viewerID, postID); err != nil {
		return nil, err
	}
	rows, err := s.store.GetPostMedia(ctx, postID)
	if err != nil {
		return nil, err
	}
	out := make([]PostMedia, 0, len(rows))
	for _, m := range rows {
		out = append(out, *m)
	}
	return out, nil
}

// LikePost 按讚：僅可見者可按讚；重複按讚冪等成功；曾取消者
// 重新生效。回傳按讚狀態與最新計數。
func (s *Service) LikePost(ctx context.Context, viewerID, postID string) (bool, int, error) {
	if _, err := s.visiblePost(ctx, viewerID, postID); err != nil {
		return false, 0, err
	}
	now := s.now().UTC()
	l, err := s.store.GetLike(ctx, postID, viewerID)
	if err != nil {
		return false, 0, err
	}
	if l == nil {
		l = &Like{PostID: postID, UserID: viewerID, LikedAt: now, CreatedAt: now, UpdatedAt: now}
	} else if !l.Active() {
		l.LikedAt = now
		l.UnlikedAt = nil
		l.UpdatedAt = now
	}
	if err := s.store.SaveLike(ctx, l); err != nil {
		return false, 0, err
	}
	count, err := s.store.CountActiveLikes(ctx, postID)
	if err != nil {
		return false, 0, err
	}
	return true, count, nil
}

// UnlikePost 取消按讚：僅可見者可操作；未按讚或已取消時冪等成功。
func (s *Service) UnlikePost(ctx context.Context, viewerID, postID string) (bool, int, error) {
	if _, err := s.visiblePost(ctx, viewerID, postID); err != nil {
		return false, 0, err
	}
	l, err := s.store.GetLike(ctx, postID, viewerID)
	if err != nil {
		return false, 0, err
	}
	if l != nil && l.Active() {
		now := s.now().UTC()
		l.UnlikedAt = &now
		l.UpdatedAt = now
		if err := s.store.SaveLike(ctx, l); err != nil {
			return false, 0, err
		}
	}
	count, err := s.store.CountActiveLikes(ctx, postID)
	if err != nil {
		return false, 0, err
	}
	return false, count, nil
}

// CreateComment 建立留言：僅可見者可留言；parent_comment_id
// 有值時須為同一貼文的留言（父留言已刪除時仍可回覆，顯示規則
// 待定稿）。留言不是訊息，不得寫入 messages。
func (s *Service) CreateComment(ctx context.Context, viewerID, postID, content, parentCommentID string) (*Comment, error) {
	if _, err := s.visiblePost(ctx, viewerID, postID); err != nil {
		return nil, err
	}
	if ve := validateContent(content, MaxCommentContentRunes, "content"); ve != nil {
		return nil, ve
	}
	if strings.TrimSpace(content) == "" {
		return nil, validationError("content", "REQUIRED", "留言內容不得為空。")
	}
	if parentCommentID != "" {
		parent, err := s.store.GetComment(ctx, parentCommentID)
		if err != nil {
			return nil, err
		}
		if parent == nil || parent.PostID != postID {
			return nil, validationError("parent_comment_id", "INVALID_VALUE", "回覆的留言不存在或不屬於此貼文。")
		}
	}
	now := s.now().UTC()
	c := &Comment{
		ID:              ids.New(),
		PostID:          postID,
		AuthorID:        viewerID,
		ParentCommentID: parentCommentID,
		Content:         content,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if c.ID == "" {
		return nil, errors.New("timeline: 識別碼產生失敗")
	}
	if err := s.store.CreateComment(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListComments 列出貼文留言：僅可見者可讀，依建立時間升序、
// 游標分頁（游標為上一頁最後一筆的 (created_at, id)）。已刪除
// 留言內容回空字串。
func (s *Service) ListComments(ctx context.Context, viewerID, postID, cursor string, limit int) ([]*Comment, string, error) {
	if _, err := s.visiblePost(ctx, viewerID, postID); err != nil {
		return nil, "", err
	}
	if _, err := decodeCursor(cursor); err != nil {
		return nil, "", err
	}
	limit = normalizeCommentLimit(limit)
	rows, err := s.store.ListComments(ctx, postID, cursor, limit+1)
	if err != nil {
		return nil, "", err
	}
	var next string
	if len(rows) > limit {
		last := rows[limit-1]
		next = encodeCursor(last.CreatedAt, last.ID)
		rows = rows[:limit]
	}
	out := make([]*Comment, 0, len(rows))
	for _, c := range rows {
		cp := *c
		if cp.IsDeleted {
			cp.Content = ""
		}
		out = append(out, &cp)
	}
	return out, next, nil
}

// DeleteComment 刪除留言：僅作者可刪除。刪除為狀態標記，子留言
// 保留；刪除後讀取時內容為空。非作者或留言所屬貼文不可見時
// 一律 404。
func (s *Service) DeleteComment(ctx context.Context, viewerID, commentID string) error {
	c, err := s.store.GetComment(ctx, commentID)
	if err != nil {
		return err
	}
	if c == nil || c.IsDeleted {
		return ErrCommentNotFound
	}
	if _, err := s.visiblePost(ctx, viewerID, c.PostID); err != nil {
		return ErrCommentNotFound
	}
	if c.AuthorID != viewerID {
		return ErrCommentNotFound
	}
	now := s.now().UTC()
	c.IsDeleted = true
	c.DeletedAt = &now
	c.UpdatedAt = now
	return s.store.SaveComment(ctx, c)
}

// MemoryStore 是 Store 的記憶體實作（開發與測試用）。
type MemoryStore struct {
	mu       sync.RWMutex
	posts    map[string]*Post
	media    map[string][]*PostMedia     // postID -> 依 position 排序
	likes    map[string]map[string]*Like // postID -> userID -> Like
	comments map[string]*Comment
}

// NewMemoryStore 建立空的記憶體動態儲存。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		posts:    map[string]*Post{},
		media:    map[string][]*PostMedia{},
		likes:    map[string]map[string]*Like{},
		comments: map[string]*Comment{},
	}
}

func clonePost(p *Post) *Post {
	if p == nil {
		return nil
	}
	cp := *p
	if p.EditedAt != nil {
		t := *p.EditedAt
		cp.EditedAt = &t
	}
	if p.DeletedAt != nil {
		t := *p.DeletedAt
		cp.DeletedAt = &t
	}
	return &cp
}

func clonePostMedia(m *PostMedia) *PostMedia {
	if m == nil {
		return nil
	}
	cp := *m
	return &cp
}

func cloneLike(l *Like) *Like {
	if l == nil {
		return nil
	}
	cp := *l
	if l.UnlikedAt != nil {
		t := *l.UnlikedAt
		cp.UnlikedAt = &t
	}
	return &cp
}

func cloneComment(c *Comment) *Comment {
	if c == nil {
		return nil
	}
	cp := *c
	if c.EditedAt != nil {
		t := *c.EditedAt
		cp.EditedAt = &t
	}
	if c.DeletedAt != nil {
		t := *c.DeletedAt
		cp.DeletedAt = &t
	}
	return &cp
}

// cursorBound 解析游標為 (created_at, id) 邊界：游標為空時回
// 零值（取全部）；格式錯誤時回錯誤（儲存層不得靜默接受壞游標）。
func cursorBound(cursor string) (time.Time, string, error) {
	if cursor == "" {
		return time.Time{}, "", nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", err
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", errors.New("timeline: 游標格式錯誤")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", err
	}
	return t, parts[1], nil
}

// afterCursor 判斷列是否在游標之後（更舊）：倒序鍵
// (created_at, id) 小於游標鍵者為之後。游標為空時一律為真。
func afterCursor(createdAt time.Time, id string, bound time.Time, boundID string) bool {
	if bound.IsZero() {
		return true
	}
	if createdAt.Before(bound) {
		return true
	}
	if createdAt.Equal(bound) && id < boundID {
		return true
	}
	return false
}

func sortPostsDesc(posts []*Post) {
	for i := 0; i < len(posts); i++ {
		for j := i + 1; j < len(posts); j++ {
			if posts[j].CreatedAt.After(posts[i].CreatedAt) ||
				(posts[j].CreatedAt.Equal(posts[i].CreatedAt) && posts[j].ID > posts[i].ID) {
				posts[i], posts[j] = posts[j], posts[i]
			}
		}
	}
}

func (s *MemoryStore) GetPost(_ context.Context, id string) (*Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return clonePost(s.posts[id]), nil
}

func (s *MemoryStore) CreatePost(_ context.Context, p *Post) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.posts[p.ID]; exists {
		return errors.New("timeline: post 已存在")
	}
	s.posts[p.ID] = clonePost(p)
	return nil
}

func (s *MemoryStore) SavePost(_ context.Context, p *Post) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.posts[p.ID]; !exists {
		return ErrPostNotFound
	}
	s.posts[p.ID] = clonePost(p)
	return nil
}

func (s *MemoryStore) ListRecentPosts(_ context.Context, cursor string, limit int) ([]*Post, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Post
	for _, p := range s.posts {
		if p.IsDeleted {
			continue
		}
		if afterCursor(p.CreatedAt, p.ID, bound, boundID) {
			out = append(out, clonePost(p))
		}
	}
	sortPostsDesc(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) ListPostsByAuthor(_ context.Context, authorID, cursor string, limit int) ([]*Post, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Post
	for _, p := range s.posts {
		if p.IsDeleted || p.AuthorID != authorID {
			continue
		}
		if afterCursor(p.CreatedAt, p.ID, bound, boundID) {
			out = append(out, clonePost(p))
		}
	}
	sortPostsDesc(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) GetPostMedia(_ context.Context, postID string) ([]*PostMedia, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := s.media[postID]
	out := make([]*PostMedia, 0, len(rows))
	for _, m := range rows {
		out = append(out, clonePostMedia(m))
	}
	return out, nil
}

func (s *MemoryStore) CreatePostMedia(_ context.Context, m *PostMedia) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.media[m.PostID] = append(s.media[m.PostID], clonePostMedia(m))
	return nil
}

func (s *MemoryStore) GetLike(_ context.Context, postID, userID string) (*Like, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneLike(s.likes[postID][userID]), nil
}

func (s *MemoryStore) SaveLike(_ context.Context, l *Like) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.likes[l.PostID] == nil {
		s.likes[l.PostID] = map[string]*Like{}
	}
	s.likes[l.PostID][l.UserID] = cloneLike(l)
	return nil
}

func (s *MemoryStore) CountActiveLikes(_ context.Context, postID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, l := range s.likes[postID] {
		if l.Active() {
			n++
		}
	}
	return n, nil
}

func (s *MemoryStore) GetComment(_ context.Context, id string) (*Comment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneComment(s.comments[id]), nil
}

func (s *MemoryStore) CreateComment(_ context.Context, c *Comment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.comments[c.ID]; exists {
		return errors.New("timeline: comment 已存在")
	}
	s.comments[c.ID] = cloneComment(c)
	return nil
}

func (s *MemoryStore) SaveComment(_ context.Context, c *Comment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.comments[c.ID]; !exists {
		return ErrCommentNotFound
	}
	s.comments[c.ID] = cloneComment(c)
	return nil
}

func (s *MemoryStore) ListComments(_ context.Context, postID, cursor string, limit int) ([]*Comment, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Comment
	for _, c := range s.comments {
		if c.PostID != postID {
			continue
		}
		// 升序游標：取建立鍵大於游標鍵者。
		if cursor != "" {
			if c.CreatedAt.Before(bound) {
				continue
			}
			if c.CreatedAt.Equal(bound) && c.ID <= boundID {
				continue
			}
		}
		out = append(out, cloneComment(c))
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.Before(out[i].CreatedAt) ||
				(out[j].CreatedAt.Equal(out[i].CreatedAt) && out[j].ID < out[i].ID) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) CountVisibleComments(_ context.Context, postID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, c := range s.comments {
		if c.PostID == postID && !c.IsDeleted {
			n++
		}
	}
	return n, nil
}
