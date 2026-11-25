/**
 * Backend API client (M2a). Talks to the Phase 03 identity endpoints
 * implemented by services/api (M1): /api/v1/auth/*, /api/v1/profiles/me
 * and /api/v1/users/me/settings. Phase 04 friends endpoints (M3) and
 * Phase 05 conversations/messages endpoints (M5) are wired below.
 *
 * Honesty rules for this layer:
 * - The conversation list and message history come from the backend
 *   in API mode (list migration milestone, see api-provider.ts and
 *   ChatsView). Sending through sendMessageAsync requires a real
 *   backend conversation id; demo ids are not backend ids and will
 *   be rejected by the server.
 * - Errors follow the ETHER envelope (ERROR_CODES.md §5.1). A 401 is
 *   surfaced as AuthError so callers can route back to sign-in. When
 *   an OIDC session is wired (see oidc.ts), the client retries once
 *   after a refresh-token grant before surfacing the 401, and
 *   refreshes proactively when the access token is about to expire.
 * - Tokens are never logged. Access tokens live in memory only
 *   (MemoryTokenStore); the localStorage store from M2a is superseded
 *   and must not be wired in the app (see below).
 */

export interface ApiFieldError {
  field: string;
  issue: string;
  message: string;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId: string | undefined;
  readonly details: ApiFieldError[];

  constructor(
    status: number,
    code: string,
    message: string,
    requestId?: string | undefined,
    details: ApiFieldError[] = [],
  ) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.requestId = requestId;
    this.details = details;
  }
}

export class AuthError extends ApiError {
  constructor(requestId?: string | undefined) {
    super(401, 'ETHER_AUTH_UNAUTHENTICATED', '未認證。', requestId);
    this.name = 'AuthError';
  }
}

/** Minimal token storage. The client only reads/writes the access token. */
export interface TokenStore {
  getToken(): string | null;
  setToken(token: string | null): void;
}

type StorageLike = {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
};

export const ACCESS_TOKEN_KEY = 'ether-access-token';

/**
 * Memory-only token store (M2d).
 *
 * This is the wired store for API mode: the access token lives only
 * in this page's memory, never in localStorage, sessionStorage, a
 * cookie, a log, or a URL. Closing or reloading the tab ends the
 * session and the user signs in again through Keycloak. The refresh
 * token is held by the OidcSession (see oidc.ts), not by this store.
 */
export class MemoryTokenStore implements TokenStore {
  private token: string | null = null;

  getToken(): string | null {
    return this.token;
  }

  setToken(token: string | null): void {
    this.token = token === null || token === '' ? null : token;
  }
}

/**
 * localStorage-backed token store.
 *
 * SUPERSEDED (M2d): the M2a provisional store. localStorage is
 * readable by any script on the origin, which conflicts with
 * IDENTITY_AND_PERMISSIONS.md §4 (an access token must not live
 * where third-party scripts can read it). The app now wires
 * MemoryTokenStore instead. This class is retained only so existing
 * tests and developer tooling keep compiling; it must not be wired
 * in App.tsx or any production sign-in path.
 */
export class LocalStorageTokenStore implements TokenStore {
  private readonly storage: StorageLike;
  private readonly memory = new Map<string, string>();

  constructor(storage?: StorageLike | undefined) {
    if (storage !== undefined) {
      this.storage = storage;
      return;
    }
    const maybe = (globalThis as { localStorage?: StorageLike }).localStorage;
    this.storage = maybe ?? {
      getItem: (key: string) => this.memory.get(key) ?? null,
      setItem: (key: string, value: string) => {
        this.memory.set(key, value);
      },
      removeItem: (key: string) => {
        this.memory.delete(key);
      },
    };
  }

  getToken(): string | null {
    return this.storage.getItem(ACCESS_TOKEN_KEY);
  }

  setToken(token: string | null): void {
    if (token === null || token === '') {
      this.storage.removeItem(ACCESS_TOKEN_KEY);
      return;
    }
    this.storage.setItem(ACCESS_TOKEN_KEY, token);
  }
}

export type FetchFn = (input: string, init?: RequestInit) => Promise<Response>;

/** Wire shapes (snake_case, exactly as services/api returns them). */
export interface MeResponse {
  user_id: string;
  keycloak_subject: string;
  email: string;
  email_verified: boolean;
  status: string;
  registered_at: string;
  last_login_at?: string | null;
}

export interface SessionResponse {
  session_id: string;
  device_id?: string;
  keycloak_session_id?: string;
  expires_at: string;
  last_active_at: string;
  is_revoked: boolean;
  revoked_reason?: string;
  created_at: string;
}

export interface DeviceResponse {
  device_id: string;
  device_name: string;
  platform: string;
  is_trusted: boolean;
  is_revoked: boolean;
  last_active_at?: string | null;
  created_at: string;
}

export interface ProfileResponse {
  user_id: string;
  display_name: string;
  public_id: string;
  avatar_media_id: string;
  bio: string;
  status_message: string;
  updated_at: string;
}

export interface SettingsResponse {
  user_id: string;
  language: string;
  theme: string;
  font_size: string;
  chat_background: string;
  updated_at: string;
}

export interface ProfilePatch {
  display_name?: string;
  public_id?: string;
  avatar_media_id?: string;
  bio?: string;
  status_message?: string;
}

export interface SettingsPatch {
  language?: string;
  theme?: string;
  font_size?: string;
  chat_background?: string;
}

export interface RegisterDeviceInput {
  device_name: string;
  platform: string;
  device_identifier_hash: string;
}

/** Wire shapes for the Phase 04 friends endpoints (M3). */
export interface FriendResponse {
  friend_id: string;
  display_name: string;
  public_id: string;
  avatar_media_id: string;
  status_message: string;
  is_hidden: boolean;
  established_at: string;
}

export interface FriendRequestResponse {
  request_id: string;
  requester_id: string;
  addressee_id: string;
  status: string;
  message: string;
  responded_at?: string | null;
  created_at: string;
  updated_at: string;
  other_user_id?: string;
  other_display_name?: string;
  other_public_id?: string;
  direction?: string;
}

export interface BlockResponse {
  blocked_user_id: string;
  display_name?: string;
  public_id?: string;
  reason: string;
  blocked_at: string;
  unblocked_at?: string | null;
}

export interface SearchUserResponse {
  user_id: string;
  display_name: string;
  public_id: string;
  avatar_media_id: string;
  status_message: string;
}

export interface SendFriendRequestInput {
  addressee_id: string;
  message?: string;
}

export interface BlockUserInput {
  blocked_user_id: string;
  reason?: string;
}

/** Wire shapes for the Phase 05 conversations/messages endpoints (M5). */
export interface ConversationResponse {
  conversation_id: string;
  type: string;
  title: string;
  created_by: string;
  last_message_id?: string;
  last_message_at?: string | null;
  unread_count: number;
  created_at: string;
  updated_at: string;
}

export interface ConversationMemberResponse {
  user_id: string;
  role: string;
  joined_at: string;
  left_at?: string | null;
  last_read_sequence: number;
}

export interface MessageResponse {
  message_id: string;
  conversation_id: string;
  sender_id: string;
  kind: string;
  content: string;
  status: string;
  client_message_id?: string;
  reply_to_message_id?: string;
  sequence_number: number;
  edit_count: number;
  edited_at?: string | null;
  recalled_at?: string | null;
  deleted_for_self: boolean;
  created_at: string;
  updated_at: string;
}

export interface MessageListResponse {
  messages: MessageResponse[];
  next_cursor?: string;
}

/** 訊息搜尋的 wire 形狀（§2v）：投影文件，不含編輯／回條狀態。 */
export interface SearchMessageResponse {
  message_id: string;
  conversation_id: string;
  sender_id: string;
  kind: string;
  content: string;
  sequence_number: number;
  created_at: string;
}

export interface SearchMessagesResponse {
  results: SearchMessageResponse[];
  next_cursor?: string;
}

export interface SearchMessagesInput {
  q: string;
  conversation_id?: string;
  sender_id?: string;
  since?: string;
  until?: string;
  limit?: number;
  cursor?: string;
}

export interface ReceiptResponse {
  message_id: string;
  user_id: string;
  receipt_type: string;
  device_id?: string;
  created_at: string;
}

export interface CreateConversationInput {
  type: 'direct' | 'group';
  other_user_id?: string;
  title?: string;
  member_ids?: string[];
}

export interface SendMessageInput {
  kind?: string;
  content: string;
  client_message_id?: string;
  reply_to_message_id?: string;
  attachment_file_ids?: string[];
}

export interface ListMessagesOptions {
  beforeSeq?: number;
  sinceSeq?: number;
  limit?: number;
}

export interface PostReceiptInput {
  receipt_type: string;
  device_id?: string;
}

/** Wire shapes for the message attachment and forward endpoints (§2s). */
export interface AttachmentResponse {
  attachment_id: string;
  message_id: string;
  media_file_id: string;
  mime_type: string;
  size_bytes: number;
  file_name: string;
  created_at: string;
}

export interface AttachmentListResponse {
  attachments: AttachmentResponse[];
}

export interface ForwardMessageInput {
  target_conversation_id: string;
  client_message_id?: string;
}

export interface ForwardResponse {
  forward_id: string;
  source_message_id: string;
  forwarded_message_id: string;
  source_conversation_id: string;
  target_conversation_id: string;
  forwarded_by: string;
  created_at: string;
}

/** Wire shapes for the message reactions and pins endpoints (§2p). */
export interface ReactionCountResponse {
  reaction: string;
  count: number;
  reacted_by_me: boolean;
}

export interface ReactionSummaryResponse {
  message_id: string;
  reactions: ReactionCountResponse[];
}

export interface SetReactionInput {
  reaction: string;
}

export interface PinResponse {
  pin_id: string;
  conversation_id: string;
  message_id: string;
  pinned_by: string;
  pinned_at: string;
  unpinned_at?: string | null;
}

export interface PinListResponse {
  pins: PinResponse[];
}

export interface PinMessageInput {
  message_id: string;
}

export interface UnpinResponse {
  conversation_id: string;
  message_id: string;
  pinned: boolean;
  unpinned_at?: string | null;
}

/** Wire shapes for the Phase 06 groups endpoints (M6). */
export interface GroupResponse {
  group_id: string;
  name: string;
  description: string;
  avatar_media_id: string;
  conversation_id?: string;
  created_by: string;
  is_disbanded: boolean;
  disbanded_at?: string | null;
  member_count: number;
  created_at: string;
  updated_at: string;
}

export interface GroupMemberResponse {
  user_id: string;
  role: string;
  joined_at: string;
  invited_by: string;
  left_at?: string | null;
}

export interface GroupInvitationResponse {
  invitation_id: string;
  group_id: string;
  group_name?: string;
  inviter_id: string;
  invitee_id: string;
  status: string;
  responded_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface GroupAnnouncementResponse {
  announcement_id: string;
  group_id: string;
  author_id: string;
  content: string;
  is_active: boolean;
  published_at: string;
  retracted_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface GroupSettingsResponse {
  group_id: string;
  invite_policy: string;
  updated_by?: string;
  updated_at?: string | null;
}

export interface CreateGroupInput {
  name: string;
  description?: string;
  avatar_media_id?: string;
  member_ids?: string[];
}

export interface UpdateGroupInput {
  name?: string;
  description?: string;
  avatar_media_id?: string;
}

export interface SetGroupMemberRoleInput {
  role: string;
}

export interface CreateGroupInvitationInput {
  invitee_id: string;
}

export interface GroupAnnouncementInput {
  content: string;
}

export interface UpdateGroupSettingsInput {
  invite_policy?: string;
}

/** Wire shapes for the Phase 06 media endpoints. */
export interface MediaFileResponse {
  file_id: string;
  mime_type: string;
  size_bytes: number;
  original_filename: string;
  width: number;
  height: number;
  duration_seconds: number;
  checksum: string;
  scan_status: string;
  scan_completed_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface MediaUploadResponse {
  upload_id: string;
  media_file_id?: string;
  status: string;
  storage_key: string;
  upload_id_external: string;
  uploaded_bytes: number;
  total_bytes: number;
  part_count: number;
  upload_url?: string;
  upload_url_expires_at?: string | null;
  completed_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface MediaThumbnailResponse {
  thumbnail_id: string;
  file_id: string;
  variant_type: string;
  mime_type: string;
  size_bytes: number;
  width: number;
  height: number;
  created_at: string;
}

export interface MediaAuthorizeResponse {
  allowed: boolean;
  scan_status: string;
  download_url?: string;
  download_url_expires_at?: string | null;
}

export interface UploadUrlResponse {
  upload_url: string;
  expires_at: string;
}

/** Wire shapes for the Timeline endpoints (§2ab). */
export interface PostMediaResponse {
  media_file_id: string;
  position: number;
  caption: string;
}

export interface PostResponse {
  post_id: string;
  author_id: string;
  content: string;
  visibility: string;
  edited_at?: string | null;
  media: PostMediaResponse[];
  like_count: number;
  comment_count: number;
  liked_by_me: boolean;
  created_at: string;
  updated_at: string;
}

export interface PostListResponse {
  posts: PostResponse[];
  next_cursor: string;
}

export interface PostLikeResponse {
  post_id: string;
  liked: boolean;
  like_count: number;
}

export interface PostCommentResponse {
  comment_id: string;
  post_id: string;
  author_id: string;
  parent_comment_id?: string;
  content: string;
  is_deleted: boolean;
  edited_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface PostCommentListResponse {
  comments: PostCommentResponse[];
  next_cursor: string;
}

export interface CreatePostInput {
  content: string;
  visibility?: string;
  media_file_ids?: string[];
}

export interface UpdatePostInput {
  content?: string;
  visibility?: string;
}

export interface CreatePostCommentInput {
  content: string;
  parent_comment_id?: string;
}

/** Wire shapes for the Call endpoints (§2ad). Event responses carry no payload. */
export interface CallResponse {
  call_id: string;
  conversation_id?: string;
  group_id?: string;
  initiator_id: string;
  call_type: string;
  status: string;
  started_at?: string | null;
  ended_at?: string | null;
  duration_seconds?: number | null;
  ended_by?: string;
  end_reason?: string;
  participant_count: number;
  created_at: string;
  updated_at: string;
}

export interface CallListResponse {
  calls: CallResponse[];
  next_cursor: string;
}

export interface CallParticipantResponse {
  user_id: string;
  joined_at?: string | null;
  left_at?: string | null;
  result: string;
  created_at: string;
}

export interface CallEventResponse {
  event_id: string;
  user_id?: string;
  event_type: string;
  occurred_at: string;
}

export interface InitiateCallWireInput {
  conversation_id?: string;
  group_id?: string;
  call_type: string;
}

/** Wire shapes for the Sticker endpoints (§2af). */
export interface StickerPackResponse {
  pack_id: string;
  creator_id: string;
  name: string;
  description: string;
  category: string;
  cover_media_id: string;
  is_free: boolean;
  status: string;
  sticker_count: number;
  favorited_by_me: boolean;
  created_at: string;
  updated_at: string;
}

export interface StickerPackListResponse {
  packs: StickerPackResponse[];
  next_cursor: string;
}

export interface StickerResponse {
  sticker_id: string;
  pack_id: string;
  pack_name: string;
  media_file_id: string;
  name: string;
  is_animated: boolean;
  position: number;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface StickerListResponse {
  stickers: StickerResponse[];
  next_cursor: string;
}

export interface CreateStickerPackInput {
  name: string;
  description: string;
  category: string;
  cover_media_id: string;
  is_free: boolean;
}

export interface UpdateStickerPackInput {
  name?: string;
  description?: string;
  category?: string;
  cover_media_id?: string;
  status?: string;
  is_free?: boolean;
}

export interface AddStickerInput {
  media_file_id: string;
  name: string;
  is_animated: boolean;
}

export interface ReportUploadProgressInput {
  uploaded_bytes: number;
  part_count: number;
}

/**
 * Complete-upload wire input: the same metadata as initiation
 * minus total_bytes — the backend rejects unknown fields, and
 * the complete body carries no total_bytes (see handler.go).
 */
export interface CompleteUploadWireInput {
  mime_type: string;
  original_filename: string;
  width: number;
  height: number;
  duration_seconds: number;
  checksum: string;
}

export interface InitiateUploadWireInput {
  mime_type: string;
  total_bytes: number;
  original_filename: string;
  width: number;
  height: number;
  duration_seconds: number;
  checksum: string;
}

export interface MediaAuthorizeInput {
  access_type: string;
}

export interface EtherApiClientOptions {
  /** Origin of the API service, e.g. http://localhost:8080. Empty means same-origin. */
  baseUrl: string;
  tokenStore: TokenStore;
  fetchFn?: FetchFn;
  /**
   * OIDC refresh hooks (wired only when an OidcSession exists).
   * shouldRefreshTokens reports that the access token is about to
   * expire; refreshTokens performs a refresh-token grant and
   * returns true when a new access token is stored. Both are
   * best-effort: a failure never blocks the request itself, the
   * 401 path below decides the outcome.
   */
  shouldRefreshTokens?: (() => boolean) | undefined;
  refreshTokens?: (() => Promise<boolean>) | undefined;
}

function joinUrl(baseUrl: string, path: string): string {
  const base = baseUrl.replace(/\/+$/, '');
  return `${base}${path}`;
}

function toApiError(status: number, data: unknown): ApiError {
  if (data !== null && typeof data === 'object') {
    const error = (data as { error?: unknown }).error;
    if (error !== null && typeof error === 'object') {
      const record = error as {
        code?: unknown;
        message?: unknown;
        request_id?: unknown;
        details?: unknown;
      };
      if (typeof record.code === 'string') {
        const details = Array.isArray(record.details) ? (record.details as ApiFieldError[]) : [];
        const requestId = typeof record.request_id === 'string' ? record.request_id : undefined;
        const message = typeof record.message === 'string' ? record.message : '請求失敗。';
        if (status === 401) {
          return new AuthError(requestId);
        }
        return new ApiError(status, record.code, message, requestId, details);
      }
    }
  }
  return new ApiError(status, 'ETHER_CLIENT_UNEXPECTED_RESPONSE', '後端回應格式不正確。');
}

export class EtherApiClient {
  private readonly baseUrl: string;
  private readonly tokenStore: TokenStore;
  private readonly fetchFn: FetchFn;
  private readonly shouldRefreshTokens: (() => boolean) | undefined;
  private readonly refreshTokens: (() => Promise<boolean>) | undefined;

  constructor(options: EtherApiClientOptions) {
    this.baseUrl = options.baseUrl;
    this.tokenStore = options.tokenStore;
    this.fetchFn = options.fetchFn ?? fetch;
    this.shouldRefreshTokens = options.shouldRefreshTokens;
    this.refreshTokens = options.refreshTokens;
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    // Proactive refresh: when the access token is about to expire,
    // try the refresh-token grant once before sending the request.
    // A failure here is not fatal; the 401 path below handles it.
    if (this.shouldRefreshTokens?.() === true && this.refreshTokens !== undefined) {
      try {
        await this.refreshTokens();
      } catch {
        // Fall through with the current token; the server decides.
      }
    }
    return this.requestAttempt<T>(method, path, body, false);
  }

  private async requestAttempt<T>(
    method: string,
    path: string,
    body: unknown,
    retried: boolean,
  ): Promise<T> {
    const headers: Record<string, string> = { Accept: 'application/json' };
    const token = this.tokenStore.getToken();
    if (token !== null && token !== '') {
      headers['Authorization'] = `Bearer ${token}`;
    }
    const init: RequestInit = { method, headers };
    if (body !== undefined) {
      headers['Content-Type'] = 'application/json';
      init.body = JSON.stringify(body);
    }
    let response: Response;
    try {
      response = await this.fetchFn(joinUrl(this.baseUrl, path), init);
    } catch {
      throw new ApiError(0, 'ETHER_CLIENT_NETWORK_ERROR', '無法連接後端服務。');
    }
    // One retry after a refresh-token grant: a 401 with a token sent
    // means the access token died; a successful refresh earns exactly
    // one replay with the new token. A second 401 surfaces as usual.
    if (
      response.status === 401 &&
      !retried &&
      token !== null &&
      token !== '' &&
      this.refreshTokens !== undefined
    ) {
      let refreshed = false;
      try {
        refreshed = await this.refreshTokens();
      } catch {
        refreshed = false;
      }
      if (refreshed) {
        return this.requestAttempt<T>(method, path, body, true);
      }
    }
    const text = await response.text();
    let data: unknown = null;
    if (text !== '') {
      try {
        data = JSON.parse(text) as unknown;
      } catch {
        data = null;
      }
    }
    if (!response.ok) {
      throw toApiError(response.status, data);
    }
    return data as T;
  }

  getMe(): Promise<MeResponse> {
    return this.request<MeResponse>('GET', '/api/v1/auth/me');
  }

  listSessions(): Promise<SessionResponse[]> {
    return this.request<SessionResponse[]>('GET', '/api/v1/auth/sessions');
  }

  revokeSession(sessionId: string): Promise<SessionResponse> {
    return this.request<SessionResponse>(
      'POST',
      `/api/v1/auth/sessions/${encodeURIComponent(sessionId)}/revoke`,
    );
  }

  logout(): Promise<{ revoked: boolean }> {
    return this.request<{ revoked: boolean }>('POST', '/api/v1/auth/logout');
  }

  listDevices(): Promise<DeviceResponse[]> {
    return this.request<DeviceResponse[]>('GET', '/api/v1/auth/devices');
  }

  registerDevice(input: RegisterDeviceInput): Promise<DeviceResponse> {
    return this.request<DeviceResponse>('POST', '/api/v1/auth/devices', input);
  }

  revokeDevice(deviceId: string): Promise<DeviceResponse> {
    return this.request<DeviceResponse>(
      'POST',
      `/api/v1/auth/devices/${encodeURIComponent(deviceId)}/revoke`,
    );
  }

  getProfile(): Promise<ProfileResponse> {
    return this.request<ProfileResponse>('GET', '/api/v1/profiles/me');
  }

  updateProfile(patch: ProfilePatch): Promise<ProfileResponse> {
    return this.request<ProfileResponse>('PATCH', '/api/v1/profiles/me', patch);
  }

  getSettings(): Promise<SettingsResponse> {
    return this.request<SettingsResponse>('GET', '/api/v1/users/me/settings');
  }

  updateSettings(patch: SettingsPatch): Promise<SettingsResponse> {
    return this.request<SettingsResponse>('PATCH', '/api/v1/users/me/settings', patch);
  }

  listFriends(includeHidden = false): Promise<FriendResponse[]> {
    const query = includeHidden ? '?include_hidden=true' : '';
    return this.request<FriendResponse[]>('GET', `/api/v1/friends${query}`);
  }

  searchFriend(publicId: string): Promise<SearchUserResponse> {
    return this.request<SearchUserResponse>(
      'GET',
      `/api/v1/friends/search?public_id=${encodeURIComponent(publicId)}`,
    );
  }

  listFriendRequests(direction: 'incoming' | 'outgoing'): Promise<FriendRequestResponse[]> {
    return this.request<FriendRequestResponse[]>(
      'GET',
      `/api/v1/friends/requests?direction=${direction}`,
    );
  }

  sendFriendRequest(input: SendFriendRequestInput): Promise<FriendRequestResponse> {
    return this.request<FriendRequestResponse>('POST', '/api/v1/friends/requests', input);
  }

  respondFriendRequest(requestId: string, accept: boolean): Promise<FriendRequestResponse> {
    const action = accept ? 'accept' : 'decline';
    return this.request<FriendRequestResponse>(
      'POST',
      `/api/v1/friends/requests/${encodeURIComponent(requestId)}/${action}`,
    );
  }

  cancelFriendRequest(requestId: string): Promise<FriendRequestResponse> {
    return this.request<FriendRequestResponse>(
      'POST',
      `/api/v1/friends/requests/${encodeURIComponent(requestId)}/cancel`,
    );
  }

  removeFriend(friendId: string): Promise<{ friend_id: string; status: string }> {
    return this.request<{ friend_id: string; status: string }>(
      'POST',
      `/api/v1/friends/${encodeURIComponent(friendId)}/remove`,
    );
  }

  setFriendHidden(
    friendId: string,
    hidden: boolean,
  ): Promise<{ friend_id: string; status: string; is_hidden?: boolean }> {
    const action = hidden ? 'hide' : 'unhide';
    return this.request<{ friend_id: string; status: string; is_hidden?: boolean }>(
      'POST',
      `/api/v1/friends/${encodeURIComponent(friendId)}/${action}`,
    );
  }

  listBlocks(): Promise<BlockResponse[]> {
    return this.request<BlockResponse[]>('GET', '/api/v1/friends/blocks');
  }

  blockUser(input: BlockUserInput): Promise<BlockResponse> {
    return this.request<BlockResponse>('POST', '/api/v1/friends/blocks', input);
  }

  unblockUser(userId: string): Promise<BlockResponse> {
    return this.request<BlockResponse>(
      'POST',
      `/api/v1/friends/blocks/${encodeURIComponent(userId)}/unblock`,
    );
  }

  listConversations(): Promise<ConversationResponse[]> {
    return this.request<ConversationResponse[]>('GET', '/api/v1/conversations');
  }

  createConversation(input: CreateConversationInput): Promise<ConversationResponse> {
    return this.request<ConversationResponse>('POST', '/api/v1/conversations', input);
  }

  getConversation(conversationId: string): Promise<ConversationResponse> {
    return this.request<ConversationResponse>(
      'GET',
      `/api/v1/conversations/${encodeURIComponent(conversationId)}`,
    );
  }

  listMembers(conversationId: string): Promise<ConversationMemberResponse[]> {
    return this.request<ConversationMemberResponse[]>(
      'GET',
      `/api/v1/conversations/${encodeURIComponent(conversationId)}/members`,
    );
  }

  addMembers(conversationId: string, memberIds: string[]): Promise<ConversationMemberResponse[]> {
    return this.request<ConversationMemberResponse[]>(
      'POST',
      `/api/v1/conversations/${encodeURIComponent(conversationId)}/members`,
      { member_ids: memberIds },
    );
  }

  leaveConversation(conversationId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>(
      'POST',
      `/api/v1/conversations/${encodeURIComponent(conversationId)}/leave`,
    );
  }

  listMessages(
    conversationId: string,
    options: ListMessagesOptions = {},
  ): Promise<MessageListResponse> {
    const params = new URLSearchParams();
    if (options.beforeSeq !== undefined) {
      params.set('before_seq', String(options.beforeSeq));
    }
    if (options.sinceSeq !== undefined) {
      params.set('since_seq', String(options.sinceSeq));
    }
    if (options.limit !== undefined) {
      params.set('limit', String(options.limit));
    }
    const query = params.toString();
    const suffix = query === '' ? '' : `?${query}`;
    return this.request<MessageListResponse>(
      'GET',
      `/api/v1/conversations/${encodeURIComponent(conversationId)}/messages${suffix}`,
    );
  }

  sendMessage(conversationId: string, input: SendMessageInput): Promise<MessageResponse> {
    return this.request<MessageResponse>(
      'POST',
      `/api/v1/conversations/${encodeURIComponent(conversationId)}/messages`,
      input,
    );
  }

  editMessage(messageId: string, content: string): Promise<MessageResponse> {
    return this.request<MessageResponse>(
      'PATCH',
      `/api/v1/messages/${encodeURIComponent(messageId)}`,
      { content },
    );
  }

  recallMessage(messageId: string): Promise<MessageResponse> {
    return this.request<MessageResponse>(
      'POST',
      `/api/v1/messages/${encodeURIComponent(messageId)}/recall`,
    );
  }

  deleteMessage(messageId: string): Promise<MessageResponse> {
    return this.request<MessageResponse>(
      'POST',
      `/api/v1/messages/${encodeURIComponent(messageId)}/delete`,
    );
  }

  postReceipt(messageId: string, input: PostReceiptInput): Promise<ReceiptResponse> {
    return this.request<ReceiptResponse>(
      'POST',
      `/api/v1/messages/${encodeURIComponent(messageId)}/receipts`,
      input,
    );
  }

  listReceipts(messageId: string): Promise<ReceiptResponse[]> {
    return this.request<ReceiptResponse[]>(
      'GET',
      `/api/v1/messages/${encodeURIComponent(messageId)}/receipts`,
    );
  }

  /**
   * Message attachments and forwards (§2s). Adding attachments is
   * an upsert: the backend always answers 200 with the full list.
   * Forwarding answers 201 for a new message and 200 for an
   * idempotent resend; both carry the new (or existing) message.
   */
  listAttachments(messageId: string): Promise<AttachmentListResponse> {
    return this.request<AttachmentListResponse>(
      'GET',
      `/api/v1/messages/${encodeURIComponent(messageId)}/attachments`,
    );
  }

  addAttachments(messageId: string, mediaFileIds: string[]): Promise<AttachmentListResponse> {
    return this.request<AttachmentListResponse>(
      'POST',
      `/api/v1/messages/${encodeURIComponent(messageId)}/attachments`,
      { media_file_ids: mediaFileIds },
    );
  }

  forwardMessage(messageId: string, input: ForwardMessageInput): Promise<MessageResponse> {
    return this.request<MessageResponse>(
      'POST',
      `/api/v1/messages/${encodeURIComponent(messageId)}/forward`,
      input,
    );
  }

  getForward(messageId: string): Promise<ForwardResponse> {
    return this.request<ForwardResponse>(
      'GET',
      `/api/v1/messages/${encodeURIComponent(messageId)}/forward`,
    );
  }

  listReactions(messageId: string): Promise<ReactionSummaryResponse> {
    return this.request<ReactionSummaryResponse>(
      'GET',
      `/api/v1/messages/${encodeURIComponent(messageId)}/reactions`,
    );
  }

  setReaction(messageId: string, input: SetReactionInput): Promise<ReactionSummaryResponse> {
    return this.request<ReactionSummaryResponse>(
      'POST',
      `/api/v1/messages/${encodeURIComponent(messageId)}/reactions`,
      input,
    );
  }

  removeReaction(messageId: string): Promise<ReactionSummaryResponse> {
    return this.request<ReactionSummaryResponse>(
      'DELETE',
      `/api/v1/messages/${encodeURIComponent(messageId)}/reactions`,
    );
  }

  listPins(conversationId: string): Promise<PinListResponse> {
    return this.request<PinListResponse>(
      'GET',
      `/api/v1/conversations/${encodeURIComponent(conversationId)}/pins`,
    );
  }

  pinMessage(conversationId: string, input: PinMessageInput): Promise<PinResponse> {
    return this.request<PinResponse>(
      'POST',
      `/api/v1/conversations/${encodeURIComponent(conversationId)}/pins`,
      input,
    );
  }

  unpinMessage(conversationId: string, messageId: string): Promise<UnpinResponse> {
    return this.request<UnpinResponse>(
      'DELETE',
      `/api/v1/conversations/${encodeURIComponent(conversationId)}/pins/${encodeURIComponent(messageId)}`,
    );
  }

  /**
   * Message search (§2v). The backend answers 200 with an empty
   * result list when the in-process index is empty (the fill loop
   * is a later milestone); an unavailable index answers 503 and
   * must not be read as "no matches".
   */
  searchMessages(input: SearchMessagesInput): Promise<SearchMessagesResponse> {
    const params = new URLSearchParams();
    params.set('q', input.q);
    if (input.conversation_id !== undefined && input.conversation_id !== '') {
      params.set('conversation_id', input.conversation_id);
    }
    if (input.sender_id !== undefined && input.sender_id !== '') {
      params.set('sender_id', input.sender_id);
    }
    if (input.since !== undefined && input.since !== '') {
      params.set('since', input.since);
    }
    if (input.until !== undefined && input.until !== '') {
      params.set('until', input.until);
    }
    if (input.limit !== undefined) {
      params.set('limit', String(input.limit));
    }
    if (input.cursor !== undefined && input.cursor !== '') {
      params.set('cursor', input.cursor);
    }
    return this.request<SearchMessagesResponse>(
      'GET',
      `/api/v1/search/messages?${params.toString()}`,
    );
  }

  listGroups(): Promise<GroupResponse[]> {
    return this.request<GroupResponse[]>('GET', '/api/v1/groups');
  }

  createGroup(input: CreateGroupInput): Promise<GroupResponse> {
    return this.request<GroupResponse>('POST', '/api/v1/groups', input);
  }

  getGroup(groupId: string): Promise<GroupResponse> {
    return this.request<GroupResponse>('GET', `/api/v1/groups/${encodeURIComponent(groupId)}`);
  }

  updateGroup(groupId: string, input: UpdateGroupInput): Promise<GroupResponse> {
    return this.request<GroupResponse>(
      'PATCH',
      `/api/v1/groups/${encodeURIComponent(groupId)}`,
      input,
    );
  }

  disbandGroup(groupId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>(
      'POST',
      `/api/v1/groups/${encodeURIComponent(groupId)}/disband`,
    );
  }

  transferGroupOwnership(groupId: string, newOwnerId: string): Promise<GroupResponse> {
    return this.request<GroupResponse>(
      'POST',
      `/api/v1/groups/${encodeURIComponent(groupId)}/transfer`,
      { new_owner_id: newOwnerId },
    );
  }

  listGroupMembers(groupId: string): Promise<GroupMemberResponse[]> {
    return this.request<GroupMemberResponse[]>(
      'GET',
      `/api/v1/groups/${encodeURIComponent(groupId)}/members`,
    );
  }

  addGroupMembers(groupId: string, memberIds: string[]): Promise<GroupMemberResponse[]> {
    return this.request<GroupMemberResponse[]>(
      'POST',
      `/api/v1/groups/${encodeURIComponent(groupId)}/members`,
      { member_ids: memberIds },
    );
  }

  setGroupMemberRole(
    groupId: string,
    userId: string,
    input: SetGroupMemberRoleInput,
  ): Promise<GroupMemberResponse> {
    return this.request<GroupMemberResponse>(
      'PATCH',
      `/api/v1/groups/${encodeURIComponent(groupId)}/members/${encodeURIComponent(userId)}`,
      input,
    );
  }

  removeGroupMember(groupId: string, userId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>(
      'POST',
      `/api/v1/groups/${encodeURIComponent(groupId)}/members/${encodeURIComponent(userId)}/remove`,
    );
  }

  leaveGroup(groupId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>(
      'POST',
      `/api/v1/groups/${encodeURIComponent(groupId)}/leave`,
    );
  }

  listGroupInvitations(groupId: string): Promise<GroupInvitationResponse[]> {
    return this.request<GroupInvitationResponse[]>(
      'GET',
      `/api/v1/groups/${encodeURIComponent(groupId)}/invitations`,
    );
  }

  createGroupInvitation(
    groupId: string,
    input: CreateGroupInvitationInput,
  ): Promise<GroupInvitationResponse> {
    return this.request<GroupInvitationResponse>(
      'POST',
      `/api/v1/groups/${encodeURIComponent(groupId)}/invitations`,
      input,
    );
  }

  listMyGroupInvitations(): Promise<GroupInvitationResponse[]> {
    return this.request<GroupInvitationResponse[]>('GET', '/api/v1/groups/invitations');
  }

  respondGroupInvitation(invitationId: string, accept: boolean): Promise<GroupInvitationResponse> {
    const action = accept ? 'accept' : 'decline';
    return this.request<GroupInvitationResponse>(
      'POST',
      `/api/v1/groups/invitations/${encodeURIComponent(invitationId)}/${action}`,
    );
  }

  cancelGroupInvitation(invitationId: string): Promise<GroupInvitationResponse> {
    return this.request<GroupInvitationResponse>(
      'POST',
      `/api/v1/groups/invitations/${encodeURIComponent(invitationId)}/cancel`,
    );
  }

  listGroupAnnouncements(groupId: string): Promise<GroupAnnouncementResponse[]> {
    return this.request<GroupAnnouncementResponse[]>(
      'GET',
      `/api/v1/groups/${encodeURIComponent(groupId)}/announcements`,
    );
  }

  createGroupAnnouncement(
    groupId: string,
    input: GroupAnnouncementInput,
  ): Promise<GroupAnnouncementResponse> {
    return this.request<GroupAnnouncementResponse>(
      'POST',
      `/api/v1/groups/${encodeURIComponent(groupId)}/announcements`,
      input,
    );
  }

  updateGroupAnnouncement(
    groupId: string,
    announcementId: string,
    input: GroupAnnouncementInput,
  ): Promise<GroupAnnouncementResponse> {
    return this.request<GroupAnnouncementResponse>(
      'PATCH',
      `/api/v1/groups/${encodeURIComponent(groupId)}/announcements/${encodeURIComponent(announcementId)}`,
      input,
    );
  }

  retractGroupAnnouncement(
    groupId: string,
    announcementId: string,
  ): Promise<GroupAnnouncementResponse> {
    return this.request<GroupAnnouncementResponse>(
      'POST',
      `/api/v1/groups/${encodeURIComponent(groupId)}/announcements/${encodeURIComponent(announcementId)}/retract`,
    );
  }

  getGroupSettings(groupId: string): Promise<GroupSettingsResponse> {
    return this.request<GroupSettingsResponse>(
      'GET',
      `/api/v1/groups/${encodeURIComponent(groupId)}/settings`,
    );
  }

  updateGroupSettings(
    groupId: string,
    input: UpdateGroupSettingsInput,
  ): Promise<GroupSettingsResponse> {
    return this.request<GroupSettingsResponse>(
      'PATCH',
      `/api/v1/groups/${encodeURIComponent(groupId)}/settings`,
      input,
    );
  }

  /**
   * Media endpoints (Phase 06 media, web wiring; byte channel §2z).
   * Upload flow: initiate (returns an upload URL) → PUT the bytes
   * to that URL → report progress → complete. The backend refuses
   * completion until the reported progress equals the declared
   * total and the stored object matches (Stat check), so progress
   * reporting is a required step, not an optional nicety.
   */
  initiateUpload(input: InitiateUploadWireInput): Promise<MediaUploadResponse> {
    return this.request<MediaUploadResponse>('POST', '/api/v1/media/uploads', input);
  }

  issueUploadUrl(uploadId: string): Promise<UploadUrlResponse> {
    return this.request<UploadUrlResponse>(
      'POST',
      `/api/v1/media/uploads/${encodeURIComponent(uploadId)}/upload-url`,
    );
  }

  /**
   * PUT raw bytes to an upload URL. Absolute URLs (MinIO presigned)
   * are fetched as-is with no Authorization header: the signature
   * covers only the host header, and a bearer token must never be
   * sent to object storage. Relative URLs (memory-stub relay) are
   * resolved against the API base URL and sent with the caller's
   * Authorization header, exactly like other API calls.
   */
  async putUploadBytes(url: string, data: Blob): Promise<void> {
    const absolute = /^https?:\/\//i.test(url);
    const headers: Record<string, string> = {};
    if (data.type !== '') {
      headers['Content-Type'] = data.type;
    }
    let target = url;
    if (!absolute) {
      const token = this.tokenStore.getToken();
      if (token !== null && token !== '') {
        headers['Authorization'] = ['Bearer', token].join(' ');
      }
      target = joinUrl(this.baseUrl, url);
    }
    let response: Response;
    try {
      response = await this.fetchFn(target, { method: 'PUT', headers, body: data });
    } catch {
      throw new ApiError(0, 'ETHER_CLIENT_NETWORK_ERROR', '無法連接上傳位址。');
    }
    if (!response.ok) {
      throw await this.toByteError(response);
    }
  }

  /**
   * GET bytes from a relay download path (relative URL, sent with
   * Authorization). Absolute presigned URLs are never fetched
   * through this method: the view opens them in a new tab.
   */
  async fetchDownloadBytes(url: string): Promise<Blob> {
    const headers: Record<string, string> = {};
    const token = this.tokenStore.getToken();
    if (token !== null && token !== '') {
      headers['Authorization'] = ['Bearer', token].join(' ');
    }
    let response: Response;
    try {
      response = await this.fetchFn(joinUrl(this.baseUrl, url), { method: 'GET', headers });
    } catch {
      throw new ApiError(0, 'ETHER_CLIENT_NETWORK_ERROR', '無法連接後端服務。');
    }
    if (!response.ok) {
      throw await this.toByteError(response);
    }
    return response.blob();
  }

  /** Translate a non-JSON byte-endpoint failure into an ApiError. */
  private async toByteError(response: Response): Promise<ApiError> {
    const text = await response.text();
    if (text !== '') {
      try {
        return toApiError(response.status, JSON.parse(text) as unknown);
      } catch {
        // Presigned endpoints answer failures with XML or an empty
        // body; fall through to the generic error below.
      }
    }
    return new ApiError(response.status, 'ETHER_CLIENT_UNEXPECTED_RESPONSE', '位元組傳輸失敗。');
  }

  reportUploadProgress(
    uploadId: string,
    input: ReportUploadProgressInput,
  ): Promise<MediaUploadResponse> {
    return this.request<MediaUploadResponse>(
      'POST',
      `/api/v1/media/uploads/${encodeURIComponent(uploadId)}/progress`,
      input,
    );
  }

  completeUpload(uploadId: string, input: CompleteUploadWireInput): Promise<MediaFileResponse> {
    return this.request<MediaFileResponse>(
      'POST',
      `/api/v1/media/uploads/${encodeURIComponent(uploadId)}/complete`,
      input,
    );
  }

  listUploads(): Promise<MediaUploadResponse[]> {
    return this.request<MediaUploadResponse[]>('GET', '/api/v1/media/uploads');
  }

  getUpload(uploadId: string): Promise<MediaUploadResponse> {
    return this.request<MediaUploadResponse>(
      'GET',
      `/api/v1/media/uploads/${encodeURIComponent(uploadId)}`,
    );
  }

  cancelUpload(uploadId: string): Promise<MediaUploadResponse> {
    return this.request<MediaUploadResponse>(
      'POST',
      `/api/v1/media/uploads/${encodeURIComponent(uploadId)}/cancel`,
    );
  }

  listFiles(): Promise<MediaFileResponse[]> {
    return this.request<MediaFileResponse[]>('GET', '/api/v1/media/files');
  }

  getFile(fileId: string): Promise<MediaFileResponse> {
    return this.request<MediaFileResponse>(
      'GET',
      `/api/v1/media/files/${encodeURIComponent(fileId)}`,
    );
  }

  deleteFile(fileId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>(
      'DELETE',
      `/api/v1/media/files/${encodeURIComponent(fileId)}`,
    );
  }

  listThumbnails(fileId: string): Promise<MediaThumbnailResponse[]> {
    return this.request<MediaThumbnailResponse[]>(
      'GET',
      `/api/v1/media/files/${encodeURIComponent(fileId)}/thumbnails`,
    );
  }

  authorizeAccess(fileId: string, input: MediaAuthorizeInput): Promise<MediaAuthorizeResponse> {
    return this.request<MediaAuthorizeResponse>(
      'POST',
      `/api/v1/media/files/${encodeURIComponent(fileId)}/authorize`,
      input,
    );
  }

  createPost(input: CreatePostInput): Promise<PostResponse> {
    return this.request<PostResponse>('POST', '/api/v1/posts', input);
  }

  getPost(postId: string): Promise<PostResponse> {
    return this.request<PostResponse>('GET', `/api/v1/posts/${encodeURIComponent(postId)}`);
  }

  updatePost(postId: string, input: UpdatePostInput): Promise<PostResponse> {
    return this.request<PostResponse>(
      'PATCH',
      `/api/v1/posts/${encodeURIComponent(postId)}`,
      input,
    );
  }

  deletePost(postId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>(
      'POST',
      `/api/v1/posts/${encodeURIComponent(postId)}/delete`,
    );
  }

  listAuthorPosts(
    authorId: string,
    options: { cursor?: string; limit?: number } = {},
  ): Promise<PostListResponse> {
    const params = new URLSearchParams({ author_id: authorId });
    if (options.cursor !== undefined && options.cursor !== '') {
      params.set('cursor', options.cursor);
    }
    if (options.limit !== undefined) {
      params.set('limit', String(options.limit));
    }
    return this.request<PostListResponse>('GET', `/api/v1/posts?${params.toString()}`);
  }

  listFeed(options: { cursor?: string; limit?: number } = {}): Promise<PostListResponse> {
    const params = new URLSearchParams();
    if (options.cursor !== undefined && options.cursor !== '') {
      params.set('cursor', options.cursor);
    }
    if (options.limit !== undefined) {
      params.set('limit', String(options.limit));
    }
    const query = params.toString();
    return this.request<PostListResponse>(
      'GET',
      query === '' ? '/api/v1/timeline' : `/api/v1/timeline?${query}`,
    );
  }

  listPostMedia(postId: string): Promise<PostMediaResponse[]> {
    return this.request<PostMediaResponse[]>(
      'GET',
      `/api/v1/posts/${encodeURIComponent(postId)}/media`,
    );
  }

  likePost(postId: string): Promise<PostLikeResponse> {
    return this.request<PostLikeResponse>(
      'POST',
      `/api/v1/posts/${encodeURIComponent(postId)}/likes`,
    );
  }

  unlikePost(postId: string): Promise<PostLikeResponse> {
    return this.request<PostLikeResponse>(
      'DELETE',
      `/api/v1/posts/${encodeURIComponent(postId)}/likes`,
    );
  }

  createPostComment(postId: string, input: CreatePostCommentInput): Promise<PostCommentResponse> {
    return this.request<PostCommentResponse>(
      'POST',
      `/api/v1/posts/${encodeURIComponent(postId)}/comments`,
      input,
    );
  }

  listPostComments(
    postId: string,
    options: { cursor?: string; limit?: number } = {},
  ): Promise<PostCommentListResponse> {
    const params = new URLSearchParams();
    if (options.cursor !== undefined && options.cursor !== '') {
      params.set('cursor', options.cursor);
    }
    if (options.limit !== undefined) {
      params.set('limit', String(options.limit));
    }
    const query = params.toString();
    return this.request<PostCommentListResponse>(
      'GET',
      query === ''
        ? `/api/v1/posts/${encodeURIComponent(postId)}/comments`
        : `/api/v1/posts/${encodeURIComponent(postId)}/comments?${query}`,
    );
  }

  deletePostComment(commentId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>(
      'POST',
      `/api/v1/comments/${encodeURIComponent(commentId)}/delete`,
    );
  }

  initiateCall(input: InitiateCallWireInput): Promise<CallResponse> {
    return this.request<CallResponse>('POST', '/api/v1/calls', input);
  }

  listCalls(options: { cursor?: string; limit?: number } = {}): Promise<CallListResponse> {
    const params = new URLSearchParams();
    if (options.cursor !== undefined && options.cursor !== '') {
      params.set('cursor', options.cursor);
    }
    if (options.limit !== undefined) {
      params.set('limit', String(options.limit));
    }
    const query = params.toString();
    return this.request<CallListResponse>(
      'GET',
      query === '' ? '/api/v1/calls' : `/api/v1/calls?${query}`,
    );
  }

  getCall(callId: string): Promise<CallResponse> {
    return this.request<CallResponse>('GET', `/api/v1/calls/${encodeURIComponent(callId)}`);
  }

  answerCall(callId: string): Promise<CallResponse> {
    return this.request<CallResponse>('POST', `/api/v1/calls/${encodeURIComponent(callId)}/answer`);
  }

  declineCall(callId: string): Promise<CallResponse> {
    return this.request<CallResponse>(
      'POST',
      `/api/v1/calls/${encodeURIComponent(callId)}/decline`,
    );
  }

  busyCall(callId: string): Promise<CallResponse> {
    return this.request<CallResponse>('POST', `/api/v1/calls/${encodeURIComponent(callId)}/busy`);
  }

  endCall(callId: string): Promise<CallResponse> {
    return this.request<CallResponse>('POST', `/api/v1/calls/${encodeURIComponent(callId)}/end`);
  }

  listCallParticipants(callId: string): Promise<CallParticipantResponse[]> {
    return this.request<CallParticipantResponse[]>(
      'GET',
      `/api/v1/calls/${encodeURIComponent(callId)}/participants`,
    );
  }

  listCallEvents(callId: string): Promise<CallEventResponse[]> {
    return this.request<CallEventResponse[]>(
      'GET',
      `/api/v1/calls/${encodeURIComponent(callId)}/events`,
    );
  }

  createStickerPack(input: CreateStickerPackInput): Promise<StickerPackResponse> {
    return this.request<StickerPackResponse>('POST', '/api/v1/sticker-packs', input);
  }

  listStickerPacks(
    options: {
      creatorId?: string;
      category?: string;
      q?: string;
      cursor?: string;
      limit?: number;
    } = {},
  ): Promise<StickerPackListResponse> {
    const params = new URLSearchParams();
    if (options.creatorId !== undefined && options.creatorId !== '') {
      params.set('creator_id', options.creatorId);
    }
    if (options.category !== undefined && options.category !== '') {
      params.set('category', options.category);
    }
    if (options.q !== undefined && options.q !== '') {
      params.set('q', options.q);
    }
    if (options.cursor !== undefined && options.cursor !== '') {
      params.set('cursor', options.cursor);
    }
    if (options.limit !== undefined) {
      params.set('limit', String(options.limit));
    }
    const query = params.toString();
    return this.request<StickerPackListResponse>(
      'GET',
      query === '' ? '/api/v1/sticker-packs' : `/api/v1/sticker-packs?${query}`,
    );
  }

  getStickerPack(packId: string): Promise<StickerPackResponse> {
    return this.request<StickerPackResponse>(
      'GET',
      `/api/v1/sticker-packs/${encodeURIComponent(packId)}`,
    );
  }

  updateStickerPack(packId: string, input: UpdateStickerPackInput): Promise<StickerPackResponse> {
    return this.request<StickerPackResponse>(
      'PATCH',
      `/api/v1/sticker-packs/${encodeURIComponent(packId)}`,
      input,
    );
  }

  deleteStickerPack(packId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>(
      'POST',
      `/api/v1/sticker-packs/${encodeURIComponent(packId)}/delete`,
    );
  }

  addSticker(packId: string, input: AddStickerInput): Promise<StickerResponse> {
    return this.request<StickerResponse>(
      'POST',
      `/api/v1/sticker-packs/${encodeURIComponent(packId)}/stickers`,
      input,
    );
  }

  listPackStickers(packId: string): Promise<StickerResponse[]> {
    return this.request<StickerResponse[]>(
      'GET',
      `/api/v1/sticker-packs/${encodeURIComponent(packId)}/stickers`,
    );
  }

  removeSticker(stickerId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>(
      'POST',
      `/api/v1/stickers/${encodeURIComponent(stickerId)}/delete`,
    );
  }

  favoriteStickerPack(packId: string): Promise<StickerPackResponse> {
    return this.request<StickerPackResponse>(
      'POST',
      `/api/v1/sticker-packs/${encodeURIComponent(packId)}/favorite`,
    );
  }

  unfavoriteStickerPack(packId: string): Promise<StickerPackResponse> {
    return this.request<StickerPackResponse>(
      'DELETE',
      `/api/v1/sticker-packs/${encodeURIComponent(packId)}/favorite`,
    );
  }

  listFavoriteStickerPacks(
    options: { cursor?: string; limit?: number } = {},
  ): Promise<StickerPackListResponse> {
    const params = new URLSearchParams();
    if (options.cursor !== undefined && options.cursor !== '') {
      params.set('cursor', options.cursor);
    }
    if (options.limit !== undefined) {
      params.set('limit', String(options.limit));
    }
    const query = params.toString();
    return this.request<StickerPackListResponse>(
      'GET',
      query === '' ? '/api/v1/stickers/favorites' : `/api/v1/stickers/favorites?${query}`,
    );
  }

  recordStickerUse(stickerId: string): Promise<StickerResponse> {
    return this.request<StickerResponse>(
      'POST',
      `/api/v1/stickers/${encodeURIComponent(stickerId)}/use`,
    );
  }

  listRecentStickers(
    options: { cursor?: string; limit?: number } = {},
  ): Promise<StickerListResponse> {
    const params = new URLSearchParams();
    if (options.cursor !== undefined && options.cursor !== '') {
      params.set('cursor', options.cursor);
    }
    if (options.limit !== undefined) {
      params.set('limit', String(options.limit));
    }
    const query = params.toString();
    return this.request<StickerListResponse>(
      'GET',
      query === '' ? '/api/v1/stickers/recent' : `/api/v1/stickers/recent?${query}`,
    );
  }

  searchStickers(
    q: string,
    options: { category?: string; cursor?: string; limit?: number } = {},
  ): Promise<StickerListResponse> {
    if (q.trim() === '') {
      return Promise.reject(new Error('搜尋關鍵字不得為空。'));
    }
    const params = new URLSearchParams({ q });
    if (options.category !== undefined && options.category !== '') {
      params.set('category', options.category);
    }
    if (options.cursor !== undefined && options.cursor !== '') {
      params.set('cursor', options.cursor);
    }
    if (options.limit !== undefined) {
      params.set('limit', String(options.limit));
    }
    return this.request<StickerListResponse>('GET', `/api/v1/stickers/search?${params.toString()}`);
  }
}
