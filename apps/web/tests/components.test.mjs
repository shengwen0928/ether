import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  renderAddMembersDialog,
  renderAttachFilesDialog,
  renderAttachmentBlock,
  renderAttachmentBlockNoDownload,
  renderAuth,
  renderAvatar,
  renderBadges,
  renderButton,
  renderChats,
  renderChatsApi,
  renderContacts,
  renderContactsApi,
  renderConversationItem,
  renderCreateConversationDialog,
  renderDialogClosed,
  renderDialogOpen,
  renderEmptyState,
  renderForwardDialog,
  renderGroups,
  renderGroupsApi,
  renderIconButton,
  renderInputError,
  renderLeaveConversationDialog,
  renderMedia,
  renderMediaApi,
  renderMembersBlock,
  renderMembersBlockUnsupported,
  renderMessageBubble,
  renderMessageOwnActions,
  renderMessageOwnActionsUnsupported,
  renderMessageSearchPanel,
  renderMessageSearchPanelUnsupported,
  renderPinsBlock,
  renderProfile,
  renderReactionBar,
  renderShowcase,
  renderTabs,
  renderTimeline,
  renderTimelineApi,
  renderPostCard,
  renderPostMediaPicker,
  renderCalls,
  renderCallsApi,
  renderCallCard,
  renderCallDetailRinging,
  renderCallDetailEnded,
  renderInitiateCallDialog,
  renderStickers,
  renderStickersApi,
  renderStickerPackCard,
  renderStickerPackDetail,
  renderStickerTile,
  renderStickerPackFormCreate,
  renderAddStickerDialog,
  renderStickerMediaPicker,
  renderTyping,
  deriveReceiptStatus,
  filterCreateCandidates,
  filterForwardTargets,
  formatAttachmentSize,
  selectDeliveredReceiptTargets,
  selectReadReceiptTarget,
  validateEditContent,
  validatePostComment,
  validatePostContent,
  visibilityLabel,
  callActionState,
  callEventLabel,
  callStatusLabel,
  callTypeLabel,
  elapsedSeconds,
  endReasonLabel,
  formatCallDuration,
  isCallTerminal,
  participantResultLabel,
  canManageStickerPack,
  stickerPackCategoryLabel,
  stickerPackStatusLabel,
  validateStickerName,
  validateStickerPackDescription,
  validateStickerPackName,
} from './.bundle/entry-components.cjs';

test('Button renders label, variant class and disabled state while loading', () => {
  const html = renderButton();
  assert.match(html, /刪除/);
  assert.match(html, /et-btn--danger/);
  assert.match(html, /disabled/);
});

test('IconButton exposes an accessible label', () => {
  assert.match(renderIconButton(), /aria-label="搜尋"/);
});

test('Input with an error renders aria-invalid and the message', () => {
  const html = renderInputError();
  assert.match(html, /Email 格式不正確。/);
  assert.match(html, /aria-invalid="true"/);
  assert.match(html, /role="alert"/);
});

test('Avatar renders initials and presence state', () => {
  const html = renderAvatar();
  assert.match(html, /林彤/);
  assert.match(html, /et-avatar__presence--online/);
});

test('Badges render tone and 99+ capping', () => {
  const html = renderBadges();
  assert.match(html, /最愛/);
  assert.match(html, /99\+/);
});

test('Dialog renders only when open', () => {
  assert.match(renderDialogOpen(), /role="dialog"/);
  assert.match(renderDialogOpen(), /展示對話框/);
  assert.equal(renderDialogClosed(), '');
});

test('Tabs expose tablist semantics and the selected tab', () => {
  const html = renderTabs();
  assert.match(html, /role="tablist"/);
  assert.match(html, /aria-selected="true"/);
});

test('MessageBubble marks own messages and shows read status', () => {
  const html = renderMessageBubble();
  assert.match(html, /et-message--own/);
  assert.match(html, /測試訊息/);
  assert.match(html, /已讀/);
});

test('MessageReactionBar renders summary chips, the picker and the pin toggle', () => {
  const html = renderReactionBar();
  assert.match(html, /et-reaction-bar/);
  assert.match(html, /👍/);
  assert.match(html, /aria-pressed="true"/);
  assert.match(html, /回應/);
  assert.match(html, /aria-label="反應表情"/);
  assert.match(html, /🎉/);
  assert.match(html, /置頂/);
  assert.match(html, /轉傳/);
});

test('PinnedMessagesBlock renders pinned entries with sender and unpin action', () => {
  const html = renderPinsBlock();
  assert.match(html, /置頂訊息/);
  assert.match(html, /陳玫/);
  assert.match(html, /週六下午三點見。/);
  assert.match(html, /取消置頂/);
});

test('MessageAttachmentBlock renders file metadata, the forward label and a download button', () => {
  const html = renderAttachmentBlock();
  assert.match(html, /陽明山雲海\.jpg/);
  assert.match(html, /image\/jpeg/);
  assert.match(html, /180 KB/);
  assert.match(html, /轉傳的訊息/);
  assert.match(html, /下載 陽明山雲海\.jpg/);
  assert.match(html, /新增附件/);
  assert.doesNotMatch(html, /目前的資料模式不提供下載/);
});

test('MessageAttachmentBlock without download support shows the no-download note', () => {
  const html = renderAttachmentBlockNoDownload();
  assert.match(html, /目前的資料模式不提供下載/);
  assert.doesNotMatch(html, /下載 陽明山雲海\.jpg/);
});

test('ForwardDialog renders targets, the search field and the no-attachments note', () => {
  const html = renderForwardDialog();
  assert.match(html, /轉傳訊息/);
  assert.match(html, /陳玫/);
  assert.match(html, /搜尋對話/);
  assert.match(html, /不帶附件與引用/);
  assert.match(html, /轉傳/);
});

test('MessageOwnActions renders edit, recall and delete buttons with an edited label', () => {
  const html = renderMessageOwnActions();
  assert.match(html, /編輯此訊息/);
  assert.match(html, /收回此訊息/);
  assert.match(html, /僅刪除自己檢視的此訊息/);
  assert.match(html, /已編輯/);
  assert.doesNotMatch(html, /<textarea/);
  assert.doesNotMatch(html, /收回後所有成員/);
});

test('MessageOwnActions renders nothing actionable when unsupported and not edited', () => {
  const html = renderMessageOwnActionsUnsupported();
  assert.doesNotMatch(html, /編輯此訊息/);
  assert.doesNotMatch(html, /收回此訊息/);
  assert.doesNotMatch(html, /僅刪除自己檢視的此訊息/);
  assert.doesNotMatch(html, /已編輯/);
});

test('edit content validation rejects empty and overlong content', () => {
  assert.equal(validateEditContent(''), '訊息內容不可為空。');
  assert.equal(validateEditContent('   '), '訊息內容不可為空。');
  assert.equal(validateEditContent('a'.repeat(4001)), '訊息內容過長。');
  assert.equal(validateEditContent('你好'), null);
});

test('ConversationMembersBlock renders members, roles and departed members', () => {
  const html = renderMembersBlock().replaceAll('<!-- -->', '');
  assert.match(html, /成員（2）/);
  assert.match(html, /管理員/);
  assert.match(html, /陳玫/);
  assert.match(html, /已離開/);
  assert.match(html, /新增成員/);
  assert.match(html, /離開對話/);
});

test('ConversationMembersBlock renders nothing when unsupported', () => {
  assert.equal(renderMembersBlockUnsupported(), '');
});

test('AddMembersDialog renders candidates with checkboxes', () => {
  const html = renderAddMembersDialog();
  assert.match(html, /新增成員/);
  assert.match(html, /陳玫/);
  assert.match(html, /王梓豪/);
  assert.match(html, /type="checkbox"/);
  assert.match(html, /僅群組對話可新增成員/);
});

test('LeaveConversationDialog explains rejoining through an admin', () => {
  const html = renderLeaveConversationDialog();
  assert.match(html, /離開對話/);
  assert.match(html, /群組管理員重新新增/);
});

test('AttachFilesDialog renders only the file list with scan-gated rows', () => {
  const html = renderAttachFilesDialog();
  assert.match(html, /選擇附件/);
  assert.match(html, /陽明山雲海\.jpg/);
  assert.match(html, /登山口集合\.mp4/);
  assert.match(html, /不可關聯/);
  assert.match(html, /不會重新上傳/);
});

test('filterForwardTargets matches titles case-insensitively and formatAttachmentSize scales', () => {
  const targets = [
    { id: 'c-1', title: '陳玫' },
    { id: 'c-2', title: '王梓豪' },
  ];
  assert.deepEqual(filterForwardTargets(targets, ''), targets);
  assert.deepEqual(filterForwardTargets(targets, '玫'), [targets[0]]);
  assert.equal(formatAttachmentSize(512), '512 B');
  assert.equal(formatAttachmentSize(184320), '180 KB');
  assert.equal(formatAttachmentSize(48234496), '46 MB');
});

test('ConversationItem renders name and preview for a non-typing conversation', () => {
  const html = renderConversationItem();
  assert.match(html, /陳玫/);
  assert.match(html, /週六下午三點/);
});

test('TypingIndicator exposes a status role', () => {
  assert.match(renderTyping(), /正在輸入/);
});

test('EmptyState renders its title', () => {
  assert.match(renderEmptyState(), /選擇一個對話/);
});

test('Chats view renders the conversation list, thread and composer', () => {
  const html = renderChats();
  assert.match(html, /王梓豪/);
  assert.match(html, /陳玫/);
  assert.match(html, /訊息輸入區/);
  assert.match(html, /傳送訊息/);
  assert.match(html, /新增對話/);
  assert.match(html, /搜尋訊息/);
});

test('Message search panel renders keyword field, filters and source note in idle state', () => {
  const html = renderMessageSearchPanel();
  assert.match(html, /搜尋訊息/);
  assert.match(html, /搜尋關鍵字/);
  assert.match(html, /全部對話/);
  assert.match(html, /陳建豪/);
  assert.match(html, /起始日期/);
  assert.match(html, /結束日期/);
  assert.match(html, /搜尋結果來自展示資料/);
  assert.match(html, /返回對話列表/);
});

test('Message search panel without support explains and offers to go back', () => {
  const html = renderMessageSearchPanelUnsupported();
  assert.match(html, /不提供訊息搜尋/);
  assert.match(html, /返回對話列表/);
});

test('Chats view in API mode renders loading state and backend label', () => {
  const html = renderChatsApi();
  assert.match(html, /載入對話中/);
  assert.match(html, /對話與訊息來自後端 API/);
  assert.match(html, /搜尋對話/);
  assert.match(html, /新增對話/);
});

test('create conversation dialog lists friends with search and a disabled create button', () => {
  const html = renderCreateConversationDialog();
  assert.match(html, /建立對話/);
  assert.match(html, /選擇一位好友/);
  assert.match(html, /搜尋好友/);
  assert.match(html, /王梓豪/);
  assert.match(html, /zihao/);
  assert.match(html, /陳玫/);
  assert.match(html, /role="listbox"/);
  assert.match(html, /aria-selected="false"/);
  assert.match(html, /disabled/);
});

test('create conversation candidates filter by name or public id', () => {
  const candidates = [
    { id: 'u-1', displayName: '王梓豪', publicId: 'zihao' },
    { id: 'u-2', displayName: '陳玫', publicId: 'mei' },
  ];
  assert.deepEqual(filterCreateCandidates(candidates, ''), candidates);
  assert.deepEqual(filterCreateCandidates(candidates, '  '), candidates);
  assert.deepEqual(
    filterCreateCandidates(candidates, '梓豪').map((item) => item.id),
    ['u-1'],
  );
  assert.deepEqual(
    filterCreateCandidates(candidates, 'MEI').map((item) => item.id),
    ['u-2'],
  );
  assert.deepEqual(filterCreateCandidates(candidates, '不存在'), []);
});

function receiptMessage(overrides = {}) {
  return {
    id: 'm-1',
    conversationId: 'c-1',
    senderId: 'u-2',
    kind: 'text',
    content: '你好',
    status: 'sent',
    clientMessageId: '',
    replyToMessageId: '',
    sequenceNumber: 3,
    editCount: 0,
    editedAt: null,
    recalledAt: null,
    deletedForSelf: false,
    createdAt: '2026-11-25T00:00:00Z',
    updatedAt: '2026-11-25T00:00:00Z',
    ...overrides,
  };
}

test('read receipt target is the newest message from another sender', () => {
  const messages = [
    receiptMessage({ id: 'm-1', sequenceNumber: 1 }),
    receiptMessage({ id: 'm-2', sequenceNumber: 2, senderId: 'u-1' }),
    receiptMessage({ id: 'm-3', sequenceNumber: 3 }),
  ];
  assert.equal(selectReadReceiptTarget(messages, 'u-1'), 'm-3');
});

test('read receipt target skips own, recalled and deleted-for-self messages', () => {
  const messages = [
    receiptMessage({ id: 'm-1', sequenceNumber: 1 }),
    receiptMessage({ id: 'm-2', sequenceNumber: 4, senderId: 'u-1' }),
    receiptMessage({ id: 'm-3', sequenceNumber: 5, status: 'recalled' }),
    receiptMessage({ id: 'm-4', sequenceNumber: 6, deletedForSelf: true }),
  ];
  assert.equal(selectReadReceiptTarget(messages, 'u-1'), 'm-1');
  assert.equal(selectReadReceiptTarget([], 'u-1'), null);
  assert.equal(selectReadReceiptTarget(messages, null), null);
});

function receiptItem(overrides = {}) {
  return {
    messageId: 'm-1',
    userId: 'u-2',
    receiptType: 'delivered',
    deviceId: '',
    createdAt: '2026-11-25T00:00:00Z',
    ...overrides,
  };
}

test('delivered receipt targets are all messages from other senders', () => {
  const messages = [
    receiptMessage({ id: 'm-1', sequenceNumber: 1 }),
    receiptMessage({ id: 'm-2', sequenceNumber: 2, senderId: 'u-1' }),
    receiptMessage({ id: 'm-3', sequenceNumber: 3, status: 'recalled' }),
    receiptMessage({ id: 'm-4', sequenceNumber: 4, deletedForSelf: true }),
    receiptMessage({ id: 'm-5', sequenceNumber: 5 }),
  ];
  assert.deepEqual(selectDeliveredReceiptTargets(messages, 'u-1'), ['m-1', 'm-5']);
  assert.deepEqual(selectDeliveredReceiptTargets(messages, null), []);
  assert.deepEqual(selectDeliveredReceiptTargets([], 'u-1'), []);
});

test('receipt status requires every other member for read and delivered', () => {
  const delivered = [receiptItem({ receiptType: 'delivered' })];
  assert.equal(deriveReceiptStatus(delivered, 'u-1', ['u-2']), 'delivered');
  const read = [receiptItem({ receiptType: 'read' })];
  assert.equal(deriveReceiptStatus(read, 'u-1', ['u-2']), 'read');
  assert.equal(deriveReceiptStatus([], 'u-1', ['u-2']), null);
  assert.equal(deriveReceiptStatus(delivered, 'u-1', []), null);
  assert.equal(deriveReceiptStatus(delivered, null, ['u-2']), null);
  // 群組：僅部分成員已讀時不顯示已讀；全部已讀才顯示。
  const partial = [
    receiptItem({ userId: 'u-2', receiptType: 'read' }),
    receiptItem({ userId: 'u-3', receiptType: 'delivered' }),
  ];
  assert.equal(deriveReceiptStatus(partial, 'u-1', ['u-2', 'u-3']), 'delivered');
  const allRead = [
    receiptItem({ userId: 'u-2', receiptType: 'read' }),
    receiptItem({ userId: 'u-3', receiptType: 'read' }),
  ];
  assert.equal(deriveReceiptStatus(allRead, 'u-1', ['u-2', 'u-3']), 'read');
  // 自己的回條列不計入。
  const ownOnly = [receiptItem({ userId: 'u-1', receiptType: 'read' })];
  assert.equal(deriveReceiptStatus(ownOnly, 'u-1', ['u-2']), null);
});

test('Auth view renders login form and demo-mode notice', () => {
  const html = renderAuth();
  assert.match(html, /歡迎回來/);
  assert.match(html, /展示模式/);
  assert.match(html, /忘記密碼/);
});

test('Contacts view renders contacts and search', () => {
  const html = renderContacts();
  assert.match(html, /聯絡人/);
  assert.match(html, /搜尋聯絡人/);
  assert.match(html, /周亭安/);
});

test('Groups view renders the title, invitations section and loading state', () => {
  const html = renderGroups();
  assert.match(html, /群組/);
  assert.match(html, /收到的群組邀請/);
  assert.match(html, /我的群組/);
  assert.match(html, /載入群組中/);
  assert.match(html, /展示資料/);
});

test('Groups view in API mode labels the backend data source', () => {
  const html = renderGroupsApi();
  assert.match(html, /群組/);
  assert.match(html, /載入群組中/);
  assert.match(html, /來自後端 API/);
});

test('Media view renders the title and loading state', () => {
  const html = renderMedia();
  assert.match(html, /媒體/);
  assert.match(html, /載入媒體檔案中/);
  assert.match(html, /展示資料/);
});

test('Media view in API mode labels the backend data source', () => {
  const html = renderMediaApi();
  assert.match(html, /媒體/);
  assert.match(html, /載入媒體檔案中/);
  assert.match(html, /來自後端 API/);
});

test('Contacts view in API mode renders search and loading state', () => {
  const html = renderContactsApi();
  assert.match(html, /聯絡人/);
  assert.match(html, /以公開識別碼搜尋/);
  assert.match(html, /載入好友與邀請中/);
});

test('Profile view renders profile form, theme options and sign-out', () => {
  const html = renderProfile();
  assert.match(html, /個人檔案與設定/);
  assert.match(html, /跟隨系統/);
  assert.match(html, /登出展示工作階段/);
  assert.match(html, /裝置與工作階段/);
  assert.match(html, /載入裝置與工作階段中/);
});

test('Showcase renders the design-system sections', () => {
  const html = renderShowcase();
  assert.match(html, /設計系統展示/);
  assert.match(html, /品牌色彩/);
  assert.match(html, /字體階層/);
  assert.match(html, /對話元件/);
});

test('Timeline view renders the title, tabs and loading state', () => {
  const html = renderTimeline();
  assert.match(html, /社群動態/);
  assert.match(html, /好友動態/);
  assert.match(html, /我的貼文/);
  assert.match(html, /載入動態中/);
  assert.match(html, /展示資料/);
  assert.match(html, /沒有即時推播/);
});

test('Timeline view in API mode labels the backend data source', () => {
  const html = renderTimelineApi();
  assert.match(html, /社群動態/);
  assert.match(html, /載入動態中/);
  assert.match(html, /來自後端 API/);
});

test('Post card renders author, content, visibility, counts and actions', () => {
  const html = renderPostCard();
  assert.match(html, /陳玫/);
  assert.match(html, /週六下午三點/);
  assert.match(html, /公開/);
  assert.match(html, /按讚/);
  assert.match(html, /留言/);
  assert.match(html, /demo-file-photo/);
  assert.match(html, /尚未開放下載/);
});

test('Post media picker lists files with scan status and no download affordance', () => {
  const html = renderPostMediaPicker();
  assert.match(html, /選擇貼文媒體/);
  assert.match(html, /陽明山雲海\.jpg/);
  assert.match(html, /掃描通過/);
  assert.doesNotMatch(html, /下載/);
});

test('Post content and comment validation follow the backend limits', () => {
  assert.equal(validatePostContent('', 0), '貼文內容與媒體不得皆為空。');
  assert.equal(validatePostContent('   ', 0), '貼文內容與媒體不得皆為空。');
  assert.equal(validatePostContent('', 1), '');
  assert.equal(validatePostContent('x'.repeat(2001), 0), '貼文內容不得超過 2000 字。');
  assert.equal(validatePostComment(''), '留言內容不得為空。');
  assert.equal(validatePostComment('   '), '留言內容不得為空。');
  assert.equal(validatePostComment('x'.repeat(1001)), '留言內容不得超過 1000 字。');
  assert.equal(validatePostComment('你好'), '');
  assert.equal(visibilityLabel('public'), '公開');
  assert.equal(visibilityLabel('friends'), '僅好友');
  assert.equal(visibilityLabel('only_me'), '僅自己');
  assert.equal(visibilityLabel('unknown'), '僅好友');
});

test('Calls view renders the title, initiate action, loading state and no-push note', () => {
  const html = renderCalls();
  assert.match(html, /通話/);
  assert.match(html, /發起通話/);
  assert.match(html, /載入通話中/);
  assert.match(html, /展示資料/);
  assert.match(html, /沒有即時來電推播/);
});

test('Calls view in API mode labels the backend data source', () => {
  const html = renderCallsApi();
  assert.match(html, /通話/);
  assert.match(html, /載入通話中/);
  assert.match(html, /來自後端 API/);
});

test('Call card renders counterpart, type, status, duration and detail action', () => {
  const html = renderCallCard();
  assert.match(html, /王梓豪/);
  assert.match(html, /語音通話/);
  assert.match(html, /已結束/);
  assert.match(html, /12 分 30 秒/);
  assert.match(html, /位參與者/);
  assert.match(html, /查看詳情/);
  assert.doesNotMatch(html, /媒體已接通/);
});

test('Ringing call detail offers answer, decline and busy, and states media is not connected', () => {
  const html = renderCallDetailRinging();
  assert.match(html, /陳玫/);
  assert.match(html, /響鈴中/);
  assert.match(html, /接聽/);
  assert.match(html, /拒接/);
  assert.match(html, /忙線/);
  assert.match(html, /媒體尚未接通，僅會話狀態/);
  assert.match(html, /發起通話/);
  assert.match(html, /待回應/);
  assert.doesNotMatch(html, /結束通話/);
});

test('Ended call detail shows duration, end reason, participants and events, with no actions', () => {
  const html = renderCallDetailEnded();
  assert.match(html, /王梓豪/);
  assert.match(html, /已結束/);
  assert.match(html, /通話時長 12 分 30 秒/);
  assert.match(html, /掛斷/);
  assert.match(html, /已加入/);
  assert.match(html, /接聽/);
  assert.doesNotMatch(html, /媒體尚未接通/);
  assert.doesNotMatch(html, /拒接/);
});

test('Initiate call dialog lists direct and group candidates with the call type choice', () => {
  const html = renderInitiateCallDialog();
  assert.match(html, /發起通話/);
  assert.match(html, /一對一對話/);
  assert.match(html, /陳玫/);
  assert.match(html, /語音通話/);
  assert.match(html, /視訊通話/);
  assert.match(html, /媒體尚未接通/);
});

test('Call labels, duration formatting and action states follow the backend rules', () => {
  assert.equal(callStatusLabel('ringing'), '響鈴中');
  assert.equal(callStatusLabel('active'), '通話中');
  assert.equal(callStatusLabel('declined'), '全員拒接');
  assert.equal(callStatusLabel('missed'), '未接');
  assert.equal(callStatusLabel('busy'), '全員忙線');
  assert.equal(callStatusLabel('ended'), '已結束');
  assert.equal(callTypeLabel('voice'), '語音通話');
  assert.equal(callTypeLabel('video'), '視訊通話');
  assert.equal(participantResultLabel('pending'), '待回應');
  assert.equal(participantResultLabel('joined'), '已加入');
  assert.equal(participantResultLabel('missed'), '未接');
  assert.equal(callEventLabel('initiated'), '發起通話');
  assert.equal(callEventLabel('answered'), '接聽');
  assert.equal(callEventLabel('ended'), '結束通話');
  assert.equal(endReasonLabel('hangup'), '掛斷');
  assert.equal(endReasonLabel(''), '');
  assert.equal(isCallTerminal('ringing'), false);
  assert.equal(isCallTerminal('active'), false);
  assert.equal(isCallTerminal('ended'), true);
  assert.equal(isCallTerminal('missed'), true);
  assert.equal(formatCallDuration(null), '—');
  assert.equal(formatCallDuration(0), '0 秒');
  assert.equal(formatCallDuration(45), '45 秒');
  assert.equal(formatCallDuration(60), '1 分');
  assert.equal(formatCallDuration(750), '12 分 30 秒');
  assert.equal(formatCallDuration(3600), '1 小時');
  assert.equal(formatCallDuration(3720), '1 小時 2 分');
  assert.equal(elapsedSeconds(null, Date.parse('2026-11-25T12:00:00+08:00')), 0);
  assert.equal(
    elapsedSeconds('2026-11-25T11:00:00+08:00', Date.parse('2026-11-25T11:01:30+08:00')),
    90,
  );
  const base = {
    id: 'call-1',
    conversationId: 'v-1',
    groupId: '',
    initiatorId: 'c-mei',
    callType: 'voice',
    status: 'ringing',
    startedAt: null,
    endedAt: null,
    durationSeconds: null,
    endedBy: '',
    endReason: '',
    participantCount: 2,
    createdAt: '2026-11-25T12:50:00+08:00',
    updatedAt: '2026-11-25T12:50:00+08:00',
  };
  const pending = {
    userId: 'me',
    joinedAt: null,
    leftAt: null,
    result: 'pending',
    createdAt: '2026-11-25T12:50:00+08:00',
  };
  const joined = { ...pending, joinedAt: '2026-11-25T12:51:00+08:00', result: 'joined' };
  assert.equal(callActionState(base, pending), 'answer');
  assert.equal(callActionState({ ...base, status: 'active' }, joined), 'end');
  assert.equal(callActionState({ ...base, status: 'ended' }, joined), 'none');
  assert.equal(callActionState(base, joined), 'end');
  assert.equal(callActionState(base, undefined), 'none');
});

test('Stickers view renders the title, tabs, search, create action, loading state and no-purchase note', () => {
  const html = renderStickers();
  assert.match(html, /貼圖/);
  assert.match(html, /瀏覽貼圖包/);
  assert.match(html, /我的收藏/);
  assert.match(html, /最近使用/);
  assert.match(html, /搜尋貼圖/);
  assert.match(html, /建立貼圖包/);
  assert.match(html, /載入貼圖中/);
  assert.match(html, /展示資料/);
  assert.match(html, /沒有購買流程/);
  assert.match(html, /沒有即時推播/);
  assert.doesNotMatch(html, /購買貼圖|立即購買|結帳/);
});

test('Stickers view in API mode labels the backend data source', () => {
  const html = renderStickersApi();
  assert.match(html, /貼圖/);
  assert.match(html, /載入貼圖中/);
  assert.match(html, /來自後端 API/);
});

test('Sticker pack card renders creator, category, free-or-granted, status and favorite action', () => {
  const html = renderStickerPackCard();
  assert.match(html, /山間小日子（付費）/);
  assert.match(html, /陳玫/);
  assert.match(html, /一般/);
  assert.match(html, /付費 · 已授權/);
  assert.match(html, /已上架/);
  assert.match(html, /4 張貼圖/);
  assert.match(html, /取消收藏/);
  assert.doesNotMatch(html, /購買/);
});

test('Sticker pack detail shows stickers as metadata tiles with author-only management', () => {
  const html = renderStickerPackDetail();
  assert.match(html, /山間小日子/);
  assert.match(html, /咖啡時間/);
  assert.match(html, /山徑漫步/);
  assert.match(html, /動態/);
  assert.match(html, /編輯貼圖包/);
  assert.match(html, /新增貼圖/);
  assert.match(html, /刪除貼圖包/);
  assert.match(html, /記錄使用/);
  assert.match(html, /不提供檔案下載/);
  assert.doesNotMatch(html, /下載按鈕|開啟檔案/);
});

test('Sticker tile renders only the name and metadata, never a download affordance', () => {
  const html = renderStickerTile();
  assert.match(html, /咖啡時間/);
  assert.match(html, /山間小日子/);
  assert.match(html, /記錄使用/);
  assert.doesNotMatch(html, /下載|<img/);
});

test('Sticker pack create form states new packs start as drafts and offers a free toggle', () => {
  const html = renderStickerPackFormCreate();
  assert.match(html, /建立貼圖包/);
  assert.match(html, /一律為草稿/);
  assert.match(html, /貼圖包名稱/);
  assert.match(html, /免費貼圖包/);
  assert.match(html, /選擇封面/);
  assert.match(html, /沒有購買流程/);
  assert.doesNotMatch(html, /立即購買|結帳|價格|定價/);
});

test('Add sticker dialog requires a media file and states bytes are not re-uploaded', () => {
  const html = renderAddStickerDialog();
  assert.match(html, /新增貼圖到/);
  assert.match(html, /貼圖名稱/);
  assert.match(html, /動態貼圖/);
  assert.match(html, /選擇媒體檔案/);
  assert.match(html, /不會重新上傳檔案位元組/);
});

test('Sticker media picker lists only existing files and marks the scan gate', () => {
  const html = renderStickerMediaPicker();
  assert.match(html, /選擇媒體檔案/);
  assert.match(html, /morning-coffee\.png/);
  assert.match(html, /掃描通過/);
  assert.match(html, /不會重新上傳檔案位元組/);
});

test('Sticker labels and validation follow the backend vocabulary and limits', () => {
  assert.equal(stickerPackCategoryLabel(''), '未分類');
  assert.equal(stickerPackCategoryLabel('general'), '一般');
  assert.equal(stickerPackCategoryLabel('animals'), '動物');
  assert.equal(stickerPackCategoryLabel('food'), '食物');
  assert.equal(stickerPackCategoryLabel('emotions'), '情緒');
  assert.equal(stickerPackCategoryLabel('festivals'), '節慶');
  assert.equal(stickerPackCategoryLabel('characters'), '角色');
  assert.equal(stickerPackCategoryLabel('other'), '其他');
  assert.equal(stickerPackCategoryLabel('bogus'), '未分類');
  assert.equal(stickerPackStatusLabel('draft'), '草稿');
  assert.equal(stickerPackStatusLabel('published'), '已上架');
  assert.equal(stickerPackStatusLabel('unlisted'), '未上架');
  assert.equal(stickerPackStatusLabel('archived'), '已封存');
  assert.equal(validateStickerPackName(''), '貼圖包名稱不得為空。');
  assert.equal(validateStickerPackName('  '), '貼圖包名稱不得為空。');
  assert.equal(validateStickerPackName('x'.repeat(65)), '貼圖包名稱過長。');
  assert.equal(validateStickerPackName('山間小日子'), '');
  assert.equal(validateStickerPackDescription('x'.repeat(501)), '貼圖包描述過長。');
  assert.equal(validateStickerPackDescription('簡介'), '');
  assert.equal(validateStickerName('x'.repeat(65)), '貼圖名稱過長。');
  assert.equal(validateStickerName(''), '');
  assert.equal(
    canManageStickerPack(
      {
        id: 'p-1',
        creatorId: 'me',
        name: '包',
        description: '',
        category: 'general',
        coverMediaId: '',
        status: 'draft',
        isFree: true,
        stickerCount: 0,
        favoritedByMe: false,
        createdAt: '2026-11-25T09:00:00Z',
        updatedAt: '2026-11-25T09:00:00Z',
      },
      'me',
    ),
    true,
  );
  assert.equal(
    canManageStickerPack(
      {
        id: 'p-1',
        creatorId: 'other',
        name: '包',
        description: '',
        category: 'general',
        coverMediaId: '',
        status: 'draft',
        isFree: true,
        stickerCount: 0,
        favoritedByMe: false,
        createdAt: '2026-11-25T09:00:00Z',
        updatedAt: '2026-11-25T09:00:00Z',
      },
      'me',
    ),
    false,
  );
});
