export type PresenceStatus = 'online' | 'away' | 'busy' | 'offline';

export type MessageStatus = 'sending' | 'sent' | 'delivered' | 'read';

export interface Contact {
  id: string;
  name: string;
  handle: string;
  statusMessage: string;
  presence: PresenceStatus;
  lastSeen: string;
  isFavorite: boolean;
  group: string;
}

export interface ChatMessage {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  sentAt: string;
  status: MessageStatus;
  kind: 'text' | 'voice' | 'attachment';
  durationSeconds?: number;
  fileName?: string;
  fileSize?: string;
  /**
   * 編輯時間（編輯／收回／刪除接線，§2u）：僅展示 provider 的
   * 快照 overlay 與 `toChatMessage` 填寫；未編輯時不帶此欄。
   */
  editedAt?: string;
  /** 是否已收回（展示快照 overlay／`toChatMessage` 填寫）。 */
  recalled?: boolean;
  /** 是否僅自己刪除（展示快照 overlay／`toChatMessage` 填寫）。 */
  deletedForSelf?: boolean;
}

export interface Conversation {
  id: string;
  contactId: string;
  lastMessagePreview: string;
  lastMessageAt: string;
  unreadCount: number;
  isPinned: boolean;
  isMuted: boolean;
  isTyping: boolean;
}

export interface DemoProfile {
  name: string;
  handle: string;
  statusMessage: string;
  presence: PresenceStatus;
}

/**
 * A signed-in session as shown in the profile view (M2c). The API
 * provider maps the wire shape (SessionResponse) to this; the demo
 * provider serves fixtures of the same shape. No token material is
 * ever carried here — only identifiers and timestamps.
 */
export interface SessionItem {
  id: string;
  deviceId: string;
  expiresAt: string;
  lastActiveAt: string;
  isRevoked: boolean;
  revokedReason: string;
}

/** A registered device as shown in the profile view (M2c). */
export interface DeviceItem {
  id: string;
  name: string;
  platform: string;
  isTrusted: boolean;
  isRevoked: boolean;
  lastActiveAt: string | null;
  createdAt: string;
}

export type ConversationFilter = 'all' | 'unread' | 'favorites';

/**
 * A friend as shown in the contacts view (M3). Mapped from the
 * backend friend list; the demo provider serves fixtures of the
 * same shape. No private fields (phone, email) are ever carried here.
 */
export interface FriendItem {
  id: string;
  displayName: string;
  publicId: string;
  avatarMediaId: string;
  statusMessage: string;
  isHidden: boolean;
  establishedAt: string;
}

/** A pending friend request, viewed from the caller's side (M3). */
export interface FriendRequestItem {
  id: string;
  direction: 'incoming' | 'outgoing';
  otherId: string;
  otherDisplayName: string;
  otherPublicId: string;
  message: string;
  createdAt: string;
}

/** A user found by public-id search (M3). */
export interface SearchedUser {
  id: string;
  displayName: string;
  publicId: string;
  avatarMediaId: string;
  statusMessage: string;
}

/** An active block as shown in the contacts view (M3). */
export interface BlockItem {
  userId: string;
  displayName: string;
  publicId: string;
  reason: string;
  blockedAt: string;
}

/**
 * A conversation as returned by the Phase 05 backend (M5). Mapped
 * from the wire shape (ConversationResponse) by the API provider;
 * the demo provider derives the same shape from local fixtures.
 * `type` and `role` vocabularies are implementation wording pending
 * finalization in the ERD (see services/api/internal/chat).
 */
export interface ConversationItem {
  id: string;
  type: 'direct' | 'group';
  title: string;
  createdBy: string;
  lastMessageId: string;
  lastMessageAt: string | null;
  unreadCount: number;
  createdAt: string;
  updatedAt: string;
}

/** A member of a conversation (Phase 05 backend, M5). */
export interface ConversationMemberItem {
  userId: string;
  role: string;
  joinedAt: string;
  leftAt: string | null;
  lastReadSequence: number;
}

/**
 * A message as returned by the Phase 05 backend (M5). `status` is
 * the server-side status only ('sent' | 'recalled'); delivery/read
 * state lives in receipts. Content is the opaque string the backend
 * stores — no E2EE cryptography is implemented or claimed.
 */
export interface MessageItem {
  id: string;
  conversationId: string;
  senderId: string;
  kind: string;
  content: string;
  status: string;
  clientMessageId: string;
  replyToMessageId: string;
  sequenceNumber: number;
  editCount: number;
  editedAt: string | null;
  recalledAt: string | null;
  deletedForSelf: boolean;
  createdAt: string;
  updatedAt: string;
}

/** One cursor page of messages (M5). The cursor is a sequence number. */
export interface MessagePage {
  messages: MessageItem[];
  nextCursor: string;
}

/**
 * A message search result (Phase 06 follow-up, §2v). The wire shape
 * mirrors the message response but carries no edit/receipt state:
 * the index holds the projected content only. Sequence numbers are
 * included so the view can jump to the message in its thread.
 */
export interface MessageSearchResultItem {
  id: string;
  conversationId: string;
  senderId: string;
  kind: string;
  content: string;
  sequenceNumber: number;
  createdAt: string;
}

/** One cursor page of message search results (§2v). */
export interface MessageSearchPage {
  results: MessageSearchResultItem[];
  nextCursor: string;
}

/** Input for a message search (§2v). Field names follow the backend query parameters. */
export interface MessageSearchInput {
  query: string;
  conversationId?: string;
  senderId?: string;
  since?: string;
  until?: string;
  limit?: number;
  cursor?: string;
}

/** A delivery/read receipt for a message (M5). */
export interface ReceiptItem {
  messageId: string;
  userId: string;
  receiptType: string;
  deviceId: string;
  createdAt: string;
}

/**
 * A media file attached to a message (§2s, Phase 05 follow-up).
 * Snapshot metadata only: the attachment references a media file
 * by id (direct FK, no storage path); bytes are never carried
 * here and no distribution URL exists yet, so an attachment is
 * shown as metadata, never as a downloadable file.
 */
export interface MessageAttachmentItem {
  id: string;
  messageId: string;
  mediaFileId: string;
  mimeType: string;
  sizeBytes: number;
  fileName: string;
  createdAt: string;
}

/**
 * The record of a forwarded message (§2s). A message is a forward
 * product exactly when such a record exists for it; the record
 * carries ids only, never content.
 */
export interface ForwardRecordItem {
  id: string;
  sourceMessageId: string;
  forwardedMessageId: string;
  sourceConversationId: string;
  targetConversationId: string;
  forwardedBy: string;
  createdAt: string;
}

/**
 * 反應允許清單（與後端 internal/message 的 AllowedEmojis 一致，
 * 見 STATUS.md 待決定 21）：8 種表情，順序即挑選器顯示順序。
 */
export const REACTION_EMOJIS: readonly string[] = ['👍', '❤️', '😂', '😮', '😢', '😡', '🙏', '🎉'];

/** 一則訊息上某一種反應的彙總（一則反應計數＋是否含自己）。 */
export interface ReactionCountItem {
  reaction: string;
  count: number;
  reactedByMe: boolean;
}

/** 一則訊息的反應彙總（後端排序：人數降序、表情升序）。 */
export interface ReactionSummaryItem {
  messageId: string;
  reactions: ReactionCountItem[];
}

/** 一則生效置頂（取消置頂後 unpinnedAt 非空，僅供紀錄）。 */
export interface PinItem {
  id: string;
  conversationId: string;
  messageId: string;
  pinnedBy: string;
  pinnedAt: string;
  unpinnedAt: string | null;
}

/**
 * A group as shown in the groups view (M6 follow-up). Mapped from
 * the backend group list; the demo provider serves fixtures of the
 * same shape. Role vocabulary (owner/admin/member) and the
 * invite-policy vocabulary (admin/all) are implementation wording
 * pending finalization in the ERD (see services/api/internal/group).
 * Disbanded groups never appear in the list (the backend answers
 * 404 for them everywhere and excludes them from ListGroups).
 */
export interface GroupItem {
  id: string;
  name: string;
  description: string;
  avatarMediaId: string;
  conversationId: string;
  createdBy: string;
  isDisbanded: boolean;
  memberCount: number;
  createdAt: string;
  updatedAt: string;
}

/** A member of a group (M6). displayName is '' when unknown. */
export interface GroupMemberItem {
  userId: string;
  displayName: string;
  role: string;
  joinedAt: string;
  invitedBy: string;
  leftAt: string | null;
}

/** A pending group invitation, viewed from either side (M6). */
export interface GroupInvitationItem {
  id: string;
  groupId: string;
  groupName: string;
  inviterId: string;
  inviteeId: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}

/** A group announcement (M6). Only one announcement is active at a time. */
export interface GroupAnnouncementItem {
  id: string;
  groupId: string;
  authorId: string;
  content: string;
  isActive: boolean;
  publishedAt: string;
  retractedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

/** Group settings (M6). Currently only the invite policy is a key. */
export interface GroupSettingsItem {
  groupId: string;
  invitePolicy: string;
  updatedBy: string;
  updatedAt: string | null;
}

/**
 * A media file as shown in the media view (Phase 06 media, web
 * wiring). Mapped from the backend file record; the demo provider
 * serves fixtures of the same shape. The scan status vocabulary
 * (pending/clean/rejected) is implementation wording pending
 * finalization in the ERD (see services/api/internal/media). A file
 * whose scan has not passed must never be presented as
 * distributable, whatever the caller asked for.
 */
export interface MediaFileItem {
  id: string;
  mimeType: string;
  sizeBytes: number;
  originalFilename: string;
  width: number;
  height: number;
  durationSeconds: number;
  checksum: string;
  scanStatus: string;
  scanCompletedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

/**
 * An upload session as shown in the media view. The storage key is
 * issued by the server. The upload URL is a transient field: only
 * the session-creation (and re-issue) responses carry it; re-listing
 * a session does not return it, so a fresh URL must be re-issued
 * before bytes can be sent (§2z).
 */
export interface MediaUploadItem {
  id: string;
  mediaFileId: string;
  status: string;
  storageKey: string;
  externalUploadId: string;
  uploadedBytes: number;
  totalBytes: number;
  partCount: number;
  uploadUrl: string | null;
  uploadUrlExpiresAt: string | null;
  completedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

/**
 * A thumbnail record as shown in the media view. Metadata only:
 * the backend returns no distribution URL, so a record is shown as
 * a placeholder, never as an image that can be opened.
 */
export interface MediaThumbnailItem {
  id: string;
  fileId: string;
  variantType: string;
  mimeType: string;
  sizeBytes: number;
  width: number;
  height: number;
  createdAt: string;
}

/**
 * The result of a file-side access check (Phase 06 media). When
 * access is allowed, the backend may also issue a download URL: an
 * absolute presigned URL in MinIO mode, or a relay path in
 * memory-stub mode. An allowed check with no URL must never be
 * presented as downloadable (§2z).
 */
export interface MediaAuthorization {
  allowed: boolean;
  scanStatus: string;
  downloadUrl: string | null;
  downloadUrlExpiresAt: string | null;
}

/** A freshly issued upload URL for an in-progress session (§2z). */
export interface MediaUploadUrl {
  uploadUrl: string;
  expiresAt: string;
}

/**
 * The outcome of resolving a file download (§2aa): a URL to open
 * in a new tab (absolute presigned URL), bytes fetched through
 * the API (relay path or local demo bytes), a scan-gate refusal,
 * or an allowed check that carries no distributable bytes.
 */
export type MediaDownloadResolution =
  | { status: 'ready-url'; url: string }
  | { status: 'ready-blob'; blob: Blob }
  | { status: 'forbidden'; scanStatus: string }
  | { status: 'unavailable' };

/** Metadata sent when an upload session is created (Phase 06 media). */
export interface InitiateUploadInput {
  mimeType: string;
  totalBytes: number;
  originalFilename: string;
  width: number;
  height: number;
  durationSeconds: number;
  checksum: string;
}

/** Post visibility (Timeline, §2ab): the backend vocabulary, pending final schema. */
export type PostVisibility = 'public' | 'friends' | 'only_me';

/** One media association on a post (§2ab). Metadata only: the post stores the media file id (direct reference, no snapshot); no distribution URL exists here, so media renders as metadata, never as a downloadable file. */
export interface PostMediaItem {
  mediaFileId: string;
  position: number;
  caption: string;
}

/**
 * A timeline post as shown in the timeline view. Counts are live
 * aggregates on the backend (likes count active rows, comments count
 * non-deleted rows); likedByMe is the caller's own like state.
 */
export interface PostItem {
  id: string;
  authorId: string;
  content: string;
  visibility: PostVisibility;
  editedAt: string | null;
  media: PostMediaItem[];
  likeCount: number;
  commentCount: number;
  likedByMe: boolean;
  createdAt: string;
  updatedAt: string;
}

/** One cursor page of posts (§2ab). The cursor is opaque; an empty nextCursor means the end. */
export interface PostPage {
  posts: PostItem[];
  nextCursor: string;
}

/**
 * A comment on a post (§2ab). Deleted comments keep their row but
 * read back with empty content; replies to a deleted parent are
 * retained. There is no comment edit endpoint.
 */
export interface PostCommentItem {
  id: string;
  postId: string;
  authorId: string;
  parentCommentId: string;
  content: string;
  isDeleted: boolean;
  editedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

/** One cursor page of comments (§2ab), oldest first. */
export interface PostCommentPage {
  comments: PostCommentItem[];
  nextCursor: string;
}

/** Input for creating a post (§2ab). An empty visibility defaults to friends on the backend. */
export interface CreatePostInput {
  content: string;
  visibility?: PostVisibility;
  mediaFileIds?: string[];
}

/** Patch for editing a post (§2ab): absent fields are left unchanged. Media associations cannot be edited. */
export interface UpdatePostInput {
  content?: string;
  visibility?: PostVisibility;
}

/** The outcome of liking or unliking a post (§2ab). */
export interface PostLikeResult {
  postId: string;
  liked: boolean;
  likeCount: number;
}

/** Call type (Call, §2ad): the backend vocabulary, pending final schema. */
export type CallType = 'voice' | 'video';

/**
 * Call status (Call, §2ad): ringing 響鈴中、active 通話中、declined
 * 全員拒接、missed 未接、busy 全員忙線、ended 已結束。Ended and
 * the three all-responded outcomes are terminal.
 */
export type CallStatus = 'ringing' | 'active' | 'declined' | 'missed' | 'busy' | 'ended';

/** Why a call ended (Call, §2ad): hangup 掛斷、declined 拒接、missed 未接、busy 忙線. */
export type CallEndReason = 'hangup' | 'declined' | 'missed' | 'busy';

/**
 * One participant's outcome (Call, §2ad): pending 待回應、joined
 * 已加入、declined 已拒接、busy 忙線、missed 未接.
 */
export type CallParticipantResult = 'pending' | 'joined' | 'declined' | 'busy' | 'missed';

/**
 * A call state event (Call, §2ad): initiated 發起、answered 接聽、
 * declined 拒接、busy 忙線、ended 結束. Events carry no payload
 * over the API: only the type, the actor and the time are exposed.
 */
export type CallEventType = 'initiated' | 'answered' | 'declined' | 'busy' | 'ended';

/**
 * A call session as shown in the calls view. Exactly one of
 * conversationId / groupId is set: a direct call hangs off a
 * conversation, a group call off a group. startedAt is the first
 * answer time (null before anyone answers); durationSeconds only
 * has a value for an ended call that had an answered period.
 * Records are never physically deleted.
 */
export interface CallItem {
  id: string;
  conversationId: string;
  groupId: string;
  initiatorId: string;
  callType: CallType;
  status: CallStatus;
  startedAt: string | null;
  endedAt: string | null;
  durationSeconds: number | null;
  endedBy: string;
  endReason: CallEndReason | '';
  participantCount: number;
  createdAt: string;
  updatedAt: string;
}

/** One cursor page of calls (§2ad). The cursor is opaque; an empty nextCursor means the end. */
export interface CallPage {
  calls: CallItem[];
  nextCursor: string;
}

/** One participant of a call (§2ad). joinedAt is null until the participant joins. */
export interface CallParticipantItem {
  userId: string;
  joinedAt: string | null;
  leftAt: string | null;
  result: CallParticipantResult;
  createdAt: string;
}

/** One call state event (§2ad), oldest first. No payload is exposed. */
export interface CallEventItem {
  id: string;
  userId: string;
  eventType: CallEventType;
  occurredAt: string;
}

/** Input for initiating a call (§2ad): exactly one scope id, plus the call type. */
export interface InitiateCallInput {
  conversationId?: string;
  groupId?: string;
  callType: CallType;
}

/**
 * Sticker pack category (Sticker, §2af): the backend vocabulary,
 * pending final schema. An empty category means uncategorized.
 */
export type StickerPackCategory =
  | ''
  | 'general'
  | 'animals'
  | 'food'
  | 'emotions'
  | 'festivals'
  | 'characters'
  | 'other';

/**
 * Sticker pack status (Sticker, §2af): draft 草稿、published 已上架、
 * unlisted 未上架、archived 已封存（刪除的狀態標記；封存的包在
 * 讀取端點一律以不存在回應）。
 */
export type StickerPackStatus = 'draft' | 'published' | 'unlisted' | 'archived';

/** A single sticker's status (Sticker, §2af): active 生效中、removed 已移除。 */
export type StickerStatus = 'active' | 'removed';

/**
 * A sticker pack as shown in the stickers view. stickerCount is a
 * live aggregate of active stickers; favoritedByMe is the caller's
 * own favorite state. A pack the caller cannot see never reaches
 * the view: the backend answers 404 and the demo provider applies
 * the same visibility rule locally.
 */
export interface StickerPackItem {
  id: string;
  creatorId: string;
  name: string;
  description: string;
  category: StickerPackCategory;
  coverMediaId: string;
  isFree: boolean;
  status: StickerPackStatus;
  stickerCount: number;
  favoritedByMe: boolean;
  createdAt: string;
  updatedAt: string;
}

/** One cursor page of sticker packs (§2af). The cursor is opaque; an empty nextCursor means the end. */
export interface StickerPackPage {
  packs: StickerPackItem[];
  nextCursor: string;
}

/**
 * A sticker as shown in the stickers view. The sticker body is only
 * a media file id (direct reference, no bytes, no distribution
 * URL): the view renders the name as a placeholder tile, never as
 * an image that can be opened or downloaded.
 */
export interface StickerItem {
  id: string;
  packId: string;
  packName: string;
  mediaFileId: string;
  name: string;
  isAnimated: boolean;
  position: number;
  status: StickerStatus;
  createdAt: string;
  updatedAt: string;
}

/** One cursor page of stickers (§2af). The cursor is opaque; an empty nextCursor means the end. */
export interface StickerPage {
  stickers: StickerItem[];
  nextCursor: string;
}

/** Input for creating a sticker pack (§2af). New packs are always drafts on the backend. */
export interface CreateStickerPackInput {
  name: string;
  description: string;
  category: StickerPackCategory;
  coverMediaId: string;
  isFree: boolean;
}

/**
 * Patch for editing a sticker pack (§2af): absent fields are left
 * unchanged. Status only accepts draft / published / unlisted;
 * archiving goes through the delete endpoint.
 */
export interface UpdateStickerPackInput {
  name?: string;
  description?: string;
  category?: StickerPackCategory;
  coverMediaId?: string;
  status?: 'draft' | 'published' | 'unlisted';
  isFree?: boolean;
}

/** Input for adding a sticker to a pack (§2af): exactly one media file body. */
export interface AddStickerInput {
  mediaFileId: string;
  name: string;
  isAnimated: boolean;
}
