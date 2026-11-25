import type {
  BlockItem,
  CallEventItem,
  CallItem,
  CallPage,
  CallParticipantItem,
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
  PinItem,
  PostCommentItem,
  PostCommentPage,
  PostItem,
  PostLikeResult,
  PostMediaItem,
  PostPage,
  ReactionCountItem,
  ReactionSummaryItem,
  ReceiptItem,
  SearchedUser,
  SessionItem,
  StickerItem,
  StickerPackItem,
  StickerPackPage,
  StickerPage,
  AddStickerInput,
  CreateStickerPackInput,
  UpdatePostInput,
  UpdateStickerPackInput,
} from './types';
import type { ApiConnectionInfo, ChatDataProvider, ChatSnapshot } from './provider';
import { createDemoProvider } from './provider';
import type {
  AttachmentListResponse,
  AttachmentResponse,
  BlockResponse,
  CallEventResponse,
  CallListResponse,
  CallParticipantResponse,
  CallResponse,
  ConversationMemberResponse,
  ConversationResponse,
  DeviceResponse,
  EtherApiClient,
  ForwardResponse,
  FriendRequestResponse,
  FriendResponse,
  GroupAnnouncementResponse,
  GroupInvitationResponse,
  GroupMemberResponse,
  GroupResponse,
  GroupSettingsResponse,
  MediaAuthorizeResponse,
  MediaFileResponse,
  MediaThumbnailResponse,
  MediaUploadResponse,
  MessageResponse,
  PinListResponse,
  PinResponse,
  PostCommentListResponse,
  PostCommentResponse,
  PostListResponse,
  PostMediaResponse,
  PostResponse,
  ProfilePatch,
  ProfileResponse,
  ReactionSummaryResponse,
  ReceiptResponse,
  SearchUserResponse,
  SessionResponse,
  StickerListResponse,
  StickerPackListResponse,
  StickerPackResponse,
  StickerResponse,
  TokenStore,
} from './api';
import { ApiError, AuthError } from './api';
import type { OidcSession, StorageLike } from './oidc';
import { beginOidcLogin, completeOidcCallback, OidcError } from './oidc';

/**
 * API-backed data provider (M2a, hybrid — labelled honestly).
 *
 * What comes from the backend (services/api, M1 identity endpoints
 * plus M3 friends endpoints plus M5 conversations/messages
 * endpoints plus M6 groups endpoints): the signed-in account (GET /api/v1/auth/me), the
 * public profile (GET/PATCH /api/v1/profiles/me), sessions/devices
 * (M2c), friends, friend requests, search and blocks
 * (GET/POST /api/v1/friends/*), conversations/messages
 * (GET/POST /api/v1/conversations/*, /api/v1/messages/*), and
 * groups, members, invitations, announcements and settings
 * (GET/POST/PATCH /api/v1/groups/*).
 *
 * What is still demonstration data: nothing in the chat views.
 * The conversation list and message history come from the backend
 * in API mode (list migration milestone); the demo snapshot only
 * backs demo mode. Sending through sendMessageAsync requires a
 * real backend conversation id — demo ids are not backend ids and
 * the server will reject them. Nothing here pretends otherwise.
 *
 * Sign-in: the OIDC authorization-code flow with PKCE (M2d, see
 * oidc.ts) when App wires an OidcSession, and developer token entry
 * as the fallback when Keycloak is not configured (or, when it is,
 * only behind the VITE_ALLOW_DEV_TOKEN_LOGIN flag). There is no
 * password login. A 401 after a failed refresh returns the app to
 * the sign-in screen. Tokens live in memory only.
 */

export interface ApiChatDataProvider extends ChatDataProvider {
  getConnection(): ApiConnectionInfo;
  initialize(): Promise<void>;
  signInWithToken(token: string): Promise<void>;
  signOut(): Promise<void>;
  refreshProfile(): Promise<void>;
  /**
   * OIDC 登入（M2d）：導向 Keycloak 授權端點。未傳入 OIDC 支援
   * 時拋錯；呼叫方（App）須先以設定判斷是否顯示此路徑。
   */
  signInWithKeycloak(): Promise<void>;
  /**
   * OIDC 回呼處理：無回呼參數時回 ignored；成功時完成登入並回
   * signed-in；失敗時回 failed 帶可讀訊息（不拋錯，App 顯示訊息）。
   */
  handleOidcCallback(search: string): Promise<OidcCallbackResult>;
}

/**
 * OIDC 支援（M2d）：App 在 Keycloak 設定齊全時傳入。未傳入時
 * provider 行為與 M2a 相同（開發者 Token 登入、401 即登出）。
 */
export interface OidcSupport {
  session: OidcSession;
  /** 登入暫存的儲存（sessionStorage）；缺失時 PKCE 登入明確拒絕。 */
  pendingStorage: StorageLike | undefined;
  navigate: (url: string) => void;
}

export interface OidcCallbackResult {
  status: 'signed-in' | 'failed' | 'ignored';
  message?: string | undefined;
}

const PLACEHOLDER_PROFILE: DemoProfile = {
  name: '',
  handle: '',
  statusMessage: '',
  presence: 'offline',
};

/**
 * Idempotency key for one send attempt (M5). Generated per attempt
 * with the platform CSPRNG when available; a repeat send of the same
 * key returns the same server-side message. This is a random
 * identifier, not a secret — it may be sent to the backend.
 */
function newClientMessageId(): string {
  const cryptoObject = (globalThis as { crypto?: { randomUUID?: () => string } }).crypto;
  if (cryptoObject?.randomUUID !== undefined) {
    return cryptoObject.randomUUID();
  }
  const bytes = new Uint8Array(16);
  const globalCrypto = (globalThis as { crypto?: { getRandomValues?: (a: Uint8Array) => void } })
    .crypto;
  if (globalCrypto?.getRandomValues !== undefined) {
    globalCrypto.getRandomValues(bytes);
  } else {
    for (let i = 0; i < bytes.length; i += 1) {
      bytes[i] = Math.floor(Math.random() * 256);
    }
  }
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
}

function toDemoProfile(meEmail: string, profile: ProfileResponse): DemoProfile {
  return {
    name: profile.display_name,
    handle: meEmail !== '' ? meEmail : profile.public_id !== '' ? `@${profile.public_id}` : '',
    statusMessage: profile.status_message,
    presence: 'online',
  };
}

function toProfilePatch(patch: Partial<DemoProfile>): ProfilePatch {
  const out: ProfilePatch = {};
  if (patch.name !== undefined) {
    out.display_name = patch.name;
  }
  if (patch.statusMessage !== undefined) {
    out.status_message = patch.statusMessage;
  }
  return out;
}

function toSessionItem(session: SessionResponse): SessionItem {
  return {
    id: session.session_id,
    deviceId: session.device_id ?? '',
    expiresAt: session.expires_at,
    lastActiveAt: session.last_active_at,
    isRevoked: session.is_revoked,
    revokedReason: session.revoked_reason ?? '',
  };
}

function toDeviceItem(device: DeviceResponse): DeviceItem {
  return {
    id: device.device_id,
    name: device.device_name,
    platform: device.platform,
    isTrusted: device.is_trusted,
    isRevoked: device.is_revoked,
    lastActiveAt: device.last_active_at ?? null,
    createdAt: device.created_at,
  };
}

function toFriendItem(friend: FriendResponse): FriendItem {
  return {
    id: friend.friend_id,
    displayName: friend.display_name,
    publicId: friend.public_id,
    avatarMediaId: friend.avatar_media_id,
    statusMessage: friend.status_message,
    isHidden: friend.is_hidden,
    establishedAt: friend.established_at,
  };
}

function toFriendRequestItem(
  request: FriendRequestResponse,
  direction: 'incoming' | 'outgoing',
): FriendRequestItem {
  return {
    id: request.request_id,
    direction:
      request.direction === 'incoming' || request.direction === 'outgoing'
        ? request.direction
        : direction,
    otherId: request.other_user_id ?? '',
    otherDisplayName: request.other_display_name ?? '',
    otherPublicId: request.other_public_id ?? '',
    message: request.message,
    createdAt: request.created_at,
  };
}

function toSearchedUser(found: SearchUserResponse): SearchedUser {
  return {
    id: found.user_id,
    displayName: found.display_name,
    publicId: found.public_id,
    avatarMediaId: found.avatar_media_id,
    statusMessage: found.status_message,
  };
}

function toBlockItem(block: BlockResponse): BlockItem {
  return {
    userId: block.blocked_user_id,
    displayName: block.display_name ?? '',
    publicId: block.public_id ?? '',
    reason: block.reason,
    blockedAt: block.blocked_at,
  };
}

function toConversationItem(conversation: ConversationResponse): ConversationItem {
  return {
    id: conversation.conversation_id,
    type: conversation.type === 'group' ? 'group' : 'direct',
    title: conversation.title,
    createdBy: conversation.created_by,
    lastMessageId: conversation.last_message_id ?? '',
    lastMessageAt: conversation.last_message_at ?? null,
    unreadCount: conversation.unread_count,
    createdAt: conversation.created_at,
    updatedAt: conversation.updated_at,
  };
}

function toConversationMemberItem(member: ConversationMemberResponse): ConversationMemberItem {
  return {
    userId: member.user_id,
    role: member.role,
    joinedAt: member.joined_at,
    leftAt: member.left_at ?? null,
    lastReadSequence: member.last_read_sequence,
  };
}

function toMessageItem(message: MessageResponse): MessageItem {
  return {
    id: message.message_id,
    conversationId: message.conversation_id,
    senderId: message.sender_id,
    kind: message.kind,
    content: message.content,
    status: message.status,
    clientMessageId: message.client_message_id ?? '',
    replyToMessageId: message.reply_to_message_id ?? '',
    sequenceNumber: message.sequence_number,
    editCount: message.edit_count,
    editedAt: message.edited_at ?? null,
    recalledAt: message.recalled_at ?? null,
    deletedForSelf: message.deleted_for_self,
    createdAt: message.created_at,
    updatedAt: message.updated_at,
  };
}

function toReceiptItem(receipt: ReceiptResponse): ReceiptItem {
  return {
    messageId: receipt.message_id,
    userId: receipt.user_id,
    receiptType: receipt.receipt_type,
    deviceId: receipt.device_id ?? '',
    createdAt: receipt.created_at,
  };
}

function toReactionSummaryItem(summary: ReactionSummaryResponse): ReactionSummaryItem {
  return {
    messageId: summary.message_id,
    reactions: summary.reactions.map(
      (count): ReactionCountItem => ({
        reaction: count.reaction,
        count: count.count,
        reactedByMe: count.reacted_by_me,
      }),
    ),
  };
}

function toPinItem(pin: PinResponse): PinItem {
  return {
    id: pin.pin_id,
    conversationId: pin.conversation_id,
    messageId: pin.message_id,
    pinnedBy: pin.pinned_by,
    pinnedAt: pin.pinned_at,
    unpinnedAt: pin.unpinned_at ?? null,
  };
}

function toPinItems(list: PinListResponse): PinItem[] {
  return list.pins.map(toPinItem);
}

function toMessageAttachmentItem(attachment: AttachmentResponse): MessageAttachmentItem {
  return {
    id: attachment.attachment_id,
    messageId: attachment.message_id,
    mediaFileId: attachment.media_file_id,
    mimeType: attachment.mime_type,
    sizeBytes: attachment.size_bytes,
    fileName: attachment.file_name,
    createdAt: attachment.created_at,
  };
}

function toMessageAttachmentItems(list: AttachmentListResponse): MessageAttachmentItem[] {
  return list.attachments.map(toMessageAttachmentItem);
}

function toForwardRecordItem(record: ForwardResponse): ForwardRecordItem {
  return {
    id: record.forward_id,
    sourceMessageId: record.source_message_id,
    forwardedMessageId: record.forwarded_message_id,
    sourceConversationId: record.source_conversation_id,
    targetConversationId: record.target_conversation_id,
    forwardedBy: record.forwarded_by,
    createdAt: record.created_at,
  };
}

function toGroupItem(group: GroupResponse): GroupItem {
  return {
    id: group.group_id,
    name: group.name,
    description: group.description,
    avatarMediaId: group.avatar_media_id,
    conversationId: group.conversation_id ?? '',
    createdBy: group.created_by,
    isDisbanded: group.is_disbanded,
    memberCount: group.member_count,
    createdAt: group.created_at,
    updatedAt: group.updated_at,
  };
}

function toGroupMemberItem(member: GroupMemberResponse): GroupMemberItem {
  return {
    userId: member.user_id,
    displayName: '',
    role: member.role,
    joinedAt: member.joined_at,
    invitedBy: member.invited_by,
    leftAt: member.left_at ?? null,
  };
}

function toGroupInvitationItem(invitation: GroupInvitationResponse): GroupInvitationItem {
  return {
    id: invitation.invitation_id,
    groupId: invitation.group_id,
    groupName: invitation.group_name ?? '',
    inviterId: invitation.inviter_id,
    inviteeId: invitation.invitee_id,
    status: invitation.status,
    createdAt: invitation.created_at,
    updatedAt: invitation.updated_at,
  };
}

function toGroupAnnouncementItem(announcement: GroupAnnouncementResponse): GroupAnnouncementItem {
  return {
    id: announcement.announcement_id,
    groupId: announcement.group_id,
    authorId: announcement.author_id,
    content: announcement.content,
    isActive: announcement.is_active,
    publishedAt: announcement.published_at,
    retractedAt: announcement.retracted_at ?? null,
    createdAt: announcement.created_at,
    updatedAt: announcement.updated_at,
  };
}

function toGroupSettingsItem(settings: GroupSettingsResponse): GroupSettingsItem {
  return {
    groupId: settings.group_id,
    invitePolicy: settings.invite_policy,
    updatedBy: settings.updated_by ?? '',
    updatedAt: settings.updated_at ?? null,
  };
}

function toMediaFileItem(file: MediaFileResponse): MediaFileItem {
  return {
    id: file.file_id,
    mimeType: file.mime_type,
    sizeBytes: file.size_bytes,
    originalFilename: file.original_filename,
    width: file.width,
    height: file.height,
    durationSeconds: file.duration_seconds,
    checksum: file.checksum,
    scanStatus: file.scan_status,
    scanCompletedAt: file.scan_completed_at ?? null,
    createdAt: file.created_at,
    updatedAt: file.updated_at,
  };
}

function toMediaUploadItem(upload: MediaUploadResponse): MediaUploadItem {
  return {
    id: upload.upload_id,
    mediaFileId: upload.media_file_id ?? '',
    status: upload.status,
    storageKey: upload.storage_key,
    externalUploadId: upload.upload_id_external,
    uploadedBytes: upload.uploaded_bytes,
    totalBytes: upload.total_bytes,
    partCount: upload.part_count,
    uploadUrl: upload.upload_url ?? null,
    uploadUrlExpiresAt: upload.upload_url_expires_at ?? null,
    completedAt: upload.completed_at ?? null,
    createdAt: upload.created_at,
    updatedAt: upload.updated_at,
  };
}

function toMediaThumbnailItem(thumbnail: MediaThumbnailResponse): MediaThumbnailItem {
  return {
    id: thumbnail.thumbnail_id,
    fileId: thumbnail.file_id,
    variantType: thumbnail.variant_type,
    mimeType: thumbnail.mime_type,
    sizeBytes: thumbnail.size_bytes,
    width: thumbnail.width,
    height: thumbnail.height,
    createdAt: thumbnail.created_at,
  };
}

function toMediaAuthorization(response: MediaAuthorizeResponse): MediaAuthorization {
  return {
    allowed: response.allowed,
    scanStatus: response.scan_status,
    downloadUrl: response.download_url ?? null,
    downloadUrlExpiresAt: response.download_url_expires_at ?? null,
  };
}

function toPostMediaItem(media: PostMediaResponse): PostMediaItem {
  return {
    mediaFileId: media.media_file_id,
    position: media.position,
    caption: media.caption,
  };
}

function toPostItem(post: PostResponse): PostItem {
  return {
    id: post.post_id,
    authorId: post.author_id,
    content: post.content,
    visibility: post.visibility as PostItem['visibility'],
    editedAt: post.edited_at ?? null,
    media: post.media.map(toPostMediaItem),
    likeCount: post.like_count,
    commentCount: post.comment_count,
    likedByMe: post.liked_by_me,
    createdAt: post.created_at,
    updatedAt: post.updated_at,
  };
}

function toPostPage(page: PostListResponse): PostPage {
  return {
    posts: page.posts.map(toPostItem),
    nextCursor: page.next_cursor,
  };
}

function toPostCommentItem(comment: PostCommentResponse): PostCommentItem {
  return {
    id: comment.comment_id,
    postId: comment.post_id,
    authorId: comment.author_id,
    parentCommentId: comment.parent_comment_id ?? '',
    content: comment.content,
    isDeleted: comment.is_deleted,
    editedAt: comment.edited_at ?? null,
    createdAt: comment.created_at,
    updatedAt: comment.updated_at,
  };
}

function toPostCommentPage(page: PostCommentListResponse): PostCommentPage {
  return {
    comments: page.comments.map(toPostCommentItem),
    nextCursor: page.next_cursor,
  };
}

function toCallItem(call: CallResponse): CallItem {
  return {
    id: call.call_id,
    conversationId: call.conversation_id ?? '',
    groupId: call.group_id ?? '',
    initiatorId: call.initiator_id,
    callType: call.call_type as CallItem['callType'],
    status: call.status as CallItem['status'],
    startedAt: call.started_at ?? null,
    endedAt: call.ended_at ?? null,
    durationSeconds: call.duration_seconds ?? null,
    endedBy: call.ended_by ?? '',
    endReason: (call.end_reason ?? '') as CallItem['endReason'],
    participantCount: call.participant_count,
    createdAt: call.created_at,
    updatedAt: call.updated_at,
  };
}

function toCallPage(page: CallListResponse): CallPage {
  return {
    calls: page.calls.map(toCallItem),
    nextCursor: page.next_cursor,
  };
}

function toCallParticipantItem(participant: CallParticipantResponse): CallParticipantItem {
  return {
    userId: participant.user_id,
    joinedAt: participant.joined_at ?? null,
    leftAt: participant.left_at ?? null,
    result: participant.result as CallParticipantItem['result'],
    createdAt: participant.created_at,
  };
}

function toCallEventItem(event: CallEventResponse): CallEventItem {
  return {
    id: event.event_id,
    userId: event.user_id ?? '',
    eventType: event.event_type as CallEventItem['eventType'],
    occurredAt: event.occurred_at,
  };
}

function toStickerPackItem(pack: StickerPackResponse): StickerPackItem {
  return {
    id: pack.pack_id,
    creatorId: pack.creator_id,
    name: pack.name,
    description: pack.description,
    category: pack.category as StickerPackItem['category'],
    coverMediaId: pack.cover_media_id,
    isFree: pack.is_free,
    status: pack.status as StickerPackItem['status'],
    stickerCount: pack.sticker_count,
    favoritedByMe: pack.favorited_by_me,
    createdAt: pack.created_at,
    updatedAt: pack.updated_at,
  };
}

function toStickerPackPage(page: StickerPackListResponse): StickerPackPage {
  return {
    packs: page.packs.map(toStickerPackItem),
    nextCursor: page.next_cursor,
  };
}

function toStickerItem(sticker: StickerResponse): StickerItem {
  return {
    id: sticker.sticker_id,
    packId: sticker.pack_id,
    packName: sticker.pack_name,
    mediaFileId: sticker.media_file_id,
    name: sticker.name,
    isAnimated: sticker.is_animated,
    position: sticker.position,
    status: sticker.status as StickerItem['status'],
    createdAt: sticker.created_at,
    updatedAt: sticker.updated_at,
  };
}

function toStickerPage(page: StickerListResponse): StickerPage {
  return {
    stickers: page.stickers.map(toStickerItem),
    nextCursor: page.next_cursor,
  };
}

export function createApiProvider(
  client: EtherApiClient,
  tokenStore: TokenStore,
  oidcSupport?: OidcSupport | undefined,
): ApiChatDataProvider {
  const demo = createDemoProvider();
  const listeners = new Set<() => void>();
  let connection: ApiConnectionInfo = { mode: 'api', state: 'connecting' };
  let apiProfile: DemoProfile = { ...PLACEHOLDER_PROFILE };
  let currentUserId: string | null = null;
  let initializing: Promise<void> | null = null;

  const oidcSession = oidcSupport?.session;

  /** 清除本地登入狀態：Access Token 儲存與 OIDC 工作階段（雙 Token）。 */
  const clearLocalSession = (): void => {
    tokenStore.setToken(null);
    oidcSession?.clear();
    currentUserId = null;
  };

  const emit = (): void => {
    for (const listener of listeners) {
      listener();
    }
  };

  const setConnection = (next: ApiConnectionInfo): void => {
    connection = next;
    emit();
  };

  const loadIdentity = async (): Promise<void> => {
    const me = await client.getMe();
    const profile = await client.getProfile();
    currentUserId = me.user_id;
    apiProfile = toDemoProfile(me.email, profile);
  };

  // A 401 from any identity-adjacent call means the token is gone
  // (the client's refresh hook already had its chance): clear the
  // local session and return the app to the sign-in screen, then
  // rethrow so the calling view can report the failure.
  const failUnauthorized = (error: unknown): never => {
    if (error instanceof AuthError) {
      clearLocalSession();
      setConnection({ mode: 'api', state: 'unauthorized' });
    }
    throw error;
  };

  const listSessionsAsync = async (): Promise<SessionItem[]> => {
    try {
      const sessions = await client.listSessions();
      return sessions.map(toSessionItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listDevicesAsync = async (): Promise<DeviceItem[]> => {
    try {
      const devices = await client.listDevices();
      return devices.map(toDeviceItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const revokeSessionAsync = async (sessionId: string): Promise<void> => {
    try {
      await client.revokeSession(sessionId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const revokeDeviceAsync = async (deviceId: string): Promise<void> => {
    try {
      await client.revokeDevice(deviceId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const listFriendsAsync = async (includeHidden = false): Promise<FriendItem[]> => {
    try {
      const friends = await client.listFriends(includeHidden);
      return friends.map(toFriendItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listFriendRequestsAsync = async (
    direction: 'incoming' | 'outgoing',
  ): Promise<FriendRequestItem[]> => {
    try {
      const requests = await client.listFriendRequests(direction);
      return requests.map((request) => toFriendRequestItem(request, direction));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const searchFriendAsync = async (publicId: string): Promise<SearchedUser | null> => {
    try {
      const found = await client.searchFriend(publicId);
      return toSearchedUser(found);
    } catch (error) {
      if (error instanceof AuthError) {
        return failUnauthorized(error);
      }
      if (error instanceof ApiError && error.status === 404) {
        return null;
      }
      throw error;
    }
  };

  const sendFriendRequestAsync = async (addresseeId: string, message = ''): Promise<void> => {
    try {
      await client.sendFriendRequest({ addressee_id: addresseeId, message });
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const respondFriendRequestAsync = async (requestId: string, accept: boolean): Promise<void> => {
    try {
      await client.respondFriendRequest(requestId, accept);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const cancelFriendRequestAsync = async (requestId: string): Promise<void> => {
    try {
      await client.cancelFriendRequest(requestId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const removeFriendAsync = async (friendId: string): Promise<void> => {
    try {
      await client.removeFriend(friendId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const setFriendHiddenAsync = async (friendId: string, hidden: boolean): Promise<void> => {
    try {
      await client.setFriendHidden(friendId, hidden);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const listBlocksAsync = async (): Promise<BlockItem[]> => {
    try {
      const blocks = await client.listBlocks();
      return blocks.map(toBlockItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const blockUserAsync = async (userId: string, reason = ''): Promise<void> => {
    try {
      await client.blockUser({ blocked_user_id: userId, reason });
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const unblockUserAsync = async (userId: string): Promise<void> => {
    try {
      await client.unblockUser(userId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const listConversationsAsync = async (): Promise<ConversationItem[]> => {
    try {
      const conversations = await client.listConversations();
      return conversations.map(toConversationItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const createConversationAsync = async (input: {
    type: 'direct' | 'group';
    otherUserId?: string;
    title?: string;
    memberIds?: string[];
  }): Promise<ConversationItem> => {
    try {
      const conversation = await client.createConversation({
        type: input.type,
        ...(input.otherUserId !== undefined ? { other_user_id: input.otherUserId } : {}),
        ...(input.title !== undefined ? { title: input.title } : {}),
        ...(input.memberIds !== undefined ? { member_ids: input.memberIds } : {}),
      });
      return toConversationItem(conversation);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listMembersAsync = async (conversationId: string): Promise<ConversationMemberItem[]> => {
    try {
      const members = await client.listMembers(conversationId);
      return members.map(toConversationMemberItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const leaveConversationAsync = async (conversationId: string): Promise<void> => {
    try {
      await client.leaveConversation(conversationId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const addMembersAsync = async (
    conversationId: string,
    memberIds: string[],
  ): Promise<ConversationMemberItem[]> => {
    try {
      const members = await client.addMembers(conversationId, memberIds);
      return members.map(toConversationMemberItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listMessagesAsync = async (
    conversationId: string,
    options: { beforeSeq?: number; sinceSeq?: number; limit?: number } = {},
  ): Promise<MessagePage> => {
    try {
      const page = await client.listMessages(conversationId, options);
      return {
        messages: page.messages.map(toMessageItem),
        nextCursor: page.next_cursor ?? '',
      };
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const sendMessageAsync = async (
    conversationId: string,
    body: string,
    options: { replyToMessageId?: string; attachmentFileIds?: string[] } = {},
  ): Promise<MessageItem> => {
    try {
      const message = await client.sendMessage(conversationId, {
        kind: 'text',
        content: body,
        client_message_id: newClientMessageId(),
        ...(options.replyToMessageId !== undefined
          ? { reply_to_message_id: options.replyToMessageId }
          : {}),
        ...(options.attachmentFileIds !== undefined && options.attachmentFileIds.length > 0
          ? { attachment_file_ids: options.attachmentFileIds }
          : {}),
      });
      return toMessageItem(message);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const editMessageAsync = async (messageId: string, content: string): Promise<MessageItem> => {
    try {
      return toMessageItem(await client.editMessage(messageId, content));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const recallMessageAsync = async (messageId: string): Promise<MessageItem> => {
    try {
      return toMessageItem(await client.recallMessage(messageId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const deleteMessageAsync = async (messageId: string): Promise<MessageItem> => {
    try {
      return toMessageItem(await client.deleteMessage(messageId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const postReceiptAsync = async (
    messageId: string,
    receiptType: 'delivered' | 'read',
  ): Promise<ReceiptItem> => {
    try {
      return toReceiptItem(await client.postReceipt(messageId, { receipt_type: receiptType }));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listReceiptsAsync = async (messageId: string): Promise<ReceiptItem[]> => {
    try {
      const receipts = await client.listReceipts(messageId);
      return receipts.map((receipt) => toReceiptItem(receipt));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listReactionsAsync = async (messageId: string): Promise<ReactionSummaryItem> => {
    try {
      return toReactionSummaryItem(await client.listReactions(messageId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const setReactionAsync = async (
    messageId: string,
    reaction: string,
  ): Promise<ReactionSummaryItem> => {
    try {
      return toReactionSummaryItem(await client.setReaction(messageId, { reaction }));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const removeReactionAsync = async (messageId: string): Promise<ReactionSummaryItem> => {
    try {
      return toReactionSummaryItem(await client.removeReaction(messageId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listPinsAsync = async (conversationId: string): Promise<PinItem[]> => {
    try {
      return toPinItems(await client.listPins(conversationId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const pinMessageAsync = async (conversationId: string, messageId: string): Promise<PinItem> => {
    try {
      return toPinItem(await client.pinMessage(conversationId, { message_id: messageId }));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const unpinMessageAsync = async (conversationId: string, messageId: string): Promise<void> => {
    try {
      await client.unpinMessage(conversationId, messageId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const searchMessagesAsync = async (input: MessageSearchInput): Promise<MessageSearchPage> => {
    try {
      const response = await client.searchMessages({
        q: input.query,
        ...(input.conversationId !== undefined ? { conversation_id: input.conversationId } : {}),
        ...(input.senderId !== undefined ? { sender_id: input.senderId } : {}),
        ...(input.since !== undefined ? { since: input.since } : {}),
        ...(input.until !== undefined ? { until: input.until } : {}),
        ...(input.limit !== undefined ? { limit: input.limit } : {}),
        ...(input.cursor !== undefined ? { cursor: input.cursor } : {}),
      });
      return {
        results: response.results.map((result) => ({
          id: result.message_id,
          conversationId: result.conversation_id,
          senderId: result.sender_id,
          kind: result.kind,
          content: result.content,
          sequenceNumber: result.sequence_number,
          createdAt: result.created_at,
        })),
        nextCursor: response.next_cursor ?? '',
      };
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listAttachmentsAsync = async (messageId: string): Promise<MessageAttachmentItem[]> => {
    try {
      return toMessageAttachmentItems(await client.listAttachments(messageId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const addAttachmentsAsync = async (
    messageId: string,
    mediaFileIds: string[],
  ): Promise<MessageAttachmentItem[]> => {
    try {
      return toMessageAttachmentItems(await client.addAttachments(messageId, mediaFileIds));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const forwardMessageAsync = async (
    messageId: string,
    targetConversationId: string,
  ): Promise<MessageItem> => {
    try {
      return toMessageItem(
        await client.forwardMessage(messageId, {
          target_conversation_id: targetConversationId,
          client_message_id: newClientMessageId(),
        }),
      );
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const getForwardAsync = async (messageId: string): Promise<ForwardRecordItem> => {
    try {
      return toForwardRecordItem(await client.getForward(messageId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listGroupsAsync = async (): Promise<GroupItem[]> => {
    try {
      const groups = await client.listGroups();
      return groups.map(toGroupItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const createGroupAsync = async (input: {
    name: string;
    description?: string;
    avatarMediaId?: string;
    memberIds?: string[];
  }): Promise<GroupItem> => {
    try {
      const group = await client.createGroup({
        name: input.name,
        ...(input.description !== undefined ? { description: input.description } : {}),
        ...(input.avatarMediaId !== undefined ? { avatar_media_id: input.avatarMediaId } : {}),
        ...(input.memberIds !== undefined ? { member_ids: input.memberIds } : {}),
      });
      return toGroupItem(group);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const getGroupAsync = async (groupId: string): Promise<GroupItem> => {
    try {
      return toGroupItem(await client.getGroup(groupId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const updateGroupAsync = async (
    groupId: string,
    patch: { name?: string; description?: string; avatarMediaId?: string },
  ): Promise<GroupItem> => {
    try {
      const group = await client.updateGroup(groupId, {
        ...(patch.name !== undefined ? { name: patch.name } : {}),
        ...(patch.description !== undefined ? { description: patch.description } : {}),
        ...(patch.avatarMediaId !== undefined ? { avatar_media_id: patch.avatarMediaId } : {}),
      });
      return toGroupItem(group);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const disbandGroupAsync = async (groupId: string): Promise<void> => {
    try {
      await client.disbandGroup(groupId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const transferGroupOwnershipAsync = async (
    groupId: string,
    newOwnerId: string,
  ): Promise<GroupItem> => {
    try {
      return toGroupItem(await client.transferGroupOwnership(groupId, newOwnerId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listGroupMembersAsync = async (groupId: string): Promise<GroupMemberItem[]> => {
    try {
      const members = await client.listGroupMembers(groupId);
      return members.map(toGroupMemberItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const addGroupMembersAsync = async (
    groupId: string,
    memberIds: string[],
  ): Promise<GroupMemberItem[]> => {
    try {
      const members = await client.addGroupMembers(groupId, memberIds);
      return members.map(toGroupMemberItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const setGroupMemberRoleAsync = async (
    groupId: string,
    userId: string,
    role: string,
  ): Promise<GroupMemberItem> => {
    try {
      return toGroupMemberItem(await client.setGroupMemberRole(groupId, userId, { role }));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const removeGroupMemberAsync = async (groupId: string, userId: string): Promise<void> => {
    try {
      await client.removeGroupMember(groupId, userId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const leaveGroupAsync = async (groupId: string): Promise<void> => {
    try {
      await client.leaveGroup(groupId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const listGroupInvitationsAsync = async (groupId: string): Promise<GroupInvitationItem[]> => {
    try {
      const invitations = await client.listGroupInvitations(groupId);
      return invitations.map(toGroupInvitationItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const createGroupInvitationAsync = async (
    groupId: string,
    inviteeId: string,
  ): Promise<GroupInvitationItem> => {
    try {
      return toGroupInvitationItem(
        await client.createGroupInvitation(groupId, { invitee_id: inviteeId }),
      );
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listMyGroupInvitationsAsync = async (): Promise<GroupInvitationItem[]> => {
    try {
      const invitations = await client.listMyGroupInvitations();
      return invitations.map(toGroupInvitationItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const respondGroupInvitationAsync = async (
    invitationId: string,
    accept: boolean,
  ): Promise<GroupInvitationItem> => {
    try {
      return toGroupInvitationItem(await client.respondGroupInvitation(invitationId, accept));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const cancelGroupInvitationAsync = async (invitationId: string): Promise<GroupInvitationItem> => {
    try {
      return toGroupInvitationItem(await client.cancelGroupInvitation(invitationId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listGroupAnnouncementsAsync = async (groupId: string): Promise<GroupAnnouncementItem[]> => {
    try {
      const announcements = await client.listGroupAnnouncements(groupId);
      return announcements.map(toGroupAnnouncementItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const createGroupAnnouncementAsync = async (
    groupId: string,
    content: string,
  ): Promise<GroupAnnouncementItem> => {
    try {
      return toGroupAnnouncementItem(await client.createGroupAnnouncement(groupId, { content }));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const updateGroupAnnouncementAsync = async (
    groupId: string,
    announcementId: string,
    content: string,
  ): Promise<GroupAnnouncementItem> => {
    try {
      return toGroupAnnouncementItem(
        await client.updateGroupAnnouncement(groupId, announcementId, { content }),
      );
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const retractGroupAnnouncementAsync = async (
    groupId: string,
    announcementId: string,
  ): Promise<GroupAnnouncementItem> => {
    try {
      return toGroupAnnouncementItem(
        await client.retractGroupAnnouncement(groupId, announcementId),
      );
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const getGroupSettingsAsync = async (groupId: string): Promise<GroupSettingsItem> => {
    try {
      return toGroupSettingsItem(await client.getGroupSettings(groupId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const updateGroupSettingsAsync = async (
    groupId: string,
    invitePolicy: string,
  ): Promise<GroupSettingsItem> => {
    try {
      return toGroupSettingsItem(
        await client.updateGroupSettings(groupId, { invite_policy: invitePolicy }),
      );
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listMediaFilesAsync = async (): Promise<MediaFileItem[]> => {
    try {
      const files = await client.listFiles();
      return files.map(toMediaFileItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listMediaUploadsAsync = async (): Promise<MediaUploadItem[]> => {
    try {
      const uploads = await client.listUploads();
      return uploads.map(toMediaUploadItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const initiateMediaUploadAsync = async (input: InitiateUploadInput): Promise<MediaUploadItem> => {
    try {
      const upload = await client.initiateUpload({
        mime_type: input.mimeType,
        total_bytes: input.totalBytes,
        original_filename: input.originalFilename,
        width: input.width,
        height: input.height,
        duration_seconds: input.durationSeconds,
        checksum: input.checksum,
      });
      return toMediaUploadItem(upload);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const cancelMediaUploadAsync = async (uploadId: string): Promise<MediaUploadItem> => {
    try {
      const upload = await client.cancelUpload(uploadId);
      return toMediaUploadItem(upload);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const deleteMediaFileAsync = async (fileId: string): Promise<void> => {
    try {
      await client.deleteFile(fileId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const listMediaThumbnailsAsync = async (fileId: string): Promise<MediaThumbnailItem[]> => {
    try {
      const thumbnails = await client.listThumbnails(fileId);
      return thumbnails.map(toMediaThumbnailItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const authorizeMediaAccessAsync = async (
    fileId: string,
    accessType: 'download' | 'preview' | 'thumbnail',
  ): Promise<MediaAuthorization> => {
    try {
      const response = await client.authorizeAccess(fileId, { access_type: accessType });
      return toMediaAuthorization(response);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const issueMediaUploadUrlAsync = async (uploadId: string): Promise<MediaUploadUrl> => {
    try {
      const response = await client.issueUploadUrl(uploadId);
      return { uploadUrl: response.upload_url, expiresAt: response.expires_at };
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const putMediaUploadBytesAsync = async (uploadUrl: string, data: Blob): Promise<void> => {
    try {
      await client.putUploadBytes(uploadUrl, data);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const reportMediaUploadProgressAsync = async (
    uploadId: string,
    uploadedBytes: number,
    partCount: number,
  ): Promise<MediaUploadItem> => {
    try {
      const upload = await client.reportUploadProgress(uploadId, {
        uploaded_bytes: uploadedBytes,
        part_count: partCount,
      });
      return toMediaUploadItem(upload);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const completeMediaUploadAsync = async (
    uploadId: string,
    input: InitiateUploadInput,
  ): Promise<MediaFileItem> => {
    try {
      const file = await client.completeUpload(uploadId, {
        mime_type: input.mimeType,
        original_filename: input.originalFilename,
        width: input.width,
        height: input.height,
        duration_seconds: input.durationSeconds,
        checksum: input.checksum,
      });
      return toMediaFileItem(file);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const resolveMediaDownloadAsync = async (fileId: string): Promise<MediaDownloadResolution> => {
    try {
      const authorization = await client.authorizeAccess(fileId, { access_type: 'download' });
      if (!authorization.allowed) {
        return { status: 'forbidden', scanStatus: authorization.scan_status };
      }
      const downloadUrl = authorization.download_url ?? null;
      if (downloadUrl === null || downloadUrl === '') {
        // 已授權但部署未提供可分發位址：不得假裝可下載。
        return { status: 'unavailable' };
      }
      if (/^https?:\/\//i.test(downloadUrl)) {
        // presigned 絕對 URL：由檢視在新分頁開啟，不經 fetch。
        return { status: 'ready-url', url: downloadUrl };
      }
      const blob = await client.fetchDownloadBytes(downloadUrl);
      return { status: 'ready-blob', blob };
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listFeedAsync = async (
    options: { cursor?: string; limit?: number } = {},
  ): Promise<PostPage> => {
    try {
      return toPostPage(await client.listFeed(options));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listAuthorPostsAsync = async (
    authorId: string,
    options: { cursor?: string; limit?: number } = {},
  ): Promise<PostPage> => {
    try {
      return toPostPage(await client.listAuthorPosts(authorId, options));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const getPostAsync = async (postId: string): Promise<PostItem> => {
    try {
      return toPostItem(await client.getPost(postId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const createPostAsync = async (input: CreatePostInput): Promise<PostItem> => {
    try {
      return toPostItem(
        await client.createPost({
          content: input.content,
          ...(input.visibility !== undefined ? { visibility: input.visibility } : {}),
          ...(input.mediaFileIds !== undefined ? { media_file_ids: input.mediaFileIds } : {}),
        }),
      );
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const updatePostAsync = async (postId: string, input: UpdatePostInput): Promise<PostItem> => {
    try {
      return toPostItem(
        await client.updatePost(postId, {
          ...(input.content !== undefined ? { content: input.content } : {}),
          ...(input.visibility !== undefined ? { visibility: input.visibility } : {}),
        }),
      );
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const deletePostAsync = async (postId: string): Promise<void> => {
    try {
      await client.deletePost(postId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const listPostMediaAsync = async (postId: string): Promise<PostMediaItem[]> => {
    try {
      const media = await client.listPostMedia(postId);
      return media.map(toPostMediaItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const likePostAsync = async (postId: string): Promise<PostLikeResult> => {
    try {
      const result = await client.likePost(postId);
      return { postId: result.post_id, liked: result.liked, likeCount: result.like_count };
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const unlikePostAsync = async (postId: string): Promise<PostLikeResult> => {
    try {
      const result = await client.unlikePost(postId);
      return { postId: result.post_id, liked: result.liked, likeCount: result.like_count };
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const createPostCommentAsync = async (
    postId: string,
    content: string,
    parentCommentId = '',
  ): Promise<PostCommentItem> => {
    try {
      return toPostCommentItem(
        await client.createPostComment(postId, {
          content,
          ...(parentCommentId !== '' ? { parent_comment_id: parentCommentId } : {}),
        }),
      );
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listPostCommentsAsync = async (
    postId: string,
    options: { cursor?: string; limit?: number } = {},
  ): Promise<PostCommentPage> => {
    try {
      return toPostCommentPage(await client.listPostComments(postId, options));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const deletePostCommentAsync = async (commentId: string): Promise<void> => {
    try {
      await client.deletePostComment(commentId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const listCallsAsync = async (
    options: { cursor?: string; limit?: number } = {},
  ): Promise<CallPage> => {
    try {
      return toCallPage(await client.listCalls(options));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const getCallAsync = async (callId: string): Promise<CallItem> => {
    try {
      return toCallItem(await client.getCall(callId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const initiateCallAsync = async (input: InitiateCallInput): Promise<CallItem> => {
    try {
      return toCallItem(
        await client.initiateCall({
          ...(input.conversationId !== undefined && input.conversationId !== ''
            ? { conversation_id: input.conversationId }
            : {}),
          ...(input.groupId !== undefined && input.groupId !== ''
            ? { group_id: input.groupId }
            : {}),
          call_type: input.callType,
        }),
      );
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const answerCallAsync = async (callId: string): Promise<CallItem> => {
    try {
      return toCallItem(await client.answerCall(callId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const declineCallAsync = async (callId: string): Promise<CallItem> => {
    try {
      return toCallItem(await client.declineCall(callId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const busyCallAsync = async (callId: string): Promise<CallItem> => {
    try {
      return toCallItem(await client.busyCall(callId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const endCallAsync = async (callId: string): Promise<CallItem> => {
    try {
      return toCallItem(await client.endCall(callId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listCallParticipantsAsync = async (callId: string): Promise<CallParticipantItem[]> => {
    try {
      const participants = await client.listCallParticipants(callId);
      return participants.map(toCallParticipantItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listCallEventsAsync = async (callId: string): Promise<CallEventItem[]> => {
    try {
      const events = await client.listCallEvents(callId);
      return events.map(toCallEventItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listStickerPacksAsync = async (
    options: {
      creatorId?: string;
      category?: string;
      q?: string;
      cursor?: string;
      limit?: number;
    } = {},
  ): Promise<StickerPackPage> => {
    try {
      return toStickerPackPage(await client.listStickerPacks(options));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const getStickerPackAsync = async (packId: string): Promise<StickerPackItem> => {
    try {
      return toStickerPackItem(await client.getStickerPack(packId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const createStickerPackAsync = async (
    input: CreateStickerPackInput,
  ): Promise<StickerPackItem> => {
    try {
      return toStickerPackItem(
        await client.createStickerPack({
          name: input.name,
          description: input.description,
          category: input.category,
          cover_media_id: input.coverMediaId,
          is_free: input.isFree,
        }),
      );
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const updateStickerPackAsync = async (
    packId: string,
    input: UpdateStickerPackInput,
  ): Promise<StickerPackItem> => {
    try {
      return toStickerPackItem(
        await client.updateStickerPack(packId, {
          ...(input.name !== undefined ? { name: input.name } : {}),
          ...(input.description !== undefined ? { description: input.description } : {}),
          ...(input.category !== undefined ? { category: input.category } : {}),
          ...(input.coverMediaId !== undefined ? { cover_media_id: input.coverMediaId } : {}),
          ...(input.status !== undefined ? { status: input.status } : {}),
          ...(input.isFree !== undefined ? { is_free: input.isFree } : {}),
        }),
      );
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const deleteStickerPackAsync = async (packId: string): Promise<void> => {
    try {
      await client.deleteStickerPack(packId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const listPackStickersAsync = async (packId: string): Promise<StickerItem[]> => {
    try {
      const stickers = await client.listPackStickers(packId);
      return stickers.map(toStickerItem);
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const addStickerAsync = async (packId: string, input: AddStickerInput): Promise<StickerItem> => {
    try {
      return toStickerItem(
        await client.addSticker(packId, {
          media_file_id: input.mediaFileId,
          name: input.name,
          is_animated: input.isAnimated,
        }),
      );
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const removeStickerAsync = async (stickerId: string): Promise<void> => {
    try {
      await client.removeSticker(stickerId);
    } catch (error) {
      failUnauthorized(error);
    }
  };

  const favoriteStickerPackAsync = async (packId: string): Promise<StickerPackItem> => {
    try {
      return toStickerPackItem(await client.favoriteStickerPack(packId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const unfavoriteStickerPackAsync = async (packId: string): Promise<StickerPackItem> => {
    try {
      return toStickerPackItem(await client.unfavoriteStickerPack(packId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listFavoriteStickerPacksAsync = async (
    options: { cursor?: string; limit?: number } = {},
  ): Promise<StickerPackPage> => {
    try {
      return toStickerPackPage(await client.listFavoriteStickerPacks(options));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const recordStickerUseAsync = async (stickerId: string): Promise<StickerItem> => {
    try {
      return toStickerItem(await client.recordStickerUse(stickerId));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const listRecentStickersAsync = async (
    options: { cursor?: string; limit?: number } = {},
  ): Promise<StickerPage> => {
    try {
      return toStickerPage(await client.listRecentStickers(options));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const searchStickersAsync = async (
    q: string,
    options: { category?: string; cursor?: string; limit?: number } = {},
  ): Promise<StickerPage> => {
    try {
      return toStickerPage(await client.searchStickers(q, options));
    } catch (error) {
      return failUnauthorized(error);
    }
  };

  const initialize = async (): Promise<void> => {
    if (initializing !== null) {
      return initializing;
    }
    initializing = (async (): Promise<void> => {
      if (tokenStore.getToken() === null) {
        setConnection({ mode: 'api', state: 'unauthorized' });
        return;
      }
      setConnection({ mode: 'api', state: 'connecting' });
      try {
        await loadIdentity();
        setConnection({ mode: 'api', state: 'ready' });
      } catch (error) {
        if (error instanceof AuthError) {
          clearLocalSession();
          setConnection({ mode: 'api', state: 'unauthorized' });
          return;
        }
        const message = error instanceof Error ? error.message : '無法連接後端服務。';
        setConnection({ mode: 'api', state: 'unreachable', message });
      }
    })();
    try {
      await initializing;
    } finally {
      initializing = null;
    }
  };

  const updateProfileAsync = async (patch: Partial<DemoProfile>): Promise<void> => {
    const previous = { ...apiProfile };
    const apiPatch = toProfilePatch(patch);
    // Presence/handle are local-only in this phase; only mapped
    // fields go to the backend.
    if (apiPatch.display_name === undefined && apiPatch.status_message === undefined) {
      apiProfile = { ...apiProfile, ...patch };
      emit();
      return;
    }
    apiProfile = { ...apiProfile, ...patch };
    emit();
    try {
      const updated = await client.updateProfile(apiPatch);
      const me = await client.getMe();
      currentUserId = me.user_id;
      apiProfile = toDemoProfile(me.email, updated);
      emit();
    } catch (error) {
      apiProfile = previous;
      emit();
      throw error;
    }
  };

  const provider: ApiChatDataProvider = {
    subscribe: (listener: () => void) => {
      listeners.add(listener);
      const unsubscribeDemo = demo.subscribe(listener);
      return () => {
        listeners.delete(listener);
        unsubscribeDemo();
      };
    },
    getSnapshot: (): ChatSnapshot => {
      const snapshot = demo.getSnapshot();
      return { ...snapshot, profile: { ...apiProfile }, apiConnection: { ...connection } };
    },
    sendMessage: (conversationId: string, body: string) => {
      // Sync fallback: delegated to the demo store. Views in API
      // mode use sendMessageAsync; this path only serves callers
      // without async messaging.
      demo.sendMessage(conversationId, body);
    },
    markConversationRead: (conversationId: string) => {
      demo.markConversationRead(conversationId);
    },
    updateProfile: (patch: Partial<DemoProfile>) => {
      // Sync interface entry point: the async implementation owns the
      // optimistic update and the revert-on-failure; failures are
      // swallowed here because this signature cannot report them.
      // Views that need the outcome use updateProfileAsync.
      void updateProfileAsync(patch).catch(() => undefined);
    },
    updateProfileAsync,
    listSessionsAsync,
    listDevicesAsync,
    revokeSessionAsync,
    revokeDeviceAsync,
    listFriendsAsync,
    listFriendRequestsAsync,
    searchFriendAsync,
    sendFriendRequestAsync,
    respondFriendRequestAsync,
    cancelFriendRequestAsync,
    removeFriendAsync,
    setFriendHiddenAsync,
    listBlocksAsync,
    blockUserAsync,
    unblockUserAsync,
    listConversationsAsync,
    createConversationAsync,
    listMembersAsync,
    addMembersAsync,
    leaveConversationAsync,
    listMessagesAsync,
    sendMessageAsync,
    editMessageAsync,
    recallMessageAsync,
    deleteMessageAsync,
    postReceiptAsync,
    listReceiptsAsync,
    listReactionsAsync,
    setReactionAsync,
    removeReactionAsync,
    listPinsAsync,
    pinMessageAsync,
    unpinMessageAsync,
    searchMessagesAsync,
    listAttachmentsAsync,
    addAttachmentsAsync,
    forwardMessageAsync,
    getForwardAsync,
    listGroupsAsync,
    createGroupAsync,
    getGroupAsync,
    updateGroupAsync,
    disbandGroupAsync,
    transferGroupOwnershipAsync,
    listGroupMembersAsync,
    addGroupMembersAsync,
    setGroupMemberRoleAsync,
    removeGroupMemberAsync,
    leaveGroupAsync,
    listGroupInvitationsAsync,
    createGroupInvitationAsync,
    listMyGroupInvitationsAsync,
    respondGroupInvitationAsync,
    cancelGroupInvitationAsync,
    listGroupAnnouncementsAsync,
    createGroupAnnouncementAsync,
    updateGroupAnnouncementAsync,
    retractGroupAnnouncementAsync,
    getGroupSettingsAsync,
    updateGroupSettingsAsync,
    listMediaFilesAsync,
    listMediaUploadsAsync,
    initiateMediaUploadAsync,
    issueMediaUploadUrlAsync,
    putMediaUploadBytesAsync,
    reportMediaUploadProgressAsync,
    completeMediaUploadAsync,
    cancelMediaUploadAsync,
    deleteMediaFileAsync,
    listMediaThumbnailsAsync,
    authorizeMediaAccessAsync,
    resolveMediaDownloadAsync,
    listFeedAsync,
    listAuthorPostsAsync,
    getPostAsync,
    createPostAsync,
    updatePostAsync,
    deletePostAsync,
    listPostMediaAsync,
    likePostAsync,
    unlikePostAsync,
    createPostCommentAsync,
    listPostCommentsAsync,
    deletePostCommentAsync,
    listCallsAsync,
    getCallAsync,
    initiateCallAsync,
    answerCallAsync,
    declineCallAsync,
    busyCallAsync,
    endCallAsync,
    listCallParticipantsAsync,
    listCallEventsAsync,
    listStickerPacksAsync,
    getStickerPackAsync,
    createStickerPackAsync,
    updateStickerPackAsync,
    deleteStickerPackAsync,
    listPackStickersAsync,
    addStickerAsync,
    removeStickerAsync,
    favoriteStickerPackAsync,
    unfavoriteStickerPackAsync,
    listFavoriteStickerPacksAsync,
    recordStickerUseAsync,
    listRecentStickersAsync,
    searchStickersAsync,
    getCurrentUserId: (): string | null => currentUserId,
    setTyping: (conversationId: string, isTyping: boolean) => {
      demo.setTyping(conversationId, isTyping);
    },
    getConnection: (): ApiConnectionInfo => ({ ...connection }),
    initialize,
    signInWithToken: async (token: string): Promise<void> => {
      const trimmed = token.trim();
      if (trimmed === '') {
        throw new Error('請貼上 Access Token。');
      }
      tokenStore.setToken(trimmed);
      await initialize();
      const state = connection.state;
      if (state === 'unauthorized') {
        throw new Error('Token 無效或已過期，請重新取得。');
      }
      if (state === 'unreachable') {
        throw new Error(connection.message ?? '無法連接後端服務。');
      }
    },
    signOut: async (): Promise<void> => {
      try {
        await client.logout();
      } catch {
        // Best effort: the local token is cleared regardless, so a
        // failed server-side logout never traps the user signed in.
      }
      tokenStore.setToken(null);
      oidcSession?.clear();
      currentUserId = null;
      apiProfile = { ...PLACEHOLDER_PROFILE };
      setConnection({ mode: 'api', state: 'unauthorized' });
    },
    signInWithKeycloak: async (): Promise<void> => {
      if (oidcSupport === undefined) {
        throw new Error('尚未設定 Keycloak 登入，請改用開發者 Token 登入或展示模式。');
      }
      await beginOidcLogin(oidcSupport.session, {
        navigate: oidcSupport.navigate,
        storage: oidcSupport.pendingStorage,
      });
    },
    handleOidcCallback: async (search: string): Promise<OidcCallbackResult> => {
      if (oidcSupport === undefined) {
        return { status: 'ignored' };
      }
      try {
        const tokens = await completeOidcCallback(
          oidcSupport.session,
          search,
          oidcSupport.pendingStorage,
        );
        if (tokens === null) {
          return { status: 'ignored' };
        }
      } catch (error) {
        clearLocalSession();
        setConnection({ mode: 'api', state: 'unauthorized' });
        const message =
          error instanceof OidcError
            ? error.message
            : error instanceof Error
              ? error.message
              : '登入失敗，請重新再試。';
        return { status: 'failed', message };
      }
      await initialize();
      if (connection.state === 'ready') {
        return { status: 'signed-in' };
      }
      if (connection.state === 'unreachable') {
        return {
          status: 'failed',
          message: connection.message ?? '無法連接後端服務。',
        };
      }
      return { status: 'failed', message: 'Token 無效或已過期，請重新登入。' };
    },
    refreshProfile: async (): Promise<void> => {
      if (connection.state !== 'ready') {
        return;
      }
      try {
        await loadIdentity();
        emit();
      } catch (error) {
        if (error instanceof AuthError) {
          clearLocalSession();
          setConnection({ mode: 'api', state: 'unauthorized' });
          return;
        }
        throw error;
      }
    },
  };

  return provider;
}
