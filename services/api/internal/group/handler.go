// Package group 的 HTTP handler：/api/v1/groups 前綴下的群組端點
// （API_STANDARDS.md §2）。全部端點皆須經 RequireAuth；非成員的
// 一切查詢皆以 404 回應，不得洩漏群組存在性（ERROR_CODES.md §6.3）。
package group

import (
	"errors"
	"net/http"
	"time"

	"ether.local/api/internal/transport"
)

// Handler 持有 groups 端點所需的服務。
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

// writeGroupError 將領域錯誤對應為 HTTP 回應：欄位驗證錯誤 →
// 400＋details；ErrGroupNotFound → 404（訊息不可區分）；其餘 → 500。
func writeGroupError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrGroupNotFound) {
		transport.WriteError(w, http.StatusNotFound, "ETHER_COMMON_NOT_FOUND", "找不到資源。")
		return
	}
	transport.WriteServiceError(w, err, nil)
}

func utcTime(t time.Time) time.Time { return t.UTC() }

func utcTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

type groupResponse struct {
	GroupID        string     `json:"group_id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	AvatarMediaID  string     `json:"avatar_media_id"`
	ConversationID string     `json:"conversation_id,omitempty"`
	CreatedBy      string     `json:"created_by"`
	IsDisbanded    bool       `json:"is_disbanded"`
	DisbandedAt    *time.Time `json:"disbanded_at,omitempty"`
	MemberCount    int        `json:"member_count"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func toGroupResponse(entry GroupEntry) groupResponse {
	return groupResponse{
		GroupID:        entry.Group.ID,
		Name:           entry.Group.Name,
		Description:    entry.Group.Description,
		AvatarMediaID:  entry.Group.AvatarMediaID,
		ConversationID: entry.Group.ConversationID,
		CreatedBy:      entry.Group.CreatedBy,
		IsDisbanded:    entry.Group.IsDisbanded,
		DisbandedAt:    utcTimePtr(entry.Group.DisbandedAt),
		MemberCount:    entry.MemberCount,
		CreatedAt:      utcTime(entry.Group.CreatedAt),
		UpdatedAt:      utcTime(entry.Group.UpdatedAt),
	}
}

// ListGroups 列出呼叫者的群組（GET /api/v1/groups）。
func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entries, err := h.Service.ListGroups(r.Context(), userID)
	if err != nil {
		writeGroupError(w, err)
		return
	}
	out := make([]groupResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toGroupResponse(e))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type createGroupBody struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	AvatarMediaID string   `json:"avatar_media_id"`
	MemberIDs     []string `json:"member_ids"`
}

// CreateGroup 建立群組（POST /api/v1/groups）。建立者為擁有者。
func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req createGroupBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	g, err := h.Service.CreateGroup(r.Context(), userID, req.Name, req.Description, req.AvatarMediaID, req.MemberIDs)
	if err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusCreated, toGroupResponse(*g))
}

// GetGroup 讀取單一群組（GET /api/v1/groups/{group_id}）。
func (h *Handler) GetGroup(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entry, err := h.Service.GetGroup(r.Context(), userID, r.PathValue("group_id"))
	if err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toGroupResponse(*entry))
}

type updateGroupBody struct {
	Name          *string `json:"name"`
	Description   *string `json:"description"`
	AvatarMediaID *string `json:"avatar_media_id"`
}

// UpdateGroup 更新群組資料（PATCH /api/v1/groups/{group_id}）。
func (h *Handler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req updateGroupBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	g, err := h.Service.UpdateGroup(r.Context(), userID, r.PathValue("group_id"), GroupPatch{
		Name:          req.Name,
		Description:   req.Description,
		AvatarMediaID: req.AvatarMediaID,
	})
	if err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toGroupResponse(*g))
}

// Disband 解散群組（POST /api/v1/groups/{group_id}/disband）。
func (h *Handler) Disband(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if err := h.Service.Disband(r.Context(), userID, r.PathValue("group_id")); err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "disbanded"})
}

type transferBody struct {
	NewOwnerID string `json:"new_owner_id"`
}

// TransferOwnership 轉移所有權（POST /api/v1/groups/{group_id}/transfer）。
func (h *Handler) TransferOwnership(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req transferBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	g, err := h.Service.TransferOwnership(r.Context(), userID, r.PathValue("group_id"), req.NewOwnerID)
	if err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toGroupResponse(*g))
}

type memberResponse struct {
	UserID    string     `json:"user_id"`
	Role      string     `json:"role"`
	JoinedAt  time.Time  `json:"joined_at"`
	InvitedBy string     `json:"invited_by"`
	LeftAt    *time.Time `json:"left_at,omitempty"`
}

func toMemberResponse(m *Member) memberResponse {
	return memberResponse{
		UserID:    m.UserID,
		Role:      string(m.Role),
		JoinedAt:  utcTime(m.JoinedAt),
		InvitedBy: m.InvitedBy,
		LeftAt:    utcTimePtr(m.LeftAt),
	}
}

// ListMembers 列出群組成員（GET /api/v1/groups/{group_id}/members）。
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	members, err := h.Service.ListMembers(r.Context(), userID, r.PathValue("group_id"))
	if err != nil {
		writeGroupError(w, err)
		return
	}
	out := make([]memberResponse, 0, len(members))
	for _, m := range members {
		out = append(out, toMemberResponse(m))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type addMembersBody struct {
	MemberIDs []string `json:"member_ids"`
}

// AddMembers 直接加入成員（POST /api/v1/groups/{group_id}/members）。
func (h *Handler) AddMembers(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req addMembersBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	added, err := h.Service.AddMembers(r.Context(), userID, r.PathValue("group_id"), req.MemberIDs)
	if err != nil {
		writeGroupError(w, err)
		return
	}
	out := make([]memberResponse, 0, len(added))
	for _, m := range added {
		out = append(out, toMemberResponse(m))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type setRoleBody struct {
	Role string `json:"role"`
}

// SetRole 變更成員角色（PATCH /api/v1/groups/{group_id}/members/{user_id}）。
func (h *Handler) SetRole(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req setRoleBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	m, err := h.Service.SetRole(r.Context(), userID, r.PathValue("group_id"), r.PathValue("user_id"), Role(req.Role))
	if err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toMemberResponse(m))
}

// RemoveMember 移除成員（POST /api/v1/groups/{group_id}/members/{user_id}/remove）。
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if err := h.Service.RemoveMember(r.Context(), userID, r.PathValue("group_id"), r.PathValue("user_id")); err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// Leave 離開群組（POST /api/v1/groups/{group_id}/leave）。
func (h *Handler) Leave(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if err := h.Service.Leave(r.Context(), userID, r.PathValue("group_id")); err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "left"})
}

type invitationResponse struct {
	InvitationID string     `json:"invitation_id"`
	GroupID      string     `json:"group_id"`
	GroupName    string     `json:"group_name,omitempty"`
	InviterID    string     `json:"inviter_id"`
	InviteeID    string     `json:"invitee_id"`
	Status       string     `json:"status"`
	RespondedAt  *time.Time `json:"responded_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func toInvitationResponse(entry InvitationEntry) invitationResponse {
	return invitationResponse{
		InvitationID: entry.Invitation.ID,
		GroupID:      entry.Invitation.GroupID,
		GroupName:    entry.GroupName,
		InviterID:    entry.Invitation.InviterID,
		InviteeID:    entry.Invitation.InviteeID,
		Status:       string(entry.Invitation.Status),
		RespondedAt:  utcTimePtr(entry.Invitation.RespondedAt),
		CreatedAt:    utcTime(entry.Invitation.CreatedAt),
		UpdatedAt:    utcTime(entry.Invitation.UpdatedAt),
	}
}

// ListGroupInvitations 列出群組的待處理邀請（GET /api/v1/groups/{group_id}/invitations）。
func (h *Handler) ListGroupInvitations(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entries, err := h.Service.ListGroupInvitations(r.Context(), userID, r.PathValue("group_id"))
	if err != nil {
		writeGroupError(w, err)
		return
	}
	out := make([]invitationResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toInvitationResponse(e))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type createInvitationBody struct {
	InviteeID string `json:"invitee_id"`
}

// CreateInvitation 建立邀請（POST /api/v1/groups/{group_id}/invitations）。
func (h *Handler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req createInvitationBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	entry, err := h.Service.CreateInvitation(r.Context(), userID, r.PathValue("group_id"), req.InviteeID)
	if err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusCreated, toInvitationResponse(*entry))
}

// ListMyInvitations 列出呼叫者收到的待處理邀請（GET /api/v1/groups/invitations）。
func (h *Handler) ListMyInvitations(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entries, err := h.Service.ListMyInvitations(r.Context(), userID)
	if err != nil {
		writeGroupError(w, err)
		return
	}
	out := make([]invitationResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toInvitationResponse(e))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

// respondInvitation 回應邀請（accept／decline 共用）。
func (h *Handler) respondInvitation(w http.ResponseWriter, r *http.Request, accept bool) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entry, err := h.Service.RespondInvitation(r.Context(), userID, r.PathValue("invitation_id"), accept)
	if err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toInvitationResponse(*entry))
}

// AcceptInvitation 接受邀請（POST /api/v1/groups/invitations/{invitation_id}/accept）。
func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	h.respondInvitation(w, r, true)
}

// DeclineInvitation 拒絕邀請（POST /api/v1/groups/invitations/{invitation_id}/decline）。
func (h *Handler) DeclineInvitation(w http.ResponseWriter, r *http.Request) {
	h.respondInvitation(w, r, false)
}

// CancelInvitation 取消邀請（POST /api/v1/groups/invitations/{invitation_id}/cancel）。
func (h *Handler) CancelInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entry, err := h.Service.CancelInvitation(r.Context(), userID, r.PathValue("invitation_id"))
	if err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toInvitationResponse(*entry))
}

type announcementResponse struct {
	AnnouncementID string     `json:"announcement_id"`
	GroupID        string     `json:"group_id"`
	AuthorID       string     `json:"author_id"`
	Content        string     `json:"content"`
	IsActive       bool       `json:"is_active"`
	PublishedAt    time.Time  `json:"published_at"`
	RetractedAt    *time.Time `json:"retracted_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func toAnnouncementResponse(a *Announcement) announcementResponse {
	return announcementResponse{
		AnnouncementID: a.ID,
		GroupID:        a.GroupID,
		AuthorID:       a.AuthorID,
		Content:        a.Content,
		IsActive:       a.IsActive,
		PublishedAt:    utcTime(a.PublishedAt),
		RetractedAt:    utcTimePtr(a.RetractedAt),
		CreatedAt:      utcTime(a.CreatedAt),
		UpdatedAt:      utcTime(a.UpdatedAt),
	}
}

// ListAnnouncements 列出群組公告（GET /api/v1/groups/{group_id}/announcements）。
func (h *Handler) ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	rows, err := h.Service.ListAnnouncements(r.Context(), userID, r.PathValue("group_id"))
	if err != nil {
		writeGroupError(w, err)
		return
	}
	out := make([]announcementResponse, 0, len(rows))
	for _, a := range rows {
		out = append(out, toAnnouncementResponse(a))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type announcementBody struct {
	Content string `json:"content"`
}

// CreateAnnouncement 發佈公告（POST /api/v1/groups/{group_id}/announcements）。
func (h *Handler) CreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req announcementBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	a, err := h.Service.CreateAnnouncement(r.Context(), userID, r.PathValue("group_id"), req.Content)
	if err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusCreated, toAnnouncementResponse(a))
}

// UpdateAnnouncement 編輯公告（PATCH /api/v1/groups/{group_id}/announcements/{announcement_id}）。
func (h *Handler) UpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req announcementBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	a, err := h.Service.UpdateAnnouncement(r.Context(), userID, r.PathValue("group_id"), r.PathValue("announcement_id"), req.Content)
	if err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toAnnouncementResponse(a))
}

// RetractAnnouncement 撤下公告（POST /api/v1/groups/{group_id}/announcements/{announcement_id}/retract）。
func (h *Handler) RetractAnnouncement(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	a, err := h.Service.RetractAnnouncement(r.Context(), userID, r.PathValue("group_id"), r.PathValue("announcement_id"))
	if err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toAnnouncementResponse(a))
}

type settingsResponse struct {
	GroupID      string     `json:"group_id"`
	InvitePolicy string     `json:"invite_policy"`
	UpdatedBy    string     `json:"updated_by,omitempty"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

func toSettingsResponse(st *Settings) settingsResponse {
	return settingsResponse{
		GroupID:      st.GroupID,
		InvitePolicy: string(st.InvitePolicy),
		UpdatedBy:    st.UpdatedBy,
		UpdatedAt:    utcTimePtr(st.UpdatedAt),
	}
}

// GetSettings 讀取群組設定（GET /api/v1/groups/{group_id}/settings）。
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	st, err := h.Service.GetSettings(r.Context(), userID, r.PathValue("group_id"))
	if err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toSettingsResponse(st))
}

type updateSettingsBody struct {
	InvitePolicy *string `json:"invite_policy"`
}

// UpdateSettings 更新群組設定（PATCH /api/v1/groups/{group_id}/settings）。
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req updateSettingsBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	var policy *InvitePolicy
	if req.InvitePolicy != nil {
		p := InvitePolicy(*req.InvitePolicy)
		policy = &p
	}
	st, err := h.Service.UpdateSettings(r.Context(), userID, r.PathValue("group_id"), policy)
	if err != nil {
		writeGroupError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toSettingsResponse(st))
}
