import { useSyncExternalStore } from 'react';
import type {
  BlockItem,
  CallEventItem,
  CallItem,
  CallPage,
  CallParticipantItem,
  ChatMessage,
  Contact,
  Conversation,
  ConversationItem,
  ConversationMemberItem,
  CreatePostInput,
  DemoProfile,
  DeviceItem,
  ForwardRecordItem,
  FriendItem,
  FriendRequestItem,
  GroupAnnouncementItem,
  GroupInvitationItem,
  GroupItem,
  GroupMemberItem,
  GroupSettingsItem,
  InitiateCallInput,
  InitiateUploadInput,
  MediaAuthorization,
  MediaDownloadResolution,
  MediaFileItem,
  MediaThumbnailItem,
  MediaUploadItem,
  MediaUploadUrl,
  MessageAttachmentItem,
  MessageItem,
  MessagePage,
  MessageSearchInput,
  MessageSearchPage,
  MessageSearchResultItem,
  PinItem,
  PostCommentItem,
  PostCommentPage,
  PostItem,
  PostLikeResult,
  PostMediaItem,
  PostPage,
  ReactionSummaryItem,
  ReceiptItem,
  SearchedUser,
  SessionItem,
  StickerItem,
  StickerPackCategory,
  StickerPackItem,
  StickerPackPage,
  StickerPage,
  AddStickerInput,
  CreateStickerPackInput,
  UpdateStickerPackInput,
  UpdatePostInput,
} from './types';
import { REACTION_EMOJIS } from './types';
import {
  compareSearchResults,
  decodeSearchCursor,
  encodeSearchCursor,
  messageMatchesSearchTerms,
  normalizeSearchTerms,
} from './logic';
import {
  DEMO_BLOCKS,
  DEMO_CALLS,
  DEMO_CALL_EVENTS,
  DEMO_CALL_PARTICIPANTS,
  DEMO_CONTACTS,
  DEMO_CONVERSATIONS,
  DEMO_DEVICES,
  DEMO_FRIEND_REQUESTS,
  DEMO_FRIENDS,
  DEMO_GROUP_ANNOUNCEMENTS,
  DEMO_GROUP_INVITATIONS,
  DEMO_GROUP_MEMBERS,
  DEMO_GROUP_SETTINGS,
  DEMO_GROUPS,
  DEMO_MEDIA_FILES,
  DEMO_MEDIA_THUMBNAILS,
  DEMO_MEDIA_UPLOADS,
  DEMO_MESSAGES,
  DEMO_POSTS,
  DEMO_POST_COMMENTS,
  DEMO_POST_LIKED_BY,
  DEMO_PROFILE,
  DEMO_SESSIONS,
  DEMO_STICKERS,
  DEMO_STICKER_FAVORITES,
  DEMO_STICKER_GRANTED_PACKS,
  DEMO_STICKER_PACKS,
  DEMO_STICKER_RECENT,
} from './demo';

/**
 * Data-layer contract. Phase 02 ships the demo implementation only; the
 * Phase 03 backend replaces this object without touching any view code.
 * All data is local and clearly labelled as demonstration data in the UI.
 *
 * M2a adds the API-backed provider (see api-provider.ts): identity,
 * profile and settings come from services/api. The snapshot carries
 * apiConnection only in API mode so views can label the data source
 * honestly.
 *
 * M5 adds the async conversation/message methods: in API mode the
 * chats view reads the conversation list and message history from
 * the backend through them (list migration milestone); in demo mode
 * the demo provider implements the same methods over local fixtures
 * and the snapshot remains the source of truth.
 */

export type ApiConnectionState = 'connecting' | 'ready' | 'unauthorized' | 'unreachable';

export interface ApiConnectionInfo {
  mode: 'api';
  state: ApiConnectionState;
  message?: string | undefined;
}

export interface ChatSnapshot {
  profile: DemoProfile;
  contacts: Contact[];
  conversations: Conversation[];
  messages: ChatMessage[];
  apiConnection?: ApiConnectionInfo | undefined;
}

export interface ChatDataProvider {
  subscribe: (listener: () => void) => () => void;
  getSnapshot: () => ChatSnapshot;
  sendMessage: (conversationId: string, body: string) => void;
  markConversationRead: (conversationId: string) => void;
  updateProfile: (patch: Partial<DemoProfile>) => void;
  /**
   * Async profile update against the backend, when the provider is
   * API-backed. Views await it to show real success/failure; the demo
   * provider leaves it undefined and views fall back to updateProfile.
   */
  updateProfileAsync?: (patch: Partial<DemoProfile>) => Promise<void>;
  /**
   * Signed-in sessions and registered devices (M2c). Optional, like
   * updateProfileAsync: the demo provider serves local fixtures and
   * the API provider reads the backend. Views treat absence as
   * "not available in this mode" rather than an error.
   */
  listSessionsAsync?: () => Promise<SessionItem[]>;
  listDevicesAsync?: () => Promise<DeviceItem[]>;
  revokeSessionAsync?: (sessionId: string) => Promise<void>;
  revokeDeviceAsync?: (deviceId: string) => Promise<void>;
  /**
   * Friends (M3). Optional, like the session/device methods: the
   * demo provider serves local fixtures and the API provider reads
   * the backend. Views treat absence as "not available in this mode".
   */
  listFriendsAsync?: (includeHidden?: boolean) => Promise<FriendItem[]>;
  listFriendRequestsAsync?: (direction: 'incoming' | 'outgoing') => Promise<FriendRequestItem[]>;
  searchFriendAsync?: (publicId: string) => Promise<SearchedUser | null>;
  sendFriendRequestAsync?: (addresseeId: string, message?: string) => Promise<void>;
  respondFriendRequestAsync?: (requestId: string, accept: boolean) => Promise<void>;
  cancelFriendRequestAsync?: (requestId: string) => Promise<void>;
  removeFriendAsync?: (friendId: string) => Promise<void>;
  setFriendHiddenAsync?: (friendId: string, hidden: boolean) => Promise<void>;
  listBlocksAsync?: () => Promise<BlockItem[]>;
  blockUserAsync?: (userId: string, reason?: string) => Promise<void>;
  unblockUserAsync?: (userId: string) => Promise<void>;
  /**
   * Conversations and messages (M5, Phase 05). Optional, like the
   * session/device/friend methods: the demo provider serves local
   * fixtures and the API provider reads the backend. Views treat
   * absence as "not available in this mode". The composer prefers
   * sendMessageAsync when present so failures can be reported;
   * sending through the API provider requires a real backend
   * conversation id (demo ids are not backend ids).
   */
  listConversationsAsync?: () => Promise<ConversationItem[]>;
  createConversationAsync?: (input: {
    type: 'direct' | 'group';
    otherUserId?: string;
    title?: string;
    memberIds?: string[];
  }) => Promise<ConversationItem>;
  listMembersAsync?: (conversationId: string) => Promise<ConversationMemberItem[]>;
  /**
   * Conversation member management (§2x). Optional, like the other
   * async methods: the API provider posts to the backend, which
   * only allows adding members to group conversations and only by
   * an admin (a direct conversation or a non-admin caller is
   * rejected, and a non-member sees a 404). The demo provider
   * mirrors those guards: its conversations are all direct, so
   * adding members is rejected with the same message.
   */
  addMembersAsync?: (
    conversationId: string,
    memberIds: string[],
  ) => Promise<ConversationMemberItem[]>;
  leaveConversationAsync?: (conversationId: string) => Promise<void>;
  listMessagesAsync?: (
    conversationId: string,
    options?: { beforeSeq?: number; sinceSeq?: number; limit?: number },
  ) => Promise<MessagePage>;
  sendMessageAsync?: (
    conversationId: string,
    body: string,
    options?: { replyToMessageId?: string; attachmentFileIds?: string[] },
  ) => Promise<MessageItem>;
  editMessageAsync?: (messageId: string, content: string) => Promise<MessageItem>;
  recallMessageAsync?: (messageId: string) => Promise<MessageItem>;
  deleteMessageAsync?: (messageId: string) => Promise<MessageItem>;
  postReceiptAsync?: (messageId: string, receiptType: 'delivered' | 'read') => Promise<ReceiptItem>;
  /**
   * Message attachments and forwards (§2s). Optional, like the
   * other async methods: the demo provider serves local fixtures
   * with the same headline rules the backend applies (attachable
   * files must be owned and scan-clean, 10 attachments per
   * message, forward depth cap 5, no forwards of recalled
   * messages) and the API provider reads the backend. Attachment
   * lists and forward records are read separately from history:
   * history responses carry no attachments.
   */
  listAttachmentsAsync?: (messageId: string) => Promise<MessageAttachmentItem[]>;
  addAttachmentsAsync?: (
    messageId: string,
    mediaFileIds: string[],
  ) => Promise<MessageAttachmentItem[]>;
  forwardMessageAsync?: (messageId: string, targetConversationId: string) => Promise<MessageItem>;
  getForwardAsync?: (messageId: string) => Promise<ForwardRecordItem>;
  /**
   * 回條列表查詢（§2r）。Optional：展示 provider 回傳本機記帳的
   * 回條，API provider 讀後端 `GET /messages/{id}/receipts`。
   * 僅成員可讀；呼叫端查詢失敗時靜默，不阻擋歷史顯示。
   */
  listReceiptsAsync?: (messageId: string) => Promise<ReceiptItem[]>;
  /**
   * Message reactions and pins (§2p). Optional, like the other
   * async methods: the demo provider serves local fixtures with the
   * same headline rules the backend applies (allowlist, one active
   * reaction per user per message, 5-pin cap, no reactions/pins on
   * recalled messages) and the API provider reads the backend.
   * No realtime push exists for reactions/pins: callers re-read
   * after writing (REST sync, see STATUS.md 待決定 21).
   */
  listReactionsAsync?: (messageId: string) => Promise<ReactionSummaryItem>;
  setReactionAsync?: (messageId: string, reaction: string) => Promise<ReactionSummaryItem>;
  removeReactionAsync?: (messageId: string) => Promise<ReactionSummaryItem>;
  listPinsAsync?: (conversationId: string) => Promise<PinItem[]>;
  pinMessageAsync?: (conversationId: string, messageId: string) => Promise<PinItem>;
  unpinMessageAsync?: (conversationId: string, messageId: string) => Promise<void>;
  /**
   * Message search (§2v). Optional, like the other async methods:
   * the demo provider searches local fixtures with the same
   * matching semantics the backend applies (terms AND, substring,
   * case-insensitive, newest first, opaque cursor) and the API
   * provider reads GET /api/v1/search/messages. In API mode an
   * empty result list is honest when the backend index has not
   * been filled yet (the fill loop is a later milestone): callers
   * must not present it as proof that no message matches. The
   * index holds projected content only; recalled messages never
   * match because their content is removed from the index.
   */
  searchMessagesAsync?: (input: MessageSearchInput) => Promise<MessageSearchPage>;
  /**
   * Signed-in user id, when the provider knows it (list migration):
   * the demo provider returns 'me'; the API provider returns the
   * user id from GET /api/v1/auth/me once initialized, and null
   * before sign-in. Views use it to tell own messages apart from
   * other senders' messages. Absence means "unknown".
   */
  getCurrentUserId?: () => string | null;
  /**
   * Groups (M6 follow-up). Optional, like the other async methods:
   * the demo provider serves local fixtures and enforces the same
   * headline permission rules the backend applies (manager-only
   * management, owner-only transfer/disband/role changes, owner
   * cannot leave); the API provider reads the backend. Views derive
   * the caller's role from the member list and hide or disable
   * actions the caller may not perform; the backend remains the
   * final authority and answers 404 when a rule is violated.
   */
  listGroupsAsync?: () => Promise<GroupItem[]>;
  createGroupAsync?: (input: {
    name: string;
    description?: string;
    avatarMediaId?: string;
    memberIds?: string[];
  }) => Promise<GroupItem>;
  getGroupAsync?: (groupId: string) => Promise<GroupItem>;
  updateGroupAsync?: (
    groupId: string,
    patch: { name?: string; description?: string; avatarMediaId?: string },
  ) => Promise<GroupItem>;
  disbandGroupAsync?: (groupId: string) => Promise<void>;
  transferGroupOwnershipAsync?: (groupId: string, newOwnerId: string) => Promise<GroupItem>;
  listGroupMembersAsync?: (groupId: string) => Promise<GroupMemberItem[]>;
  addGroupMembersAsync?: (groupId: string, memberIds: string[]) => Promise<GroupMemberItem[]>;
  setGroupMemberRoleAsync?: (
    groupId: string,
    userId: string,
    role: string,
  ) => Promise<GroupMemberItem>;
  removeGroupMemberAsync?: (groupId: string, userId: string) => Promise<void>;
  leaveGroupAsync?: (groupId: string) => Promise<void>;
  listGroupInvitationsAsync?: (groupId: string) => Promise<GroupInvitationItem[]>;
  createGroupInvitationAsync?: (groupId: string, inviteeId: string) => Promise<GroupInvitationItem>;
  listMyGroupInvitationsAsync?: () => Promise<GroupInvitationItem[]>;
  respondGroupInvitationAsync?: (
    invitationId: string,
    accept: boolean,
  ) => Promise<GroupInvitationItem>;
  cancelGroupInvitationAsync?: (invitationId: string) => Promise<GroupInvitationItem>;
  listGroupAnnouncementsAsync?: (groupId: string) => Promise<GroupAnnouncementItem[]>;
  createGroupAnnouncementAsync?: (
    groupId: string,
    content: string,
  ) => Promise<GroupAnnouncementItem>;
  updateGroupAnnouncementAsync?: (
    groupId: string,
    announcementId: string,
    content: string,
  ) => Promise<GroupAnnouncementItem>;
  retractGroupAnnouncementAsync?: (
    groupId: string,
    announcementId: string,
  ) => Promise<GroupAnnouncementItem>;
  getGroupSettingsAsync?: (groupId: string) => Promise<GroupSettingsItem>;
  updateGroupSettingsAsync?: (groupId: string, invitePolicy: string) => Promise<GroupSettingsItem>;
  /**
   * Media files and upload sessions (Phase 06 media, web wiring;
   * byte channel §2aa). Optional, like the other async methods:
   * the demo provider serves local fixtures and the API provider
   * reads the backend. The byte flow is initiate → PUT bytes to
   * the issued upload URL → report progress → complete; the scan
   * gate decides whether a file may be presented as distributable.
   * Views treat absence as "not available in this mode".
   * Deliberately unwired: thumbnail registration, which would
   * claim thumbnails were generated (see STATUS).
   */
  listMediaFilesAsync?: () => Promise<MediaFileItem[]>;
  listMediaUploadsAsync?: () => Promise<MediaUploadItem[]>;
  initiateMediaUploadAsync?: (input: InitiateUploadInput) => Promise<MediaUploadItem>;
  issueMediaUploadUrlAsync?: (uploadId: string) => Promise<MediaUploadUrl>;
  putMediaUploadBytesAsync?: (uploadUrl: string, data: Blob) => Promise<void>;
  reportMediaUploadProgressAsync?: (
    uploadId: string,
    uploadedBytes: number,
    partCount: number,
  ) => Promise<MediaUploadItem>;
  completeMediaUploadAsync?: (
    uploadId: string,
    input: InitiateUploadInput,
  ) => Promise<MediaFileItem>;
  cancelMediaUploadAsync?: (uploadId: string) => Promise<MediaUploadItem>;
  deleteMediaFileAsync?: (fileId: string) => Promise<void>;
  listMediaThumbnailsAsync?: (fileId: string) => Promise<MediaThumbnailItem[]>;
  authorizeMediaAccessAsync?: (
    fileId: string,
    accessType: 'download' | 'preview' | 'thumbnail',
  ) => Promise<MediaAuthorization>;
  resolveMediaDownloadAsync?: (fileId: string) => Promise<MediaDownloadResolution>;
  /**
   * Timeline posts, comments and likes (§2ab). Optional, like the
   * other async methods: the demo provider serves local fixtures
   * with the same headline rules the backend applies (visibility
   * filter with either-direction blocks hidden, author-only edit
   * and delete, soft-deleted comments read back empty, live
   * counts, opaque cursors, 2000/1000 rune limits, 9-media cap
   * through the scan gate) and the API provider reads the
   * backend. No realtime push exists for posts: callers re-read
   * after writing (REST sync, see STATUS.md 待決定 29).
   */
  listFeedAsync?: (options?: { cursor?: string; limit?: number }) => Promise<PostPage>;
  listAuthorPostsAsync?: (
    authorId: string,
    options?: { cursor?: string; limit?: number },
  ) => Promise<PostPage>;
  getPostAsync?: (postId: string) => Promise<PostItem>;
  createPostAsync?: (input: CreatePostInput) => Promise<PostItem>;
  updatePostAsync?: (postId: string, input: UpdatePostInput) => Promise<PostItem>;
  deletePostAsync?: (postId: string) => Promise<void>;
  listPostMediaAsync?: (postId: string) => Promise<PostMediaItem[]>;
  likePostAsync?: (postId: string) => Promise<PostLikeResult>;
  unlikePostAsync?: (postId: string) => Promise<PostLikeResult>;
  createPostCommentAsync?: (
    postId: string,
    content: string,
    parentCommentId?: string,
  ) => Promise<PostCommentItem>;
  listPostCommentsAsync?: (
    postId: string,
    options?: { cursor?: string; limit?: number },
  ) => Promise<PostCommentPage>;
  deletePostCommentAsync?: (commentId: string) => Promise<void>;
  /**
   * Call sessions (§2ad). Optional, like the other async methods:
   * the demo provider serves local fixtures with the same headline
   * rules the backend applies (ringing → active / declined / busy /
   * missed / ended, exactly one scope, duplicate-active guard,
   * either-direction blocks hidden as not found, started_at only
   * on first answer, duration only for ended calls with an
   * answered period, events carry no payload) and the API provider
   * reads the backend. There is no realtime signaling: callers
   * re-read after writing (REST sync, see STATUS.md 待決定 31).
   * No media is ever involved at this layer.
   */
  listCallsAsync?: (options?: { cursor?: string; limit?: number }) => Promise<CallPage>;
  getCallAsync?: (callId: string) => Promise<CallItem>;
  initiateCallAsync?: (input: InitiateCallInput) => Promise<CallItem>;
  answerCallAsync?: (callId: string) => Promise<CallItem>;
  declineCallAsync?: (callId: string) => Promise<CallItem>;
  busyCallAsync?: (callId: string) => Promise<CallItem>;
  endCallAsync?: (callId: string) => Promise<CallItem>;
  listCallParticipantsAsync?: (callId: string) => Promise<CallParticipantItem[]>;
  listCallEventsAsync?: (callId: string) => Promise<CallEventItem[]>;
  /**
   * Sticker packs, stickers, favorites and recent use (§2af).
   * Optional, like the other async methods: the demo provider
   * serves local fixtures with the same headline rules the
   * backend applies (author-only management, draft/unlisted only
   * for the author, archived never readable, paid published packs
   * only with a grant row, favorite idempotency, media scan gate,
   * 100-per-pack cap, 30 recent rows, token AND substring search
   * over the sticker-plus-pack name union, opaque cursors taking
   * the last returned row) and the API provider reads the
   * backend. No purchase flow exists at this layer: paid packs
   * without a grant simply do not appear. No realtime push
   * exists for stickers: callers re-read after writing.
   */
  listStickerPacksAsync?: (options?: {
    creatorId?: string;
    category?: string;
    q?: string;
    cursor?: string;
    limit?: number;
  }) => Promise<StickerPackPage>;
  getStickerPackAsync?: (packId: string) => Promise<StickerPackItem>;
  createStickerPackAsync?: (input: CreateStickerPackInput) => Promise<StickerPackItem>;
  updateStickerPackAsync?: (
    packId: string,
    input: UpdateStickerPackInput,
  ) => Promise<StickerPackItem>;
  deleteStickerPackAsync?: (packId: string) => Promise<void>;
  listPackStickersAsync?: (packId: string) => Promise<StickerItem[]>;
  addStickerAsync?: (packId: string, input: AddStickerInput) => Promise<StickerItem>;
  removeStickerAsync?: (stickerId: string) => Promise<void>;
  favoriteStickerPackAsync?: (packId: string) => Promise<StickerPackItem>;
  unfavoriteStickerPackAsync?: (packId: string) => Promise<StickerPackItem>;
  listFavoriteStickerPacksAsync?: (options?: {
    cursor?: string;
    limit?: number;
  }) => Promise<StickerPackPage>;
  recordStickerUseAsync?: (stickerId: string) => Promise<StickerItem>;
  listRecentStickersAsync?: (options?: { cursor?: string; limit?: number }) => Promise<StickerPage>;
  searchStickersAsync?: (
    q: string,
    options?: { category?: string; cursor?: string; limit?: number },
  ) => Promise<StickerPage>;
  setTyping: (conversationId: string, isTyping: boolean) => void;
}

function cloneSnapshot(snapshot: ChatSnapshot): ChatSnapshot {
  return {
    profile: { ...snapshot.profile },
    contacts: snapshot.contacts.map((contact) => ({ ...contact })),
    conversations: snapshot.conversations.map((conversation) => ({ ...conversation })),
    messages: snapshot.messages.map((message) => ({ ...message })),
    ...(snapshot.apiConnection !== undefined
      ? { apiConnection: { ...snapshot.apiConnection } }
      : {}),
  };
}

export function createDemoProvider(): ChatDataProvider {
  let snapshot: ChatSnapshot = {
    profile: { ...DEMO_PROFILE },
    contacts: DEMO_CONTACTS.map((contact) => ({ ...contact })),
    conversations: DEMO_CONVERSATIONS.map((conversation) => ({ ...conversation })),
    messages: DEMO_MESSAGES.map((message) => ({ ...message })),
  };
  const listeners = new Set<() => void>();
  let counter = 100;
  // Local demonstration sessions/devices (M2c): mutable copies of
  // the fixtures; revoking only flips the local revoked flag.
  let demoSessions: SessionItem[] = DEMO_SESSIONS.map((item) => ({ ...item }));
  let demoDevices: DeviceItem[] = DEMO_DEVICES.map((item) => ({ ...item }));
  // Local demonstration friends/requests/blocks (M3): mutable copies
  // of the fixtures; actions only change local state.
  let demoFriends: FriendItem[] = DEMO_FRIENDS.map((item) => ({ ...item }));
  let demoRequests: FriendRequestItem[] = DEMO_FRIEND_REQUESTS.map((item) => ({ ...item }));
  let demoBlocks: BlockItem[] = DEMO_BLOCKS.map((item) => ({ ...item }));
  let demoRequestCounter = 100;
  // Local demonstration receipts (§2r): receipts posted through this
  // provider, keyed by message id. Starts empty; the demo view keeps
  // its fixture message statuses, so demo behaviour is unchanged.
  let demoReceipts: ReceiptItem[] = [];

  // Local demonstration groups (M6 follow-up): mutable copies of
  // the fixtures; actions only change local state. Permission rules
  // mirror the backend's headline rules, judged for 'me'.
  let demoGroups: GroupItem[] = DEMO_GROUPS.map((item) => ({ ...item }));
  let demoGroupMembers: Record<string, GroupMemberItem[]> = Object.fromEntries(
    Object.entries(DEMO_GROUP_MEMBERS).map(([groupId, members]) => [
      groupId,
      members.map((item) => ({ ...item })),
    ]),
  );
  let demoGroupInvitations: GroupInvitationItem[] = DEMO_GROUP_INVITATIONS.map((item) => ({
    ...item,
  }));
  let demoGroupAnnouncements: GroupAnnouncementItem[] = DEMO_GROUP_ANNOUNCEMENTS.map((item) => ({
    ...item,
  }));
  let demoGroupSettings: GroupSettingsItem[] = DEMO_GROUP_SETTINGS.map((item) => ({ ...item }));
  let demoGroupCounter = 100;

  // Local demonstration media (Phase 06 media, web wiring): mutable
  // copies of the fixtures. The demo byte channel keeps uploaded
  // bytes in memory only, keyed by session id and then by file id;
  // fixture files have no bytes, so they resolve as unavailable.
  // The scan gate (clean only) decides the authorization answer.
  let demoMediaFiles: MediaFileItem[] = DEMO_MEDIA_FILES.map((item) => ({ ...item }));
  let demoMediaUploads: MediaUploadItem[] = DEMO_MEDIA_UPLOADS.map((item) => ({ ...item }));
  let demoMediaCounter = 100;
  const demoMediaBytes = new Map<string, Blob>();

  // Local demonstration timeline (Phase 08, web wiring): mutable
  // copies of the fixtures; actions only change local state.
  // Visibility, soft-delete and live-count rules mirror the
  // backend's headline rules, judged for 'me'. Deleted posts are
  // kept in a set (soft delete): every read path treats them as
  // not found, like the backend's 404-after-delete rule.
  let demoPosts: PostItem[] = DEMO_POSTS.map((item) => ({
    ...item,
    media: item.media.map((media) => ({ ...media })),
  }));
  let demoPostComments: PostCommentItem[] = DEMO_POST_COMMENTS.map((item) => ({ ...item }));
  const demoPostLikes = new Map<string, Set<string>>(
    Object.entries(DEMO_POST_LIKED_BY).map(([postId, userIds]) => [postId, new Set(userIds)]),
  );
  const demoDeletedPostIds = new Set<string>();
  let demoPostCounter = 100;
  let demoCommentCounter = 100;

  const demoPostCounts = (postId: string): { likeCount: number; commentCount: number } => ({
    likeCount: demoPostLikes.get(postId)?.size ?? 0,
    commentCount: demoPostComments.filter(
      (comment) => comment.postId === postId && !comment.isDeleted,
    ).length,
  });

  const demoPostView = (post: PostItem): PostItem => ({
    ...post,
    media: post.media.map((media) => ({ ...media })),
    ...demoPostCounts(post.id),
    likedByMe: demoPostLikes.get(post.id)?.has('me') ?? false,
  });

  // 展示能見度裁決（與後端 canView 同規則，檢視者恆為 'me'）：
  // 作者本人恆可見；任一方向封鎖一律不可見（展示封鎖清單即
  // 本人封鎖的對象）；public 全體可見；friends 僅好友可見；
  // only_me 僅作者可見。
  const demoCanViewPost = (post: PostItem): boolean => {
    if (post.authorId === 'me') {
      return true;
    }
    if (demoBlocks.some((block) => block.userId === post.authorId)) {
      return false;
    }
    if (post.visibility === 'public') {
      return true;
    }
    if (post.visibility === 'friends') {
      return demoFriends.some((friend) => friend.id === post.authorId);
    }
    return false;
  };

  const demoVisiblePost = (postId: string): PostItem | null => {
    const post = demoPosts.find((item) => item.id === postId);
    if (post === undefined || demoDeletedPostIds.has(postId) || !demoCanViewPost(post)) {
      return null;
    }
    return post;
  };

  // 展示游標：與後端同格式（(created_at, id) 的 base64url 編碼，
  // 僅展示資料內部使用）。貼文倒序鍵、留言升序鍵共用此編解碼。
  const demoEncodeCursor = (createdAt: string, id: string): string => {
    const bytes = new TextEncoder().encode(`${createdAt}|${id}`);
    let binary = '';
    bytes.forEach((byte) => {
      binary += String.fromCharCode(byte);
    });
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  };

  const demoDecodeCursor = (cursor: string): { createdAt: string; id: string } | null => {
    try {
      const padded = cursor.replace(/-/g, '+').replace(/_/g, '/');
      const raw = atob(padded);
      const separator = raw.lastIndexOf('|');
      if (separator <= 0 || separator === raw.length - 1) {
        return null;
      }
      return { createdAt: raw.slice(0, separator), id: raw.slice(separator + 1) };
    } catch {
      return null;
    }
  };

  const demoPostPage = (candidates: PostItem[], cursor: string, limit: number): PostPage => {
    const take = limit > 0 && limit <= 50 ? limit : 20;
    const bound = cursor === '' ? null : demoDecodeCursor(cursor);
    if (cursor !== '' && bound === null) {
      throw new Error('游標格式錯誤。');
    }
    const ordered = [...candidates].sort((a, b) => {
      const timeDelta = Date.parse(b.createdAt) - Date.parse(a.createdAt);
      return timeDelta !== 0 ? timeDelta : b.id.localeCompare(a.id);
    });
    const after =
      bound === null
        ? ordered
        : ordered.filter((post) => {
            const timeDelta = Date.parse(post.createdAt) - Date.parse(bound.createdAt);
            return timeDelta < 0 || (timeDelta === 0 && post.id < bound.id);
          });
    const page = after.slice(0, take);
    const last = page[page.length - 1];
    const nextCursor =
      after.length > take && last !== undefined ? demoEncodeCursor(last.createdAt, last.id) : '';
    return { posts: page.map(demoPostView), nextCursor };
  };

  const demoCommentView = (comment: PostCommentItem): PostCommentItem =>
    comment.isDeleted ? { ...comment, content: '' } : { ...comment };

  const demoCommentPage = (postId: string, cursor: string, limit: number): PostCommentPage => {
    const take = limit > 0 && limit <= 100 ? limit : 30;
    const bound = cursor === '' ? null : demoDecodeCursor(cursor);
    if (cursor !== '' && bound === null) {
      throw new Error('游標格式錯誤。');
    }
    const ordered = demoPostComments
      .filter((comment) => comment.postId === postId)
      .sort((a, b) => {
        const timeDelta = Date.parse(a.createdAt) - Date.parse(b.createdAt);
        return timeDelta !== 0 ? timeDelta : a.id.localeCompare(b.id);
      });
    const after =
      bound === null
        ? ordered
        : ordered.filter((comment) => {
            const timeDelta = Date.parse(comment.createdAt) - Date.parse(bound.createdAt);
            return timeDelta > 0 || (timeDelta === 0 && comment.id > bound.id);
          });
    const page = after.slice(0, take);
    const last = page[page.length - 1];
    const nextCursor =
      after.length > take && last !== undefined ? demoEncodeCursor(last.createdAt, last.id) : '';
    return { comments: page.map(demoCommentView), nextCursor };
  };

  const demoUploadUrl = (uploadId: string): MediaUploadUrl => ({
    uploadUrl: `demo://uploads/${uploadId}`,
    expiresAt: new Date(Date.now() + 15 * 60 * 1000).toISOString(),
  });

  // Local demonstration calls (Phase 07, web wiring): mutable
  // copies of the fixtures; actions only change local state.
  // State machine, visibility and record rules mirror the
  // backend's headline rules, judged for 'me'. Call records are
  // append-only: no delete path exists, matching the backend.
  // Demo invitees never answer on their own: a ringing call only
  // resolves when the caller acts, exactly as the backend's
  // REST-only rules allow.
  let demoCalls: CallItem[] = DEMO_CALLS.map((item) => ({ ...item }));
  const demoCallParticipants: Record<string, CallParticipantItem[]> = Object.fromEntries(
    Object.entries(DEMO_CALL_PARTICIPANTS).map(([callId, participants]) => [
      callId,
      participants.map((item) => ({ ...item })),
    ]),
  );
  const demoCallEvents: Record<string, CallEventItem[]> = Object.fromEntries(
    Object.entries(DEMO_CALL_EVENTS).map(([callId, events]) => [
      callId,
      events.map((item) => ({ ...item })),
    ]),
  );
  let demoCallCounter = 100;
  let demoCallEventCounter = 100;

  const demoCallView = (call: CallItem): CallItem => ({
    ...call,
    participantCount: (demoCallParticipants[call.id] ?? []).length,
  });

  const demoCallForParticipant = (callId: string): CallItem | null => {
    const call = demoCalls.find((item) => item.id === callId);
    if (call === undefined) {
      return null;
    }
    const participants = demoCallParticipants[call.id] ?? [];
    if (!participants.some((participant) => participant.userId === 'me')) {
      return null;
    }
    return call;
  };

  const demoRecordCallEvent = (
    callId: string,
    userId: string,
    eventType: CallEventItem['eventType'],
    occurredAt: string,
  ): void => {
    demoCallEventCounter += 1;
    const event: CallEventItem = {
      id: `demo-call-event-local-${demoCallEventCounter}`,
      userId,
      eventType,
      occurredAt,
    };
    demoCallEvents[callId] = [...(demoCallEvents[callId] ?? []), event];
  };

  const demoSaveCall = (next: CallItem): void => {
    demoCalls = demoCalls.map((item) => (item.id === next.id ? next : item));
  };

  const demoSaveCallParticipant = (callId: string, next: CallParticipantItem): void => {
    demoCallParticipants[callId] = (demoCallParticipants[callId] ?? []).map((item) =>
      item.userId === next.userId ? next : item,
    );
  };

  // 展示拒接／忙線回應（與後端 respond 同規則）：僅待回應的受邀者、
  // 僅響鈴中的通話可回應；全員皆已回應且無人接聽時按拒接優先彙總
  // 為 declined（有拒接者）或 busy（全員忙線），並記結束時間。
  const demoRespondCall = async (
    callId: string,
    result: 'declined' | 'busy',
    eventType: 'declined' | 'busy',
  ): Promise<CallItem> => {
    const call = demoCallForParticipant(callId);
    if (call === null) {
      throw new Error('找不到通話。');
    }
    if (call.status !== 'ringing') {
      throw new Error('通話已不在可回應狀態。');
    }
    const mine = (demoCallParticipants[call.id] ?? []).find((p) => p.userId === 'me');
    if (mine === undefined || mine.result !== 'pending') {
      throw new Error('參與者已回應此次來電。');
    }
    if (call.initiatorId === 'me') {
      throw new Error('發起人請以結束通話代替回應。');
    }
    const now = new Date().toISOString();
    demoSaveCallParticipant(call.id, { ...mine, result });
    demoRecordCallEvent(call.id, 'me', eventType, now);
    const participants = demoCallParticipants[call.id] ?? [];
    const allResponded = participants.every(
      (p) => p.userId === call.initiatorId || (p.result !== 'pending' && p.result !== 'joined'),
    );
    if (allResponded) {
      const anyDeclined = participants.some(
        (p) => p.userId !== call.initiatorId && p.result === 'declined',
      );
      const next: CallItem = {
        ...call,
        status: anyDeclined ? 'declined' : 'busy',
        endReason: anyDeclined ? 'declined' : 'busy',
        endedAt: now,
        endedBy: 'me',
        updatedAt: now,
      };
      demoSaveCall(next);
      demoRecordCallEvent(call.id, 'me', 'ended', now);
      emit();
      return demoCallView(next);
    }
    emit();
    return demoCallView(call);
  };

  // Local demonstration stickers (Phase 08, web wiring): mutable
  // copies of the fixtures; actions only change local state.
  // Visibility, author-only management, favorite, grant and
  // scan-gate rules mirror the backend's headline rules, judged
  // for 'me'. Archived packs and removed stickers are kept in
  // sets: every read path treats them as not found, like the
  // backend's 404-after-delete rule. No purchase flow exists: a
  // paid pack without a grant row simply never appears.
  let demoStickerPacks: StickerPackItem[] = DEMO_STICKER_PACKS.map((item) => ({ ...item }));
  let demoStickers: StickerItem[] = DEMO_STICKERS.map((item) => ({ ...item }));
  const demoStickerGrants = new Set<string>(DEMO_STICKER_GRANTED_PACKS);
  const demoStickerFavorites = new Map<string, string>(
    DEMO_STICKER_FAVORITES.map((packId) => [packId, '2026-11-25T10:00:00+08:00']),
  );
  let demoStickerRecent: { stickerId: string; lastUsedAt: string }[] = DEMO_STICKER_RECENT.map(
    (item) => ({ ...item }),
  );
  const demoArchivedPackIds = new Set<string>(
    DEMO_STICKER_PACKS.filter((pack) => pack.status === 'archived').map((pack) => pack.id),
  );
  const demoRemovedStickerIds = new Set<string>(
    DEMO_STICKERS.filter((sticker) => sticker.status === 'removed').map((sticker) => sticker.id),
  );
  let demoStickerPackCounter = 100;
  let demoStickerCounter = 100;

  const demoStickerPackView = (pack: StickerPackItem): StickerPackItem => ({
    ...pack,
    stickerCount: demoStickers.filter(
      (sticker) =>
        sticker.packId === pack.id &&
        sticker.status === 'active' &&
        !demoRemovedStickerIds.has(sticker.id),
    ).length,
    favoritedByMe: demoStickerFavorites.has(pack.id),
  });

  // 展示可見性裁決（與後端 canView 同規則，檢視者恆為 'me'）：
  // 作者本人恆可見；已封存一律不可見；草稿／未上架僅作者；
  // 已上架免費包全體可見，付費包須授權發放紀錄。
  const demoCanViewPack = (pack: StickerPackItem): boolean => {
    if (demoArchivedPackIds.has(pack.id)) {
      return false;
    }
    if (pack.creatorId === 'me') {
      return true;
    }
    if (pack.status !== 'published') {
      return false;
    }
    if (pack.isFree) {
      return true;
    }
    return demoStickerGrants.has(pack.id);
  };

  const demoVisiblePack = (packId: string): StickerPackItem | null => {
    const pack = demoStickerPacks.find((item) => item.id === packId);
    if (pack === undefined || !demoCanViewPack(pack)) {
      return null;
    }
    return pack;
  };

  const demoStickerView = (sticker: StickerItem): StickerItem => {
    const pack = demoStickerPacks.find((item) => item.id === sticker.packId);
    return { ...sticker, packName: pack?.name ?? sticker.packName };
  };

  // 展示游標分頁（與後端同語義）：排序鍵倒序、游標取最後一筆
  // 回傳列，僅展示資料內部使用。呼叫方先過濾候選列，再分頁。
  const demoStickerCursorPage = <T extends { id: string }>(
    rows: { item: T; sortAt: string }[],
    cursor: string,
    limit: number,
  ): { items: T[]; nextCursor: string } => {
    const take = limit > 0 && limit <= 50 ? limit : 20;
    const bound = cursor === '' ? null : demoDecodeCursor(cursor);
    if (cursor !== '' && bound === null) {
      throw new Error('游標格式錯誤。');
    }
    const ordered = [...rows].sort((a, b) => {
      const timeDelta = Date.parse(b.sortAt) - Date.parse(a.sortAt);
      return timeDelta !== 0 ? timeDelta : b.item.id.localeCompare(a.item.id);
    });
    const after =
      bound === null
        ? ordered
        : ordered.filter(({ item, sortAt }) => {
            const timeDelta = Date.parse(sortAt) - Date.parse(bound.createdAt);
            return timeDelta < 0 || (timeDelta === 0 && item.id < bound.id);
          });
    const page = after.slice(0, take);
    const last = page[page.length - 1];
    const nextCursor =
      after.length > take && last !== undefined ? demoEncodeCursor(last.sortAt, last.item.id) : '';
    return { items: page.map(({ item }) => item), nextCursor };
  };

  const demoStickerSearchTokens = (q: string): string[] => {
    const seen = new Set<string>();
    const tokens: string[] = [];
    for (const token of q.toLowerCase().split(/\s+/)) {
      if (token !== '' && !seen.has(token)) {
        seen.add(token);
        tokens.push(token);
      }
    }
    return tokens;
  };

  const demoAuthorizeMedia = (fileId: string): MediaAuthorization => {
    const file = demoMediaFiles.find((item) => item.id === fileId);
    const scanStatus = file?.scanStatus ?? 'pending';
    return {
      allowed: scanStatus === 'clean',
      scanStatus,
      downloadUrl: null,
      downloadUrlExpiresAt: null,
    };
  };

  const demoGroupMemberCount = (groupId: string): number =>
    (demoGroupMembers[groupId] ?? []).filter((item) => item.leftAt === null).length;

  const demoGroupView = (group: GroupItem): GroupItem => ({
    ...group,
    memberCount: demoGroupMemberCount(group.id),
  });

  const demoFindGroup = (groupId: string): GroupItem => {
    const group = demoGroups.find((item) => item.id === groupId && !item.isDisbanded);
    if (group === undefined) {
      throw new Error('找不到群組。');
    }
    return group;
  };

  const demoMyGroupRole = (groupId: string): string => {
    const self = (demoGroupMembers[groupId] ?? []).find(
      (item) => item.userId === 'me' && item.leftAt === null,
    );
    return self?.role ?? '';
  };

  const demoRequireGroupMember = (groupId: string): void => {
    demoFindGroup(groupId);
    if (demoMyGroupRole(groupId) === '') {
      throw new Error('找不到群組。');
    }
  };

  const demoRequireGroupManager = (groupId: string): void => {
    demoRequireGroupMember(groupId);
    const role = demoMyGroupRole(groupId);
    if (role !== 'owner' && role !== 'admin') {
      throw new Error('找不到群組。');
    }
  };

  const demoRequireGroupOwner = (groupId: string): void => {
    demoRequireGroupMember(groupId);
    if (demoMyGroupRole(groupId) !== 'owner') {
      throw new Error('找不到群組。');
    }
  };

  const demoContactName = (userId: string): string => {
    if (userId === 'me') {
      return DEMO_PROFILE.name;
    }
    return DEMO_CONTACTS.find((item) => item.id === userId)?.name ?? userId;
  };

  const demoResolveUserId = (raw: string): string => {
    const trimmed = raw.trim().replace(/^@/, '');
    if (trimmed === '') {
      return '';
    }
    const contact = DEMO_CONTACTS.find(
      (item) => item.id === trimmed || item.handle.replace(/^@/, '') === trimmed,
    );
    return contact?.id ?? trimmed;
  };

  const emit = (): void => {
    for (const listener of listeners) {
      listener();
    }
  };

  // Demo-only message overlays (M5): edits/recalls/deletes are local
  // fixture mutations, tracked separately from the snapshot messages.
  const demoEdits = new Map<string, { editCount: number; editedAt: string }>();
  const demoRecalled = new Map<string, string>();
  const demoDeleted = new Set<string>();

  // Demo-only reactions and pins (§2q): local overlays following the
  // backend's headline rules (allowlist, one active reaction per user
  // per message, 5-pin cap per conversation). Reactions keyed by
  // message id, then user id; pins keyed by conversation id, then
  // message id (value: pinned-at timestamp).
  const demoReactions = new Map<string, Map<string, string>>();
  const demoPins = new Map<string, Map<string, string>>();

  // Demo-only attachments and forwards (§2t): local overlays
  // following the backend's headline rules (attachable files must
  // be owned and scan-clean, 10 attachments per message, forward
  // depth cap 5). Attachments keyed by message id; forward records
  // keyed by the forwarded (new) message id; depths keyed by
  // message id (original messages count as depth 1).
  const demoAttachments = new Map<string, MessageAttachmentItem[]>();
  const demoForwards = new Map<string, ForwardRecordItem>();
  const demoForwardDepth = new Map<string, number>();

  /** 展示反應彙總：人數降序、表情升序，與後端排序一致。 */
  const demoReactionSummary = (messageId: string): ReactionSummaryItem => {
    const counts = new Map<string, { count: number; reactedByMe: boolean }>();
    for (const [userId, emoji] of demoReactions.get(messageId) ?? []) {
      const entry = counts.get(emoji) ?? { count: 0, reactedByMe: false };
      entry.count += 1;
      if (userId === 'me') {
        entry.reactedByMe = true;
      }
      counts.set(emoji, entry);
    }
    const reactions = [...counts.entries()]
      .map(([reaction, entry]) => ({
        reaction,
        count: entry.count,
        reactedByMe: entry.reactedByMe,
      }))
      .sort((a, b) => b.count - a.count || a.reaction.localeCompare(b.reaction));
    return { messageId, reactions };
  };

  /** 展示守衛：收回訊息與作者僅自己刪除的訊息不得新增反應／置頂／附件。 */
  const demoRequireReactable = (message: ChatMessage, action: '反應' | '置頂' | '附件'): void => {
    if (demoRecalled.has(message.id)) {
      throw new Error(`此訊息已收回，無法新增${action}。`);
    }
    if (message.senderId === 'me' && demoDeleted.has(message.id)) {
      throw new Error(`此訊息已刪除，無法新增${action}。`);
    }
  };

  /**
   * 展示附件守衛：檔案須存在且掃描通過（clean），與後端
   * CheckAttachable 的 headline 規則一致；展示檔案一律視為
   * 本人所有。回傳去重後的檔案清單（維持傳入順序）。
   */
  const demoResolveAttachable = (mediaFileIds: string[]): MediaFileItem[] => {
    const resolved: MediaFileItem[] = [];
    const seen = new Set<string>();
    for (const fileId of mediaFileIds) {
      if (seen.has(fileId)) {
        continue;
      }
      seen.add(fileId);
      const file = demoMediaFiles.find((item) => item.id === fileId);
      if (file === undefined) {
        throw new Error('找不到要關聯的媒體檔案。');
      }
      if (file.scanStatus !== 'clean') {
        throw new Error(`檔案「${file.originalFilename}」尚未通過掃描，無法關聯。`);
      }
      resolved.push(file);
    }
    return resolved;
  };

  /** 展示附件寫入：快照詮釋資料，上限每訊息 10 筆，重複關聯收斂。 */
  const demoAttachFiles = (messageId: string, mediaFileIds: string[]): MessageAttachmentItem[] => {
    const files = demoResolveAttachable(mediaFileIds);
    const existing = demoAttachments.get(messageId) ?? [];
    const attachedIds = new Set(existing.map((item) => item.mediaFileId));
    const now = new Date().toISOString();
    const appended = files
      .filter((file) => !attachedIds.has(file.id))
      .map((file) => ({
        id: `att-local-${messageId}-${file.id}`,
        messageId,
        mediaFileId: file.id,
        mimeType: file.mimeType,
        sizeBytes: file.sizeBytes,
        fileName: file.originalFilename,
        createdAt: now,
      }));
    const merged = [...existing, ...appended];
    if (merged.length > 10) {
      throw new Error('每則訊息最多關聯 10 個附件。');
    }
    demoAttachments.set(messageId, merged);
    return merged.map((item) => ({ ...item }));
  };

  const appendMessage = (conversationId: string, body: string): ChatMessage | null => {
    const trimmed = body.trim();
    if (trimmed === '') {
      return null;
    }
    counter += 1;
    const now = new Date().toISOString();
    const message: ChatMessage = {
      id: `m-local-${counter}`,
      conversationId,
      senderId: 'me',
      body: trimmed,
      sentAt: now,
      status: 'sent',
      kind: 'text',
    };
    snapshot = {
      ...snapshot,
      messages: [...snapshot.messages, message],
      conversations: snapshot.conversations.map((conversation) =>
        conversation.id === conversationId
          ? {
              ...conversation,
              lastMessagePreview: trimmed,
              lastMessageAt: now,
              unreadCount: 0,
            }
          : conversation,
      ),
    };
    emit();
    return message;
  };

  const conversationSequence = (conversationId: string): ChatMessage[] =>
    snapshot.messages
      .filter((message) => message.conversationId === conversationId)
      // 依實際時間排序：字串比較會被不同時區字尾（Z 與 +08:00）
      // 誤導，須先解析為時間戳。
      .sort((a, b) => Date.parse(a.sentAt) - Date.parse(b.sentAt));

  const toMessageItem = (message: ChatMessage, sequence: number): MessageItem => {
    const edit = demoEdits.get(message.id);
    const recalledAt = demoRecalled.get(message.id);
    const deleted = demoDeleted.has(message.id);
    return {
      id: message.id,
      conversationId: message.conversationId,
      senderId: message.senderId,
      kind: message.kind,
      content: recalledAt !== undefined || deleted ? '' : message.body,
      status: recalledAt !== undefined ? 'recalled' : message.status,
      clientMessageId: '',
      replyToMessageId: '',
      sequenceNumber: sequence,
      editCount: edit?.editCount ?? 0,
      editedAt: edit?.editedAt ?? null,
      recalledAt: recalledAt ?? null,
      deletedForSelf: deleted,
      createdAt: message.sentAt,
      updatedAt: edit?.editedAt ?? message.sentAt,
    };
  };

  const toConversationItem = (conversation: Conversation): ConversationItem => {
    const contact = snapshot.contacts.find((item) => item.id === conversation.contactId);
    return {
      id: conversation.id,
      type: 'direct',
      title: contact?.name ?? '',
      createdBy: 'me',
      lastMessageId: '',
      lastMessageAt: conversation.lastMessageAt,
      unreadCount: conversation.unreadCount,
      createdAt: conversation.lastMessageAt,
      updatedAt: conversation.lastMessageAt,
    };
  };

  return {
    subscribe: (listener: () => void) => {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    // 快照 overlay（編輯／收回／刪除接線，§2u）：收回與僅自己刪除
    // 的訊息在檢視層只顯示佔位文字（與後端回應層一致，見
    // toMessageResponse）；內部快照維持原文，async 方法的 overlay
    //（toMessageItem）不受影響。編輯時間取自 demoEdits。
    getSnapshot: () => {
      const cloned = cloneSnapshot(snapshot);
      return {
        ...cloned,
        messages: cloned.messages.map((message) => {
          if (demoRecalled.has(message.id)) {
            return { ...message, body: '此訊息已收回', recalled: true };
          }
          if (demoDeleted.has(message.id)) {
            return { ...message, body: '此訊息已刪除', deletedForSelf: true };
          }
          const edit = demoEdits.get(message.id);
          return edit === undefined ? message : { ...message, editedAt: edit.editedAt };
        }),
      };
    },
    sendMessage: (conversationId: string, body: string) => {
      appendMessage(conversationId, body);
    },
    markConversationRead: (conversationId: string) => {
      snapshot = {
        ...snapshot,
        conversations: snapshot.conversations.map((conversation) =>
          conversation.id === conversationId ? { ...conversation, unreadCount: 0 } : conversation,
        ),
      };
      emit();
    },
    updateProfile: (patch: Partial<DemoProfile>) => {
      snapshot = { ...snapshot, profile: { ...snapshot.profile, ...patch } };
      emit();
    },
    listSessionsAsync: async (): Promise<SessionItem[]> =>
      demoSessions.map((item) => ({ ...item })),
    listDevicesAsync: async (): Promise<DeviceItem[]> => demoDevices.map((item) => ({ ...item })),
    revokeSessionAsync: async (sessionId: string): Promise<void> => {
      demoSessions = demoSessions.map((item) =>
        item.id === sessionId ? { ...item, isRevoked: true, revokedReason: 'revoked' } : item,
      );
      emit();
    },
    revokeDeviceAsync: async (deviceId: string): Promise<void> => {
      demoDevices = demoDevices.map((item) =>
        item.id === deviceId ? { ...item, isRevoked: true } : item,
      );
      emit();
    },
    listFriendsAsync: async (includeHidden = false): Promise<FriendItem[]> =>
      demoFriends.filter((item) => includeHidden || !item.isHidden).map((item) => ({ ...item })),
    listFriendRequestsAsync: async (
      direction: 'incoming' | 'outgoing',
    ): Promise<FriendRequestItem[]> =>
      demoRequests.filter((item) => item.direction === direction).map((item) => ({ ...item })),
    searchFriendAsync: async (publicId: string): Promise<SearchedUser | null> => {
      const found = DEMO_CONTACTS.find(
        (contact) => contact.handle.replace(/^@/, '') === publicId.trim(),
      );
      if (found === undefined) {
        return null;
      }
      return {
        id: found.id,
        displayName: found.name,
        publicId: found.handle.replace(/^@/, ''),
        avatarMediaId: '',
        statusMessage: found.statusMessage,
      };
    },
    sendFriendRequestAsync: async (addresseeId: string, message = ''): Promise<void> => {
      const contact = DEMO_CONTACTS.find((item) => item.id === addresseeId);
      demoRequestCounter += 1;
      demoRequests = [
        ...demoRequests,
        {
          id: `demo-request-out-${demoRequestCounter}`,
          direction: 'outgoing',
          otherId: addresseeId,
          otherDisplayName: contact?.name ?? addresseeId,
          otherPublicId: contact?.handle.replace(/^@/, '') ?? '',
          message,
          createdAt: new Date().toISOString(),
        },
      ];
      emit();
    },
    respondFriendRequestAsync: async (requestId: string, accept: boolean): Promise<void> => {
      const target = demoRequests.find((item) => item.id === requestId);
      demoRequests = demoRequests.filter((item) => item.id !== requestId);
      if (
        accept &&
        target !== undefined &&
        !demoFriends.some((item) => item.id === target.otherId)
      ) {
        demoFriends = [
          ...demoFriends,
          {
            id: target.otherId,
            displayName: target.otherDisplayName,
            publicId: target.otherPublicId,
            avatarMediaId: '',
            statusMessage: '',
            isHidden: false,
            establishedAt: new Date().toISOString(),
          },
        ];
      }
      emit();
    },
    cancelFriendRequestAsync: async (requestId: string): Promise<void> => {
      demoRequests = demoRequests.filter((item) => item.id !== requestId);
      emit();
    },
    removeFriendAsync: async (friendId: string): Promise<void> => {
      demoFriends = demoFriends.filter((item) => item.id !== friendId);
      emit();
    },
    setFriendHiddenAsync: async (friendId: string, hidden: boolean): Promise<void> => {
      demoFriends = demoFriends.map((item) =>
        item.id === friendId ? { ...item, isHidden: hidden } : item,
      );
      emit();
    },
    listBlocksAsync: async (): Promise<BlockItem[]> => demoBlocks.map((item) => ({ ...item })),
    blockUserAsync: async (userId: string, reason = ''): Promise<void> => {
      demoFriends = demoFriends.filter((item) => item.id !== userId);
      demoRequests = demoRequests.filter((item) => item.otherId !== userId);
      if (!demoBlocks.some((item) => item.userId === userId)) {
        const contact = DEMO_CONTACTS.find((item) => item.id === userId);
        demoBlocks = [
          ...demoBlocks,
          {
            userId,
            displayName: contact?.name ?? userId,
            publicId: contact?.handle.replace(/^@/, '') ?? '',
            reason,
            blockedAt: new Date().toISOString(),
          },
        ];
      }
      emit();
    },
    unblockUserAsync: async (userId: string): Promise<void> => {
      demoBlocks = demoBlocks.filter((item) => item.userId !== userId);
      emit();
    },
    listConversationsAsync: async (): Promise<ConversationItem[]> =>
      snapshot.conversations.map(toConversationItem),
    createConversationAsync: async (input: {
      type: 'direct' | 'group';
      otherUserId?: string;
      title?: string;
      memberIds?: string[];
    }): Promise<ConversationItem> => {
      // 對齊後端 CreateDirect 語義：已有一對一對話時回傳既有資源，
      // 不另建重複對話（後端回 200＋既有對話；展示模式以同一規則
      // 避免同一聯絡人出現兩列對話）。
      if (input.type === 'direct' && input.otherUserId !== undefined) {
        const existing = snapshot.conversations.find(
          (conversation) => conversation.contactId === input.otherUserId,
        );
        if (existing !== undefined) {
          return { ...toConversationItem(existing), type: 'direct' };
        }
      }
      const contactId = input.type === 'direct' ? (input.otherUserId ?? '') : '';
      const now = new Date().toISOString();
      counter += 1;
      const conversation: Conversation = {
        id: `c-local-${counter}`,
        contactId,
        lastMessagePreview: '',
        lastMessageAt: now,
        unreadCount: 0,
        isPinned: false,
        isMuted: false,
        isTyping: false,
      };
      snapshot = { ...snapshot, conversations: [...snapshot.conversations, conversation] };
      emit();
      return { ...toConversationItem(conversation), type: input.type, title: input.title ?? '' };
    },
    listMembersAsync: async (conversationId: string): Promise<ConversationMemberItem[]> => {
      const conversation = snapshot.conversations.find((item) => item.id === conversationId);
      const now = new Date().toISOString();
      const members: ConversationMemberItem[] = [
        { userId: 'me', role: 'admin', joinedAt: now, leftAt: null, lastReadSequence: 0 },
      ];
      if (conversation !== undefined && conversation.contactId !== '') {
        members.push({
          userId: conversation.contactId,
          role: 'member',
          joinedAt: now,
          leftAt: null,
          lastReadSequence: 0,
        });
      }
      return members;
    },
    leaveConversationAsync: async (conversationId: string): Promise<void> => {
      snapshot = {
        ...snapshot,
        conversations: snapshot.conversations.filter((item) => item.id !== conversationId),
      };
      emit();
    },
    addMembersAsync: async (
      conversationId: string,
      memberIds: string[],
    ): Promise<ConversationMemberItem[]> => {
      void memberIds;
      // 展示資料的對話皆為一對一（toConversationItem 固定 direct），
      // 與後端 AddMembers 守衛一致：一對一對話不可新增成員。
      // 順序亦與後端一致：先查對話存在，再判對話類型。
      const conversation = snapshot.conversations.find((item) => item.id === conversationId);
      if (conversation === undefined) {
        throw new Error('找不到對話。');
      }
      throw new Error('一對一對話不可新增成員。');
    },
    listMessagesAsync: async (
      conversationId: string,
      options: { beforeSeq?: number; sinceSeq?: number; limit?: number } = {},
    ): Promise<MessagePage> => {
      const ordered = conversationSequence(conversationId);
      const withSeq = ordered.map((message, index) => toMessageItem(message, index + 1));
      const limit = options.limit ?? 30;
      const beforeSeq = options.beforeSeq;
      const sinceSeq = options.sinceSeq;
      let page: MessageItem[];
      let nextCursor = '';
      if (beforeSeq !== undefined) {
        page = withSeq
          .filter((item) => item.sequenceNumber < beforeSeq)
          .reverse()
          .slice(0, limit);
        const last = page[page.length - 1];
        nextCursor = last !== undefined ? String(last.sequenceNumber) : '';
        return { messages: page, nextCursor };
      }
      if (sinceSeq !== undefined) {
        page = withSeq.filter((item) => item.sequenceNumber > sinceSeq).slice(0, limit);
      } else {
        page = withSeq.slice(0, limit);
      }
      const last = page[page.length - 1];
      nextCursor = last !== undefined ? String(last.sequenceNumber) : '';
      return { messages: page, nextCursor };
    },
    sendMessageAsync: async (
      conversationId: string,
      body: string,
      options: { replyToMessageId?: string; attachmentFileIds?: string[] } = {},
    ): Promise<MessageItem> => {
      // 發送時關聯附件（§2t）：先驗證可關聯性再寫入，與後端
      // 同寫入路徑提交的語義一致，驗證失敗時不產生訊息。
      const attachable =
        options.attachmentFileIds !== undefined && options.attachmentFileIds.length > 0
          ? demoResolveAttachable(options.attachmentFileIds)
          : [];
      const message = appendMessage(conversationId, body);
      if (message === null) {
        throw new Error('訊息不可為空。');
      }
      if (attachable.length > 0) {
        demoAttachFiles(
          message.id,
          attachable.map((file) => file.id),
        );
      }
      const sequence = conversationSequence(conversationId).findIndex(
        (item) => item.id === message.id,
      );
      return toMessageItem(message, sequence + 1);
    },
    editMessageAsync: async (messageId: string, content: string): Promise<MessageItem> => {
      const trimmed = content.trim();
      if (trimmed === '') {
        throw new Error('訊息不可為空。');
      }
      const ordered = snapshot.messages;
      const index = ordered.findIndex((item) => item.id === messageId);
      const target = index === -1 ? undefined : ordered[index];
      if (target === undefined || target.senderId !== 'me') {
        throw new Error('找不到可編輯的訊息。');
      }
      // 後端守衛（見 Service.Edit）：收回或僅自己刪除的訊息不可編輯。
      if (demoRecalled.has(messageId) || demoDeleted.has(messageId)) {
        throw new Error('此訊息不可編輯。');
      }
      const now = new Date().toISOString();
      const previous = demoEdits.get(messageId);
      demoEdits.set(messageId, { editCount: (previous?.editCount ?? 0) + 1, editedAt: now });
      const updated: ChatMessage = { ...target, body: trimmed, editedAt: now };
      snapshot = {
        ...snapshot,
        messages: ordered.map((item) => (item.id === messageId ? updated : item)),
      };
      emit();
      const sequence =
        conversationSequence(updated.conversationId).findIndex((item) => item.id === messageId) + 1;
      return toMessageItem(updated, sequence);
    },
    recallMessageAsync: async (messageId: string): Promise<MessageItem> => {
      const message = snapshot.messages.find((item) => item.id === messageId);
      if (message === undefined || message.senderId !== 'me') {
        throw new Error('找不到可收回的訊息。');
      }
      demoRecalled.set(messageId, new Date().toISOString());
      emit();
      const sequence =
        conversationSequence(message.conversationId).findIndex((item) => item.id === messageId) + 1;
      return toMessageItem(message, sequence);
    },
    deleteMessageAsync: async (messageId: string): Promise<MessageItem> => {
      const message = snapshot.messages.find((item) => item.id === messageId);
      if (message === undefined || message.senderId !== 'me') {
        throw new Error('找不到可刪除的訊息。');
      }
      demoDeleted.add(messageId);
      emit();
      const sequence =
        conversationSequence(message.conversationId).findIndex((item) => item.id === messageId) + 1;
      return toMessageItem(message, sequence);
    },
    postReceiptAsync: async (
      messageId: string,
      receiptType: 'delivered' | 'read',
    ): Promise<ReceiptItem> => {
      const message = snapshot.messages.find((item) => item.id === messageId);
      if (message === undefined) {
        throw new Error('找不到訊息。');
      }
      if (receiptType === 'read') {
        snapshot = {
          ...snapshot,
          conversations: snapshot.conversations.map((conversation) =>
            conversation.id === message.conversationId
              ? { ...conversation, unreadCount: 0 }
              : conversation,
          ),
        };
        emit();
      }
      // 回條記帳（§2r）：同一訊息＋使用者＋型別只記一則（冪等，
      // 與後端一致）；回傳既有列或新列。
      const receipt: ReceiptItem = {
        messageId,
        userId: 'me',
        receiptType,
        deviceId: '',
        createdAt: new Date().toISOString(),
      };
      const existing = demoReceipts.find(
        (item) =>
          item.messageId === messageId &&
          item.userId === receipt.userId &&
          item.receiptType === receiptType,
      );
      if (existing === undefined) {
        demoReceipts = [...demoReceipts, receipt];
      }
      return existing ?? receipt;
    },
    listReceiptsAsync: async (messageId: string): Promise<ReceiptItem[]> => {
      if (snapshot.messages.find((item) => item.id === messageId) === undefined) {
        throw new Error('找不到訊息。');
      }
      return demoReceipts.filter((item) => item.messageId === messageId);
    },
    listReactionsAsync: async (messageId: string): Promise<ReactionSummaryItem> => {
      if (snapshot.messages.find((item) => item.id === messageId) === undefined) {
        throw new Error('找不到訊息。');
      }
      return demoReactionSummary(messageId);
    },
    setReactionAsync: async (messageId: string, reaction: string): Promise<ReactionSummaryItem> => {
      const message = snapshot.messages.find((item) => item.id === messageId);
      if (message === undefined) {
        throw new Error('找不到訊息。');
      }
      if (!REACTION_EMOJIS.includes(reaction)) {
        throw new Error('不支援的反應表情。');
      }
      demoRequireReactable(message, '反應');
      const byUser = demoReactions.get(messageId) ?? new Map<string, string>();
      byUser.set('me', reaction);
      demoReactions.set(messageId, byUser);
      return demoReactionSummary(messageId);
    },
    removeReactionAsync: async (messageId: string): Promise<ReactionSummaryItem> => {
      if (snapshot.messages.find((item) => item.id === messageId) === undefined) {
        throw new Error('找不到訊息。');
      }
      demoReactions.get(messageId)?.delete('me');
      return demoReactionSummary(messageId);
    },
    listPinsAsync: async (conversationId: string): Promise<PinItem[]> => {
      if (snapshot.conversations.find((item) => item.id === conversationId) === undefined) {
        throw new Error('找不到對話。');
      }
      const pins = demoPins.get(conversationId) ?? new Map<string, string>();
      return [...pins.entries()]
        .map(([messageId, pinnedAt]) => ({
          id: `pin-local-${conversationId}-${messageId}`,
          conversationId,
          messageId,
          pinnedBy: 'me',
          pinnedAt,
          unpinnedAt: null,
        }))
        .sort((a, b) => a.pinnedAt.localeCompare(b.pinnedAt));
    },
    pinMessageAsync: async (conversationId: string, messageId: string): Promise<PinItem> => {
      const message = snapshot.messages.find((item) => item.id === messageId);
      if (
        message === undefined ||
        message.conversationId !== conversationId ||
        snapshot.conversations.find((item) => item.id === conversationId) === undefined
      ) {
        throw new Error('找不到訊息。');
      }
      demoRequireReactable(message, '置頂');
      const pins = demoPins.get(conversationId) ?? new Map<string, string>();
      const existing = pins.get(messageId);
      if (existing !== undefined) {
        return {
          id: `pin-local-${conversationId}-${messageId}`,
          conversationId,
          messageId,
          pinnedBy: 'me',
          pinnedAt: existing,
          unpinnedAt: null,
        };
      }
      if (pins.size >= 5) {
        throw new Error('置頂訊息已達上限（5 則），請先取消一則置頂。');
      }
      const pinnedAt = new Date().toISOString();
      pins.set(messageId, pinnedAt);
      demoPins.set(conversationId, pins);
      return {
        id: `pin-local-${conversationId}-${messageId}`,
        conversationId,
        messageId,
        pinnedBy: 'me',
        pinnedAt,
        unpinnedAt: null,
      };
    },
    unpinMessageAsync: async (conversationId: string, messageId: string): Promise<void> => {
      demoPins.get(conversationId)?.delete(messageId);
    },
    searchMessagesAsync: async (input: MessageSearchInput): Promise<MessageSearchPage> => {
      // 展示搜尋（§2v）：與後端相同的比對語義——AND、子字串、
      // 大小寫不敏感、建立時間降序＋識別碼降序、游標分頁。
      // 收回與僅自己刪除的訊息不參與比對：收回訊息的內容已自
      // 索引移除，僅自己刪除的內容在目前檢視中為空。
      const terms = normalizeSearchTerms(input.query);
      if (terms.length === 0) {
        throw new Error('關鍵字不可為空。');
      }
      if ([...input.query].length > 200) {
        throw new Error('關鍵字過長。');
      }
      const limit = input.limit === undefined ? 20 : Math.min(Math.max(input.limit, 1), 50);
      const sinceMs = input.since === undefined ? null : Date.parse(input.since);
      const untilMs = input.until === undefined ? null : Date.parse(input.until);
      const cursor = input.cursor === undefined ? null : decodeSearchCursor(input.cursor);
      const candidates: MessageSearchResultItem[] = [];
      for (const message of snapshot.messages) {
        if (demoRecalled.has(message.id) || demoDeleted.has(message.id)) {
          continue;
        }
        if (input.conversationId !== undefined && message.conversationId !== input.conversationId) {
          continue;
        }
        if (input.senderId !== undefined && message.senderId !== input.senderId) {
          continue;
        }
        const sentMs = Date.parse(message.sentAt);
        if (sinceMs !== null && !Number.isNaN(sinceMs) && sentMs < sinceMs) {
          continue;
        }
        if (untilMs !== null && !Number.isNaN(untilMs) && sentMs > untilMs) {
          continue;
        }
        const content = message.body;
        if (!messageMatchesSearchTerms(content, terms)) {
          continue;
        }
        const sequence =
          conversationSequence(message.conversationId).findIndex((item) => item.id === message.id) +
          1;
        candidates.push({
          id: message.id,
          conversationId: message.conversationId,
          senderId: message.senderId,
          kind: message.kind,
          content,
          sequenceNumber: sequence,
          createdAt: message.sentAt,
        });
      }
      candidates.sort(compareSearchResults);
      const afterCursor =
        cursor === null
          ? candidates
          : candidates.filter((item) => {
              const itemMs = Date.parse(item.createdAt);
              const cursorMs = Date.parse(cursor.createdAt);
              if (itemMs < cursorMs) {
                return true;
              }
              return itemMs === cursorMs && item.id < cursor.messageId;
            });
      const page = afterCursor.slice(0, limit);
      const last = page[page.length - 1];
      return {
        results: page,
        nextCursor:
          last !== undefined && afterCursor.length > limit
            ? encodeSearchCursor(last.createdAt, last.id)
            : '',
      };
    },
    listAttachmentsAsync: async (messageId: string): Promise<MessageAttachmentItem[]> => {
      if (snapshot.messages.find((item) => item.id === messageId) === undefined) {
        throw new Error('找不到訊息。');
      }
      return (demoAttachments.get(messageId) ?? []).map((item) => ({ ...item }));
    },
    addAttachmentsAsync: async (
      messageId: string,
      mediaFileIds: string[],
    ): Promise<MessageAttachmentItem[]> => {
      const message = snapshot.messages.find((item) => item.id === messageId);
      if (message === undefined) {
        throw new Error('找不到訊息。');
      }
      if (message.senderId !== 'me') {
        throw new Error('只有作者可以為訊息新增附件。');
      }
      demoRequireReactable(message, '附件');
      return demoAttachFiles(messageId, mediaFileIds);
    },
    forwardMessageAsync: async (
      messageId: string,
      targetConversationId: string,
    ): Promise<MessageItem> => {
      const source = snapshot.messages.find((item) => item.id === messageId);
      const target = snapshot.conversations.find((item) => item.id === targetConversationId);
      if (source === undefined || target === undefined) {
        throw new Error('找不到訊息或對話。');
      }
      if (demoRecalled.has(source.id)) {
        throw new Error('收回的訊息不可轉傳。');
      }
      if (source.senderId === 'me' && demoDeleted.has(source.id)) {
        throw new Error('已刪除的訊息不可轉傳。');
      }
      const depth = (demoForwardDepth.get(source.id) ?? 1) + 1;
      if (depth > 5) {
        throw new Error('轉傳鏈深度已達上限（5）。');
      }
      const forwarded = appendMessage(targetConversationId, source.body);
      if (forwarded === null) {
        throw new Error('訊息不可為空。');
      }
      const now = new Date().toISOString();
      demoForwards.set(forwarded.id, {
        id: `fwd-local-${forwarded.id}`,
        sourceMessageId: source.id,
        forwardedMessageId: forwarded.id,
        sourceConversationId: source.conversationId,
        targetConversationId,
        forwardedBy: 'me',
        createdAt: now,
      });
      demoForwardDepth.set(forwarded.id, depth);
      const sequence = conversationSequence(targetConversationId).findIndex(
        (item) => item.id === forwarded.id,
      );
      return toMessageItem(forwarded, sequence + 1);
    },
    getForwardAsync: async (messageId: string): Promise<ForwardRecordItem> => {
      const record = demoForwards.get(messageId);
      if (record === undefined) {
        throw new Error('找不到轉傳紀錄。');
      }
      return { ...record };
    },
    listGroupsAsync: async (): Promise<GroupItem[]> =>
      demoGroups.filter((item) => !item.isDisbanded).map(demoGroupView),
    createGroupAsync: async (input: {
      name: string;
      description?: string;
      avatarMediaId?: string;
      memberIds?: string[];
    }): Promise<GroupItem> => {
      const name = input.name.trim();
      if (name === '') {
        throw new Error('請輸入群組名稱。');
      }
      const now = new Date().toISOString();
      demoGroupCounter += 1;
      const groupId = `demo-group-local-${demoGroupCounter}`;
      const memberIds = [...new Set((input.memberIds ?? []).map(demoResolveUserId))].filter(
        (id) => id !== '' && id !== 'me',
      );
      const members: GroupMemberItem[] = [
        {
          userId: 'me',
          displayName: demoContactName('me'),
          role: 'owner',
          joinedAt: now,
          invitedBy: 'me',
          leftAt: null,
        },
        ...memberIds.map((userId) => ({
          userId,
          displayName: demoContactName(userId),
          role: 'member',
          joinedAt: now,
          invitedBy: 'me',
          leftAt: null,
        })),
      ];
      const group: GroupItem = {
        id: groupId,
        name,
        description: input.description ?? '',
        avatarMediaId: input.avatarMediaId ?? '',
        conversationId: '',
        createdBy: 'me',
        isDisbanded: false,
        memberCount: members.length,
        createdAt: now,
        updatedAt: now,
      };
      demoGroups = [group, ...demoGroups];
      demoGroupMembers = { ...demoGroupMembers, [groupId]: members };
      demoGroupSettings = [
        ...demoGroupSettings,
        { groupId, invitePolicy: 'admin', updatedBy: 'me', updatedAt: now },
      ];
      emit();
      return demoGroupView(group);
    },
    getGroupAsync: async (groupId: string): Promise<GroupItem> =>
      demoGroupView(demoFindGroup(groupId)),
    updateGroupAsync: async (
      groupId: string,
      patch: { name?: string; description?: string; avatarMediaId?: string },
    ): Promise<GroupItem> => {
      demoRequireGroupManager(groupId);
      const group = demoFindGroup(groupId);
      const next = { ...group };
      if (patch.name !== undefined) {
        if (patch.name.trim() === '') {
          throw new Error('群組名稱不可為空。');
        }
        next.name = patch.name.trim();
      }
      if (patch.description !== undefined) {
        next.description = patch.description;
      }
      if (patch.avatarMediaId !== undefined) {
        next.avatarMediaId = patch.avatarMediaId;
      }
      next.updatedAt = new Date().toISOString();
      demoGroups = demoGroups.map((item) => (item.id === groupId ? next : item));
      emit();
      return demoGroupView(next);
    },
    disbandGroupAsync: async (groupId: string): Promise<void> => {
      demoRequireGroupOwner(groupId);
      const now = new Date().toISOString();
      demoGroups = demoGroups.map((item) =>
        item.id === groupId ? { ...item, isDisbanded: true, updatedAt: now } : item,
      );
      emit();
    },
    transferGroupOwnershipAsync: async (
      groupId: string,
      newOwnerId: string,
    ): Promise<GroupItem> => {
      demoRequireGroupOwner(groupId);
      const members = demoGroupMembers[groupId] ?? [];
      const target = members.find((item) => item.userId === newOwnerId && item.leftAt === null);
      if (target === undefined || newOwnerId === 'me') {
        throw new Error('找不到群組。');
      }
      demoGroupMembers = {
        ...demoGroupMembers,
        [groupId]: members.map((item) => {
          if (item.userId === 'me') {
            return { ...item, role: 'admin' };
          }
          if (item.userId === newOwnerId) {
            return { ...item, role: 'owner' };
          }
          return item;
        }),
      };
      const group = demoFindGroup(groupId);
      const next = { ...group, updatedAt: new Date().toISOString() };
      demoGroups = demoGroups.map((item) => (item.id === groupId ? next : item));
      emit();
      return demoGroupView(next);
    },
    listGroupMembersAsync: async (groupId: string): Promise<GroupMemberItem[]> => {
      demoRequireGroupMember(groupId);
      return (demoGroupMembers[groupId] ?? []).map((item) => ({ ...item }));
    },
    addGroupMembersAsync: async (
      groupId: string,
      memberIds: string[],
    ): Promise<GroupMemberItem[]> => {
      demoRequireGroupManager(groupId);
      const now = new Date().toISOString();
      const members = [...(demoGroupMembers[groupId] ?? [])];
      const added: GroupMemberItem[] = [];
      for (const raw of memberIds) {
        const userId = demoResolveUserId(raw);
        if (userId === '' || userId === 'me') {
          continue;
        }
        const existing = members.find((item) => item.userId === userId);
        if (existing !== undefined) {
          if (existing.leftAt !== null) {
            existing.leftAt = null;
            existing.role = 'member';
            existing.joinedAt = now;
            existing.invitedBy = 'me';
            added.push({ ...existing });
          }
          continue;
        }
        const member: GroupMemberItem = {
          userId,
          displayName: demoContactName(userId),
          role: 'member',
          joinedAt: now,
          invitedBy: 'me',
          leftAt: null,
        };
        members.push(member);
        added.push({ ...member });
      }
      demoGroupMembers = { ...demoGroupMembers, [groupId]: members };
      emit();
      return added;
    },
    setGroupMemberRoleAsync: async (
      groupId: string,
      userId: string,
      role: string,
    ): Promise<GroupMemberItem> => {
      demoRequireGroupOwner(groupId);
      if (role !== 'admin' && role !== 'member') {
        throw new Error('角色須為管理員或一般成員。');
      }
      const members = demoGroupMembers[groupId] ?? [];
      const target = members.find((item) => item.userId === userId && item.leftAt === null);
      if (target === undefined || userId === 'me' || target.role === 'owner') {
        throw new Error('找不到群組。');
      }
      const next = { ...target, role };
      demoGroupMembers = {
        ...demoGroupMembers,
        [groupId]: members.map((item) => (item.userId === userId ? next : item)),
      };
      emit();
      return { ...next };
    },
    removeGroupMemberAsync: async (groupId: string, userId: string): Promise<void> => {
      demoRequireGroupManager(groupId);
      if (userId === 'me') {
        throw new Error('不得移除自己，請使用離開流程。');
      }
      const members = demoGroupMembers[groupId] ?? [];
      const target = members.find((item) => item.userId === userId && item.leftAt === null);
      const selfRole = demoMyGroupRole(groupId);
      if (
        target === undefined ||
        target.role === 'owner' ||
        (selfRole === 'admin' && target.role === 'admin')
      ) {
        throw new Error('找不到群組。');
      }
      demoGroupMembers = {
        ...demoGroupMembers,
        [groupId]: members.map((item) =>
          item.userId === userId ? { ...item, leftAt: new Date().toISOString() } : item,
        ),
      };
      emit();
    },
    leaveGroupAsync: async (groupId: string): Promise<void> => {
      demoRequireGroupMember(groupId);
      if (demoMyGroupRole(groupId) === 'owner') {
        throw new Error('擁有者離開前須先轉移所有權或解散群組。');
      }
      demoGroupMembers = {
        ...demoGroupMembers,
        [groupId]: (demoGroupMembers[groupId] ?? []).map((item) =>
          item.userId === 'me' ? { ...item, leftAt: new Date().toISOString() } : item,
        ),
      };
      emit();
    },
    listGroupInvitationsAsync: async (groupId: string): Promise<GroupInvitationItem[]> => {
      demoRequireGroupManager(groupId);
      return demoGroupInvitations
        .filter((item) => item.groupId === groupId && item.status === 'pending')
        .map((item) => ({ ...item }));
    },
    createGroupInvitationAsync: async (
      groupId: string,
      inviteeId: string,
    ): Promise<GroupInvitationItem> => {
      demoRequireGroupMember(groupId);
      const role = demoMyGroupRole(groupId);
      const settings = demoGroupSettings.find((item) => item.groupId === groupId);
      const canInvite = role === 'owner' || role === 'admin' || settings?.invitePolicy === 'all';
      if (!canInvite) {
        throw new Error('找不到群組。');
      }
      const resolved = demoResolveUserId(inviteeId);
      if (resolved === '' || resolved === 'me') {
        throw new Error('請提供被邀請者識別碼。');
      }
      const members = demoGroupMembers[groupId] ?? [];
      if (members.some((item) => item.userId === resolved && item.leftAt === null)) {
        throw new Error('對方已是群組成員。');
      }
      if (
        demoGroupInvitations.some(
          (item) =>
            item.groupId === groupId && item.inviteeId === resolved && item.status === 'pending',
        )
      ) {
        throw new Error('已送出待處理的群組邀請。');
      }
      const group = demoFindGroup(groupId);
      const now = new Date().toISOString();
      demoGroupCounter += 1;
      const invitation: GroupInvitationItem = {
        id: `demo-group-inv-${demoGroupCounter}`,
        groupId,
        groupName: group.name,
        inviterId: 'me',
        inviteeId: resolved,
        status: 'pending',
        createdAt: now,
        updatedAt: now,
      };
      demoGroupInvitations = [...demoGroupInvitations, invitation];
      emit();
      return { ...invitation };
    },
    listMyGroupInvitationsAsync: async (): Promise<GroupInvitationItem[]> =>
      demoGroupInvitations
        .filter((item) => item.inviteeId === 'me' && item.status === 'pending')
        .map((item) => ({ ...item })),
    respondGroupInvitationAsync: async (
      invitationId: string,
      accept: boolean,
    ): Promise<GroupInvitationItem> => {
      const invitation = demoGroupInvitations.find((item) => item.id === invitationId);
      if (
        invitation === undefined ||
        invitation.inviteeId !== 'me' ||
        invitation.status !== 'pending'
      ) {
        throw new Error('找不到邀請。');
      }
      const now = new Date().toISOString();
      const next = { ...invitation, status: accept ? 'accepted' : 'declined', updatedAt: now };
      demoGroupInvitations = demoGroupInvitations.map((item) =>
        item.id === invitationId ? next : item,
      );
      if (accept) {
        const existing = demoGroups.find((item) => item.id === invitation.groupId);
        if (existing === undefined || existing.isDisbanded) {
          const group: GroupItem = {
            id: invitation.groupId,
            name: invitation.groupName,
            description: '',
            avatarMediaId: '',
            conversationId: '',
            createdBy: invitation.inviterId,
            isDisbanded: false,
            memberCount: 2,
            createdAt: now,
            updatedAt: now,
          };
          demoGroups = [group, ...demoGroups];
          demoGroupMembers = {
            ...demoGroupMembers,
            [invitation.groupId]: [
              {
                userId: invitation.inviterId,
                displayName: demoContactName(invitation.inviterId),
                role: 'owner',
                joinedAt: now,
                invitedBy: invitation.inviterId,
                leftAt: null,
              },
              {
                userId: 'me',
                displayName: demoContactName('me'),
                role: 'member',
                joinedAt: now,
                invitedBy: invitation.inviterId,
                leftAt: null,
              },
            ],
          };
          demoGroupSettings = [
            ...demoGroupSettings,
            {
              groupId: invitation.groupId,
              invitePolicy: 'admin',
              updatedBy: invitation.inviterId,
              updatedAt: now,
            },
          ];
        } else {
          const members = [...(demoGroupMembers[invitation.groupId] ?? [])];
          const self = members.find((item) => item.userId === 'me');
          if (self === undefined) {
            members.push({
              userId: 'me',
              displayName: demoContactName('me'),
              role: 'member',
              joinedAt: now,
              invitedBy: invitation.inviterId,
              leftAt: null,
            });
          } else if (self.leftAt !== null) {
            self.leftAt = null;
            self.role = 'member';
            self.joinedAt = now;
          }
          demoGroupMembers = { ...demoGroupMembers, [invitation.groupId]: members };
        }
      }
      emit();
      return { ...next };
    },
    cancelGroupInvitationAsync: async (invitationId: string): Promise<GroupInvitationItem> => {
      const invitation = demoGroupInvitations.find((item) => item.id === invitationId);
      if (invitation === undefined || invitation.status !== 'pending') {
        throw new Error('找不到邀請。');
      }
      const isManager =
        demoMyGroupRole(invitation.groupId) === 'owner' ||
        demoMyGroupRole(invitation.groupId) === 'admin';
      if (invitation.inviterId !== 'me' && !isManager) {
        throw new Error('找不到邀請。');
      }
      const next = { ...invitation, status: 'cancelled', updatedAt: new Date().toISOString() };
      demoGroupInvitations = demoGroupInvitations.map((item) =>
        item.id === invitationId ? next : item,
      );
      emit();
      return { ...next };
    },
    listGroupAnnouncementsAsync: async (groupId: string): Promise<GroupAnnouncementItem[]> => {
      demoRequireGroupMember(groupId);
      return demoGroupAnnouncements
        .filter((item) => item.groupId === groupId)
        .map((item) => ({ ...item }))
        .sort((a, b) => Date.parse(b.publishedAt) - Date.parse(a.publishedAt));
    },
    createGroupAnnouncementAsync: async (
      groupId: string,
      content: string,
    ): Promise<GroupAnnouncementItem> => {
      demoRequireGroupManager(groupId);
      const trimmed = content.trim();
      if ([...trimmed].length === 0 || [...trimmed].length > 1000) {
        throw new Error('公告內容須為 1 至 1000 字。');
      }
      const now = new Date().toISOString();
      demoGroupAnnouncements = demoGroupAnnouncements.map((item) =>
        item.groupId === groupId && item.isActive
          ? { ...item, isActive: false, retractedAt: now, updatedAt: now }
          : item,
      );
      demoGroupCounter += 1;
      const announcement: GroupAnnouncementItem = {
        id: `demo-group-ann-${demoGroupCounter}`,
        groupId,
        authorId: 'me',
        content: trimmed,
        isActive: true,
        publishedAt: now,
        retractedAt: null,
        createdAt: now,
        updatedAt: now,
      };
      demoGroupAnnouncements = [announcement, ...demoGroupAnnouncements];
      emit();
      return { ...announcement };
    },
    updateGroupAnnouncementAsync: async (
      groupId: string,
      announcementId: string,
      content: string,
    ): Promise<GroupAnnouncementItem> => {
      demoRequireGroupManager(groupId);
      const trimmed = content.trim();
      if ([...trimmed].length === 0 || [...trimmed].length > 1000) {
        throw new Error('公告內容須為 1 至 1000 字。');
      }
      const target = demoGroupAnnouncements.find(
        (item) => item.id === announcementId && item.groupId === groupId && item.isActive,
      );
      if (target === undefined) {
        throw new Error('找不到公告。');
      }
      const next = { ...target, content: trimmed, updatedAt: new Date().toISOString() };
      demoGroupAnnouncements = demoGroupAnnouncements.map((item) =>
        item.id === announcementId ? next : item,
      );
      emit();
      return { ...next };
    },
    retractGroupAnnouncementAsync: async (
      groupId: string,
      announcementId: string,
    ): Promise<GroupAnnouncementItem> => {
      demoRequireGroupManager(groupId);
      const target = demoGroupAnnouncements.find(
        (item) => item.id === announcementId && item.groupId === groupId && item.isActive,
      );
      if (target === undefined) {
        throw new Error('找不到公告。');
      }
      const now = new Date().toISOString();
      const next = { ...target, isActive: false, retractedAt: now, updatedAt: now };
      demoGroupAnnouncements = demoGroupAnnouncements.map((item) =>
        item.id === announcementId ? next : item,
      );
      emit();
      return { ...next };
    },
    getGroupSettingsAsync: async (groupId: string): Promise<GroupSettingsItem> => {
      demoRequireGroupMember(groupId);
      const settings = demoGroupSettings.find((item) => item.groupId === groupId);
      return settings !== undefined
        ? { ...settings }
        : { groupId, invitePolicy: 'admin', updatedBy: '', updatedAt: null };
    },
    updateGroupSettingsAsync: async (
      groupId: string,
      invitePolicy: string,
    ): Promise<GroupSettingsItem> => {
      demoRequireGroupManager(groupId);
      if (invitePolicy !== 'admin' && invitePolicy !== 'all') {
        throw new Error('邀請政策須為 admin 或 all。');
      }
      const now = new Date().toISOString();
      const existing = demoGroupSettings.find((item) => item.groupId === groupId);
      const next: GroupSettingsItem =
        existing !== undefined
          ? { ...existing, invitePolicy, updatedBy: 'me', updatedAt: now }
          : { groupId, invitePolicy, updatedBy: 'me', updatedAt: now };
      demoGroupSettings = [...demoGroupSettings.filter((item) => item.groupId !== groupId), next];
      emit();
      return { ...next };
    },
    listMediaFilesAsync: async (): Promise<MediaFileItem[]> =>
      demoMediaFiles.map((item) => ({ ...item })),
    listMediaUploadsAsync: async (): Promise<MediaUploadItem[]> =>
      demoMediaUploads.map((item) => ({ ...item })),
    initiateMediaUploadAsync: async (input: InitiateUploadInput): Promise<MediaUploadItem> => {
      demoMediaCounter += 1;
      const now = new Date().toISOString();
      const id = `demo-upload-local-${demoMediaCounter}`;
      const issued = demoUploadUrl(id);
      const session: MediaUploadItem = {
        id,
        mediaFileId: '',
        status: 'uploading',
        storageKey: `media/demo/${id}`,
        externalUploadId: '',
        uploadedBytes: 0,
        totalBytes: input.totalBytes,
        partCount: 0,
        uploadUrl: issued.uploadUrl,
        uploadUrlExpiresAt: issued.expiresAt,
        completedAt: null,
        createdAt: now,
        updatedAt: now,
      };
      demoMediaUploads = [session, ...demoMediaUploads];
      emit();
      return { ...session };
    },
    issueMediaUploadUrlAsync: async (uploadId: string): Promise<MediaUploadUrl> => {
      const target = demoMediaUploads.find((item) => item.id === uploadId);
      if (target === undefined || target.status !== 'uploading') {
        throw new Error('找不到進行中的上傳會話。');
      }
      return demoUploadUrl(uploadId);
    },
    putMediaUploadBytesAsync: async (uploadUrl: string, data: Blob): Promise<void> => {
      // 僅接受本機 pseudo-URL；位元組大小須等於宣告總量，否則
      // 拒絕寫入（與後端中繼端點的 fail-closed 語義一致）。
      const match = /^demo:\/\/uploads\/(.+)$/.exec(uploadUrl);
      const target =
        match === null
          ? undefined
          : demoMediaUploads.find((item) => item.id === match[1] && item.status === 'uploading');
      if (target === undefined) {
        throw new Error('找不到進行中的上傳會話。');
      }
      if (data.size !== target.totalBytes) {
        throw new Error('位元組大小須等於宣告總量。');
      }
      demoMediaBytes.set(target.id, data);
    },
    reportMediaUploadProgressAsync: async (
      uploadId: string,
      uploadedBytes: number,
      partCount: number,
    ): Promise<MediaUploadItem> => {
      const target = demoMediaUploads.find((item) => item.id === uploadId);
      if (target === undefined || target.status !== 'uploading') {
        throw new Error('找不到進行中的上傳會話。');
      }
      if (
        uploadedBytes < target.uploadedBytes ||
        uploadedBytes > target.totalBytes ||
        partCount < 0
      ) {
        throw new Error('上傳進度不合法。');
      }
      const next = {
        ...target,
        uploadedBytes,
        partCount,
        updatedAt: new Date().toISOString(),
      };
      demoMediaUploads = demoMediaUploads.map((item) => (item.id === uploadId ? next : item));
      emit();
      return { ...next };
    },
    completeMediaUploadAsync: async (
      uploadId: string,
      input: InitiateUploadInput,
    ): Promise<MediaFileItem> => {
      const target = demoMediaUploads.find((item) => item.id === uploadId);
      if (target === undefined) {
        throw new Error('找不到上傳會話。');
      }
      if (target.status === 'completed' && target.mediaFileId !== '') {
        const existing = demoMediaFiles.find((item) => item.id === target.mediaFileId);
        if (existing !== undefined) {
          return { ...existing };
        }
        throw new Error('找不到檔案。');
      }
      if (target.status !== 'uploading') {
        throw new Error('上傳會話已不在進行中。');
      }
      // Fail-closed：無位元組或大小與回報進度不一致時不得完成
      // （與後端 Stat 校驗語義一致）。
      const stored = demoMediaBytes.get(uploadId);
      if (
        stored === undefined ||
        stored.size !== target.totalBytes ||
        target.uploadedBytes !== target.totalBytes
      ) {
        throw new Error('物件尚未上傳，不得完成。');
      }
      demoMediaCounter += 1;
      const now = new Date().toISOString();
      const file: MediaFileItem = {
        id: `demo-file-local-${demoMediaCounter}`,
        mimeType: input.mimeType,
        sizeBytes: stored.size,
        originalFilename: input.originalFilename,
        width: input.width,
        height: input.height,
        durationSeconds: input.durationSeconds,
        checksum: input.checksum,
        scanStatus: 'pending',
        scanCompletedAt: null,
        createdAt: now,
        updatedAt: now,
      };
      demoMediaFiles = [file, ...demoMediaFiles];
      demoMediaBytes.set(file.id, stored);
      demoMediaBytes.delete(uploadId);
      const next = {
        ...target,
        mediaFileId: file.id,
        status: 'completed',
        uploadUrl: null,
        uploadUrlExpiresAt: null,
        completedAt: now,
        updatedAt: now,
      };
      demoMediaUploads = demoMediaUploads.map((item) => (item.id === uploadId ? next : item));
      emit();
      return { ...file };
    },
    cancelMediaUploadAsync: async (uploadId: string): Promise<MediaUploadItem> => {
      const target = demoMediaUploads.find((item) => item.id === uploadId);
      if (target === undefined || target.status !== 'uploading') {
        throw new Error('找不到進行中的上傳會話。');
      }
      demoMediaBytes.delete(uploadId);
      const next = {
        ...target,
        status: 'cancelled',
        uploadUrl: null,
        uploadUrlExpiresAt: null,
        updatedAt: new Date().toISOString(),
      };
      demoMediaUploads = demoMediaUploads.map((item) => (item.id === uploadId ? next : item));
      emit();
      return { ...next };
    },
    deleteMediaFileAsync: async (fileId: string): Promise<void> => {
      if (!demoMediaFiles.some((item) => item.id === fileId)) {
        throw new Error('找不到檔案。');
      }
      demoMediaFiles = demoMediaFiles.filter((item) => item.id !== fileId);
      demoMediaBytes.delete(fileId);
      emit();
    },
    listMediaThumbnailsAsync: async (fileId: string): Promise<MediaThumbnailItem[]> =>
      DEMO_MEDIA_THUMBNAILS.filter((item) => item.fileId === fileId).map((item) => ({
        ...item,
      })),
    authorizeMediaAccessAsync: async (
      fileId: string,
      accessType: 'download' | 'preview' | 'thumbnail',
    ): Promise<MediaAuthorization> => {
      // 掃描閘門適用全部存取類型（與後端一致），故不依類型分流。
      void accessType;
      return demoAuthorizeMedia(fileId);
    },
    resolveMediaDownloadAsync: async (fileId: string): Promise<MediaDownloadResolution> => {
      const authorization = demoAuthorizeMedia(fileId);
      if (!authorization.allowed) {
        return { status: 'forbidden', scanStatus: authorization.scanStatus };
      }
      const stored = demoMediaBytes.get(fileId);
      if (stored === undefined) {
        // 展示 fixture 沒有位元組：已授權但無可分發內容，不得
        // 假裝可下載。
        return { status: 'unavailable' };
      }
      return { status: 'ready-blob', blob: stored };
    },
    listFeedAsync: async (options: { cursor?: string; limit?: number } = {}): Promise<PostPage> =>
      demoPostPage(
        demoPosts.filter((post) => !demoDeletedPostIds.has(post.id) && demoCanViewPost(post)),
        options.cursor ?? '',
        options.limit ?? 0,
      ),
    listAuthorPostsAsync: async (
      authorId: string,
      options: { cursor?: string; limit?: number } = {},
    ): Promise<PostPage> => {
      const knownAuthor =
        authorId === 'me' || DEMO_CONTACTS.some((contact) => contact.id === authorId);
      if (!knownAuthor) {
        throw new Error('找不到貼文。');
      }
      return demoPostPage(
        demoPosts.filter(
          (post) =>
            post.authorId === authorId && !demoDeletedPostIds.has(post.id) && demoCanViewPost(post),
        ),
        options.cursor ?? '',
        options.limit ?? 0,
      );
    },
    getPostAsync: async (postId: string): Promise<PostItem> => {
      const post = demoVisiblePost(postId);
      if (post === null) {
        throw new Error('找不到貼文。');
      }
      return demoPostView(post);
    },
    createPostAsync: async (input: CreatePostInput): Promise<PostItem> => {
      const content = input.content;
      if ([...content].length > 2000) {
        throw new Error('貼文內容不得超過 2000 字。');
      }
      const visibility = input.visibility ?? 'friends';
      if (visibility !== 'public' && visibility !== 'friends' && visibility !== 'only_me') {
        throw new Error('能見度須為 public、friends 或 only_me。');
      }
      const attached =
        input.mediaFileIds !== undefined && input.mediaFileIds.length > 0
          ? demoResolveAttachable(input.mediaFileIds)
          : [];
      if (attached.length > 9) {
        throw new Error('單則貼文最多關聯 9 個媒體檔案。');
      }
      if (content.trim() === '' && attached.length === 0) {
        throw new Error('貼文內容與媒體不得皆為空。');
      }
      demoPostCounter += 1;
      const now = new Date().toISOString();
      const post: PostItem = {
        id: `demo-post-local-${demoPostCounter}`,
        authorId: 'me',
        content,
        visibility,
        editedAt: null,
        media: attached.map((file, index) => ({
          mediaFileId: file.id,
          position: index,
          caption: '',
        })),
        likeCount: 0,
        commentCount: 0,
        likedByMe: false,
        createdAt: now,
        updatedAt: now,
      };
      demoPosts = [post, ...demoPosts];
      emit();
      return demoPostView(post);
    },
    updatePostAsync: async (postId: string, input: UpdatePostInput): Promise<PostItem> => {
      const post = demoPosts.find((item) => item.id === postId);
      if (post === undefined || demoDeletedPostIds.has(postId) || post.authorId !== 'me') {
        throw new Error('找不到貼文。');
      }
      const next = { ...post, media: post.media.map((media) => ({ ...media })) };
      if (input.content !== undefined) {
        if ([...input.content].length > 2000) {
          throw new Error('貼文內容不得超過 2000 字。');
        }
        next.content = input.content;
      }
      if (input.visibility !== undefined) {
        if (
          input.visibility !== 'public' &&
          input.visibility !== 'friends' &&
          input.visibility !== 'only_me'
        ) {
          throw new Error('能見度須為 public、friends 或 only_me。');
        }
        next.visibility = input.visibility;
      }
      if (next.content.trim() === '' && next.media.length === 0) {
        throw new Error('貼文內容與媒體不得皆為空。');
      }
      const now = new Date().toISOString();
      next.editedAt = now;
      next.updatedAt = now;
      demoPosts = demoPosts.map((item) => (item.id === postId ? next : item));
      emit();
      return demoPostView(next);
    },
    deletePostAsync: async (postId: string): Promise<void> => {
      const post = demoPosts.find((item) => item.id === postId);
      if (post === undefined || demoDeletedPostIds.has(postId) || post.authorId !== 'me') {
        throw new Error('找不到貼文。');
      }
      demoDeletedPostIds.add(postId);
      emit();
    },
    listPostMediaAsync: async (postId: string): Promise<PostMediaItem[]> => {
      const post = demoVisiblePost(postId);
      if (post === null) {
        throw new Error('找不到貼文。');
      }
      return post.media.map((media) => ({ ...media }));
    },
    likePostAsync: async (postId: string): Promise<PostLikeResult> => {
      if (demoVisiblePost(postId) === null) {
        throw new Error('找不到貼文。');
      }
      let likers = demoPostLikes.get(postId);
      if (likers === undefined) {
        likers = new Set();
        demoPostLikes.set(postId, likers);
      }
      likers.add('me');
      emit();
      return { postId, liked: true, likeCount: likers.size };
    },
    unlikePostAsync: async (postId: string): Promise<PostLikeResult> => {
      if (demoVisiblePost(postId) === null) {
        throw new Error('找不到貼文。');
      }
      const likers = demoPostLikes.get(postId);
      likers?.delete('me');
      emit();
      return { postId, liked: false, likeCount: likers?.size ?? 0 };
    },
    createPostCommentAsync: async (
      postId: string,
      content: string,
      parentCommentId = '',
    ): Promise<PostCommentItem> => {
      if (demoVisiblePost(postId) === null) {
        throw new Error('找不到貼文。');
      }
      if ([...content].length > 1000) {
        throw new Error('留言內容不得超過 1000 字。');
      }
      if (content.trim() === '') {
        throw new Error('留言內容不得為空。');
      }
      if (parentCommentId !== '') {
        const parent = demoPostComments.find((comment) => comment.id === parentCommentId);
        if (parent === undefined || parent.postId !== postId) {
          throw new Error('回覆的留言不存在或不屬於此貼文。');
        }
      }
      demoCommentCounter += 1;
      const now = new Date().toISOString();
      const comment: PostCommentItem = {
        id: `demo-comment-local-${demoCommentCounter}`,
        postId,
        authorId: 'me',
        parentCommentId,
        content,
        isDeleted: false,
        editedAt: null,
        createdAt: now,
        updatedAt: now,
      };
      demoPostComments = [...demoPostComments, comment];
      emit();
      return { ...comment };
    },
    listPostCommentsAsync: async (
      postId: string,
      options: { cursor?: string; limit?: number } = {},
    ): Promise<PostCommentPage> => {
      if (demoVisiblePost(postId) === null) {
        throw new Error('找不到貼文。');
      }
      return demoCommentPage(postId, options.cursor ?? '', options.limit ?? 0);
    },
    deletePostCommentAsync: async (commentId: string): Promise<void> => {
      const comment = demoPostComments.find((item) => item.id === commentId);
      if (
        comment === undefined ||
        comment.isDeleted ||
        comment.authorId !== 'me' ||
        demoVisiblePost(comment.postId) === null
      ) {
        throw new Error('找不到留言。');
      }
      demoPostComments = demoPostComments.map((item) =>
        item.id === commentId
          ? { ...item, isDeleted: true, updatedAt: new Date().toISOString() }
          : item,
      );
      emit();
    },
    listCallsAsync: async (
      options: { cursor?: string; limit?: number } = {},
    ): Promise<CallPage> => {
      const limit = options.limit ?? 0;
      const take = limit > 0 && limit <= 50 ? limit : 20;
      const cursor = options.cursor ?? '';
      const bound = cursor === '' ? null : demoDecodeCursor(cursor);
      if (cursor !== '' && bound === null) {
        throw new Error('游標格式錯誤。');
      }
      const ordered = demoCalls
        .filter((call) => (demoCallParticipants[call.id] ?? []).some((p) => p.userId === 'me'))
        .sort((a, b) => {
          const timeDelta = Date.parse(b.createdAt) - Date.parse(a.createdAt);
          return timeDelta !== 0 ? timeDelta : b.id.localeCompare(a.id);
        });
      const after =
        bound === null
          ? ordered
          : ordered.filter((call) => {
              const timeDelta = Date.parse(call.createdAt) - Date.parse(bound.createdAt);
              return timeDelta < 0 || (timeDelta === 0 && call.id < bound.id);
            });
      const page = after.slice(0, take);
      const last = page[page.length - 1];
      const nextCursor =
        after.length > take && last !== undefined ? demoEncodeCursor(last.createdAt, last.id) : '';
      return { calls: page.map(demoCallView), nextCursor };
    },
    getCallAsync: async (callId: string): Promise<CallItem> => {
      const call = demoCallForParticipant(callId);
      if (call === null) {
        throw new Error('找不到通話。');
      }
      return demoCallView(call);
    },
    initiateCallAsync: async (input: InitiateCallInput): Promise<CallItem> => {
      if (input.callType !== 'voice' && input.callType !== 'video') {
        throw new Error('通話類型須為 voice 或 video。');
      }
      const conversationId = input.conversationId ?? '';
      const groupId = input.groupId ?? '';
      if ((conversationId === '' && groupId === '') || (conversationId !== '' && groupId !== '')) {
        throw new Error('須指定 conversation_id 或 group_id 其中之一。');
      }
      let inviteeIds: string[] = [];
      if (conversationId !== '') {
        const conversation = snapshot.conversations.find((item) => item.id === conversationId);
        if (conversation === undefined) {
          throw new Error('找不到通話。');
        }
        if (demoBlocks.some((block) => block.userId === conversation.contactId)) {
          throw new Error('找不到通話。');
        }
        inviteeIds = [conversation.contactId];
      } else {
        const group = demoGroups.find((item) => item.id === groupId && !item.isDisbanded);
        const members = demoGroupMembers[groupId] ?? [];
        if (group === undefined || !members.some((m) => m.userId === 'me' && m.leftAt === null)) {
          throw new Error('找不到通話。');
        }
        inviteeIds = members
          .filter((member) => member.userId !== 'me' && member.leftAt === null)
          .map((member) => member.userId);
      }
      if (inviteeIds.length === 0) {
        throw new Error('通話須至少有一名受邀者。');
      }
      const active = demoCalls.find(
        (call) =>
          (call.status === 'ringing' || call.status === 'active') &&
          (conversationId !== ''
            ? call.conversationId === conversationId
            : call.groupId === groupId),
      );
      if (active !== undefined) {
        throw new Error('同一範圍內已有進行中的通話。');
      }
      demoCallCounter += 1;
      const now = new Date().toISOString();
      const call: CallItem = {
        id: `demo-call-local-${demoCallCounter}`,
        conversationId,
        groupId,
        initiatorId: 'me',
        callType: input.callType,
        status: 'ringing',
        startedAt: null,
        endedAt: null,
        durationSeconds: null,
        endedBy: '',
        endReason: '',
        participantCount: inviteeIds.length + 1,
        createdAt: now,
        updatedAt: now,
      };
      demoCalls = [call, ...demoCalls];
      demoCallParticipants[call.id] = [
        {
          userId: 'me',
          joinedAt: now,
          leftAt: null,
          result: 'joined',
          createdAt: now,
        },
        ...inviteeIds.map((userId) => ({
          userId,
          joinedAt: null,
          leftAt: null,
          result: 'pending' as const,
          createdAt: now,
        })),
      ];
      demoRecordCallEvent(call.id, 'me', 'initiated', now);
      emit();
      return demoCallView(call);
    },
    answerCallAsync: async (callId: string): Promise<CallItem> => {
      const call = demoCallForParticipant(callId);
      if (call === null) {
        throw new Error('找不到通話。');
      }
      if (call.status !== 'ringing') {
        throw new Error('通話已不在可接聽狀態。');
      }
      const mine = (demoCallParticipants[call.id] ?? []).find((p) => p.userId === 'me');
      if (mine === undefined || mine.result !== 'pending') {
        throw new Error('參與者已回應此次來電。');
      }
      if (call.initiatorId === 'me') {
        throw new Error('發起人無須接聽自己的來電。');
      }
      const now = new Date().toISOString();
      demoSaveCallParticipant(call.id, { ...mine, joinedAt: now, result: 'joined' });
      const next: CallItem = {
        ...call,
        status: 'active',
        startedAt: call.startedAt ?? now,
        updatedAt: now,
      };
      demoSaveCall(next);
      demoRecordCallEvent(call.id, 'me', 'answered', now);
      emit();
      return demoCallView(next);
    },
    declineCallAsync: async (callId: string): Promise<CallItem> =>
      demoRespondCall(callId, 'declined', 'declined'),
    busyCallAsync: async (callId: string): Promise<CallItem> =>
      demoRespondCall(callId, 'busy', 'busy'),
    endCallAsync: async (callId: string): Promise<CallItem> => {
      const call = demoCallForParticipant(callId);
      if (call === null) {
        throw new Error('找不到通話。');
      }
      if (
        call.status === 'declined' ||
        call.status === 'missed' ||
        call.status === 'busy' ||
        call.status === 'ended'
      ) {
        throw new Error('通話已結束。');
      }
      const mine = (demoCallParticipants[call.id] ?? []).find((p) => p.userId === 'me');
      if (mine === undefined || mine.result !== 'joined') {
        throw new Error('找不到通話。');
      }
      const now = new Date().toISOString();
      const participants = demoCallParticipants[call.id] ?? [];
      const anyJoined = participants.some(
        (p) => p.userId !== call.initiatorId && p.result === 'joined',
      );
      const next: CallItem = {
        ...call,
        status: anyJoined ? 'ended' : 'missed',
        endReason: anyJoined ? 'hangup' : 'missed',
        endedAt: now,
        endedBy: 'me',
        updatedAt: now,
        durationSeconds:
          call.startedAt === null
            ? null
            : Math.max(0, Math.floor((Date.parse(now) - Date.parse(call.startedAt)) / 1000)),
      };
      demoSaveCall(next);
      participants.forEach((participant) => {
        if (participant.result === 'pending') {
          demoSaveCallParticipant(call.id, { ...participant, result: 'missed' });
        } else if (participant.result === 'joined' && participant.leftAt === null) {
          demoSaveCallParticipant(call.id, { ...participant, leftAt: now });
        }
      });
      demoRecordCallEvent(call.id, 'me', 'ended', now);
      emit();
      return demoCallView(next);
    },
    listCallParticipantsAsync: async (callId: string): Promise<CallParticipantItem[]> => {
      if (demoCallForParticipant(callId) === null) {
        throw new Error('找不到通話。');
      }
      return (demoCallParticipants[callId] ?? []).map((item) => ({ ...item }));
    },
    listCallEventsAsync: async (callId: string): Promise<CallEventItem[]> => {
      if (demoCallForParticipant(callId) === null) {
        throw new Error('找不到通話。');
      }
      return [...(demoCallEvents[callId] ?? [])]
        .sort((a, b) => {
          const timeDelta = Date.parse(a.occurredAt) - Date.parse(b.occurredAt);
          return timeDelta !== 0 ? timeDelta : a.id.localeCompare(b.id);
        })
        .map((item) => ({ ...item }));
    },
    listStickerPacksAsync: async (
      options: {
        creatorId?: string;
        category?: string;
        q?: string;
        cursor?: string;
        limit?: number;
      } = {},
    ): Promise<StickerPackPage> => {
      const creatorId = options.creatorId ?? '';
      if (creatorId !== '') {
        const knownAuthor =
          creatorId === 'me' || DEMO_CONTACTS.some((contact) => contact.id === creatorId);
        if (!knownAuthor) {
          throw new Error('找不到貼圖包。');
        }
      }
      const category = (options.category ?? '') as StickerPackCategory;
      if (
        category !== '' &&
        category !== 'general' &&
        category !== 'animals' &&
        category !== 'food' &&
        category !== 'emotions' &&
        category !== 'festivals' &&
        category !== 'characters' &&
        category !== 'other'
      ) {
        throw new Error(
          '分類須為 general、animals、food、emotions、festivals、characters 或 other。',
        );
      }
      const tokens = demoStickerSearchTokens(options.q ?? '');
      const candidates = demoStickerPacks.filter((pack) => {
        if (!demoCanViewPack(pack)) {
          return false;
        }
        if (creatorId !== '' && pack.creatorId !== creatorId) {
          return false;
        }
        if (category !== '' && pack.category !== category) {
          return false;
        }
        if (
          tokens.length > 0 &&
          !tokens.every((token) => `${pack.name} ${pack.description}`.toLowerCase().includes(token))
        ) {
          return false;
        }
        return true;
      });
      const page = demoStickerCursorPage(
        candidates.map((pack) => ({ item: pack, sortAt: pack.createdAt })),
        options.cursor ?? '',
        options.limit ?? 0,
      );
      return { packs: page.items.map(demoStickerPackView), nextCursor: page.nextCursor };
    },
    getStickerPackAsync: async (packId: string): Promise<StickerPackItem> => {
      const pack = demoVisiblePack(packId);
      if (pack === null) {
        throw new Error('找不到貼圖包。');
      }
      return demoStickerPackView(pack);
    },
    createStickerPackAsync: async (input: CreateStickerPackInput): Promise<StickerPackItem> => {
      if (input.name.trim() === '') {
        throw new Error('貼圖包名稱不得為空。');
      }
      if ([...input.name].length > 64) {
        throw new Error('貼圖包名稱過長。');
      }
      if ([...input.description].length > 500) {
        throw new Error('貼圖包描述過長。');
      }
      if (
        input.category !== '' &&
        input.category !== 'general' &&
        input.category !== 'animals' &&
        input.category !== 'food' &&
        input.category !== 'emotions' &&
        input.category !== 'festivals' &&
        input.category !== 'characters' &&
        input.category !== 'other'
      ) {
        throw new Error(
          '分類須為 general、animals、food、emotions、festivals、characters 或 other。',
        );
      }
      if (input.coverMediaId !== '') {
        demoResolveAttachable([input.coverMediaId]);
      }
      demoStickerPackCounter += 1;
      const now = new Date().toISOString();
      const pack: StickerPackItem = {
        id: `demo-pack-local-${demoStickerPackCounter}`,
        creatorId: 'me',
        name: input.name,
        description: input.description,
        category: input.category,
        coverMediaId: input.coverMediaId,
        isFree: input.isFree,
        status: 'draft',
        stickerCount: 0,
        favoritedByMe: false,
        createdAt: now,
        updatedAt: now,
      };
      demoStickerPacks = [pack, ...demoStickerPacks];
      emit();
      return demoStickerPackView(pack);
    },
    updateStickerPackAsync: async (
      packId: string,
      input: UpdateStickerPackInput,
    ): Promise<StickerPackItem> => {
      const pack = demoStickerPacks.find((item) => item.id === packId);
      if (pack === undefined || demoArchivedPackIds.has(packId) || pack.creatorId !== 'me') {
        throw new Error('找不到貼圖包。');
      }
      const next = { ...pack };
      if (input.name !== undefined) {
        if (input.name.trim() === '') {
          throw new Error('貼圖包名稱不得為空。');
        }
        if ([...input.name].length > 64) {
          throw new Error('貼圖包名稱過長。');
        }
        next.name = input.name;
      }
      if (input.description !== undefined) {
        if ([...input.description].length > 500) {
          throw new Error('貼圖包描述過長。');
        }
        next.description = input.description;
      }
      if (input.category !== undefined) {
        if (
          input.category !== '' &&
          input.category !== 'general' &&
          input.category !== 'animals' &&
          input.category !== 'food' &&
          input.category !== 'emotions' &&
          input.category !== 'festivals' &&
          input.category !== 'characters' &&
          input.category !== 'other'
        ) {
          throw new Error(
            '分類須為 general、animals、food、emotions、festivals、characters 或 other。',
          );
        }
        next.category = input.category;
      }
      if (input.coverMediaId !== undefined) {
        if (input.coverMediaId !== '') {
          demoResolveAttachable([input.coverMediaId]);
        }
        next.coverMediaId = input.coverMediaId;
      }
      if (input.status !== undefined) {
        next.status = input.status;
      }
      if (input.isFree !== undefined) {
        next.isFree = input.isFree;
      }
      next.updatedAt = new Date().toISOString();
      demoStickerPacks = demoStickerPacks.map((item) => (item.id === packId ? next : item));
      demoStickers = demoStickers.map((sticker) =>
        sticker.packId === packId ? { ...sticker, packName: next.name } : sticker,
      );
      emit();
      return demoStickerPackView(next);
    },
    deleteStickerPackAsync: async (packId: string): Promise<void> => {
      const pack = demoStickerPacks.find((item) => item.id === packId);
      if (pack === undefined || demoArchivedPackIds.has(packId) || pack.creatorId !== 'me') {
        throw new Error('找不到貼圖包。');
      }
      demoArchivedPackIds.add(packId);
      demoStickerPacks = demoStickerPacks.map((item) =>
        item.id === packId ? { ...item, status: 'archived' as const } : item,
      );
      emit();
    },
    listPackStickersAsync: async (packId: string): Promise<StickerItem[]> => {
      const pack = demoVisiblePack(packId);
      if (pack === null) {
        throw new Error('找不到貼圖包。');
      }
      return demoStickers
        .filter(
          (sticker) =>
            sticker.packId === packId &&
            sticker.status === 'active' &&
            !demoRemovedStickerIds.has(sticker.id),
        )
        .sort((a, b) => a.position - b.position || a.id.localeCompare(b.id))
        .map((sticker) => demoStickerView({ ...sticker, packName: pack.name }));
    },
    addStickerAsync: async (packId: string, input: AddStickerInput): Promise<StickerItem> => {
      const pack = demoStickerPacks.find((item) => item.id === packId);
      if (pack === undefined || demoArchivedPackIds.has(packId) || pack.creatorId !== 'me') {
        throw new Error('找不到貼圖包。');
      }
      if ([...input.name].length > 64) {
        throw new Error('貼圖名稱過長。');
      }
      if (input.mediaFileId === '') {
        throw new Error('貼圖須關聯媒體檔案。');
      }
      const activeCount = demoStickers.filter(
        (sticker) =>
          sticker.packId === packId &&
          sticker.status === 'active' &&
          !demoRemovedStickerIds.has(sticker.id),
      ).length;
      if (activeCount >= 100) {
        throw new Error('單一貼圖包最多 100 張生效貼圖。');
      }
      demoResolveAttachable([input.mediaFileId]);
      const position =
        demoStickers
          .filter((sticker) => sticker.packId === packId)
          .reduce((max, sticker) => Math.max(max, sticker.position), -1) + 1;
      demoStickerCounter += 1;
      const now = new Date().toISOString();
      const sticker: StickerItem = {
        id: `demo-sticker-local-${demoStickerCounter}`,
        packId,
        packName: pack.name,
        mediaFileId: input.mediaFileId,
        name: input.name,
        isAnimated: input.isAnimated,
        position,
        status: 'active',
        createdAt: now,
        updatedAt: now,
      };
      demoStickers = [...demoStickers, sticker];
      emit();
      return demoStickerView(sticker);
    },
    removeStickerAsync: async (stickerId: string): Promise<void> => {
      const sticker = demoStickers.find((item) => item.id === stickerId);
      if (
        sticker === undefined ||
        sticker.status === 'removed' ||
        demoRemovedStickerIds.has(stickerId)
      ) {
        throw new Error('找不到貼圖。');
      }
      const pack = demoStickerPacks.find((item) => item.id === sticker.packId);
      if (pack === undefined || demoArchivedPackIds.has(pack.id) || pack.creatorId !== 'me') {
        throw new Error('找不到貼圖。');
      }
      demoRemovedStickerIds.add(stickerId);
      demoStickers = demoStickers.map((item) =>
        item.id === stickerId ? { ...item, status: 'removed' as const } : item,
      );
      emit();
    },
    favoriteStickerPackAsync: async (packId: string): Promise<StickerPackItem> => {
      const pack = demoVisiblePack(packId);
      if (pack === null) {
        throw new Error('找不到貼圖包。');
      }
      demoStickerFavorites.set(packId, new Date().toISOString());
      emit();
      return demoStickerPackView(pack);
    },
    unfavoriteStickerPackAsync: async (packId: string): Promise<StickerPackItem> => {
      const pack = demoStickerPacks.find((item) => item.id === packId);
      if (pack === undefined) {
        throw new Error('找不到貼圖包。');
      }
      const hadFavorite = demoStickerFavorites.has(packId);
      if (hadFavorite) {
        demoStickerFavorites.delete(packId);
        emit();
        return demoStickerPackView(pack);
      }
      if (demoVisiblePack(packId) === null) {
        throw new Error('找不到貼圖包。');
      }
      emit();
      return demoStickerPackView(pack);
    },
    listFavoriteStickerPacksAsync: async (
      options: { cursor?: string; limit?: number } = {},
    ): Promise<StickerPackPage> => {
      const rows = [...demoStickerFavorites.entries()]
        .map(([packId, favoriteAt]) => {
          const pack = demoVisiblePack(packId);
          return pack === null ? null : { item: pack, sortAt: favoriteAt };
        })
        .filter((row): row is { item: StickerPackItem; sortAt: string } => row !== null);
      const page = demoStickerCursorPage(rows, options.cursor ?? '', options.limit ?? 0);
      return { packs: page.items.map(demoStickerPackView), nextCursor: page.nextCursor };
    },
    recordStickerUseAsync: async (stickerId: string): Promise<StickerItem> => {
      const sticker = demoStickers.find((item) => item.id === stickerId);
      if (
        sticker === undefined ||
        sticker.status === 'removed' ||
        demoRemovedStickerIds.has(stickerId)
      ) {
        throw new Error('找不到貼圖。');
      }
      const pack = demoVisiblePack(sticker.packId);
      if (pack === null) {
        throw new Error('找不到貼圖。');
      }
      const now = new Date().toISOString();
      demoStickerRecent = [
        { stickerId, lastUsedAt: now },
        ...demoStickerRecent.filter((row) => row.stickerId !== stickerId),
      ].slice(0, 30);
      emit();
      return demoStickerView({ ...sticker, packName: pack.name });
    },
    listRecentStickersAsync: async (
      options: { cursor?: string; limit?: number } = {},
    ): Promise<StickerPage> => {
      const rows = demoStickerRecent
        .map((row) => {
          const sticker = demoStickers.find((item) => item.id === row.stickerId);
          if (
            sticker === undefined ||
            sticker.status === 'removed' ||
            demoRemovedStickerIds.has(sticker.id) ||
            demoVisiblePack(sticker.packId) === null
          ) {
            return null;
          }
          return { item: demoStickerView(sticker), sortAt: row.lastUsedAt };
        })
        .filter((row): row is { item: StickerItem; sortAt: string } => row !== null);
      const page = demoStickerCursorPage(rows, options.cursor ?? '', options.limit ?? 0);
      return { stickers: page.items, nextCursor: page.nextCursor };
    },
    searchStickersAsync: async (
      q: string,
      options: { category?: string; cursor?: string; limit?: number } = {},
    ): Promise<StickerPage> => {
      if (q.trim() === '') {
        throw new Error('搜尋關鍵字不得為空。');
      }
      if ([...q].length > 100) {
        throw new Error('搜尋關鍵字過長。');
      }
      const category = (options.category ?? '') as StickerPackCategory;
      if (
        category !== '' &&
        category !== 'general' &&
        category !== 'animals' &&
        category !== 'food' &&
        category !== 'emotions' &&
        category !== 'festivals' &&
        category !== 'characters' &&
        category !== 'other'
      ) {
        throw new Error(
          '分類須為 general、animals、food、emotions、festivals、characters 或 other。',
        );
      }
      const tokens = demoStickerSearchTokens(q);
      const rows = demoStickers
        .filter((sticker) => {
          if (sticker.status !== 'active' || demoRemovedStickerIds.has(sticker.id)) {
            return false;
          }
          const pack = demoVisiblePack(sticker.packId);
          if (pack === null) {
            return false;
          }
          if (category !== '' && pack.category !== category) {
            return false;
          }
          return tokens.every((token) =>
            `${sticker.name} ${pack.name}`.toLowerCase().includes(token),
          );
        })
        .map((sticker) => {
          const pack = demoVisiblePack(sticker.packId);
          return {
            item: demoStickerView({ ...sticker, packName: pack?.name ?? sticker.packName }),
            sortAt: sticker.createdAt,
          };
        });
      const page = demoStickerCursorPage(rows, options.cursor ?? '', options.limit ?? 0);
      return { stickers: page.items, nextCursor: page.nextCursor };
    },
    getCurrentUserId: (): string | null => 'me',
    setTyping: (conversationId: string, isTyping: boolean) => {
      snapshot = {
        ...snapshot,
        conversations: snapshot.conversations.map((conversation) =>
          conversation.id === conversationId ? { ...conversation, isTyping } : conversation,
        ),
      };
      emit();
    },
  };
}

export function useChatData(provider: ChatDataProvider): ChatSnapshot {
  return useSyncExternalStore(provider.subscribe, provider.getSnapshot, provider.getSnapshot);
}
