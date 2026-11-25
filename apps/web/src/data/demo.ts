import type {
  BlockItem,
  CallEventItem,
  CallItem,
  CallParticipantItem,
  ChatMessage,
  Contact,
  Conversation,
  DemoProfile,
  DeviceItem,
  FriendItem,
  FriendRequestItem,
  GroupAnnouncementItem,
  GroupInvitationItem,
  GroupItem,
  GroupMemberItem,
  GroupSettingsItem,
  MediaFileItem,
  MediaThumbnailItem,
  MediaUploadItem,
  PostCommentItem,
  PostItem,
  SessionItem,
  StickerItem,
  StickerPackItem,
} from './types';

/**
 * Demonstration data only. Phase 02 has no backend: these records stand in
 * for the future data layer and are always labelled as demo data in the UI.
 * Nothing here is presented as a real account, contact or message history.
 */

export const DEMO_PROFILE: DemoProfile = {
  name: '林曉彤',
  handle: '@xiaotong',
  statusMessage: '在城市與山之間，記得回訊息。',
  presence: 'online',
};

export const DEMO_CONTACTS: Contact[] = [
  {
    id: 'c-mei',
    name: '陳玫',
    handle: '@mei',
    statusMessage: '開會中，晚點回覆',
    presence: 'busy',
    lastSeen: '10 分鐘前上線',
    isFavorite: true,
    group: '家人',
  },
  {
    id: 'c-hao',
    name: '王梓豪',
    handle: '@zihao',
    statusMessage: '今天在台北出差',
    presence: 'online',
    lastSeen: '線上',
    isFavorite: true,
    group: '工作',
  },
  {
    id: 'c-yu',
    name: '張語晨',
    handle: '@yuchen',
    statusMessage: '跑步的時候不要找我',
    presence: 'away',
    lastSeen: '32 分鐘前上線',
    isFavorite: false,
    group: '朋友',
  },
  {
    id: 'c-wei',
    name: '李維哲',
    handle: '@weizhe',
    statusMessage: '',
    presence: 'offline',
    lastSeen: '昨天 22:41 上線',
    isFavorite: false,
    group: '工作',
  },
  {
    id: 'c-ting',
    name: '周亭安',
    handle: '@tingan',
    statusMessage: '新書讀到第三章',
    presence: 'online',
    lastSeen: '線上',
    isFavorite: true,
    group: '朋友',
  },
  {
    id: 'c-jun',
    name: '何俊傑',
    handle: '@junjie',
    statusMessage: '露營裝備清單整理中',
    presence: 'away',
    lastSeen: '1 小時前上線',
    isFavorite: false,
    group: '家人',
  },
];

export const DEMO_CONVERSATIONS: Conversation[] = [
  {
    id: 'v-hao',
    contactId: 'c-hao',
    lastMessagePreview: '提案第三版我剛剛寄出了，你看一下。',
    lastMessageAt: '2026-11-25T12:41:00+08:00',
    unreadCount: 2,
    isPinned: true,
    isMuted: false,
    isTyping: true,
  },
  {
    id: 'v-mei',
    contactId: 'c-mei',
    lastMessagePreview: '週六下午三點，媽媽家吃飯，別遲到。',
    lastMessageAt: '2026-11-25T11:05:00+08:00',
    unreadCount: 0,
    isPinned: true,
    isMuted: false,
    isTyping: false,
  },
  {
    id: 'v-ting',
    contactId: 'c-ting',
    lastMessagePreview: '你傳的語音我聽了，想法很不錯。',
    lastMessageAt: '2026-11-25T09:52:00+08:00',
    unreadCount: 1,
    isPinned: false,
    isMuted: false,
    isTyping: false,
  },
  {
    id: 'v-yu',
    contactId: 'c-yu',
    lastMessagePreview: '照片收到了，謝謝！',
    lastMessageAt: '2026-11-24T21:18:00+08:00',
    unreadCount: 0,
    isPinned: false,
    isMuted: true,
    isTyping: false,
  },
  {
    id: 'v-wei',
    contactId: 'c-wei',
    lastMessagePreview: '會議記錄整理好了。',
    lastMessageAt: '2026-11-24T18:03:00+08:00',
    unreadCount: 0,
    isPinned: false,
    isMuted: false,
    isTyping: false,
  },
];

export const DEMO_MESSAGES: ChatMessage[] = [
  {
    id: 'm-hao-1',
    conversationId: 'v-hao',
    senderId: 'c-hao',
    body: '早安，提案第二版客戶看過了，有三個地方要調整。',
    sentAt: '2026-11-25T09:12:00+08:00',
    status: 'read',
    kind: 'text',
  },
  {
    id: 'm-hao-2',
    conversationId: 'v-hao',
    senderId: 'me',
    body: '收到，我中午前把調整清單列給你。',
    sentAt: '2026-11-25T09:20:00+08:00',
    status: 'read',
    kind: 'text',
  },
  {
    id: 'm-hao-3',
    conversationId: 'v-hao',
    senderId: 'c-hao',
    body: '提案第三版我剛剛寄出了，你看一下。',
    sentAt: '2026-11-25T12:41:00+08:00',
    status: 'delivered',
    kind: 'text',
  },
  {
    id: 'm-mei-1',
    conversationId: 'v-mei',
    senderId: 'c-mei',
    body: '週六下午三點，媽媽家吃飯，別遲到。',
    sentAt: '2026-11-25T11:05:00+08:00',
    status: 'read',
    kind: 'text',
  },
  {
    id: 'm-mei-2',
    conversationId: 'v-mei',
    senderId: 'me',
    body: '好，我會提早到，順便帶水果過去。',
    sentAt: '2026-11-25T11:09:00+08:00',
    status: 'read',
    kind: 'text',
  },
  {
    id: 'm-ting-1',
    conversationId: 'v-ting',
    senderId: 'c-ting',
    body: '你傳的語音我聽了，想法很不錯。',
    sentAt: '2026-11-25T09:52:00+08:00',
    status: 'read',
    kind: 'voice',
    durationSeconds: 42,
  },
  {
    id: 'm-ting-2',
    conversationId: 'v-ting',
    senderId: 'me',
    body: '謝謝！我把完整版整理成文件再傳給你。',
    sentAt: '2026-11-25T09:55:00+08:00',
    status: 'read',
    kind: 'text',
  },
  {
    id: 'm-yu-1',
    conversationId: 'v-yu',
    senderId: 'me',
    body: '週末爬山的照片在這裡。',
    sentAt: '2026-11-24T21:14:00+08:00',
    status: 'read',
    kind: 'attachment',
    fileName: 'mountain-weekend.jpg',
    fileSize: '2.4 MB',
  },
  {
    id: 'm-yu-2',
    conversationId: 'v-yu',
    senderId: 'c-yu',
    body: '照片收到了，謝謝！',
    sentAt: '2026-11-24T21:18:00+08:00',
    status: 'read',
    kind: 'text',
  },
];

/**
 * Demonstration sessions and devices (M2c). Fictional records in the
 * same shape as the API-backed provider returns; revoking one only
 * changes this local demonstration state. Nothing here is a real
 * account, device or session.
 */
export const DEMO_SESSIONS: SessionItem[] = [
  {
    id: 'demo-session-phone',
    deviceId: 'demo-device-phone',
    expiresAt: '2026-11-26T12:00:00+08:00',
    lastActiveAt: '2026-11-25T14:20:00+08:00',
    isRevoked: false,
    revokedReason: '',
  },
  {
    id: 'demo-session-laptop',
    deviceId: 'demo-device-laptop',
    expiresAt: '2026-11-26T09:00:00+08:00',
    lastActiveAt: '2026-11-25T10:05:00+08:00',
    isRevoked: true,
    revokedReason: 'logout',
  },
];

export const DEMO_DEVICES: DeviceItem[] = [
  {
    id: 'demo-device-phone',
    name: '展示手機',
    platform: 'android',
    isTrusted: true,
    isRevoked: false,
    lastActiveAt: '2026-11-25T14:20:00+08:00',
    createdAt: '2026-11-20T09:00:00+08:00',
  },
  {
    id: 'demo-device-laptop',
    name: '展示筆電',
    platform: 'web',
    isTrusted: false,
    isRevoked: true,
    lastActiveAt: '2026-11-25T10:05:00+08:00',
    createdAt: '2026-11-18T20:30:00+08:00',
  },
];

/** Demonstration friends (M3): the demo contacts, as friend entries. */
export const DEMO_FRIENDS: FriendItem[] = DEMO_CONTACTS.map((contact, index) => ({
  id: contact.id,
  displayName: contact.name,
  publicId: contact.handle.replace(/^@/, ''),
  avatarMediaId: '',
  statusMessage: contact.statusMessage,
  isHidden: index === DEMO_CONTACTS.length - 1,
  establishedAt: '2026-11-20T09:00:00+08:00',
}));

/** Demonstration pending friend requests (M3, fictitious records). */
export const DEMO_FRIEND_REQUESTS: FriendRequestItem[] = [
  {
    id: 'demo-request-in-1',
    direction: 'incoming',
    otherId: 'c-mei',
    otherDisplayName: '陳玫',
    otherPublicId: 'mei',
    message: '好久不見，加個好友吧。',
    createdAt: '2026-11-25T09:30:00+08:00',
  },
  {
    id: 'demo-request-out-1',
    direction: 'outgoing',
    otherId: 'c-yu',
    otherDisplayName: '張語晨',
    otherPublicId: 'yuchen',
    message: '',
    createdAt: '2026-11-25T08:15:00+08:00',
  },
];

/** Demonstration active blocks (M3, fictitious records). */
export const DEMO_BLOCKS: BlockItem[] = [
  {
    userId: 'c-wei',
    displayName: '李維哲',
    publicId: 'weizhe',
    reason: '廣告訊息過多',
    blockedAt: '2026-11-24T18:00:00+08:00',
  },
];

/**
 * Demonstration groups (M6 follow-up). Fictitious records in the
 * same shape as the API-backed provider returns. The signed-in user
 * ('me') owns the first group and is an ordinary member of the
 * second; group actions only change this local demonstration state.
 */
export const DEMO_GROUPS: GroupItem[] = [
  {
    id: 'demo-group-hiking',
    name: '週末爬山小隊',
    description: '約爬山、分享路線與裝備清單。',
    avatarMediaId: '',
    conversationId: '',
    createdBy: 'me',
    isDisbanded: false,
    memberCount: 4,
    createdAt: '2026-11-20T09:00:00+08:00',
    updatedAt: '2026-11-24T18:30:00+08:00',
  },
  {
    id: 'demo-group-reading',
    name: '讀書會',
    description: '每月一本書，週五晚上線上討論。',
    avatarMediaId: '',
    conversationId: '',
    createdBy: 'c-ting',
    isDisbanded: false,
    memberCount: 3,
    createdAt: '2026-11-18T20:00:00+08:00',
    updatedAt: '2026-11-23T21:00:00+08:00',
  },
];

/** Demonstration group members, keyed by group id (M6 follow-up). */
export const DEMO_GROUP_MEMBERS: Record<string, GroupMemberItem[]> = {
  'demo-group-hiking': [
    {
      userId: 'me',
      displayName: '林曉彤',
      role: 'owner',
      joinedAt: '2026-11-20T09:00:00+08:00',
      invitedBy: 'me',
      leftAt: null,
    },
    {
      userId: 'c-yu',
      displayName: '張語晨',
      role: 'admin',
      joinedAt: '2026-11-20T10:15:00+08:00',
      invitedBy: 'me',
      leftAt: null,
    },
    {
      userId: 'c-jun',
      displayName: '何俊傑',
      role: 'member',
      joinedAt: '2026-11-21T14:00:00+08:00',
      invitedBy: 'c-yu',
      leftAt: null,
    },
    {
      userId: 'c-hao',
      displayName: '王梓豪',
      role: 'member',
      joinedAt: '2026-11-22T09:30:00+08:00',
      invitedBy: 'me',
      leftAt: null,
    },
  ],
  'demo-group-reading': [
    {
      userId: 'c-ting',
      displayName: '周亭安',
      role: 'owner',
      joinedAt: '2026-11-18T20:00:00+08:00',
      invitedBy: 'c-ting',
      leftAt: null,
    },
    {
      userId: 'me',
      displayName: '林曉彤',
      role: 'member',
      joinedAt: '2026-11-19T08:00:00+08:00',
      invitedBy: 'c-ting',
      leftAt: null,
    },
    {
      userId: 'c-mei',
      displayName: '陳玫',
      role: 'member',
      joinedAt: '2026-11-19T09:20:00+08:00',
      invitedBy: 'c-ting',
      leftAt: null,
    },
  ],
};

/** Demonstration group invitations (M6 follow-up, fictitious records). */
export const DEMO_GROUP_INVITATIONS: GroupInvitationItem[] = [
  {
    id: 'demo-group-inv-1',
    groupId: 'demo-group-hiking',
    groupName: '週末爬山小隊',
    inviterId: 'me',
    inviteeId: 'c-wei',
    status: 'pending',
    createdAt: '2026-11-24T19:00:00+08:00',
    updatedAt: '2026-11-24T19:00:00+08:00',
  },
  {
    id: 'demo-group-inv-2',
    groupId: 'demo-group-piano',
    groupName: '鋼琴同好會',
    inviterId: 'c-jun',
    inviteeId: 'me',
    status: 'pending',
    createdAt: '2026-11-25T08:30:00+08:00',
    updatedAt: '2026-11-25T08:30:00+08:00',
  },
];

/** Demonstration group announcements (M6 follow-up). */
export const DEMO_GROUP_ANNOUNCEMENTS: GroupAnnouncementItem[] = [
  {
    id: 'demo-group-ann-1',
    groupId: 'demo-group-hiking',
    authorId: 'c-yu',
    content: '本週六陽明山東西大縱走，七點在劍潭站集合，記得帶水與雨具。',
    isActive: true,
    publishedAt: '2026-11-24T18:30:00+08:00',
    retractedAt: null,
    createdAt: '2026-11-24T18:30:00+08:00',
    updatedAt: '2026-11-24T18:30:00+08:00',
  },
  {
    id: 'demo-group-ann-2',
    groupId: 'demo-group-hiking',
    authorId: 'me',
    content: '上週的裝備清單已整理在群組檔案區。',
    isActive: false,
    publishedAt: '2026-11-21T12:00:00+08:00',
    retractedAt: '2026-11-24T18:30:00+08:00',
    createdAt: '2026-11-21T12:00:00+08:00',
    updatedAt: '2026-11-24T18:30:00+08:00',
  },
];

/** Demonstration group settings (M6 follow-up). */
export const DEMO_GROUP_SETTINGS: GroupSettingsItem[] = [
  {
    groupId: 'demo-group-hiking',
    invitePolicy: 'admin',
    updatedBy: 'me',
    updatedAt: '2026-11-20T09:00:00+08:00',
  },
  {
    groupId: 'demo-group-reading',
    invitePolicy: 'all',
    updatedBy: 'c-ting',
    updatedAt: '2026-11-18T20:00:00+08:00',
  },
];

/**
 * Demonstration media files (Phase 06 media, web wiring). Fictitious
 * records in the same shape as the API-backed provider returns. One
 * file per scan state, so the scan gate is visible in every state:
 * only the clean file may be presented as distributable.
 */
export const DEMO_MEDIA_FILES: MediaFileItem[] = [
  {
    id: 'demo-file-photo',
    mimeType: 'image/jpeg',
    sizeBytes: 184320,
    originalFilename: '陽明山雲海.jpg',
    width: 1920,
    height: 1080,
    durationSeconds: 0,
    checksum: '',
    scanStatus: 'clean',
    scanCompletedAt: '2026-11-24T19:05:00+08:00',
    createdAt: '2026-11-24T19:00:00+08:00',
    updatedAt: '2026-11-24T19:05:00+08:00',
  },
  {
    id: 'demo-file-video',
    mimeType: 'video/mp4',
    sizeBytes: 48234496,
    originalFilename: '登山口集合.mp4',
    width: 1920,
    height: 1080,
    durationSeconds: 96,
    checksum: '',
    scanStatus: 'pending',
    scanCompletedAt: null,
    createdAt: '2026-11-25T08:12:00+08:00',
    updatedAt: '2026-11-25T08:12:00+08:00',
  },
  {
    id: 'demo-file-doc',
    mimeType: 'application/pdf',
    sizeBytes: 245760,
    originalFilename: '裝備清單.pdf',
    width: 0,
    height: 0,
    durationSeconds: 0,
    checksum: '',
    scanStatus: 'rejected',
    scanCompletedAt: '2026-11-24T20:40:00+08:00',
    createdAt: '2026-11-24T20:30:00+08:00',
    updatedAt: '2026-11-24T20:40:00+08:00',
  },
];

/** Demonstration upload sessions (Phase 06 media, web wiring). */
export const DEMO_MEDIA_UPLOADS: MediaUploadItem[] = [
  {
    id: 'demo-upload-1',
    mediaFileId: '',
    status: 'uploading',
    storageKey: 'media/demo/demo-upload-1',
    externalUploadId: '',
    uploadedBytes: 0,
    totalBytes: 48234496,
    partCount: 0,
    uploadUrl: null,
    uploadUrlExpiresAt: null,
    completedAt: null,
    createdAt: '2026-11-25T08:12:00+08:00',
    updatedAt: '2026-11-25T08:12:00+08:00',
  },
  {
    id: 'demo-upload-2',
    mediaFileId: 'demo-file-photo',
    status: 'completed',
    storageKey: 'media/demo/demo-file-photo',
    externalUploadId: '',
    uploadedBytes: 184320,
    totalBytes: 184320,
    partCount: 1,
    uploadUrl: null,
    uploadUrlExpiresAt: null,
    completedAt: '2026-11-24T19:00:00+08:00',
    createdAt: '2026-11-24T18:58:00+08:00',
    updatedAt: '2026-11-24T19:00:00+08:00',
  },
];

/**
 * Demonstration thumbnail records (Phase 06 media, web wiring).
 * Metadata only: no distribution URL exists, so the view shows a
 * placeholder for each record.
 */
export const DEMO_MEDIA_THUMBNAILS: MediaThumbnailItem[] = [
  {
    id: 'demo-thumb-1',
    fileId: 'demo-file-photo',
    variantType: 'thumbnail',
    mimeType: 'image/jpeg',
    sizeBytes: 18432,
    width: 320,
    height: 180,
    createdAt: '2026-11-24T19:05:00+08:00',
  },
  {
    id: 'demo-thumb-2',
    fileId: 'demo-file-photo',
    variantType: 'preview',
    mimeType: 'image/jpeg',
    sizeBytes: 92160,
    width: 1280,
    height: 720,
    createdAt: '2026-11-24T19:05:00+08:00',
  },
];

/**
 * Demonstration timeline posts (Phase 08, web wiring). Fictitious
 * records in the same shape as the API-backed provider returns. One
 * post per visibility value, so the visibility filter is visible:
 * the blocked author's public post and the only_me post are never
 * shown to the signed-in user ('me'), mirroring the backend guards.
 * Counts are derived by the demo provider from the like/comment
 * fixtures below, like the backend's live aggregation.
 */
export const DEMO_POSTS: PostItem[] = [
  {
    id: 'demo-post-1',
    authorId: 'c-mei',
    content: '週六下午三點，媽媽家吃飯，別遲到。照片是上次聚餐拍的。',
    visibility: 'public',
    editedAt: null,
    media: [{ mediaFileId: 'demo-file-photo', position: 0, caption: '' }],
    likeCount: 0,
    commentCount: 0,
    likedByMe: false,
    createdAt: '2026-11-25T10:30:00+08:00',
    updatedAt: '2026-11-25T10:30:00+08:00',
  },
  {
    id: 'demo-post-2',
    authorId: 'c-hao',
    content: '提案第三版終於寄出了，台北出差的最後一天。',
    visibility: 'friends',
    editedAt: '2026-11-25T11:00:00+08:00',
    media: [],
    likeCount: 0,
    commentCount: 0,
    likedByMe: false,
    createdAt: '2026-11-25T09:00:00+08:00',
    updatedAt: '2026-11-25T11:00:00+08:00',
  },
  {
    id: 'demo-post-3',
    authorId: 'me',
    content: '爬山照片整理好了，雲海那張最喜歡。',
    visibility: 'friends',
    editedAt: null,
    media: [{ mediaFileId: 'demo-file-photo', position: 0, caption: '' }],
    likeCount: 0,
    commentCount: 0,
    likedByMe: false,
    createdAt: '2026-11-24T20:00:00+08:00',
    updatedAt: '2026-11-24T20:00:00+08:00',
  },
  {
    id: 'demo-post-4',
    authorId: 'c-wei',
    content: '限時優惠，錯過不再有。',
    visibility: 'public',
    editedAt: null,
    media: [],
    likeCount: 0,
    commentCount: 0,
    likedByMe: false,
    createdAt: '2026-11-25T12:00:00+08:00',
    updatedAt: '2026-11-25T12:00:00+08:00',
  },
  {
    id: 'demo-post-5',
    authorId: 'c-yu',
    content: '只給自己看的跑步紀錄。',
    visibility: 'only_me',
    editedAt: null,
    media: [],
    likeCount: 0,
    commentCount: 0,
    likedByMe: false,
    createdAt: '2026-11-25T07:00:00+08:00',
    updatedAt: '2026-11-25T07:00:00+08:00',
  },
];

/** Demonstration active likes, keyed by post id (Phase 08). */
export const DEMO_POST_LIKED_BY: Record<string, string[]> = {
  'demo-post-1': ['c-hao', 'c-ting'],
  'demo-post-2': ['me'],
  'demo-post-3': ['c-mei'],
};

/**
 * Demonstration post comments (Phase 08). Includes a reply and a
 * deleted comment whose content reads back empty, mirroring the
 * backend's soft-delete display rule.
 */
export const DEMO_POST_COMMENTS: PostCommentItem[] = [
  {
    id: 'demo-comment-1',
    postId: 'demo-post-1',
    authorId: 'c-hao',
    parentCommentId: '',
    content: '收到，週六見！',
    isDeleted: false,
    editedAt: null,
    createdAt: '2026-11-25T10:35:00+08:00',
    updatedAt: '2026-11-25T10:35:00+08:00',
  },
  {
    id: 'demo-comment-2',
    postId: 'demo-post-1',
    authorId: 'c-mei',
    parentCommentId: 'demo-comment-1',
    content: '記得帶水果過來。',
    isDeleted: false,
    editedAt: null,
    createdAt: '2026-11-25T10:40:00+08:00',
    updatedAt: '2026-11-25T10:40:00+08:00',
  },
  {
    id: 'demo-comment-3',
    postId: 'demo-post-1',
    authorId: 'c-yu',
    parentCommentId: '',
    content: '這句話已經刪除了。',
    isDeleted: true,
    editedAt: null,
    createdAt: '2026-11-25T10:45:00+08:00',
    updatedAt: '2026-11-25T10:50:00+08:00',
  },
  {
    id: 'demo-comment-4',
    postId: 'demo-post-2',
    authorId: 'me',
    parentCommentId: '',
    content: '辛苦了，客戶回覆如何？',
    isDeleted: false,
    editedAt: null,
    createdAt: '2026-11-25T09:30:00+08:00',
    updatedAt: '2026-11-25T09:30:00+08:00',
  },
];

/**
 * Demonstration call sessions (Phase 07, web wiring). One of each
 * headline outcome: an answered call that ended, a missed call, a
 * declined call, and an incoming ringing call the caller has not
 * answered yet (so the answer / decline / busy actions are
 * reachable in demo mode). No media is attached to any fixture:
 * the demo provider mirrors the backend's session-only rules.
 */
export const DEMO_CALLS: CallItem[] = [
  {
    id: 'demo-call-1',
    conversationId: 'v-hao',
    groupId: '',
    initiatorId: 'me',
    callType: 'voice',
    status: 'ended',
    startedAt: '2026-11-25T11:02:00+08:00',
    endedAt: '2026-11-25T11:14:30+08:00',
    durationSeconds: 750,
    endedBy: 'me',
    endReason: 'hangup',
    participantCount: 2,
    createdAt: '2026-11-25T11:01:00+08:00',
    updatedAt: '2026-11-25T11:14:30+08:00',
  },
  {
    id: 'demo-call-2',
    conversationId: 'v-ting',
    groupId: '',
    initiatorId: 'c-ting',
    callType: 'video',
    status: 'missed',
    startedAt: null,
    endedAt: '2026-11-25T09:58:00+08:00',
    durationSeconds: null,
    endedBy: 'c-ting',
    endReason: 'missed',
    participantCount: 2,
    createdAt: '2026-11-25T09:55:00+08:00',
    updatedAt: '2026-11-25T09:58:00+08:00',
  },
  {
    id: 'demo-call-3',
    conversationId: '',
    groupId: 'demo-group-hiking',
    initiatorId: 'me',
    callType: 'voice',
    status: 'declined',
    startedAt: null,
    endedAt: '2026-11-25T08:20:00+08:00',
    durationSeconds: null,
    endedBy: 'c-yu',
    endReason: 'declined',
    participantCount: 4,
    createdAt: '2026-11-25T08:18:00+08:00',
    updatedAt: '2026-11-25T08:20:00+08:00',
  },
  {
    id: 'demo-call-4',
    conversationId: 'v-mei',
    groupId: '',
    initiatorId: 'c-mei',
    callType: 'video',
    status: 'ringing',
    startedAt: null,
    endedAt: null,
    durationSeconds: null,
    endedBy: '',
    endReason: '',
    participantCount: 2,
    createdAt: '2026-11-25T12:50:00+08:00',
    updatedAt: '2026-11-25T12:50:00+08:00',
  },
];

/** Demonstration call participants, keyed by call id (Phase 07). */
export const DEMO_CALL_PARTICIPANTS: Record<string, CallParticipantItem[]> = {
  'demo-call-1': [
    {
      userId: 'me',
      joinedAt: '2026-11-25T11:01:00+08:00',
      leftAt: '2026-11-25T11:14:30+08:00',
      result: 'joined',
      createdAt: '2026-11-25T11:01:00+08:00',
    },
    {
      userId: 'c-hao',
      joinedAt: '2026-11-25T11:02:00+08:00',
      leftAt: '2026-11-25T11:14:30+08:00',
      result: 'joined',
      createdAt: '2026-11-25T11:01:00+08:00',
    },
  ],
  'demo-call-2': [
    {
      userId: 'c-ting',
      joinedAt: '2026-11-25T09:55:00+08:00',
      leftAt: '2026-11-25T09:58:00+08:00',
      result: 'joined',
      createdAt: '2026-11-25T09:55:00+08:00',
    },
    {
      userId: 'me',
      joinedAt: null,
      leftAt: null,
      result: 'missed',
      createdAt: '2026-11-25T09:55:00+08:00',
    },
  ],
  'demo-call-3': [
    {
      userId: 'me',
      joinedAt: '2026-11-25T08:18:00+08:00',
      leftAt: '2026-11-25T08:20:00+08:00',
      result: 'joined',
      createdAt: '2026-11-25T08:18:00+08:00',
    },
    {
      userId: 'c-yu',
      joinedAt: null,
      leftAt: null,
      result: 'declined',
      createdAt: '2026-11-25T08:18:00+08:00',
    },
    {
      userId: 'c-jun',
      joinedAt: null,
      leftAt: null,
      result: 'busy',
      createdAt: '2026-11-25T08:18:00+08:00',
    },
    {
      userId: 'c-hao',
      joinedAt: null,
      leftAt: null,
      result: 'declined',
      createdAt: '2026-11-25T08:18:00+08:00',
    },
  ],
  'demo-call-4': [
    {
      userId: 'c-mei',
      joinedAt: '2026-11-25T12:50:00+08:00',
      leftAt: null,
      result: 'joined',
      createdAt: '2026-11-25T12:50:00+08:00',
    },
    {
      userId: 'me',
      joinedAt: null,
      leftAt: null,
      result: 'pending',
      createdAt: '2026-11-25T12:50:00+08:00',
    },
  ],
};

/** Demonstration call events, keyed by call id (Phase 07). No payloads. */
export const DEMO_CALL_EVENTS: Record<string, CallEventItem[]> = {
  'demo-call-1': [
    {
      id: 'demo-call-event-1',
      userId: 'me',
      eventType: 'initiated',
      occurredAt: '2026-11-25T11:01:00+08:00',
    },
    {
      id: 'demo-call-event-2',
      userId: 'c-hao',
      eventType: 'answered',
      occurredAt: '2026-11-25T11:02:00+08:00',
    },
    {
      id: 'demo-call-event-3',
      userId: 'me',
      eventType: 'ended',
      occurredAt: '2026-11-25T11:14:30+08:00',
    },
  ],
  'demo-call-2': [
    {
      id: 'demo-call-event-4',
      userId: 'c-ting',
      eventType: 'initiated',
      occurredAt: '2026-11-25T09:55:00+08:00',
    },
    {
      id: 'demo-call-event-5',
      userId: 'c-ting',
      eventType: 'ended',
      occurredAt: '2026-11-25T09:58:00+08:00',
    },
  ],
  'demo-call-3': [
    {
      id: 'demo-call-event-6',
      userId: 'me',
      eventType: 'initiated',
      occurredAt: '2026-11-25T08:18:00+08:00',
    },
    {
      id: 'demo-call-event-7',
      userId: 'c-yu',
      eventType: 'declined',
      occurredAt: '2026-11-25T08:19:00+08:00',
    },
    {
      id: 'demo-call-event-8',
      userId: 'c-jun',
      eventType: 'busy',
      occurredAt: '2026-11-25T08:19:30+08:00',
    },
    {
      id: 'demo-call-event-9',
      userId: 'c-hao',
      eventType: 'declined',
      occurredAt: '2026-11-25T08:20:00+08:00',
    },
    {
      id: 'demo-call-event-10',
      userId: 'c-hao',
      eventType: 'ended',
      occurredAt: '2026-11-25T08:20:00+08:00',
    },
  ],
  'demo-call-4': [
    {
      id: 'demo-call-event-11',
      userId: 'c-mei',
      eventType: 'initiated',
      occurredAt: '2026-11-25T12:50:00+08:00',
    },
  ],
};

/**
 * Demonstration sticker packs (Phase 08). Counts and favorite
 * state are computed live by the demo provider, like post counts:
 * the fixtures carry zeros here. Visibility cases, judged for
 * 'me': a free published pack (visible), a paid published pack
 * with a grant row (visible), a paid published pack without one
 * (hidden, fail-closed), an own draft (visible to its author
 * only), another author's unlisted pack (hidden), and an
 * archived pack (hidden everywhere). Sticker bodies only reference
 * the demo photo file id: tiles render names, never bytes.
 */
export const DEMO_STICKER_PACKS: StickerPackItem[] = [
  {
    id: 'demo-pack-1',
    creatorId: 'c-mei',
    name: '山間小日子',
    description: '爬山、咖啡與雨天的日常貼圖。',
    category: 'general',
    coverMediaId: 'demo-file-photo',
    isFree: true,
    status: 'published',
    stickerCount: 0,
    favoritedByMe: false,
    createdAt: '2026-11-25T10:00:00+08:00',
    updatedAt: '2026-11-25T10:00:00+08:00',
  },
  {
    id: 'demo-pack-2',
    creatorId: 'c-hao',
    name: '貓咪日常',
    description: '家裡的貓每天都在做的事。',
    category: 'animals',
    coverMediaId: '',
    isFree: true,
    status: 'published',
    stickerCount: 0,
    favoritedByMe: false,
    createdAt: '2026-11-25T09:00:00+08:00',
    updatedAt: '2026-11-25T09:00:00+08:00',
  },
  {
    id: 'demo-pack-3',
    creatorId: 'c-ting',
    name: '心情點點',
    description: '付費貼圖包，展示資料已授權給目前使用者。',
    category: 'emotions',
    coverMediaId: '',
    isFree: false,
    status: 'published',
    stickerCount: 0,
    favoritedByMe: false,
    createdAt: '2026-11-25T08:00:00+08:00',
    updatedAt: '2026-11-25T08:00:00+08:00',
  },
  {
    id: 'demo-pack-4',
    creatorId: 'c-yu',
    name: '煙火節慶',
    description: '付費貼圖包，展示資料未授權，不可見。',
    category: 'festivals',
    coverMediaId: '',
    isFree: false,
    status: 'published',
    stickerCount: 0,
    favoritedByMe: false,
    createdAt: '2026-11-25T07:00:00+08:00',
    updatedAt: '2026-11-25T07:00:00+08:00',
  },
  {
    id: 'demo-pack-5',
    creatorId: 'me',
    name: '我的草稿包',
    description: '尚未上架，只有作者看得到。',
    category: 'other',
    coverMediaId: '',
    isFree: true,
    status: 'draft',
    stickerCount: 0,
    favoritedByMe: false,
    createdAt: '2026-11-25T11:00:00+08:00',
    updatedAt: '2026-11-25T11:00:00+08:00',
  },
  {
    id: 'demo-pack-6',
    creatorId: 'c-wei',
    name: '下架舊包',
    description: '曾上架後下架，只有作者看得到。',
    category: 'food',
    coverMediaId: '',
    isFree: true,
    status: 'unlisted',
    stickerCount: 0,
    favoritedByMe: false,
    createdAt: '2026-11-24T18:00:00+08:00',
    updatedAt: '2026-11-24T18:00:00+08:00',
  },
  {
    id: 'demo-pack-7',
    creatorId: 'c-mei',
    name: '已封存包',
    description: '已刪除的狀態標記，讀取一律不存在。',
    category: 'characters',
    coverMediaId: '',
    isFree: true,
    status: 'archived',
    stickerCount: 0,
    favoritedByMe: false,
    createdAt: '2026-11-24T12:00:00+08:00',
    updatedAt: '2026-11-24T12:00:00+08:00',
  },
];

/** Demonstration stickers, flat by pack (Phase 08). One removed row is kept for the position rule. */
export const DEMO_STICKERS: StickerItem[] = [
  {
    id: 'demo-sticker-1',
    packId: 'demo-pack-1',
    packName: '山間小日子',
    mediaFileId: 'demo-file-photo',
    name: '咖啡時間',
    isAnimated: false,
    position: 0,
    status: 'active',
    createdAt: '2026-11-25T10:00:00+08:00',
    updatedAt: '2026-11-25T10:00:00+08:00',
  },
  {
    id: 'demo-sticker-2',
    packId: 'demo-pack-1',
    packName: '山間小日子',
    mediaFileId: 'demo-file-photo',
    name: '雨天散步',
    isAnimated: false,
    position: 1,
    status: 'active',
    createdAt: '2026-11-25T10:01:00+08:00',
    updatedAt: '2026-11-25T10:01:00+08:00',
  },
  {
    id: 'demo-sticker-3',
    packId: 'demo-pack-1',
    packName: '山間小日子',
    mediaFileId: 'demo-file-photo',
    name: '山頂雲海',
    isAnimated: true,
    position: 2,
    status: 'active',
    createdAt: '2026-11-25T10:02:00+08:00',
    updatedAt: '2026-11-25T10:02:00+08:00',
  },
  {
    id: 'demo-sticker-12',
    packId: 'demo-pack-1',
    packName: '山間小日子',
    mediaFileId: 'demo-file-photo',
    name: '已移除貼圖',
    isAnimated: false,
    position: 3,
    status: 'removed',
    createdAt: '2026-11-25T10:03:00+08:00',
    updatedAt: '2026-11-25T10:03:00+08:00',
  },
  {
    id: 'demo-sticker-4',
    packId: 'demo-pack-2',
    packName: '貓咪日常',
    mediaFileId: 'demo-file-photo',
    name: '伸懶腰',
    isAnimated: false,
    position: 0,
    status: 'active',
    createdAt: '2026-11-25T09:00:00+08:00',
    updatedAt: '2026-11-25T09:00:00+08:00',
  },
  {
    id: 'demo-sticker-5',
    packId: 'demo-pack-2',
    packName: '貓咪日常',
    mediaFileId: 'demo-file-photo',
    name: '打哈欠',
    isAnimated: false,
    position: 1,
    status: 'active',
    createdAt: '2026-11-25T09:01:00+08:00',
    updatedAt: '2026-11-25T09:01:00+08:00',
  },
  {
    id: 'demo-sticker-6',
    packId: 'demo-pack-3',
    packName: '心情點點',
    mediaFileId: 'demo-file-photo',
    name: '開心',
    isAnimated: false,
    position: 0,
    status: 'active',
    createdAt: '2026-11-25T08:00:00+08:00',
    updatedAt: '2026-11-25T08:00:00+08:00',
  },
  {
    id: 'demo-sticker-7',
    packId: 'demo-pack-3',
    packName: '心情點點',
    mediaFileId: 'demo-file-photo',
    name: '加油',
    isAnimated: false,
    position: 1,
    status: 'active',
    createdAt: '2026-11-25T08:01:00+08:00',
    updatedAt: '2026-11-25T08:01:00+08:00',
  },
  {
    id: 'demo-sticker-8',
    packId: 'demo-pack-4',
    packName: '煙火節慶',
    mediaFileId: 'demo-file-photo',
    name: '煙火',
    isAnimated: false,
    position: 0,
    status: 'active',
    createdAt: '2026-11-25T07:00:00+08:00',
    updatedAt: '2026-11-25T07:00:00+08:00',
  },
  {
    id: 'demo-sticker-9',
    packId: 'demo-pack-5',
    packName: '我的草稿包',
    mediaFileId: 'demo-file-photo',
    name: '草稿貼圖',
    isAnimated: false,
    position: 0,
    status: 'active',
    createdAt: '2026-11-25T11:00:00+08:00',
    updatedAt: '2026-11-25T11:00:00+08:00',
  },
  {
    id: 'demo-sticker-10',
    packId: 'demo-pack-6',
    packName: '下架舊包',
    mediaFileId: 'demo-file-photo',
    name: '舊貼圖',
    isAnimated: false,
    position: 0,
    status: 'active',
    createdAt: '2026-11-24T18:00:00+08:00',
    updatedAt: '2026-11-24T18:00:00+08:00',
  },
  {
    id: 'demo-sticker-11',
    packId: 'demo-pack-7',
    packName: '已封存包',
    mediaFileId: 'demo-file-photo',
    name: '封存貼圖',
    isAnimated: false,
    position: 0,
    status: 'active',
    createdAt: '2026-11-24T12:00:00+08:00',
    updatedAt: '2026-11-24T12:00:00+08:00',
  },
];

/** Demonstration grant rows: pack ids the demo user holds a grant for (paid-pack visibility). */
export const DEMO_STICKER_GRANTED_PACKS: string[] = ['demo-pack-3'];

/** Demonstration favorite rows: pack ids the demo user favorited. */
export const DEMO_STICKER_FAVORITES: string[] = ['demo-pack-1'];

/** Demonstration recent-use rows, newest first (Phase 08). */
export const DEMO_STICKER_RECENT: { stickerId: string; lastUsedAt: string }[] = [
  { stickerId: 'demo-sticker-6', lastUsedAt: '2026-11-25T12:00:00+08:00' },
  { stickerId: 'demo-sticker-1', lastUsedAt: '2026-11-25T09:30:00+08:00' },
];
