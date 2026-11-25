package sticker

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

// ErrPackNotFound 表示查無貼圖包、呼叫者無權限、貼圖包已下架，
// 或權限不足而以「不存在」回應的情形。對外一律 404，不得區分原因，
// 不得洩漏貼圖包存在性（ERROR_CODES.md §6.3）。
var ErrPackNotFound = errors.New("sticker: pack not found")

// ErrStickerNotFound 表示查無貼圖、呼叫者無權限、貼圖已移除，或
// 權限不足而以「不存在」回應的情形。對外一律 404。
var ErrStickerNotFound = errors.New("sticker: sticker not found")

// ErrCheckerUnavailable 表示權限裁決所需的唯讀介面未接線：
// fail-closed，不得放行（呼叫方須回 503）。
var ErrCheckerUnavailable = errors.New("sticker: checker unavailable")

// PackStatus 是貼圖包狀態。值域為本里程碑實作詞彙（ERD §6.7.1
// 稱值域待定稿，例如上架／下架／審核中），待端點 Schema 定稿覆核。
type PackStatus string

const (
	// PackStatusDraft 草稿：僅作者可見。
	PackStatusDraft PackStatus = "draft"
	// PackStatusPublished 已上架：依免費／付費規則可見。
	PackStatusPublished PackStatus = "published"
	// PackStatusUnlisted 未上架：僅作者可見（曾上架後下架）。
	PackStatusUnlisted PackStatus = "unlisted"
	// PackStatusArchived 已封存：刪除的狀態標記，一律 404，列保留。
	PackStatusArchived PackStatus = "archived"
)

// StickerStatus 是貼圖單張狀態。值域為本里程碑實作詞彙（ERD
// §6.7.2 稱值域待定稿），待端點 Schema 定稿覆核。
type StickerStatus string

const (
	// StickerStatusActive 生效中。
	StickerStatusActive StickerStatus = "active"
	// StickerStatusRemoved 已移除：狀態標記，一律 404，列保留。
	StickerStatusRemoved StickerStatus = "removed"
)

// UserStickerSource 是 user_stickers 列的來源語義：一列只承載一種
// 來源（ERD §6.7.3 稱值域待定稿，例如下載／購買／收藏）。
type UserStickerSource string

const (
	// SourceFavorite 使用者主動收藏的包（is_favorite 為真時）。
	SourceFavorite UserStickerSource = "favorite"
	// SourceGranted 經授權發放介面取得的包（付款完成後由 Payment
	// 模組呼叫 GrantPack 寫入；本模組無 HTTP 端點可寫入此來源）。
	SourceGranted UserStickerSource = "granted"
	// SourceRecent 最近使用過的單張貼圖（last_used_at 為準）。
	SourceRecent UserStickerSource = "recent"
)

// PackCategory 是貼圖包分類。值域為本里程碑實作詞彙（ERD §6.7.1
// 稱值域待定稿），待端點 Schema 定稿覆核；空值表示未分類。
type PackCategory string

const (
	// CategoryGeneral 一般。
	CategoryGeneral PackCategory = "general"
	// CategoryAnimals 動物。
	CategoryAnimals PackCategory = "animals"
	// CategoryFood 食物。
	CategoryFood PackCategory = "food"
	// CategoryEmotions 情緒。
	CategoryEmotions PackCategory = "emotions"
	// CategoryFestivals 節慶。
	CategoryFestivals PackCategory = "festivals"
	// CategoryCharacters 角色。
	CategoryCharacters PackCategory = "characters"
	// CategoryOther 其他。
	CategoryOther PackCategory = "other"
)

// Pack 是 sticker_packs 的一列。刪除為 archived 狀態標記，不得
// 物理刪除（已授權／已收藏使用者的權益需要歷史）；author_name
// 欄位（ERD 草案）本里程碑不建，作者顯示名稱由 User 模組公開
// 檔案解析，屬後續里程碑。
type Pack struct {
	ID           string
	CreatorID    string
	Name         string
	Description  string
	Category     PackCategory
	CoverMediaID string
	IsFree       bool
	Status       PackStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Sticker 是 stickers 的一列：貼圖本體只存 media_file_id 直接參照
// （沿 M10 附件關聯與 Timeline 貼文媒體先例，不存 storage_path，
// 不存位元組）。移除為狀態標記，不得物理刪除（已傳送訊息的參照
// 需要歷史）；position 由服務層依序指派，移除列保留原序號，新
// 增貼圖一律接在最大序號之後，不得重用已移除列的序號。
type Sticker struct {
	ID          string
	PackID      string
	MediaFileID string
	Name        string
	IsAnimated  bool
	Position    int
	Status      StickerStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// UserSticker 是 user_stickers 的一列：包級關聯（StickerID 為空）
// 或單張級關聯（PackID 為空），二者恰居其一（資料庫 CHECK 約束
// 強制）。包級列承載收藏與授權發放（source 為 favorite／granted，
// is_favorite 獨立於 source：已授權的包亦可收藏）；單張級列承載
// 最近使用（source 為 recent，last_used_at 為準）。使用者移除包級
// 關聯為硬刪除列（ERD 草案），不得刪除貼圖包本身。
type UserSticker struct {
	ID         string
	UserID     string
	PackID     string
	StickerID  string
	Source     UserStickerSource
	LastUsedAt *time.Time
	IsFavorite bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ValidationError 承載欄位驗證失敗清單。
type ValidationError struct {
	Fields []transport.FieldError
}

func (e *ValidationError) Error() string { return "sticker: validation failed" }

// FieldErrors 供 transport.WriteServiceError 寫出 details。
func (e *ValidationError) FieldErrors() []transport.FieldError { return e.Fields }

func validationError(field, issue, message string) *ValidationError {
	return &ValidationError{Fields: []transport.FieldError{{Field: field, Issue: issue, Message: message}}}
}

// UserResolver 是 Sticker 對 User 模組的最小依賴（依賴 DAG：
// Sticker --> User）：僅確認使用者存在。
type UserResolver interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// MediaAccess 是 Sticker 對 Media 模組的最小依賴（依賴 DAG：
// Sticker --> Media）：僅驗證檔案可被呼叫者關聯為貼圖本體或封面。
// 檔案存在、未刪除、為呼叫者所有、掃描已通過四項條件缺一不可，
// 任一不符回傳的錯誤語義由 Media 模組決定。本模組不得直讀
// media_files 表。
type MediaAccess interface {
	CheckAttachable(ctx context.Context, accessorID, fileID string) (*media.MediaFile, error)
}

// Store 是 Sticker 三張表的儲存介面。游標分頁一律以
// (created_at, id) 倒序鍵為準，唯收藏清單以 (updated_at, id)、
// 最近使用清單以 (last_used_at, id) 為鍵；游標為服務層編碼的
// 不透明字串，儲存層解碼後以元組比較分頁。
type Store interface {
	GetPack(ctx context.Context, id string) (*Pack, error)
	CreatePack(ctx context.Context, p *Pack) error
	SavePack(ctx context.Context, p *Pack) error
	// ListPacks 列出未封存貼圖包，依建立時間倒序。
	ListPacks(ctx context.Context, cursor string, limit int) ([]*Pack, error)
	// ListPacksByCreator 列出某作者的未封存貼圖包，依建立時間倒序。
	ListPacksByCreator(ctx context.Context, creatorID, cursor string, limit int) ([]*Pack, error)

	GetSticker(ctx context.Context, id string) (*Sticker, error)
	CreateSticker(ctx context.Context, s *Sticker) error
	SaveSticker(ctx context.Context, s *Sticker) error
	// ListStickersByPack 列出某包的全部貼圖（含已移除列，服務層
	// 過濾），依 position 升序。
	ListStickersByPack(ctx context.Context, packID string) ([]*Sticker, error)
	// NextStickerPosition 回傳某包下一個可用的序號（現有最大序號
	// +1，含已移除列，不得重用序號）。
	NextStickerPosition(ctx context.Context, packID string) (int, error)
	// CountActiveStickers 回傳某包生效中貼圖數。
	CountActiveStickers(ctx context.Context, packID string) (int, error)
	// ListAllStickers 列出生效中貼圖（全部包），依建立時間倒序，
	// 供搜尋掃描用。
	ListAllStickers(ctx context.Context, cursor string, limit int) ([]*Sticker, error)

	GetUserStickerByPack(ctx context.Context, userID, packID string) (*UserSticker, error)
	GetUserStickerBySticker(ctx context.Context, userID, stickerID string) (*UserSticker, error)
	SaveUserSticker(ctx context.Context, u *UserSticker) error
	DeleteUserSticker(ctx context.Context, id string) error
	// ListFavorites 列出使用者的收藏列（is_favorite 為真），依
	// 更新時間倒序。
	ListFavorites(ctx context.Context, userID, cursor string, limit int) ([]*UserSticker, error)
	// ListRecent 列出使用者的最近使用列（source 為 recent），依
	// last_used_at 倒序。
	ListRecent(ctx context.Context, userID, cursor string, limit int) ([]*UserSticker, error)
	// PruneRecent 刪除使用者超出 keep 筆的舊最近使用列（依
	// last_used_at 倒序保留最新者）。
	PruneRecent(ctx context.Context, userID string, keep int) error
}

// Service 是 Sticker 領域服務。
type Service struct {
	store Store
	users UserResolver
	media MediaAccess // 可為空：未接線時媒體關聯一律拒絕。
	now   func() time.Time
}

// NewService 建立 Service。now 為空時用 time.Now；users 為空時
// 作者存在檢查 fail-closed（回 ErrCheckerUnavailable）。
func NewService(store Store, users UserResolver, media MediaAccess, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, users: users, media: media, now: now}
}

// MaxPackNameRunes 是貼圖包名稱上限（本里程碑實作預設，待定稿覆核）。
const MaxPackNameRunes = 64

// MaxPackDescriptionRunes 是貼圖包描述上限（本里程碑實作預設，待定稿覆核）。
const MaxPackDescriptionRunes = 500

// MaxStickerNameRunes 是貼圖名稱上限（本里程碑實作預設，待定稿覆核）。
const MaxStickerNameRunes = 64

// MaxStickersPerPack 是單一貼圖包生效貼圖上限（本里程碑實作預設，待定稿覆核）。
const MaxStickersPerPack = 100

// DefaultPackLimit 是貼圖包清單的預設分頁大小。
const DefaultPackLimit = 20

// MaxPackLimit 是貼圖包清單的分頁上限。
const MaxPackLimit = 50

// DefaultRecentLimit 是最近使用清單的預設分頁大小。
const DefaultRecentLimit = 20

// MaxRecentLimit 是最近使用清單的分頁上限。
const MaxRecentLimit = 50

// MaxRecentStickers 是每位使用者保留的最近使用筆數上限（本里程碑實作預設，待定稿覆核）。
const MaxRecentStickers = 30

// MaxSearchQueryRunes 是搜尋關鍵字上限（本里程碑實作預設，待定稿覆核）。
const MaxSearchQueryRunes = 100

func validatePackName(name string) *ValidationError {
	if strings.TrimSpace(name) == "" {
		return validationError("name", "REQUIRED", "貼圖包名稱不得為空。")
	}
	if utf8.RuneCountInString(name) > MaxPackNameRunes {
		return validationError("name", "TOO_LONG", "貼圖包名稱過長。")
	}
	return nil
}

func validateCategory(c PackCategory) *ValidationError {
	switch c {
	case "", CategoryGeneral, CategoryAnimals, CategoryFood, CategoryEmotions, CategoryFestivals, CategoryCharacters, CategoryOther:
		return nil
	default:
		return validationError("category", "INVALID_VALUE", "分類須為 general、animals、food、emotions、festivals、characters 或 other。")
	}
}

func validatePackStatus(s PackStatus) *ValidationError {
	switch s {
	case PackStatusDraft, PackStatusPublished, PackStatusUnlisted:
		return nil
	default:
		return validationError("status", "INVALID_VALUE", "狀態須為 draft、published 或 unlisted（封存請用刪除端點）。")
	}
}

// checkMedia 驗證檔案可被呼叫者關聯：Media 未接線時 fail-closed；
// 查無檔案／非擁有者／已刪除轉為 ErrPackNotFound（404，不得洩漏
// 檔案存在性，沿 message 附件先例）；掃描未通過的欄位錯誤原樣回傳。
func (s *Service) checkMedia(ctx context.Context, accessorID, fileID string) error {
	if s.media == nil {
		return ErrCheckerUnavailable
	}
	if _, err := s.media.CheckAttachable(ctx, accessorID, fileID); err != nil {
		if errors.Is(err, media.ErrMediaNotFound) {
			return ErrPackNotFound
		}
		return err
	}
	return nil
}

// loadPack 讀取貼圖包列：不存在或已封存皆回 ErrPackNotFound。
func (s *Service) loadPack(ctx context.Context, packID string) (*Pack, error) {
	p, err := s.store.GetPack(ctx, packID)
	if err != nil {
		return nil, err
	}
	if p == nil || p.Status == PackStatusArchived {
		return nil, ErrPackNotFound
	}
	return p, nil
}

// hasGrant 查詢使用者是否持有某包的授權發放紀錄（source 為
// granted 的包級列）。無紀錄時回 false，不得放行付費包。
func (s *Service) hasGrant(ctx context.Context, userID string, p *Pack) (bool, error) {
	row, err := s.store.GetUserStickerByPack(ctx, userID, p.ID)
	if err != nil {
		return false, err
	}
	return row != nil && row.Source == SourceGranted, nil
}

// canView 裁決呼叫者是否可見該貼圖包：作者本人恆可見（草稿與
// 未上架亦然）；已上架包：免費者全體可見，付費者須授權發放紀錄；
// 草稿／未上架包僅作者可見。授權查詢失敗時 fail-closed。
func (s *Service) canView(ctx context.Context, viewerID string, p *Pack) (bool, error) {
	if viewerID == p.CreatorID {
		return true, nil
	}
	if p.Status != PackStatusPublished {
		return false, nil
	}
	if p.IsFree {
		return true, nil
	}
	return s.hasGrant(ctx, viewerID, p)
}

// visiblePack 讀取貼圖包並裁決可見性：不可見一律回
// ErrPackNotFound，不得區分原因。
func (s *Service) visiblePack(ctx context.Context, viewerID, packID string) (*Pack, error) {
	p, err := s.loadPack(ctx, packID)
	if err != nil {
		return nil, err
	}
	ok, err := s.canView(ctx, viewerID, p)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrPackNotFound
	}
	return p, nil
}

// PackEntry 是貼圖包查詢的一項：貼圖包列＋生效貼圖數＋呼叫者
// 收藏狀態。計數一律即時彙總，不得以快取代替。
type PackEntry struct {
	Pack          Pack
	StickerCount  int
	FavoritedByMe bool
}

func (s *Service) entryOf(ctx context.Context, viewerID string, p *Pack) (*PackEntry, error) {
	count, err := s.store.CountActiveStickers(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	favorited := false
	if viewerID != "" {
		row, err := s.store.GetUserStickerByPack(ctx, viewerID, p.ID)
		if err != nil {
			return nil, err
		}
		favorited = row != nil && row.IsFavorite
	}
	return &PackEntry{Pack: *p, StickerCount: count, FavoritedByMe: favorited}, nil
}

// CreatePack 建立貼圖包：新包一律為草稿（draft），須經 PATCH
// 上架；封面媒體可為空，有值時經 Media 模組驗證。is_free 預設
// 為真（呼叫方未指定時以建立參數為準，本方法簽名明示，不得以
// 零值偷渡付費語義）。
func (s *Service) CreatePack(ctx context.Context, creatorID, name, description string, category PackCategory, coverMediaID string, isFree bool) (*PackEntry, error) {
	if ve := validatePackName(name); ve != nil {
		return nil, ve
	}
	if utf8.RuneCountInString(description) > MaxPackDescriptionRunes {
		return nil, validationError("description", "TOO_LONG", "貼圖包描述過長。")
	}
	if ve := validateCategory(category); ve != nil {
		return nil, ve
	}
	if coverMediaID != "" {
		if err := s.checkMedia(ctx, creatorID, coverMediaID); err != nil {
			return nil, err
		}
	}
	now := s.now().UTC()
	p := &Pack{
		ID:           ids.New(),
		CreatorID:    creatorID,
		Name:         name,
		Description:  description,
		Category:     category,
		CoverMediaID: coverMediaID,
		IsFree:       isFree,
		Status:       PackStatusDraft,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if p.ID == "" {
		return nil, errors.New("sticker: 識別碼產生失敗")
	}
	if err := s.store.CreatePack(ctx, p); err != nil {
		return nil, err
	}
	return s.entryOf(ctx, creatorID, p)
}

// GetPack 讀取單一貼圖包：僅可見者可讀，不可見一律 404。
func (s *Service) GetPack(ctx context.Context, viewerID, packID string) (*PackEntry, error) {
	p, err := s.visiblePack(ctx, viewerID, packID)
	if err != nil {
		return nil, err
	}
	return s.entryOf(ctx, viewerID, p)
}

// PackPatch 是貼圖包更新的指標語義：nil 表示不改。
type PackPatch struct {
	Name         *string
	Description  *string
	Category     *PackCategory
	CoverMediaID *string
	Status       *PackStatus
	IsFree       *bool
}

// UpdatePack 編輯貼圖包：僅作者可編輯；已封存貼圖包不可編輯。
// 狀態只接受 draft／published／unlisted 三者互相轉換（封存請用
// DeletePack；已封存列不得以 PATCH 復活）。
func (s *Service) UpdatePack(ctx context.Context, userID, packID string, patch PackPatch) (*PackEntry, error) {
	p, err := s.loadPack(ctx, packID)
	if err != nil {
		return nil, err
	}
	if p.CreatorID != userID {
		return nil, ErrPackNotFound
	}
	if patch.Name != nil {
		if ve := validatePackName(*patch.Name); ve != nil {
			return nil, ve
		}
		p.Name = *patch.Name
	}
	if patch.Description != nil {
		if utf8.RuneCountInString(*patch.Description) > MaxPackDescriptionRunes {
			return nil, validationError("description", "TOO_LONG", "貼圖包描述過長。")
		}
		p.Description = *patch.Description
	}
	if patch.Category != nil {
		if ve := validateCategory(*patch.Category); ve != nil {
			return nil, ve
		}
		p.Category = *patch.Category
	}
	if patch.CoverMediaID != nil {
		if *patch.CoverMediaID != "" {
			if err := s.checkMedia(ctx, userID, *patch.CoverMediaID); err != nil {
				return nil, err
			}
		}
		p.CoverMediaID = *patch.CoverMediaID
	}
	if patch.Status != nil {
		if ve := validatePackStatus(*patch.Status); ve != nil {
			return nil, ve
		}
		p.Status = *patch.Status
	}
	if patch.IsFree != nil {
		p.IsFree = *patch.IsFree
	}
	p.UpdatedAt = s.now().UTC()
	if err := s.store.SavePack(ctx, p); err != nil {
		return nil, err
	}
	return s.entryOf(ctx, userID, p)
}

// DeletePack 刪除貼圖包：僅作者可刪除。刪除為 archived 狀態標記，
// 貼圖包列、貼圖列、收藏與授權列皆保留（不得連帶物理刪除）；刪除
// 後一切讀取端點皆以 404 回應。
func (s *Service) DeletePack(ctx context.Context, userID, packID string) error {
	p, err := s.loadPack(ctx, packID)
	if err != nil {
		return err
	}
	if p.CreatorID != userID {
		return ErrPackNotFound
	}
	p.Status = PackStatusArchived
	p.UpdatedAt = s.now().UTC()
	return s.store.SavePack(ctx, p)
}

// encodeCursor 將 (created_at, id) 排序鍵編為不透明游標。
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

func normalizePackLimit(limit int) int {
	if limit <= 0 {
		return DefaultPackLimit
	}
	if limit > MaxPackLimit {
		return MaxPackLimit
	}
	return limit
}

func normalizeRecentLimit(limit int) int {
	if limit <= 0 {
		return DefaultRecentLimit
	}
	if limit > MaxRecentLimit {
		return MaxRecentLimit
	}
	return limit
}

// searchTokens 將關鍵字正規化為 AND 比對的 token 清單：去空白、
// 小寫、空白切分、去重（沿訊息搜尋模組的查詢語義，不得以 OR 代替）。
func searchTokens(q string) []string {
	seen := map[string]bool{}
	var out []string
	for _, tok := range strings.Fields(strings.ToLower(q)) {
		if !seen[tok] {
			seen[tok] = true
			out = append(out, tok)
		}
	}
	return out
}

func matchTokens(haystack string, tokens []string) bool {
	lower := strings.ToLower(haystack)
	for _, tok := range tokens {
		if !strings.Contains(lower, tok) {
			return false
		}
	}
	return true
}

// packPage 掃描候選貼圖包並以可見性與篩選條件過濾，累積至
// limit 筆或候選耗盡：Timeline 時間線同款「掃描後過濾」作法。
// 游標語義（與 Timeline 的 feedPage 不同，見 STATUS §2af 已知限制）：
// 回傳的 nextCursor 取最後一筆回傳列的排序鍵，不得取最後一個候選
// 列——候選列未必回傳，以其為游標會使下一頁漏掉該列（實證：
// 五筆全可見、limit 2 時以候選列為游標合計只取回四筆）。每輪多取
// 一列以判斷是否還有下一頁候選；掃描游標以最後候選列推進，僅供
// 本函式內續掃使用，不得外洩為回傳游標。
func (s *Service) packPage(ctx context.Context, viewerID, cursor string, limit int, list func(ctx context.Context, cursor string, limit int) ([]*Pack, error), filter func(*Pack) bool) ([]*Pack, string, error) {
	var out []*Pack
	scanCursor := cursor
	var lastReturned string
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
			if ok && filter(p) && len(out) < limit {
				out = append(out, p)
				lastReturned = encodeCursor(p.CreatedAt, p.ID)
			}
		}
		last := candidates[len(candidates)-1]
		scanCursor = encodeCursor(last.CreatedAt, last.ID)
		if len(candidates) < want {
			return out, "", nil
		}
		if len(out) >= limit {
			return out, lastReturned, nil
		}
	}
	return out, "", nil
}

func (s *Service) packEntriesOf(ctx context.Context, viewerID string, packs []*Pack, next string) ([]*PackEntry, string, error) {
	out := make([]*PackEntry, 0, len(packs))
	for _, p := range packs {
		e, err := s.entryOf(ctx, viewerID, p)
		if err != nil {
			return nil, "", err
		}
		out = append(out, e)
	}
	return out, next, nil
}

// ListPacks 列出呼叫者可見的貼圖包：依建立時間倒序、游標分頁；
// category 與 q（名稱／描述子字串 AND 比對）為選填篩選。
func (s *Service) ListPacks(ctx context.Context, viewerID string, category PackCategory, q, cursor string, limit int) ([]*PackEntry, string, error) {
	if _, err := decodeCursor(cursor); err != nil {
		return nil, "", err
	}
	if ve := validateCategory(category); ve != nil {
		return nil, "", ve
	}
	tokens := searchTokens(q)
	limit = normalizePackLimit(limit)
	packs, next, err := s.packPage(ctx, viewerID, cursor, limit, s.store.ListPacks, func(p *Pack) bool {
		if category != "" && p.Category != category {
			return false
		}
		if len(tokens) > 0 && !matchTokens(p.Name+" "+p.Description, tokens) {
			return false
		}
		return true
	})
	if err != nil {
		return nil, "", err
	}
	return s.packEntriesOf(ctx, viewerID, packs, next)
}

// ListPacksByCreator 列出某作者的貼圖包：作者不存在時 404；僅回傳
// 呼叫者可見者（作者本人可見自己的草稿與未上架包），依建立時間
// 倒序、游標分頁。
func (s *Service) ListPacksByCreator(ctx context.Context, viewerID, creatorID, cursor string, limit int) ([]*PackEntry, string, error) {
	if _, err := decodeCursor(cursor); err != nil {
		return nil, "", err
	}
	if s.users == nil {
		return nil, "", ErrCheckerUnavailable
	}
	exists, err := s.users.Exists(ctx, creatorID)
	if err != nil {
		return nil, "", err
	}
	if !exists {
		return nil, "", ErrPackNotFound
	}
	limit = normalizePackLimit(limit)
	packs, next, err := s.packPage(ctx, viewerID, cursor, limit, func(ctx context.Context, c string, n int) ([]*Pack, error) {
		return s.store.ListPacksByCreator(ctx, creatorID, c, n)
	}, func(*Pack) bool { return true })
	if err != nil {
		return nil, "", err
	}
	return s.packEntriesOf(ctx, viewerID, packs, next)
}

// AddSticker 新增貼圖至貼圖包：僅包作者可新增；貼圖本體經 Media
// 模組驗證（存在、未刪除、為呼叫者所有、掃描已通過，缺一不可）；
// 序號由服務層接在現有最大序號之後；生效貼圖達上限時拒絕。
// 貼圖包列與貼圖列分步寫入，跨模組無交易邊界（為已知限制）。
func (s *Service) AddSticker(ctx context.Context, userID, packID, mediaFileID, name string, isAnimated bool) (*Sticker, error) {
	p, err := s.loadPack(ctx, packID)
	if err != nil {
		return nil, err
	}
	if p.CreatorID != userID {
		return nil, ErrPackNotFound
	}
	if utf8.RuneCountInString(name) > MaxStickerNameRunes {
		return nil, validationError("name", "TOO_LONG", "貼圖名稱過長。")
	}
	if mediaFileID == "" {
		return nil, validationError("media_file_id", "REQUIRED", "貼圖須關聯媒體檔案。")
	}
	count, err := s.store.CountActiveStickers(ctx, packID)
	if err != nil {
		return nil, err
	}
	if count >= MaxStickersPerPack {
		return nil, validationError("media_file_id", "TOO_LONG", "單一貼圖包最多 100 張生效貼圖。")
	}
	if err := s.checkMedia(ctx, userID, mediaFileID); err != nil {
		return nil, err
	}
	position, err := s.store.NextStickerPosition(ctx, packID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	st := &Sticker{
		ID:          ids.New(),
		PackID:      packID,
		MediaFileID: mediaFileID,
		Name:        name,
		IsAnimated:  isAnimated,
		Position:    position,
		Status:      StickerStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if st.ID == "" {
		return nil, errors.New("sticker: 識別碼產生失敗")
	}
	if err := s.store.CreateSticker(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

// StickerEntry 是貼圖查詢的一項：貼圖列＋所屬包名稱（搜尋與
// 最近使用清單顯示用）。
type StickerEntry struct {
	Sticker  Sticker
	PackName string
}

// ListStickers 列出某包的生效貼圖：僅包可見者可讀，依 position
// 升序；已移除貼圖不列出。
func (s *Service) ListStickers(ctx context.Context, viewerID, packID string) ([]StickerEntry, error) {
	p, err := s.visiblePack(ctx, viewerID, packID)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListStickersByPack(ctx, packID)
	if err != nil {
		return nil, err
	}
	out := make([]StickerEntry, 0, len(rows))
	for _, st := range rows {
		if st.Status != StickerStatusActive {
			continue
		}
		out = append(out, StickerEntry{Sticker: *st, PackName: p.Name})
	}
	return out, nil
}

// RemoveSticker 移除貼圖：僅包作者可移除。移除為狀態標記，列保留
// （已傳送訊息的參照需要歷史）；移除後讀取一律 404。
func (s *Service) RemoveSticker(ctx context.Context, userID, stickerID string) error {
	st, err := s.store.GetSticker(ctx, stickerID)
	if err != nil {
		return err
	}
	if st == nil || st.Status == StickerStatusRemoved {
		return ErrStickerNotFound
	}
	p, err := s.loadPack(ctx, st.PackID)
	if err != nil {
		return ErrStickerNotFound
	}
	if p.CreatorID != userID {
		return ErrStickerNotFound
	}
	st.Status = StickerStatusRemoved
	st.UpdatedAt = s.now().UTC()
	return s.store.SaveSticker(ctx, st)
}

// loadVisibleSticker 讀取貼圖並裁決其所屬包的可見性：貼圖不存在、
// 已移除、或包不可見時一律回 ErrStickerNotFound。
func (s *Service) loadVisibleSticker(ctx context.Context, viewerID, stickerID string) (*Sticker, *Pack, error) {
	st, err := s.store.GetSticker(ctx, stickerID)
	if err != nil {
		return nil, nil, err
	}
	if st == nil || st.Status == StickerStatusRemoved {
		return nil, nil, ErrStickerNotFound
	}
	p, err := s.visiblePack(ctx, viewerID, st.PackID)
	if err != nil {
		return nil, nil, ErrStickerNotFound
	}
	return st, p, nil
}

// FavoritePack 收藏貼圖包：僅可見者可收藏；重複收藏冪等成功。
// 已授權列（source 為 granted）收藏時保留來源、僅立 is_favorite
// 旗標；新列來源為 favorite。
func (s *Service) FavoritePack(ctx context.Context, userID, packID string) (*PackEntry, error) {
	p, err := s.visiblePack(ctx, userID, packID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	row, err := s.store.GetUserStickerByPack(ctx, userID, packID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		row = &UserSticker{
			ID:         ids.New(),
			UserID:     userID,
			PackID:     packID,
			Source:     SourceFavorite,
			IsFavorite: true,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if row.ID == "" {
			return nil, errors.New("sticker: 識別碼產生失敗")
		}
	} else {
		row.IsFavorite = true
		row.UpdatedAt = now
	}
	if err := s.store.SaveUserSticker(ctx, row); err != nil {
		return nil, err
	}
	return s.entryOf(ctx, userID, p)
}

// UnfavoritePack 取消收藏：貼圖包列須存在（任何狀態皆可取消，
// 含已封存包，屬清理操作）；未收藏時冪等成功。來源為 favorite
// 的列取消後硬刪除（ERD 草案）；來源為 granted 的列保留、僅清
// is_favorite 旗標（授權紀錄不得因取消收藏而遺失）。回應：曾有
// 收藏列者回包檢視（呼叫者先前已知此包，不構成洩漏）；從未收藏
// 且包不可見者一律 404，不得洩漏存在性。
func (s *Service) UnfavoritePack(ctx context.Context, userID, packID string) (*PackEntry, error) {
	p, err := s.store.GetPack(ctx, packID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPackNotFound
	}
	row, err := s.store.GetUserStickerByPack(ctx, userID, packID)
	if err != nil {
		return nil, err
	}
	hadFavorite := row != nil && row.IsFavorite
	if hadFavorite {
		now := s.now().UTC()
		if row.Source == SourceGranted {
			row.IsFavorite = false
			row.UpdatedAt = now
			if err := s.store.SaveUserSticker(ctx, row); err != nil {
				return nil, err
			}
		} else {
			if err := s.store.DeleteUserSticker(ctx, row.ID); err != nil {
				return nil, err
			}
		}
		return s.entryOf(ctx, userID, p)
	}
	visible, err := s.visiblePack(ctx, userID, packID)
	if err != nil {
		return nil, err
	}
	return s.entryOf(ctx, userID, visible)
}

// ListFavorites 列出使用者的收藏貼圖包：依收藏更新時間倒序、游標
// 分頁；包已封存或不再可見者靜默略過（掃描後過濾）。回傳游標取
// 最後一筆回傳列的 (updated_at, id) 鍵（見 packPage 游標語義）。
func (s *Service) ListFavorites(ctx context.Context, userID, cursor string, limit int) ([]*PackEntry, string, error) {
	if _, err := decodeCursor(cursor); err != nil {
		return nil, "", err
	}
	limit = normalizePackLimit(limit)
	var out []*PackEntry
	scanCursor := cursor
	var lastReturned string
	for len(out) < limit {
		want := limit - len(out) + 1
		rows, err := s.store.ListFavorites(ctx, userID, scanCursor, want)
		if err != nil {
			return nil, "", err
		}
		if len(rows) == 0 {
			return out, "", nil
		}
		for _, row := range rows {
			p, err := s.visiblePack(ctx, userID, row.PackID)
			if err != nil {
				if errors.Is(err, ErrPackNotFound) {
					continue
				}
				return nil, "", err
			}
			if len(out) < limit {
				e, err := s.entryOf(ctx, userID, p)
				if err != nil {
					return nil, "", err
				}
				out = append(out, e)
				lastReturned = encodeCursor(row.UpdatedAt, row.ID)
			}
		}
		last := rows[len(rows)-1]
		scanCursor = encodeCursor(last.UpdatedAt, last.ID)
		if len(rows) < want {
			return out, "", nil
		}
		if len(out) >= limit {
			return out, lastReturned, nil
		}
	}
	return out, "", nil
}

// GrantPack 授權發放貼圖包（供 Payment 模組付款完成後呼叫，無
// HTTP 端點）：為使用者建立或保留包級授權列（source 為 granted）。
// 已收藏列授權時保留 is_favorite 旗標、來源改為 granted；重複
// 發放冪等成功。付費包的可見性以此列為準，fail-closed。
func (s *Service) GrantPack(ctx context.Context, userID, packID string) error {
	p, err := s.store.GetPack(ctx, packID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrPackNotFound
	}
	now := s.now().UTC()
	row, err := s.store.GetUserStickerByPack(ctx, userID, packID)
	if err != nil {
		return err
	}
	if row == nil {
		row = &UserSticker{
			ID:        ids.New(),
			UserID:    userID,
			PackID:    packID,
			Source:    SourceGranted,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if row.ID == "" {
			return errors.New("sticker: 識別碼產生失敗")
		}
	} else {
		row.Source = SourceGranted
		row.UpdatedAt = now
	}
	return s.store.SaveUserSticker(ctx, row)
}

// RevokeGrant 撤回授權發放（供 Payment 模組退款／撤銷後呼叫，無
// HTTP 端點）：授權列撤回時，已收藏者來源改回 favorite、保留
// is_favorite 旗標；未收藏者硬刪除列。最近使用列不受影響。
func (s *Service) RevokeGrant(ctx context.Context, userID, packID string) error {
	row, err := s.store.GetUserStickerByPack(ctx, userID, packID)
	if err != nil {
		return err
	}
	if row == nil || row.Source != SourceGranted {
		return nil
	}
	if row.IsFavorite {
		row.Source = SourceFavorite
		row.UpdatedAt = s.now().UTC()
		return s.store.SaveUserSticker(ctx, row)
	}
	return s.store.DeleteUserSticker(ctx, row.ID)
}

// RecordStickerUse 記錄最近使用：僅包可見者可記錄；同張貼圖重複
// 記錄時更新 last_used_at（去重）；每位使用者保留最新 30 筆，舊列
// 由儲存層裁剪。
func (s *Service) RecordStickerUse(ctx context.Context, userID, stickerID string) (*StickerEntry, error) {
	st, p, err := s.loadVisibleSticker(ctx, userID, stickerID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	row, err := s.store.GetUserStickerBySticker(ctx, userID, stickerID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		row = &UserSticker{
			ID:         ids.New(),
			UserID:     userID,
			StickerID:  stickerID,
			Source:     SourceRecent,
			LastUsedAt: &now,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if row.ID == "" {
			return nil, errors.New("sticker: 識別碼產生失敗")
		}
	} else {
		row.Source = SourceRecent
		row.LastUsedAt = &now
		row.UpdatedAt = now
	}
	if err := s.store.SaveUserSticker(ctx, row); err != nil {
		return nil, err
	}
	if err := s.store.PruneRecent(ctx, userID, MaxRecentStickers); err != nil {
		return nil, err
	}
	return &StickerEntry{Sticker: *st, PackName: p.Name}, nil
}

// ListRecentStickers 列出使用者的最近使用貼圖：依 last_used_at
// 倒序、游標分頁；貼圖已移除或包不再可見者靜默略過。回傳游標取
// 最後一筆回傳列的 (last_used_at, id) 鍵（見 packPage 游標語義）。
func (s *Service) ListRecentStickers(ctx context.Context, userID, cursor string, limit int) ([]*StickerEntry, string, error) {
	if _, err := decodeCursor(cursor); err != nil {
		return nil, "", err
	}
	limit = normalizeRecentLimit(limit)
	var out []*StickerEntry
	scanCursor := cursor
	var lastReturned string
	for len(out) < limit {
		want := limit - len(out) + 1
		rows, err := s.store.ListRecent(ctx, userID, scanCursor, want)
		if err != nil {
			return nil, "", err
		}
		if len(rows) == 0 {
			return out, "", nil
		}
		for _, row := range rows {
			if row.LastUsedAt == nil {
				continue
			}
			st, p, err := s.loadVisibleSticker(ctx, userID, row.StickerID)
			if err != nil {
				if errors.Is(err, ErrStickerNotFound) {
					continue
				}
				return nil, "", err
			}
			if len(out) < limit {
				cp := *st
				out = append(out, &StickerEntry{Sticker: cp, PackName: p.Name})
				lastReturned = encodeCursor(*row.LastUsedAt, row.ID)
			}
		}
		last := rows[len(rows)-1]
		lastTime := last.UpdatedAt
		if last.LastUsedAt != nil {
			lastTime = *last.LastUsedAt
		}
		scanCursor = encodeCursor(lastTime, last.ID)
		if len(rows) < want {
			return out, "", nil
		}
		if len(out) >= limit {
			return out, lastReturned, nil
		}
	}
	return out, "", nil
}

// SearchStickers 搜尋貼圖：q 必填（1–100 runes），以空白切分
// token 做 AND 子字串比對（沿訊息搜尋模組語義，含 CJK 子字串）；
// token 同時比對貼圖名稱與所屬包名稱的聯集（單一 haystack，
// 非逐欄 AND）；category 選填篩選包分類；僅回傳包可見者的貼圖，
// 依建立時間倒序、游標分頁。回傳游標取最後一筆回傳列的排序鍵
// （見 packPage 游標語義）。掃描後過濾（與時間線同款限制）。
func (s *Service) SearchStickers(ctx context.Context, viewerID, q string, category PackCategory, cursor string, limit int) ([]*StickerEntry, string, error) {
	if _, err := decodeCursor(cursor); err != nil {
		return nil, "", err
	}
	if ve := validateCategory(category); ve != nil {
		return nil, "", ve
	}
	if strings.TrimSpace(q) == "" {
		return nil, "", validationError("q", "REQUIRED", "搜尋關鍵字不得為空。")
	}
	if utf8.RuneCountInString(q) > MaxSearchQueryRunes {
		return nil, "", validationError("q", "TOO_LONG", "搜尋關鍵字過長。")
	}
	tokens := searchTokens(q)
	limit = normalizePackLimit(limit)
	var out []*StickerEntry
	scanCursor := cursor
	var lastReturned string
	for len(out) < limit {
		want := limit - len(out) + 1
		candidates, err := s.store.ListAllStickers(ctx, scanCursor, want)
		if err != nil {
			return nil, "", err
		}
		if len(candidates) == 0 {
			return out, "", nil
		}
		for _, st := range candidates {
			p, err := s.visiblePack(ctx, viewerID, st.PackID)
			if err != nil {
				if errors.Is(err, ErrPackNotFound) {
					continue
				}
				return nil, "", err
			}
			if category != "" && p.Category != category {
				continue
			}
			if !matchTokens(st.Name+" "+p.Name, tokens) {
				continue
			}
			if len(out) < limit {
				cp := *st
				out = append(out, &StickerEntry{Sticker: cp, PackName: p.Name})
				lastReturned = encodeCursor(st.CreatedAt, st.ID)
			}
		}
		last := candidates[len(candidates)-1]
		scanCursor = encodeCursor(last.CreatedAt, last.ID)
		if len(candidates) < want {
			return out, "", nil
		}
		if len(out) >= limit {
			return out, lastReturned, nil
		}
	}
	return out, "", nil
}

// MemoryStore 是 Store 的記憶體實作（開發與測試用）。
type MemoryStore struct {
	mu           sync.RWMutex
	packs        map[string]*Pack
	stickers     map[string]*Sticker
	userStickers map[string]*UserSticker
}

// NewMemoryStore 建立空的記憶體貼圖儲存。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		packs:        map[string]*Pack{},
		stickers:     map[string]*Sticker{},
		userStickers: map[string]*UserSticker{},
	}
}

func clonePack(p *Pack) *Pack {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

func cloneSticker(st *Sticker) *Sticker {
	if st == nil {
		return nil
	}
	cp := *st
	return &cp
}

func cloneUserSticker(u *UserSticker) *UserSticker {
	if u == nil {
		return nil
	}
	cp := *u
	if u.LastUsedAt != nil {
		t := *u.LastUsedAt
		cp.LastUsedAt = &t
	}
	return &cp
}

// cursorBound 解析游標為 (time, id) 邊界：游標為空時回零值
// （取全部）；格式錯誤時回錯誤（儲存層不得靜默接受壞游標）。
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
		return time.Time{}, "", errors.New("sticker: 游標格式錯誤")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", err
	}
	return t, parts[1], nil
}

// afterCursorDesc 判斷列是否在倒序游標之後（排序鍵小於游標鍵者
// 為之後）。游標為空時一律為真。
func afterCursorDesc(createdAt time.Time, id string, bound time.Time, boundID string) bool {
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

func sortPacksDesc(packs []*Pack) {
	for i := 0; i < len(packs); i++ {
		for j := i + 1; j < len(packs); j++ {
			if packs[j].CreatedAt.After(packs[i].CreatedAt) ||
				(packs[j].CreatedAt.Equal(packs[i].CreatedAt) && packs[j].ID > packs[i].ID) {
				packs[i], packs[j] = packs[j], packs[i]
			}
		}
	}
}

func (s *MemoryStore) GetPack(_ context.Context, id string) (*Pack, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return clonePack(s.packs[id]), nil
}

func (s *MemoryStore) CreatePack(_ context.Context, p *Pack) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.packs[p.ID]; exists {
		return errors.New("sticker: pack 已存在")
	}
	s.packs[p.ID] = clonePack(p)
	return nil
}

func (s *MemoryStore) SavePack(_ context.Context, p *Pack) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.packs[p.ID]; !exists {
		return ErrPackNotFound
	}
	s.packs[p.ID] = clonePack(p)
	return nil
}

func (s *MemoryStore) ListPacks(_ context.Context, cursor string, limit int) ([]*Pack, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Pack
	for _, p := range s.packs {
		if p.Status == PackStatusArchived {
			continue
		}
		if afterCursorDesc(p.CreatedAt, p.ID, bound, boundID) {
			out = append(out, clonePack(p))
		}
	}
	sortPacksDesc(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) ListPacksByCreator(_ context.Context, creatorID, cursor string, limit int) ([]*Pack, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Pack
	for _, p := range s.packs {
		if p.Status == PackStatusArchived || p.CreatorID != creatorID {
			continue
		}
		if afterCursorDesc(p.CreatedAt, p.ID, bound, boundID) {
			out = append(out, clonePack(p))
		}
	}
	sortPacksDesc(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) GetSticker(_ context.Context, id string) (*Sticker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSticker(s.stickers[id]), nil
}

func (s *MemoryStore) CreateSticker(_ context.Context, st *Sticker) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.stickers[st.ID]; exists {
		return errors.New("sticker: sticker 已存在")
	}
	s.stickers[st.ID] = cloneSticker(st)
	return nil
}

func (s *MemoryStore) SaveSticker(_ context.Context, st *Sticker) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.stickers[st.ID]; !exists {
		return ErrStickerNotFound
	}
	s.stickers[st.ID] = cloneSticker(st)
	return nil
}

func (s *MemoryStore) ListStickersByPack(_ context.Context, packID string) ([]*Sticker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Sticker
	for _, st := range s.stickers {
		if st.PackID == packID {
			out = append(out, cloneSticker(st))
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Position < out[i].Position ||
				(out[j].Position == out[i].Position && out[j].ID < out[i].ID) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (s *MemoryStore) NextStickerPosition(_ context.Context, packID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	max := -1
	for _, st := range s.stickers {
		if st.PackID == packID && st.Position > max {
			max = st.Position
		}
	}
	return max + 1, nil
}

func (s *MemoryStore) CountActiveStickers(_ context.Context, packID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, st := range s.stickers {
		if st.PackID == packID && st.Status == StickerStatusActive {
			n++
		}
	}
	return n, nil
}

func (s *MemoryStore) ListAllStickers(_ context.Context, cursor string, limit int) ([]*Sticker, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Sticker
	for _, st := range s.stickers {
		if st.Status != StickerStatusActive {
			continue
		}
		if afterCursorDesc(st.CreatedAt, st.ID, bound, boundID) {
			out = append(out, cloneSticker(st))
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.After(out[i].CreatedAt) ||
				(out[j].CreatedAt.Equal(out[i].CreatedAt) && out[j].ID > out[i].ID) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) GetUserStickerByPack(_ context.Context, userID, packID string) (*UserSticker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.userStickers {
		if u.UserID == userID && u.PackID == packID {
			return cloneUserSticker(u), nil
		}
	}
	return nil, nil
}

func (s *MemoryStore) GetUserStickerBySticker(_ context.Context, userID, stickerID string) (*UserSticker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.userStickers {
		if u.UserID == userID && u.StickerID == stickerID {
			return cloneUserSticker(u), nil
		}
	}
	return nil, nil
}

func (s *MemoryStore) SaveUserSticker(_ context.Context, u *UserSticker) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u.ID == "" {
		return errors.New("sticker: 識別碼不得為空")
	}
	// 同 (user, pack) 或 (user, sticker) 的舊列以新列取代（upsert
	// 語義，與 PG ON CONFLICT 對映）；舊列識別碼不同時刪除舊列。
	for id, old := range s.userStickers {
		if old.UserID != u.UserID {
			continue
		}
		if u.PackID != "" && old.PackID == u.PackID && id != u.ID {
			delete(s.userStickers, id)
		}
		if u.StickerID != "" && old.StickerID == u.StickerID && id != u.ID {
			delete(s.userStickers, id)
		}
	}
	s.userStickers[u.ID] = cloneUserSticker(u)
	return nil
}

func (s *MemoryStore) DeleteUserSticker(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.userStickers, id)
	return nil
}

func (s *MemoryStore) ListFavorites(_ context.Context, userID, cursor string, limit int) ([]*UserSticker, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*UserSticker
	for _, u := range s.userStickers {
		if u.UserID != userID || !u.IsFavorite || u.PackID == "" {
			continue
		}
		if afterCursorDesc(u.UpdatedAt, u.ID, bound, boundID) {
			out = append(out, cloneUserSticker(u))
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].UpdatedAt.After(out[i].UpdatedAt) ||
				(out[j].UpdatedAt.Equal(out[i].UpdatedAt) && out[j].ID > out[i].ID) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) ListRecent(_ context.Context, userID, cursor string, limit int) ([]*UserSticker, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*UserSticker
	for _, u := range s.userStickers {
		if u.UserID != userID || u.Source != SourceRecent || u.LastUsedAt == nil {
			continue
		}
		if afterCursorDesc(*u.LastUsedAt, u.ID, bound, boundID) {
			out = append(out, cloneUserSticker(u))
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].LastUsedAt.After(*out[i].LastUsedAt) ||
				(out[j].LastUsedAt.Equal(*out[i].LastUsedAt) && out[j].ID > out[i].ID) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) PruneRecent(_ context.Context, userID string, keep int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var rows []*UserSticker
	for _, u := range s.userStickers {
		if u.UserID == userID && u.Source == SourceRecent && u.LastUsedAt != nil {
			rows = append(rows, u)
		}
	}
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].LastUsedAt.After(*rows[i].LastUsedAt) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	if len(rows) <= keep {
		return nil
	}
	for _, u := range rows[keep:] {
		delete(s.userStickers, u.ID)
	}
	return nil
}
