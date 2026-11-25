// Package timeline 的 HTTP handler：/api/v1/posts・/api/v1/timeline・
// /api/v1/comments 前綴下的動態端點（API_STANDARDS.md §2，
// Timeline 擁有 /api/v1/posts 前綴）。全部端點皆須經 RequireAuth；
// 不可見的貼文／留言一律以 404 回應，不得洩漏存在性
// （ERROR_CODES.md §6.3）。動態無對應 WebSocket 事件名（§21
// 的 15 個事件皆為訊息事件），本里程碑不寫 Outbox 事件，客戶端
// 以 REST 查詢同步。
package timeline

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"ether.local/api/internal/transport"
)

// Handler 持有動態端點所需的服務。
type Handler struct {
	Service     *Service
	CurrentUser func(r *http.Request) (string, bool)
}

func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := h.CurrentUser(r)
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return "", false
	}
	return userID, true
}

// writeTimelineError 將領域錯誤對應為 HTTP 回應：欄位驗證錯誤 →
// 400＋details；ErrPostNotFound／ErrCommentNotFound → 404（訊息
// 不可區分）；ErrCheckerUnavailable → 503 fail-closed；其餘 → 500。
func writeTimelineError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrCheckerUnavailable) {
		transport.WriteError(w, http.StatusServiceUnavailable,
			"ETHER_COMMON_UNAVAILABLE", "服務暫時無法使用。")
		return
	}
	transport.WriteServiceError(w, err, ErrPostNotFound)
}

func utcTime(t time.Time) time.Time { return t.UTC() }

func utcTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

type postMediaResponse struct {
	MediaFileID string `json:"media_file_id"`
	Position    int    `json:"position"`
	Caption     string `json:"caption"`
}

type postResponse struct {
	PostID       string              `json:"post_id"`
	AuthorID     string              `json:"author_id"`
	Content      string              `json:"content"`
	Visibility   string              `json:"visibility"`
	EditedAt     *time.Time          `json:"edited_at,omitempty"`
	Media        []postMediaResponse `json:"media"`
	LikeCount    int                 `json:"like_count"`
	CommentCount int                 `json:"comment_count"`
	LikedByMe    bool                `json:"liked_by_me"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

func toPostResponse(entry *PostEntry) postResponse {
	mediaOut := make([]postMediaResponse, 0, len(entry.Media))
	for _, m := range entry.Media {
		mediaOut = append(mediaOut, postMediaResponse{
			MediaFileID: m.MediaFileID,
			Position:    m.Position,
			Caption:     m.Caption,
		})
	}
	return postResponse{
		PostID:       entry.Post.ID,
		AuthorID:     entry.Post.AuthorID,
		Content:      entry.Post.Content,
		Visibility:   string(entry.Post.Visibility),
		EditedAt:     utcTimePtr(entry.Post.EditedAt),
		Media:        mediaOut,
		LikeCount:    entry.LikeCount,
		CommentCount: entry.CommentCount,
		LikedByMe:    entry.LikedByMe,
		CreatedAt:    utcTime(entry.Post.CreatedAt),
		UpdatedAt:    utcTime(entry.Post.UpdatedAt),
	}
}

type createPostBody struct {
	Content      string   `json:"content"`
	Visibility   string   `json:"visibility"`
	MediaFileIDs []string `json:"media_file_ids"`
}

// CreatePost 建立貼文（POST /api/v1/posts）。
func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req createPostBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	entry, err := h.Service.CreatePost(r.Context(), userID, req.Content, Visibility(req.Visibility), req.MediaFileIDs)
	if err != nil {
		writeTimelineError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusCreated, toPostResponse(entry))
}

// GetPost 讀取單一貼文（GET /api/v1/posts/{post_id}）。
func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entry, err := h.Service.GetPost(r.Context(), userID, r.PathValue("post_id"))
	if err != nil {
		writeTimelineError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toPostResponse(entry))
}

type updatePostBody struct {
	Content    *string `json:"content"`
	Visibility *string `json:"visibility"`
}

// UpdatePost 編輯貼文（PATCH /api/v1/posts/{post_id}）。
func (h *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req updatePostBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	patch := PostPatch{Content: req.Content}
	if req.Visibility != nil {
		v := Visibility(*req.Visibility)
		patch.Visibility = &v
	}
	entry, err := h.Service.UpdatePost(r.Context(), userID, r.PathValue("post_id"), patch)
	if err != nil {
		writeTimelineError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toPostResponse(entry))
}

// DeletePost 刪除貼文（POST /api/v1/posts/{post_id}/delete）。
// 刪除為狀態標記，刪除後一切讀取皆以 404 回應。
func (h *Handler) DeletePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if err := h.Service.DeletePost(r.Context(), userID, r.PathValue("post_id")); err != nil {
		writeTimelineError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// parseLimit 解析 limit 查詢參數：缺省為 0（由服務層取預設值）；
// 非數字或負數為欄位錯誤。
func parseLimit(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, errors.New("limit 須為非負整數")
	}
	return n, nil
}

type postListResponse struct {
	Posts      []postResponse `json:"posts"`
	NextCursor string         `json:"next_cursor"`
}

// ListAuthorPosts 列出某作者的貼文（GET /api/v1/posts?author_id=…）。
func (h *Handler) ListAuthorPosts(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	authorID := r.URL.Query().Get("author_id")
	if authorID == "" {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請提供 author_id。")
		return
	}
	limit, err := parseLimit(r)
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "limit 參數錯誤。")
		return
	}
	entries, next, err := h.Service.ListAuthorPosts(r.Context(), userID, authorID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeTimelineError(w, err)
		return
	}
	out := make([]postResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toPostResponse(e))
	}
	transport.WriteJSON(w, http.StatusOK, postListResponse{Posts: out, NextCursor: next})
}

// ListFeed 列出呼叫者的好友時間線（GET /api/v1/timeline）。
func (h *Handler) ListFeed(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	limit, err := parseLimit(r)
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "limit 參數錯誤。")
		return
	}
	entries, next, err := h.Service.ListFeed(r.Context(), userID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeTimelineError(w, err)
		return
	}
	out := make([]postResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toPostResponse(e))
	}
	transport.WriteJSON(w, http.StatusOK, postListResponse{Posts: out, NextCursor: next})
}

// ListPostMedia 列出貼文的媒體關聯（GET /api/v1/posts/{post_id}/media）。
func (h *Handler) ListPostMedia(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	rows, err := h.Service.ListPostMedia(r.Context(), userID, r.PathValue("post_id"))
	if err != nil {
		writeTimelineError(w, err)
		return
	}
	out := make([]postMediaResponse, 0, len(rows))
	for _, m := range rows {
		out = append(out, postMediaResponse{MediaFileID: m.MediaFileID, Position: m.Position, Caption: m.Caption})
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type likeResponse struct {
	PostID    string `json:"post_id"`
	Liked     bool   `json:"liked"`
	LikeCount int    `json:"like_count"`
}

// LikePost 按讚（POST /api/v1/posts/{post_id}/likes）。重複按讚冪等成功。
func (h *Handler) LikePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	liked, count, err := h.Service.LikePost(r.Context(), userID, r.PathValue("post_id"))
	if err != nil {
		writeTimelineError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, likeResponse{PostID: r.PathValue("post_id"), Liked: liked, LikeCount: count})
}

// UnlikePost 取消按讚（DELETE /api/v1/posts/{post_id}/likes）。未按讚時冪等成功。
func (h *Handler) UnlikePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	liked, count, err := h.Service.UnlikePost(r.Context(), userID, r.PathValue("post_id"))
	if err != nil {
		writeTimelineError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, likeResponse{PostID: r.PathValue("post_id"), Liked: liked, LikeCount: count})
}

type commentResponse struct {
	CommentID       string     `json:"comment_id"`
	PostID          string     `json:"post_id"`
	AuthorID        string     `json:"author_id"`
	ParentCommentID string     `json:"parent_comment_id,omitempty"`
	Content         string     `json:"content"`
	IsDeleted       bool       `json:"is_deleted"`
	EditedAt        *time.Time `json:"edited_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func toCommentResponse(c *Comment) commentResponse {
	return commentResponse{
		CommentID:       c.ID,
		PostID:          c.PostID,
		AuthorID:        c.AuthorID,
		ParentCommentID: c.ParentCommentID,
		Content:         c.Content,
		IsDeleted:       c.IsDeleted,
		EditedAt:        utcTimePtr(c.EditedAt),
		CreatedAt:       utcTime(c.CreatedAt),
		UpdatedAt:       utcTime(c.UpdatedAt),
	}
}

type createCommentBody struct {
	Content         string `json:"content"`
	ParentCommentID string `json:"parent_comment_id"`
}

// CreateComment 建立留言（POST /api/v1/posts/{post_id}/comments）。
func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req createCommentBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	c, err := h.Service.CreateComment(r.Context(), userID, r.PathValue("post_id"), req.Content, req.ParentCommentID)
	if err != nil {
		writeTimelineError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusCreated, toCommentResponse(c))
}

type commentListResponse struct {
	Comments   []commentResponse `json:"comments"`
	NextCursor string            `json:"next_cursor"`
}

// ListComments 列出貼文留言（GET /api/v1/posts/{post_id}/comments）。
func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	limit, err := parseLimit(r)
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "limit 參數錯誤。")
		return
	}
	rows, next, err := h.Service.ListComments(r.Context(), userID, r.PathValue("post_id"), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeTimelineError(w, err)
		return
	}
	out := make([]commentResponse, 0, len(rows))
	for _, c := range rows {
		out = append(out, toCommentResponse(c))
	}
	transport.WriteJSON(w, http.StatusOK, commentListResponse{Comments: out, NextCursor: next})
}

// DeleteComment 刪除留言（POST /api/v1/comments/{comment_id}/delete）。
// 刪除為狀態標記，子留言保留。
func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if err := h.Service.DeleteComment(r.Context(), userID, r.PathValue("comment_id")); err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			transport.WriteError(w, http.StatusNotFound, "ETHER_COMMON_NOT_FOUND", "找不到資源。")
			return
		}
		writeTimelineError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
