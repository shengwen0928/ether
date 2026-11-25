import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ReactElement } from 'react';
import {
  filterConversations,
  formatMessageTime,
  groupMessagesByDay,
  presenceLabel,
  searchDateToSince,
  searchDateToUntil,
  sortConversations,
  totalUnread,
} from '../data/logic';
import type { ChatDataProvider } from '../data/provider';
import { useChatData } from '../data/provider';
import type {
  ChatMessage,
  ConversationFilter,
  ConversationItem as ConversationItemModel,
  ConversationMemberItem,
  ForwardRecordItem,
  FriendItem,
  MediaFileItem,
  MessageAttachmentItem,
  MessageItem,
  MessageSearchInput,
  MessageSearchPage,
  MessageSearchResultItem,
  PinItem,
  ReactionSummaryItem,
  ReceiptItem,
} from '../data/types';
import { REACTION_EMOJIS } from '../data/types';
import { Avatar } from '../components/Avatar';
import { Badge, CountBadge } from '../components/Badge';
import { Button } from '../components/Button';
import { ConversationItem } from '../components/ConversationItem';
import { Dialog } from '../components/Dialog';
import { Dropdown } from '../components/Dropdown';
import { EmptyState } from '../components/EmptyState';
import { BackIcon, DotsIcon, PhoneIcon, PlusIcon, VideoIcon } from '../components/icons';
import { IconButton } from '../components/IconButton';
import { Input } from '../components/Input';
import { MessageBubble } from '../components/MessageBubble';
import { MessageComposer } from '../components/MessageComposer';
import { SearchField } from '../components/SearchField';
import { Select } from '../components/Select';
import { Tabs } from '../components/Tabs';
import { useToast } from '../components/Toast';
import { TypingIndicator } from '../components/TypingIndicator';
import { useIsMobile } from '../hooks/useMediaQuery';
import { downloadBlob, openDownloadUrl } from './downloadFile';

export interface ChatsViewProps {
  provider: ChatDataProvider;
  selectedId: string | null;
  onSelect: (conversationId: string | null) => void;
  /** 資料模式（ContactsView 相同分流）：預設展示模式。 */
  dataMode?: 'demo' | 'api';
  /** API 模式下回報後端未讀總數（供 App 側邊列徽章）。 */
  onUnreadChange?: (count: number) => void;
}

function DemoChatsView({ provider, selectedId, onSelect }: ChatsViewProps): ReactElement {
  const { notify } = useToast();
  const snapshot = useChatData(provider);
  const isMobile = useIsMobile();
  const [query, setQuery] = useState('');
  const [filter, setFilter] = useState<ConversationFilter>('all');
  const [detailOpen, setDetailOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [highlightMessageId, setHighlightMessageId] = useState<string | null>(null);
  /** 醒目訊息所屬的對話：手動切換對話時清除，跳轉時保留。 */
  const highlightForRef = useRef<string | null>(null);
  const [createCandidates, setCreateCandidates] = useState<FriendItem[]>([]);
  const [createCandidatesState, setCreateCandidatesState] =
    useState<CreateCandidatesState>('loading');
  const messagesRef = useRef<HTMLDivElement | null>(null);
  const now = useMemo(() => new Date(), []);

  // 建對話人選：開啟對話框時向 provider 取好友清單（預設不含隱藏
  // 好友，與 ContactsView 好友清單一致）；展示 provider 取本地資料。
  const loadCreateCandidates = useCallback(() => {
    const listFriends = provider.listFriendsAsync;
    if (listFriends === undefined) {
      setCreateCandidatesState('unsupported');
      return;
    }
    setCreateCandidatesState('loading');
    let cancelled = false;
    void listFriends()
      .then((fetched) => {
        if (!cancelled) {
          setCreateCandidates(fetched);
          setCreateCandidatesState('ready');
        }
      })
      .catch(() => {
        if (!cancelled) {
          setCreateCandidatesState('error');
        }
      });
    return () => {
      cancelled = true;
    };
  }, [provider]);

  useEffect(() => {
    if (!createOpen) {
      return undefined;
    }
    return loadCreateCandidates();
  }, [createOpen, loadCreateCandidates]);

  // 對話成員管理（§2x）：成員清單經 hook 載入；新增人選沿用
  // 建對話好友清單（開啟新增對話框時載入）。
  const [addMembersOpen, setAddMembersOpen] = useState(false);
  const [leaveOpen, setLeaveOpen] = useState(false);
  const conversationMembers = useConversationMembers(provider, selectedId);

  useEffect(() => {
    if (!addMembersOpen) {
      return undefined;
    }
    return loadCreateCandidates();
  }, [addMembersOpen, loadCreateCandidates]);

  const handleAddMembers = async (memberIds: string[]): Promise<void> => {
    const addMembers = provider.addMembersAsync;
    if (addMembers === undefined || selectedId === null) {
      throw new Error('目前的資料模式不提供新增成員。');
    }
    await addMembers(selectedId, memberIds);
    conversationMembers.reload();
    notify({ message: `已新增 ${memberIds.length} 位成員。`, tone: 'success' });
  };

  const handleLeaveConversation = async (): Promise<void> => {
    const leave = provider.leaveConversationAsync;
    if (leave === undefined || selectedId === null) {
      throw new Error('目前的資料模式不提供離開對話。');
    }
    await leave(selectedId);
    onSelect(null);
    notify({ message: '已離開對話。', tone: 'success' });
  };

  const handleCreateConversation = async (userId: string): Promise<void> => {
    const create = provider.createConversationAsync;
    if (create === undefined) {
      throw new Error('目前的資料模式不提供建立對話。');
    }
    // 展示 provider 寫入快照後經訂閱重繪；既有 DM 回傳既有對話。
    const created = await create({ type: 'direct', otherUserId: userId });
    onSelect(created.id);
  };

  const contactsById = useMemo(
    () => new Map(snapshot.contacts.map((contact) => [contact.id, contact])),
    [snapshot.contacts],
  );

  const myId = provider.getCurrentUserId?.() ?? 'me';

  // 成員顯示名稱取自聯絡人清單；新增人選排除既有生效成員與自己。
  const memberNames = useMemo(
    () => new Map(snapshot.contacts.map((contact) => [contact.id, contact.name])),
    [snapshot.contacts],
  );

  const addMemberCandidates = useMemo<CreateConversationCandidate[]>(
    () =>
      createCandidates
        .filter((friend) => friend.id !== myId)
        .filter(
          (friend) =>
            !conversationMembers.members.some(
              (member) => member.leftAt === null && member.userId === friend.id,
            ),
        )
        .map((friend) => ({
          id: friend.id,
          displayName: friend.displayName,
          publicId: friend.publicId,
        })),
    [createCandidates, conversationMembers.members, myId],
  );

  const visible = useMemo(
    () =>
      sortConversations(filterConversations(snapshot.conversations, contactsById, query, filter)),
    [snapshot.conversations, contactsById, query, filter],
  );

  const selected = snapshot.conversations.find((conversation) => conversation.id === selectedId);
  const selectedContact = selected !== undefined ? contactsById.get(selected.contactId) : undefined;
  const selectedMessages = useMemo(
    () =>
      snapshot.messages
        .filter((message) => message.conversationId === selectedId)
        .sort((a, b) => a.sentAt.localeCompare(b.sentAt)),
    [snapshot.messages, selectedId],
  );
  const grouped = useMemo(() => groupMessagesByDay(selectedMessages, now), [selectedMessages, now]);
  const unread = totalUnread(snapshot.conversations);

  // 反應與置頂（§2q）：彙總與清單以 REST 另查，無即時推播。
  const demoMessageIds = useMemo(
    () => selectedMessages.map((message) => message.id),
    [selectedMessages],
  );
  const reactions = useReactionsAndPins(provider, selectedId, demoMessageIds);
  const pinnedEntries = useMemo<PinnedMessageEntry[]>(
    () =>
      reactions.pins.map((pin) => {
        const item = selectedMessages.find((message) => message.id === pin.messageId);
        if (item === undefined) {
          return { pin, senderName: '', body: '置頂訊息尚未載入。' };
        }
        const senderName =
          item.senderId === 'me' ? '你' : (contactsById.get(item.senderId)?.name ?? '');
        return { pin, senderName, body: item.body };
      }),
    [reactions.pins, selectedMessages, contactsById],
  );

  // 附件與轉傳（§2t）：附件清單與轉傳紀錄以 REST 另查，無即時推播。
  const attachments = useMessageAttachments(provider, selectedId, demoMessageIds);
  const downloadAttachment = useMediaDownload(provider);
  const [forwardFor, setForwardFor] = useState<string | null>(null);
  const [attachPickerOpen, setAttachPickerOpen] = useState(false);
  const [attachPickerMode, setAttachPickerMode] = useState<'compose' | 'add'>('compose');
  const [attachTargetId, setAttachTargetId] = useState<string | null>(null);
  const [pendingAttachments, setPendingAttachments] = useState<MediaFileItem[]>([]);
  const [mediaFiles, setMediaFiles] = useState<MediaFileItem[]>([]);
  const [mediaFilesState, setMediaFilesState] = useState<AttachFilesState>('loading');

  // 附加檔案人選：開啟挑選對話框時向 provider 取媒體檔案清單；
  // 展示 provider 取本地資料。僅掃描通過的檔案可選（對話框內
  // 把關，見 AttachFilesDialog）。
  const loadMediaFiles = useCallback(() => {
    const listFiles = provider.listMediaFilesAsync;
    if (listFiles === undefined) {
      setMediaFilesState('unsupported');
      return;
    }
    setMediaFilesState('loading');
    let cancelled = false;
    void listFiles()
      .then((fetched) => {
        if (!cancelled) {
          setMediaFiles(fetched);
          setMediaFilesState('ready');
        }
      })
      .catch(() => {
        if (!cancelled) {
          setMediaFilesState('error');
        }
      });
    return () => {
      cancelled = true;
    };
  }, [provider]);

  useEffect(() => {
    if (!attachPickerOpen) {
      return undefined;
    }
    return loadMediaFiles();
  }, [attachPickerOpen, loadMediaFiles]);

  const openAttachPicker = (mode: 'compose' | 'add', messageId: string | null): void => {
    if (provider.listMediaFilesAsync === undefined) {
      demoNotice('附加檔案尚未接線：媒體功能在 Phase 06 實作。');
      return;
    }
    setAttachPickerMode(mode);
    setAttachTargetId(messageId);
    setAttachPickerOpen(true);
  };

  const handleAttachConfirm = async (mediaFileIds: string[]): Promise<void> => {
    if (attachPickerMode === 'compose') {
      const picked = mediaFiles.filter((file) => mediaFileIds.includes(file.id));
      setPendingAttachments(picked);
      return;
    }
    if (attachTargetId === null || provider.addAttachmentsAsync === undefined) {
      throw new Error('目前的資料模式不提供新增附件。');
    }
    await attachments.addAttachments(attachTargetId, mediaFileIds);
    notify({ message: '已新增附件。', tone: 'success' });
  };

  const handleForward = async (targetConversationId: string): Promise<void> => {
    if (forwardFor === null || provider.forwardMessageAsync === undefined) {
      throw new Error('目前的資料模式不提供轉傳。');
    }
    // 展示 provider 寫入快照後經訂閱重繪；轉傳訊息落在目標對話。
    await provider.forwardMessageAsync(forwardFor, targetConversationId);
    notify({ message: '已轉傳訊息。', tone: 'success' });
  };

  // 自己訊息的操作（編輯／收回／僅自己刪除，§2u）：展示 provider
  // 寫入 overlay 後經訂閱重繪，狀態更新不由本層另行維護。
  const ownActionsSupported =
    provider.editMessageAsync !== undefined &&
    provider.recallMessageAsync !== undefined &&
    provider.deleteMessageAsync !== undefined;

  const handleEditMessage = async (messageId: string, content: string): Promise<void> => {
    if (provider.editMessageAsync === undefined) {
      throw new Error('目前的資料模式不提供編輯訊息。');
    }
    await provider.editMessageAsync(messageId, content);
  };

  const handleRecallMessage = async (messageId: string): Promise<void> => {
    if (provider.recallMessageAsync === undefined) {
      throw new Error('目前的資料模式不提供收回訊息。');
    }
    await provider.recallMessageAsync(messageId);
  };

  const handleDeleteMessage = async (messageId: string): Promise<void> => {
    if (provider.deleteMessageAsync === undefined) {
      throw new Error('目前的資料模式不提供刪除訊息。');
    }
    await provider.deleteMessageAsync(messageId);
  };

  const forwardTargets = useMemo<ForwardTarget[]>(
    () =>
      snapshot.conversations.map((conversation) => ({
        id: conversation.id,
        title: contactsById.get(conversation.contactId)?.name ?? '對話',
      })),
    [snapshot.conversations, contactsById],
  );

  const handleSend = (body: string): void => {
    if (selected === undefined) {
      return;
    }
    // Prefer the async send path when the provider offers it (M5):
    // failures are reported instead of silently appending locally.
    // The sync path stays as the fallback for providers without
    // async messaging. Pending attachments ride along when present.
    const attachedIds = pendingAttachments.map((file) => file.id);
    if (provider.sendMessageAsync !== undefined) {
      const pending = attachedIds;
      void provider
        .sendMessageAsync(
          selected.id,
          body,
          pending.length > 0 ? { attachmentFileIds: pending } : {},
        )
        .then(() => {
          setPendingAttachments([]);
        })
        .catch((error: unknown) => {
          notify({
            title: '傳送失敗',
            message: error instanceof Error ? error.message : '訊息傳送失敗。',
            tone: 'error',
          });
        });
      return;
    }
    provider.sendMessage(selected.id, body);
    setPendingAttachments([]);
  };

  useEffect(() => {
    if (!isMobile && selectedId === null && visible.length > 0) {
      const first = visible[0];
      if (first !== undefined) {
        onSelect(first.id);
      }
    }
  }, [isMobile, selectedId, visible, onSelect]);

  useEffect(() => {
    const pane = messagesRef.current;
    if (pane !== null) {
      pane.scrollTop = pane.scrollHeight;
    }
  }, [selectedId, selectedMessages.length, selected?.isTyping]);

  const choose = (conversationId: string): void => {
    onSelect(conversationId);
    provider.markConversationRead(conversationId);
  };

  // 搜尋跳轉的醒目提示（§2v）：手動切換對話時清除，跳轉時保留；
  // 展示資料全在本機快照，訊息已在畫面上，直接捲入視野。
  useEffect(() => {
    if (selectedId === null || highlightForRef.current !== selectedId) {
      if (highlightMessageId !== null) {
        setHighlightMessageId(null);
      }
      return;
    }
    if (highlightMessageId === null) {
      return;
    }
    const target = messagesRef.current?.querySelector(`[data-message-id="${highlightMessageId}"]`);
    target?.scrollIntoView({ block: 'center' });
  }, [selectedId, highlightMessageId]);

  // 訊息搜尋（§2v）：面板共用 MessageSearchPanel；展示模式以本機
  // 比對搜尋展示資料，跳轉時開啟對話並醒目提示該訊息。
  const handleSearchMessages = async (input: MessageSearchInput): Promise<MessageSearchPage> => {
    if (provider.searchMessagesAsync === undefined) {
      throw new Error('目前的資料模式不提供訊息搜尋。');
    }
    return provider.searchMessagesAsync(input);
  };

  const jumpToSearchResult = (result: MessageSearchResultItem): void => {
    highlightForRef.current = result.conversationId;
    setHighlightMessageId(result.id);
    setSearchOpen(false);
    choose(result.conversationId);
  };

  const searchConversationOptions = useMemo<MessageSearchOption[]>(
    () =>
      snapshot.conversations.map((conversation) => ({
        value: conversation.id,
        label: contactsById.get(conversation.contactId)?.name ?? '對話',
      })),
    [snapshot.conversations, contactsById],
  );

  const searchSenderOptions = useMemo<MessageSearchOption[]>(
    () => [
      { value: 'me', label: '我' },
      ...snapshot.contacts.map((contact) => ({ value: contact.id, label: contact.name })),
    ],
    [snapshot.contacts],
  );

  const demoNotice = (message: string): void => {
    notify({ title: '展示模式', message, tone: 'warning' });
  };

  return (
    <div className={selectedId !== null ? 'et-chats et-chats--show-thread' : 'et-chats'}>
      <section className="et-chats__list" aria-label="對話列表">
        <div className="et-chats__list-tools">
          <SearchField
            value={query}
            onChange={setQuery}
            placeholder="搜尋對話或聯絡人"
            label="搜尋對話"
          />
          <Tabs
            label="對話篩選"
            value={filter}
            onChange={(id) => setFilter(id as ConversationFilter)}
            tabs={[
              { id: 'all', label: '全部' },
              {
                id: 'unread',
                label: '未讀',
                badge:
                  unread > 0 ? <CountBadge count={unread} label={`${unread} 則未讀`} /> : undefined,
              },
              { id: 'favorites', label: '最愛' },
            ]}
          />
        </div>
        {searchOpen ? (
          <MessageSearchPanel
            conversations={searchConversationOptions}
            senders={searchSenderOptions}
            supported={provider.searchMessagesAsync !== undefined}
            sourceNote="搜尋結果來自展示資料（本機比對）。"
            onSearch={handleSearchMessages}
            onJump={jumpToSearchResult}
            onClose={() => {
              setSearchOpen(false);
            }}
          />
        ) : (
          <div className="et-chats__items et-scroll" role="list" aria-label="對話">
            {visible.length === 0 ? (
              <p className="et-chats__empty">
                {query !== '' || filter !== 'all'
                  ? '沒有符合條件的對話。試試其他關鍵字或篩選。'
                  : '目前沒有對話。'}
              </p>
            ) : (
              visible.map((conversation) => {
                const contact = contactsById.get(conversation.contactId);
                if (contact === undefined) {
                  return null;
                }
                return (
                  <ConversationItem
                    key={conversation.id}
                    conversation={conversation}
                    contact={contact}
                    active={conversation.id === selectedId}
                    now={now}
                    onSelect={choose}
                  />
                );
              })
            )}
          </div>
        )}
        <div className="et-chats__list-tools">
          <button
            type="button"
            className="et-btn et-btn--secondary et-btn--md et-btn--full"
            onClick={() => {
              setSearchOpen((open) => !open);
            }}
            aria-expanded={searchOpen}
          >
            搜尋訊息
          </button>
          <button
            type="button"
            className="et-btn et-btn--secondary et-btn--md et-btn--full"
            onClick={() => {
              setCreateOpen(true);
            }}
          >
            <PlusIcon size={16} />
            新增對話
          </button>
        </div>
      </section>

      <CreateConversationDialog
        open={createOpen}
        candidates={createCandidates}
        candidatesState={createCandidatesState}
        onRetryCandidates={loadCreateCandidates}
        onClose={() => {
          setCreateOpen(false);
        }}
        onCreate={handleCreateConversation}
      />

      <AddMembersDialog
        open={addMembersOpen}
        candidates={addMemberCandidates}
        candidatesState={createCandidatesState}
        onRetryCandidates={loadCreateCandidates}
        onClose={() => {
          setAddMembersOpen(false);
        }}
        onAdd={handleAddMembers}
      />

      {selected !== undefined && selectedContact !== undefined ? (
        <LeaveConversationDialog
          open={leaveOpen}
          conversationTitle={selectedContact.name}
          conversationType="direct"
          onClose={() => {
            setLeaveOpen(false);
          }}
          onConfirm={handleLeaveConversation}
        />
      ) : null}

      <section className="et-chats__thread" aria-label="對話內容">
        {selected === undefined || selectedContact === undefined ? (
          <div className="et-thread__empty">
            <EmptyState
              title="選擇一個對話"
              description="從左側列表挑一位聯絡人，開始閱讀與回覆訊息。"
            />
          </div>
        ) : (
          <>
            <header className="et-thread__header">
              <span className="et-thread__back">
                <IconButton
                  label="回到對話列表"
                  onClick={() => {
                    onSelect(null);
                  }}
                >
                  <BackIcon size={20} />
                </IconButton>
              </span>
              <button
                type="button"
                className="et-thread__identity"
                onClick={() => {
                  setDetailOpen(true);
                }}
              >
                <Avatar name={selectedContact.name} size="md" presence={selectedContact.presence} />
                <span className="et-thread__identity-text">
                  <span className="et-thread__name">{selectedContact.name}</span>
                  <span className="et-thread__presence">
                    {selected.isTyping
                      ? '正在輸入…'
                      : `${presenceLabel(selectedContact.presence)} · ${selectedContact.lastSeen}`}
                  </span>
                </span>
              </button>
              <div className="et-thread__actions">
                <IconButton
                  label={`語音通話給 ${selectedContact.name}（展示）`}
                  onClick={() => demoNotice('語音通話尚未接線：通話功能在 Phase 07 實作。')}
                >
                  <PhoneIcon size={20} />
                </IconButton>
                <IconButton
                  label={`視訊通話給 ${selectedContact.name}（展示）`}
                  onClick={() => demoNotice('視訊通話尚未接線：通話功能在 Phase 07 實作。')}
                >
                  <VideoIcon size={20} />
                </IconButton>
                <Dropdown
                  label={<DotsIcon size={20} />}
                  ariaLabel="對話選項"
                  items={[
                    { id: 'detail', label: '檢視聯絡資訊' },
                    { id: 'mute', label: selected.isMuted ? '取消靜音' : '靜音對話' },
                    { id: 'read', label: '標示全部已讀' },
                  ]}
                  onSelect={(id) => {
                    if (id === 'detail') {
                      setDetailOpen(true);
                    } else if (id === 'read') {
                      provider.markConversationRead(selected.id);
                      notify({ message: '已標示全部已讀（本機展示）。', tone: 'success' });
                    } else {
                      demoNotice('靜音設定尚未接線：通知功能在 Phase 04 之後實作。');
                    }
                  }}
                />
              </div>
            </header>
            {selected !== undefined && provider.listMembersAsync !== undefined ? (
              <ConversationMembersBlock
                members={conversationMembers.members}
                membersState={conversationMembers.state}
                onRetryMembers={conversationMembers.reload}
                memberNames={memberNames}
                myId={myId}
                canAddMembers={provider.addMembersAsync !== undefined}
                canLeave={provider.leaveConversationAsync !== undefined}
                onAddMembers={() => {
                  setAddMembersOpen(true);
                }}
                onLeave={() => {
                  setLeaveOpen(true);
                }}
              />
            ) : null}
            <PinnedMessagesBlock
              entries={pinnedEntries}
              state={reactions.pinsState}
              onRetry={reactions.reloadPins}
              onUnpin={(messageId) => {
                void reactions.togglePin(messageId);
              }}
            />
            <div className="et-thread__messages et-scroll" ref={messagesRef} aria-live="off">
              {grouped.map((group) => (
                <div key={group.label}>
                  <div className="et-day-divider">
                    <span className="et-day-divider__label">{group.label}</span>
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
                    {group.messages.map((message) => {
                      const senderName =
                        message.senderId === 'me'
                          ? '你'
                          : (contactsById.get(message.senderId)?.name ?? '');
                      return (
                        <div
                          key={message.id}
                          data-message-id={message.id}
                          className={
                            highlightMessageId === message.id
                              ? 'et-message--jump-target'
                              : undefined
                          }
                        >
                          <MessageBubble
                            message={message}
                            isOwn={message.senderId === 'me'}
                            senderName={senderName}
                            showSenderName={message.senderId !== 'me'}
                            now={now}
                            onAttachmentDownload={() => {
                              demoNotice('檔案下載尚未接線：媒體功能在 Phase 06 實作。');
                            }}
                            actions={
                              <>
                                {attachments.supported ? (
                                  <MessageAttachmentBlock
                                    attachments={attachments.attachmentsById[message.id] ?? []}
                                    isForwarded={attachments.forwardsById[message.id] !== undefined}
                                    canAddAttachments={message.senderId === 'me'}
                                    onAddAttachments={() => {
                                      openAttachPicker('add', message.id);
                                    }}
                                    downloadSupported={downloadAttachment !== null}
                                    onDownloadAttachment={
                                      downloadAttachment === null ? undefined : downloadAttachment
                                    }
                                  />
                                ) : null}
                                {reactions.supported ? (
                                  <MessageReactionBar
                                    summary={reactions.summaries[message.id]}
                                    isPinned={reactions.pins.some(
                                      (pin) => pin.messageId === message.id,
                                    )}
                                    pickerOpen={reactions.pickerFor === message.id}
                                    onTogglePicker={() => {
                                      reactions.setPickerFor(
                                        reactions.pickerFor === message.id ? null : message.id,
                                      );
                                    }}
                                    onPickReaction={(emoji) => {
                                      void reactions.pickReaction(message.id, emoji);
                                    }}
                                    onRemoveReaction={() => {
                                      void reactions.removeReaction(message.id);
                                    }}
                                    onTogglePin={() => {
                                      void reactions.togglePin(message.id);
                                    }}
                                    onForward={
                                      attachments.supported
                                        ? () => {
                                            setForwardFor(message.id);
                                          }
                                        : undefined
                                    }
                                  />
                                ) : undefined}
                                {message.senderId === 'me' &&
                                !message.recalled &&
                                !message.deletedForSelf ? (
                                  <MessageOwnActions
                                    messageId={message.id}
                                    content={message.body}
                                    edited={message.editedAt !== undefined}
                                    supported={ownActionsSupported}
                                    onEdit={(content) => handleEditMessage(message.id, content)}
                                    onRecall={() => handleRecallMessage(message.id)}
                                    onDelete={() => handleDeleteMessage(message.id)}
                                  />
                                ) : null}
                              </>
                            }
                          />
                        </div>
                      );
                    })}
                  </div>
                </div>
              ))}
              {selected.isTyping ? (
                <div className="et-thread__typing-row">
                  <Avatar name={selectedContact.name} size="sm" />
                  <span>
                    {selectedContact.name} 正在輸入
                    <TypingIndicator name={selectedContact.name} />
                  </span>
                </div>
              ) : null}
            </div>
            {pendingAttachments.length > 0 ? (
              <div className="et-pending-attachments" aria-label="已選擇的附件">
                <span className="et-pending-attachments__label">
                  已選擇 {pendingAttachments.length} 個附件（關聯既有檔案，不重新上傳）：
                </span>
                {pendingAttachments.map((file) => (
                  <span key={file.id} className="et-pending-attachments__chip">
                    {file.originalFilename}
                    <button
                      type="button"
                      aria-label={`移除附件 ${file.originalFilename}`}
                      onClick={() => {
                        setPendingAttachments((prev) => prev.filter((item) => item.id !== file.id));
                      }}
                    >
                      ×
                    </button>
                  </span>
                ))}
              </div>
            ) : null}
            <MessageComposer
              onSend={handleSend}
              onAttach={() => {
                openAttachPicker('compose', null);
              }}
              onEmoji={() => {
                demoNotice('表情符號面板尚未接線：貼圖功能在 Phase 08 實作。');
              }}
            />
          </>
        )}
      </section>

      <ForwardDialog
        open={forwardFor !== null}
        targets={forwardTargets}
        onClose={() => {
          setForwardFor(null);
        }}
        onForward={handleForward}
      />
      <AttachFilesDialog
        open={attachPickerOpen}
        files={mediaFiles}
        filesState={mediaFilesState}
        onRetryFiles={loadMediaFiles}
        onClose={() => {
          setAttachPickerOpen(false);
        }}
        onConfirm={handleAttachConfirm}
        confirmLabel={attachPickerMode === 'compose' ? '選擇附件' : '新增附件'}
      />

      {selectedContact !== undefined ? (
        <Dialog
          open={detailOpen}
          onClose={() => {
            setDetailOpen(false);
          }}
          title={selectedContact.name}
          description="聯絡人資訊（展示資料）"
          footer={
            <Button
              variant="secondary"
              onClick={() => {
                setDetailOpen(false);
              }}
            >
              關閉
            </Button>
          }
        >
          <div className="et-contact-detail">
            <Avatar name={selectedContact.name} size="xl" presence={selectedContact.presence} />
            <div className="et-contact-detail__row">
              <span className="et-contact-detail__key">帳號</span>
              <span className="et-contact-detail__value">{selectedContact.handle}</span>
            </div>
            <div className="et-contact-detail__row">
              <span className="et-contact-detail__key">狀態</span>
              <span className="et-contact-detail__value">
                {selectedContact.statusMessage !== ''
                  ? selectedContact.statusMessage
                  : presenceLabel(selectedContact.presence)}
              </span>
            </div>
            <div className="et-contact-detail__row">
              <span className="et-contact-detail__key">上次上線</span>
              <span className="et-contact-detail__value">{selectedContact.lastSeen}</span>
            </div>
            <div className="et-contact-detail__row">
              <span className="et-contact-detail__key">分類</span>
              <span className="et-contact-detail__value">{selectedContact.group}</span>
            </div>
          </div>
        </Dialog>
      ) : null}
    </div>
  );
}

/**
 * 後端對話清單與歷史（清單遷移里程碑，API 模式）。
 *
 * 對話清單來自 GET /api/v1/conversations（未讀數取後端資料，
 * 依後端 updated_at 降序）；歷史來自
 * GET /api/v1/conversations/{id}/messages，預設取最新一頁、
 * 以 before_seq 游標載入更早訊息（wire 順序為序號降序，
 * 顯示前以序號排序為升序）。發送走 sendMessageAsync，成功後
 * 以伺服器回應的訊息為準更新歷史與清單。已讀回條（§2n）：
 * 歷史載入完成後，對序號最大的他人訊息送一則 read 回條，後端
 * 已讀水位即推進至該序號；成功後重載清單取權威未讀數，不在本
 * 機扣減。建立對話（本里程碑）：經好友清單挑選人選，
 * POST /api/v1/conversations（type=direct）；既有 DM 後端回 200＋
 * 既有對話，直接選取並重載清單；新建回 201，重載後選取。反應
 * 與置頂（§2q）：彙總與清單以 REST 另查，寫入後以伺服器回應
 * 更新本機狀態；無即時推播。無即時
 * 推播：新訊息須以「重新載入」手動更新（realtime stub 不動）。
 * 狀態機與 M2c／M3 的四態模式一致：載入／錯誤＋重試／不支援／
 * 就緒；重新載入失敗時保留既有畫面。
 */

type ApiListState = 'loading' | 'ready' | 'error' | 'unsupported';
type ApiHistoryState = 'idle' | 'loading' | 'ready' | 'error' | 'unsupported';
type ApiListFilter = 'all' | 'unread';

interface HistoryEntry {
  messages: MessageItem[];
  nextCursor: string;
  hasMore: boolean;
}

const HISTORY_PAGE_LIMIT = 30;

/** 依序號升序合併兩頁訊息（去重，wire 順序不得假設）。 */
function mergeMessages(existing: MessageItem[], incoming: MessageItem[]): MessageItem[] {
  const byId = new Map<string, MessageItem>();
  for (const message of [...existing, ...incoming]) {
    byId.set(message.id, message);
  }
  return [...byId.values()].sort((a, b) => a.sequenceNumber - b.sequenceNumber);
}

/**
 * 挑出已讀回條的目標訊息：已載入歷史中序號最大的一則他人訊息。
 * 後端 read 回條會把已讀水位推進至該訊息序號（不得倒退），故一則
 * 回條即涵蓋其前全部訊息；自己的訊息後端拒寫回條、收回訊息與僅
 * 自己刪除的訊息無須回條，一律排除。身分未知（myId 為 null）時
 * 無法排除自己的訊息，回 null、不送回條。
 */
export function selectReadReceiptTarget(
  messages: MessageItem[],
  myId: string | null,
): string | null {
  if (myId === null) {
    return null;
  }
  let target: MessageItem | null = null;
  for (const message of messages) {
    if (message.senderId === myId || message.status === 'recalled' || message.deletedForSelf) {
      continue;
    }
    if (target === null || message.sequenceNumber > target.sequenceNumber) {
      target = message;
    }
  }
  return target?.id ?? null;
}

/**
 * 挑出送達回條的目標訊息：已載入歷史中全部他人訊息（逐則送
 * delivered，後端無批次端點；delivered 不推進已讀水位，故不與
 * read 回條合併）。自己的訊息後端拒寫回條、收回訊息與僅自己
 * 刪除的訊息無須回條，一律排除。身分未知（myId 為 null）時回
 * 空陣列、不送回條。
 */
export function selectDeliveredReceiptTargets(
  messages: MessageItem[],
  myId: string | null,
): string[] {
  if (myId === null) {
    return [];
  }
  const targets: string[] = [];
  for (const message of messages) {
    if (message.senderId === myId || message.status === 'recalled' || message.deletedForSelf) {
      continue;
    }
    targets.push(message.id);
  }
  return targets;
}

/**
 * 由回條列表推導自己訊息的送達狀態：已讀須全部其他在線成員皆
 * 有 read 回條；已送達須全部其他在線成員皆有 delivered 或 read
 * 回條（已讀蘊含已送達）。成員未知（otherMemberIds 為空）或未
 * 達標時回 null（維持「已傳送」，未知不顯示）。回條中的自己列
 * （若有）不計入。
 */
export function deriveReceiptStatus(
  receipts: ReceiptItem[],
  myId: string | null,
  otherMemberIds: readonly string[],
): 'delivered' | 'read' | null {
  if (myId === null || otherMemberIds.length === 0) {
    return null;
  }
  const readBy = new Set<string>();
  const deliveredBy = new Set<string>();
  for (const receipt of receipts) {
    if (receipt.userId === myId) {
      continue;
    }
    if (receipt.receiptType === 'read') {
      readBy.add(receipt.userId);
      deliveredBy.add(receipt.userId);
    } else if (receipt.receiptType === 'delivered') {
      deliveredBy.add(receipt.userId);
    }
  }
  if (otherMemberIds.every((userId) => readBy.has(userId))) {
    return 'read';
  }
  if (otherMemberIds.every((userId) => deliveredBy.has(userId))) {
    return 'delivered';
  }
  return null;
}

function sortConversationsByUpdated(items: ConversationItemModel[]): ConversationItemModel[] {
  return [...items].sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt));
}

/** 建立一對一對話的人選：只取好友（非好友不得建 DM，見後端 CreateDirect）。 */
export interface CreateConversationCandidate {
  id: string;
  displayName: string;
  publicId: string;
}

/**
 * 依顯示名稱／公開識別碼過濾建對話人選（大小寫不敏感、前後空白
 * 忽略；空查詢回傳全部人選）。
 */
export function filterCreateCandidates(
  candidates: CreateConversationCandidate[],
  query: string,
): CreateConversationCandidate[] {
  const needle = query.trim().toLocaleLowerCase();
  if (needle === '') {
    return candidates;
  }
  return candidates.filter(
    (candidate) =>
      candidate.displayName.toLocaleLowerCase().includes(needle) ||
      candidate.publicId.toLocaleLowerCase().includes(needle),
  );
}

type CreateCandidatesState = 'loading' | 'ready' | 'error' | 'unsupported';

interface CreateConversationDialogProps {
  open: boolean;
  candidates: CreateConversationCandidate[];
  candidatesState: CreateCandidatesState;
  onRetryCandidates: () => void;
  onClose: () => void;
  /** 建立成功時 resolve（父層負責重載清單與選取對話）；失敗時 reject。 */
  onCreate: (userId: string) => Promise<void>;
}

/**
 * 建立一對一對話對話框（展示／API 雙模式共用）：人選一律來自好友
 * 清單，父層傳入；搜尋過濾、單選、建立失敗時對話框內顯示錯誤文字
 * 並 toast（可重試），成功時由父層關閉。開啟時重置查詢與選取。
 */
export function CreateConversationDialog({
  open,
  candidates,
  candidatesState,
  onRetryCandidates,
  onClose,
  onCreate,
}: CreateConversationDialogProps): ReactElement {
  const { notify } = useToast();
  const [query, setQuery] = useState('');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    if (open) {
      setQuery('');
      setSelectedId(null);
      setError('');
      setCreating(false);
    }
  }, [open]);

  const filtered = useMemo(() => filterCreateCandidates(candidates, query), [candidates, query]);

  const submit = (): void => {
    if (selectedId === null) {
      setError('請先選擇一位好友。');
      return;
    }
    if (creating) {
      return;
    }
    const userId = selectedId;
    setCreating(true);
    setError('');
    void onCreate(userId)
      .then(() => {
        onClose();
      })
      .catch((createError: unknown) => {
        const message =
          createError instanceof Error ? createError.message : '建立對話失敗，請再試一次。';
        setError(message);
        notify({ title: '建立對話失敗', message, tone: 'error' });
      })
      .finally(() => {
        setCreating(false);
      });
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="建立對話"
      description="選擇一位好友，開始一對一對話。已有對話時會直接開啟既有對話。"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={creating}>
            取消
          </Button>
          <Button disabled={creating || selectedId === null} loading={creating} onClick={submit}>
            {creating ? '建立中…' : '建立對話'}
          </Button>
        </>
      }
    >
      {candidatesState === 'loading' ? (
        <p className="et-chats__empty">載入好友中…</p>
      ) : candidatesState === 'error' ? (
        <div className="et-chats__empty">
          <p>載入好友失敗，請檢查連線後再試一次。</p>
          <Button type="button" variant="secondary" size="sm" onClick={onRetryCandidates}>
            重新載入
          </Button>
        </div>
      ) : candidatesState === 'unsupported' ? (
        <p className="et-chats__empty">目前的資料模式不提供好友清單。</p>
      ) : (
        <>
          <SearchField
            value={query}
            onChange={(value) => {
              setQuery(value);
            }}
            placeholder="搜尋好友名稱或帳號"
            label="搜尋好友"
          />
          {candidates.length === 0 ? (
            <p className="et-chats__empty">還沒有好友。先到「聯絡人」新增好友，再回來建立對話。</p>
          ) : filtered.length === 0 ? (
            <p className="et-chats__empty">沒有符合條件的好友。試試其他關鍵字。</p>
          ) : (
            <div role="listbox" aria-label="好友清單" className="et-friend-list">
              {filtered.map((candidate) => (
                <button
                  key={candidate.id}
                  type="button"
                  role="option"
                  aria-selected={selectedId === candidate.id}
                  className="et-friend-row et-friend-row--pick"
                  onClick={() => {
                    setSelectedId(candidate.id);
                    setError('');
                  }}
                >
                  <Avatar name={candidate.displayName} size="md" />
                  <span className="et-friend-row__main">
                    <span className="et-friend-row__name">{candidate.displayName}</span>
                    <span className="et-friend-row__meta">@{candidate.publicId}</span>
                  </span>
                  {selectedId === candidate.id ? <Badge tone="accent">已選擇</Badge> : null}
                </button>
              ))}
            </div>
          )}
          {error !== '' ? (
            <p className="et-field__error" role="alert">
              {error}
            </p>
          ) : null}
        </>
      )}
    </Dialog>
  );
}

type PinsState = 'idle' | 'loading' | 'ready' | 'error' | 'unsupported';

/** 對話成員載入狀態：與既有清單／歷史狀態同一組詞彙。 */
export type MembersState = 'idle' | 'loading' | 'ready' | 'error' | 'unsupported';

/**
 * 對話成員共用 hook（§2x）：切換對話時重置，載入失敗時保留空
 * 成員清單（標題與回條推導退回通用值，不阻擋歷史顯示）；
 * reload 供新增成員後重查。provider 未提供 listMembersAsync
 * 時為 unsupported，呼叫端不渲染成員區塊。
 */
export function useConversationMembers(
  provider: ChatDataProvider,
  conversationId: string | null,
): {
  members: ConversationMemberItem[];
  state: MembersState;
  reload: () => void;
} {
  const [members, setMembers] = useState<ConversationMemberItem[]>([]);
  const [state, setState] = useState<MembersState>('idle');
  const [epoch, setEpoch] = useState(0);

  useEffect(() => {
    setMembers([]);
    if (conversationId === null) {
      setState('idle');
      return undefined;
    }
    const listMembers = provider.listMembersAsync;
    if (listMembers === undefined) {
      setState('unsupported');
      return undefined;
    }
    setState('loading');
    let cancelled = false;
    void listMembers(conversationId)
      .then((fetched) => {
        if (!cancelled) {
          setMembers(fetched);
          setState('ready');
        }
      })
      .catch(() => {
        if (!cancelled) {
          setState('error');
        }
      });
    return () => {
      cancelled = true;
    };
  }, [provider, conversationId, epoch]);

  const reload = useCallback(() => {
    setEpoch((value) => value + 1);
  }, []);

  return { members, state, reload };
}

function memberDisplayName(
  member: ConversationMemberItem,
  memberNames: ReadonlyMap<string, string>,
  myId: string | null,
): string {
  if (myId !== null && member.userId === myId) {
    return '你';
  }
  return memberNames.get(member.userId) ?? `使用者 ${member.userId.slice(0, 8)}`;
}

function memberRoleLabel(role: string): string {
  if (role === 'admin') {
    return '管理員';
  }
  if (role === 'owner') {
    return '擁有者';
  }
  return '成員';
}

interface ConversationMembersBlockProps {
  members: ConversationMemberItem[];
  membersState: MembersState;
  onRetryMembers: () => void;
  memberNames: ReadonlyMap<string, string>;
  myId: string | null;
  canAddMembers: boolean;
  canLeave: boolean;
  onAddMembers: () => void;
  onLeave: () => void;
}

/**
 * 對話成員區塊（§2x，展示／API 雙模式共用）：顯示生效成員與已
 * 離開成員、新增成員與離開對話的操作入口。成員清單順序以後端
 * 回應為準（已離開者列於末尾）。不提供移除他人成員的操作（後
 * 端無對應端點）。
 */
export function ConversationMembersBlock({
  members,
  membersState,
  onRetryMembers,
  memberNames,
  myId,
  canAddMembers,
  canLeave,
  onAddMembers,
  onLeave,
}: ConversationMembersBlockProps): ReactElement | null {
  if (membersState === 'unsupported' || membersState === 'idle') {
    return null;
  }
  const activeCount = members.filter((member) => member.leftAt === null).length;
  return (
    <section className="et-members" aria-label="對話成員">
      <div className="et-members__header">
        <h2 className="et-members__title">成員（{activeCount}）</h2>
        <div className="et-members__actions">
          {canAddMembers ? (
            <Button type="button" variant="secondary" size="sm" onClick={onAddMembers}>
              新增成員
            </Button>
          ) : null}
          {canLeave ? (
            <Button type="button" variant="danger" size="sm" onClick={onLeave}>
              離開對話
            </Button>
          ) : null}
        </div>
      </div>
      {membersState === 'loading' ? (
        <p className="et-members__empty">載入成員中…</p>
      ) : membersState === 'error' ? (
        <div className="et-members__empty">
          <p>載入成員失敗，請檢查連線後再試一次。</p>
          <Button type="button" variant="secondary" size="sm" onClick={onRetryMembers}>
            重新載入
          </Button>
        </div>
      ) : members.length === 0 ? (
        <p className="et-members__empty">目前沒有成員資料。</p>
      ) : (
        <ul className="et-members__list">
          {members.map((member) => (
            <li key={member.userId} className="et-members__row">
              <Avatar name={memberDisplayName(member, memberNames, myId)} size="sm" />
              <span className="et-members__name">
                {memberDisplayName(member, memberNames, myId)}
              </span>
              <span className="et-members__role">{memberRoleLabel(member.role)}</span>
              {member.leftAt !== null ? <span className="et-members__left">已離開</span> : null}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

interface AddMembersDialogProps {
  open: boolean;
  candidates: CreateConversationCandidate[];
  candidatesState: CreateCandidatesState;
  onRetryCandidates: () => void;
  onClose: () => void;
  /** 新增成功時 resolve（父層負責重載成員）；失敗時 reject。 */
  onAdd: (memberIds: string[]) => Promise<void>;
}

/**
 * 新增成員對話框（§2x，展示／API 雙模式共用）：人選一律來自好友
 * 清單（父層已排除既有成員與自己），多選、搜尋過濾；新增失敗
 * 時對話框內顯示錯誤文字並 toast（可重試），成功時由父層關閉。
 * 一對一對話與非管理員的新增請求由後端守衛拒絕，錯誤誠實顯示。
 * 開啟時重置查詢與選取。
 */
export function AddMembersDialog({
  open,
  candidates,
  candidatesState,
  onRetryCandidates,
  onClose,
  onAdd,
}: AddMembersDialogProps): ReactElement {
  const { notify } = useToast();
  const [query, setQuery] = useState('');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [error, setError] = useState('');
  const [adding, setAdding] = useState(false);

  useEffect(() => {
    if (open) {
      setQuery('');
      setSelectedIds([]);
      setError('');
      setAdding(false);
    }
  }, [open]);

  const filtered = useMemo(() => filterCreateCandidates(candidates, query), [candidates, query]);

  const toggle = (id: string): void => {
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((item) => item !== id) : [...prev, id],
    );
    setError('');
  };

  const submit = (): void => {
    if (selectedIds.length === 0) {
      setError('請先選擇至少一位好友。');
      return;
    }
    if (adding) {
      return;
    }
    const memberIds = selectedIds;
    setAdding(true);
    setError('');
    void onAdd(memberIds)
      .then(() => {
        onClose();
      })
      .catch((addError: unknown) => {
        const message = addError instanceof Error ? addError.message : '新增成員失敗，請再試一次。';
        setError(message);
        notify({ title: '新增成員失敗', message, tone: 'error' });
      })
      .finally(() => {
        setAdding(false);
      });
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="新增成員"
      description="選擇好友加入這個對話。僅群組對話可新增成員，且須由管理員操作。"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={adding}>
            取消
          </Button>
          <Button disabled={adding || selectedIds.length === 0} loading={adding} onClick={submit}>
            {adding ? '新增中…' : `新增 ${selectedIds.length} 位成員`}
          </Button>
        </>
      }
    >
      {candidatesState === 'loading' ? (
        <p className="et-chats__empty">載入好友中…</p>
      ) : candidatesState === 'error' ? (
        <div className="et-chats__empty">
          <p>載入好友失敗，請檢查連線後再試一次。</p>
          <Button type="button" variant="secondary" size="sm" onClick={onRetryCandidates}>
            重新載入
          </Button>
        </div>
      ) : candidatesState === 'unsupported' ? (
        <p className="et-chats__empty">目前的資料模式不提供好友清單。</p>
      ) : (
        <>
          <SearchField
            value={query}
            onChange={(value) => {
              setQuery(value);
            }}
            placeholder="搜尋好友名稱或帳號"
            label="搜尋好友"
          />
          {candidates.length === 0 ? (
            <p className="et-chats__empty">
              沒有可新增的好友。先到「聯絡人」新增好友，或既有成員已全部加入。
            </p>
          ) : filtered.length === 0 ? (
            <p className="et-chats__empty">沒有符合條件的好友。試試其他關鍵字。</p>
          ) : (
            <div role="group" aria-label="好友清單" className="et-friend-list">
              {filtered.map((candidate) => {
                const checked = selectedIds.includes(candidate.id);
                return (
                  <label key={candidate.id} className="et-friend-row et-friend-row--pick">
                    <input
                      type="checkbox"
                      checked={checked}
                      onChange={() => {
                        toggle(candidate.id);
                      }}
                      aria-label={candidate.displayName}
                    />
                    <Avatar name={candidate.displayName} size="md" />
                    <span className="et-friend-row__main">
                      <span className="et-friend-row__name">{candidate.displayName}</span>
                      <span className="et-friend-row__meta">@{candidate.publicId}</span>
                    </span>
                    {checked ? <Badge tone="accent">已選擇</Badge> : null}
                  </label>
                );
              })}
            </div>
          )}
          {error !== '' ? (
            <p className="et-field__error" role="alert">
              {error}
            </p>
          ) : null}
        </>
      )}
    </Dialog>
  );
}

interface LeaveConversationDialogProps {
  open: boolean;
  conversationTitle: string;
  conversationType: 'direct' | 'group';
  onClose: () => void;
  /** 離開成功時 resolve（父層負責清單與選取）；失敗時 reject。 */
  onConfirm: () => Promise<void>;
}

/**
 * 離開對話確認對話框（§2x，展示／API 雙模式共用）：明載離開後
 * 果——對話自清單移除、訊息紀錄仍保留在伺服器；一對一對話可
 * 經「建立對話」重新開啟（須仍為好友，見後端 CreateDirect），
 * 群組對話須由管理員重新新增。失敗時對話框內顯示錯誤並 toast。
 */
export function LeaveConversationDialog({
  open,
  conversationTitle,
  conversationType,
  onClose,
  onConfirm,
}: LeaveConversationDialogProps): ReactElement {
  const { notify } = useToast();
  const [error, setError] = useState('');
  const [leaving, setLeaving] = useState(false);

  useEffect(() => {
    if (open) {
      setError('');
      setLeaving(false);
    }
  }, [open]);

  const submit = (): void => {
    if (leaving) {
      return;
    }
    setLeaving(true);
    setError('');
    void onConfirm()
      .then(() => {
        onClose();
      })
      .catch((leaveError: unknown) => {
        const message =
          leaveError instanceof Error ? leaveError.message : '離開對話失敗，請再試一次。';
        setError(message);
        notify({ title: '離開對話失敗', message, tone: 'error' });
      })
      .finally(() => {
        setLeaving(false);
      });
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="離開對話"
      description={`確定要離開「${conversationTitle}」嗎？`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={leaving}>
            取消
          </Button>
          <Button variant="danger" disabled={leaving} loading={leaving} onClick={submit}>
            {leaving ? '處理中…' : '離開對話'}
          </Button>
        </>
      }
    >
      <p>
        {conversationType === 'group'
          ? '離開後，這個群組對話會從你的對話清單移除，訊息紀錄仍保留在伺服器上。要再次加入，須由群組管理員重新新增你。'
          : '離開後，這個對話會從你的對話清單移除，訊息紀錄仍保留在伺服器上。之後若仍為好友，可經「建立對話」重新開啟。'}
      </p>
      {error !== '' ? (
        <p className="et-field__error" role="alert">
          {error}
        </p>
      ) : null}
    </Dialog>
  );
}

export interface MessageReactionBarProps {
  summary: ReactionSummaryItem | undefined;
  isPinned: boolean;
  pickerOpen: boolean;
  onTogglePicker: () => void;
  onPickReaction: (emoji: string) => void;
  onRemoveReaction: () => void;
  onTogglePin: () => void;
  /** 轉傳此訊息（§2t）：未提供時不渲染轉傳按鈕。 */
  onForward?: (() => void) | undefined;
}

/**
 * 訊息反應列（§2q，展示／API 雙模式共用）：彙總 chip、表情挑選
 * 器、置頂切換。點按已有自己反應的 chip 即取消；點按他人反應的
 * chip 改為該表情；挑選器點按自己的現有反應亦取消。轉傳按鈕
 * （§2t）由呼叫端傳入 onForward 時渲染，點按後由父層開啟
 * 轉傳對話框。
 */
export function MessageReactionBar({
  summary,
  isPinned,
  pickerOpen,
  onTogglePicker,
  onPickReaction,
  onRemoveReaction,
  onTogglePin,
  onForward,
}: MessageReactionBarProps): ReactElement {
  const myReaction = summary?.reactions.find((item) => item.reactedByMe)?.reaction;
  return (
    <div className="et-reaction-bar">
      {summary?.reactions.map((item) => (
        <button
          key={item.reaction}
          type="button"
          className={
            item.reactedByMe ? 'et-reaction-chip et-reaction-chip--mine' : 'et-reaction-chip'
          }
          aria-pressed={item.reactedByMe}
          aria-label={`${item.reaction}，${item.count} 則反應${
            item.reactedByMe ? '（你的反應，點按取消）' : '，點按改為此反應'
          }`}
          onClick={() => {
            if (item.reactedByMe) {
              onRemoveReaction();
            } else {
              onPickReaction(item.reaction);
            }
          }}
        >
          <span aria-hidden="true">{item.reaction}</span> {item.count}
        </button>
      ))}
      <button
        type="button"
        className="et-reaction-btn"
        aria-expanded={pickerOpen}
        aria-label="選擇反應表情"
        onClick={onTogglePicker}
      >
        回應
      </button>
      {pickerOpen ? (
        <span className="et-reaction-picker" role="group" aria-label="反應表情">
          {REACTION_EMOJIS.map((emoji) => (
            <button
              key={emoji}
              type="button"
              className="et-reaction-picker__btn"
              aria-pressed={myReaction === emoji}
              aria-label={`反應 ${emoji}${myReaction === emoji ? '（點按取消）' : ''}`}
              onClick={() => {
                onPickReaction(emoji);
              }}
            >
              {emoji}
            </button>
          ))}
        </span>
      ) : null}
      <button
        type="button"
        className="et-reaction-btn"
        aria-pressed={isPinned}
        aria-label={isPinned ? '取消置頂此訊息' : '置頂此訊息'}
        onClick={onTogglePin}
      >
        {isPinned ? '取消置頂' : '置頂'}
      </button>
      {onForward !== undefined ? (
        <button
          type="button"
          className="et-reaction-btn"
          aria-label="轉傳此訊息"
          onClick={onForward}
        >
          轉傳
        </button>
      ) : null}
    </div>
  );
}

export interface PinnedMessageEntry {
  pin: PinItem;
  senderName: string;
  body: string;
}

export interface PinnedMessagesBlockProps {
  entries: PinnedMessageEntry[];
  state: PinsState;
  onRetry: () => void;
  onUnpin: (messageId: string) => void;
}

/**
 * 置頂訊息區塊（§2q，展示／API 雙模式共用）：列出對話的生效置
 * 頂，內容取自已載入的歷史；尚未載入的置頂訊息顯示佔位文字，
 * 不假裝內容已知。載入失敗不阻擋歷史顯示，只留重試。
 */
export function PinnedMessagesBlock({
  entries,
  state,
  onRetry,
  onUnpin,
}: PinnedMessagesBlockProps): ReactElement | null {
  if (state === 'idle' || state === 'unsupported') {
    return null;
  }
  if (state === 'loading') {
    return (
      <section className="et-pins" aria-label="置頂訊息">
        <p className="et-pins__empty">載入置頂訊息中…</p>
      </section>
    );
  }
  if (state === 'error') {
    return (
      <section className="et-pins" aria-label="置頂訊息">
        <p className="et-pins__empty">
          載入置頂訊息失敗。
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={() => {
              onRetry();
            }}
          >
            重新載入
          </Button>
        </p>
      </section>
    );
  }
  if (entries.length === 0) {
    return null;
  }
  return (
    <section className="et-pins" aria-label="置頂訊息">
      <h3 className="et-pins__title">置頂訊息（共 {entries.length} 則）</h3>
      <ul className="et-pins__list">
        {entries.map(({ pin, senderName, body }) => (
          <li key={pin.id} className="et-pins__item">
            <span className="et-pins__body">
              {senderName !== '' ? <span className="et-pins__sender">{senderName}</span> : null}
              {body}
            </span>
            <button
              type="button"
              className="et-reaction-btn"
              aria-label="取消置頂此訊息"
              onClick={() => {
                onUnpin(pin.messageId);
              }}
            >
              取消置頂
            </button>
          </li>
        ))}
      </ul>
    </section>
  );
}

export function formatAttachmentSize(sizeBytes: number): string {
  if (sizeBytes < 1024) {
    return `${sizeBytes} B`;
  }
  const units = ['KB', 'MB', 'GB'];
  let value = sizeBytes / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
}

export interface MessageAttachmentBlockProps {
  attachments: MessageAttachmentItem[];
  /** 此訊息是否為轉傳產物（有轉傳紀錄）：顯示「轉傳的訊息」標示。 */
  isForwarded: boolean;
  /** 是否顯示「新增附件」按鈕（僅作者可事後新增，見 §2s）。 */
  canAddAttachments: boolean;
  onAddAttachments?: () => void;
  /** 是否提供下載（provider 有下載解析方法時為真）。 */
  downloadSupported: boolean;
  /** 點按「下載」時呼叫（父層負責授權解析與檔案儲存）。 */
  onDownloadAttachment?: ((attachment: MessageAttachmentItem) => Promise<void>) | undefined;
}

/**
 * 訊息附件區塊（§2t／§2aa，展示／API 雙模式共用）：列出訊息
 * 關聯的媒體檔案詮釋資料（檔名、類型、大小）與下載按鈕（經
 * 授權解析後分發；掃描未通過者由父層回呼拒絕）；轉傳產物另
 * 顯示標示。無附件、非轉傳產物、不可新增且不提供下載時不渲染。
 */
export function MessageAttachmentBlock({
  attachments,
  isForwarded,
  canAddAttachments,
  onAddAttachments,
  downloadSupported,
  onDownloadAttachment,
}: MessageAttachmentBlockProps): ReactElement | null {
  if (attachments.length === 0 && !isForwarded && !canAddAttachments) {
    return null;
  }
  return (
    <div className="et-attachment-block">
      {isForwarded ? <p className="et-forward-label">轉傳的訊息</p> : null}
      {attachments.length > 0 ? (
        <ul className="et-attachment-block__list" aria-label="訊息附件">
          {attachments.map((attachment) => (
            <li key={attachment.id} className="et-attachment-block__item">
              <span className="et-attachment-block__name">{attachment.fileName}</span>
              <span className="et-attachment-block__meta">
                {attachment.mimeType} · {formatAttachmentSize(attachment.sizeBytes)}
              </span>
              {downloadSupported && onDownloadAttachment !== undefined ? (
                <button
                  type="button"
                  className="et-reaction-btn"
                  aria-label={`下載 ${attachment.fileName}`}
                  onClick={() => {
                    onDownloadAttachment(attachment);
                  }}
                >
                  下載
                </button>
              ) : (
                <span className="et-attachment-block__note">目前的資料模式不提供下載</span>
              )}
            </li>
          ))}
        </ul>
      ) : null}
      {canAddAttachments ? (
        <button
          type="button"
          className="et-reaction-btn"
          aria-label="為此訊息新增附件"
          onClick={onAddAttachments}
        >
          新增附件
        </button>
      ) : null}
    </div>
  );
}

export interface ForwardTarget {
  id: string;
  title: string;
}

/** 編輯內容長度上限（與後端 MaxContentRunes 一致，見 message.go）。 */
export const MAX_EDIT_CONTENT_RUNES = 4000;

/**
 * 編輯內容檢查（純函式）：與後端 validateContent 同規則——不可為
 * 空、至多 4000 runes；回傳錯誤文字或 null（可送出）。
 */
export function validateEditContent(content: string): string | null {
  const trimmed = content.trim();
  if (trimmed === '') {
    return '訊息內容不可為空。';
  }
  if ([...trimmed].length > MAX_EDIT_CONTENT_RUNES) {
    return '訊息內容過長。';
  }
  return null;
}

export interface MessageOwnActionsProps {
  messageId: string;
  /** 訊息目前的文字內容（編輯框的初始值）。 */
  content: string;
  /** 是否曾編輯：顯示「已編輯」標示。 */
  edited: boolean;
  /** 三個操作是否可用（provider 未提供任一方法時不顯示按鈕）。 */
  supported: boolean;
  /** 編輯成功時 resolve；失敗時 reject（父層負責狀態更新）。 */
  onEdit: (content: string) => Promise<void>;
  /** 收回成功時 resolve；失敗時 reject。 */
  onRecall: () => Promise<void>;
  /** 僅自己刪除成功時 resolve；失敗時 reject。 */
  onDelete: () => Promise<void>;
}

type OwnConfirmAction = 'recall' | 'delete';

/**
 * 自己訊息的操作列（展示／API 雙模式共用）：編輯、收回、僅自己
 * 刪除。僅作者自己的互動訊息顯示（收回／僅自己刪除的訊息由父
 * 層排除）；收回與刪除屬不可逆操作，須經確認對話框；編輯以行
 * 內輸入框進行，失敗時行內顯示錯誤並 toast、可重試。操作成功
 * 後的狀態更新由父層回呼負責；本元件不假裝即時推播。
 */
export function MessageOwnActions({
  messageId,
  content,
  edited,
  supported,
  onEdit,
  onRecall,
  onDelete,
}: MessageOwnActionsProps): ReactElement | null {
  const { notify } = useToast();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState('');
  const [editError, setEditError] = useState('');
  const [saving, setSaving] = useState(false);
  const [confirm, setConfirm] = useState<OwnConfirmAction | null>(null);
  const [confirmError, setConfirmError] = useState('');
  const [confirming, setConfirming] = useState(false);

  // 切換訊息時重置編輯與確認狀態，不得把上一則的草稿帶過來。
  useEffect(() => {
    setEditing(false);
    setDraft('');
    setEditError('');
    setConfirm(null);
    setConfirmError('');
  }, [messageId]);

  if (!supported && !edited) {
    return null;
  }

  const startEdit = (): void => {
    setDraft(content);
    setEditError('');
    setEditing(true);
  };

  const saveEdit = (): void => {
    const error = validateEditContent(draft);
    if (error !== null) {
      setEditError(error);
      return;
    }
    if (saving) {
      return;
    }
    const trimmed = draft.trim();
    setSaving(true);
    setEditError('');
    void onEdit(trimmed)
      .then(() => {
        setEditing(false);
        notify({ message: '已儲存編輯。', tone: 'success' });
      })
      .catch((editError: unknown) => {
        const message = editError instanceof Error ? editError.message : '編輯失敗，請再試一次。';
        setEditError(message);
        notify({ title: '編輯失敗', message, tone: 'error' });
      })
      .finally(() => {
        setSaving(false);
      });
  };

  const submitConfirm = (): void => {
    if (confirm === null || confirming) {
      return;
    }
    const action = confirm === 'recall' ? onRecall : onDelete;
    setConfirming(true);
    setConfirmError('');
    void action()
      .then(() => {
        setConfirm(null);
        notify({
          message: confirm === 'recall' ? '已收回訊息。' : '已刪除訊息（僅自己檢視）。',
          tone: 'success',
        });
      })
      .catch((actionError: unknown) => {
        const message =
          actionError instanceof Error ? actionError.message : '操作失敗，請再試一次。';
        setConfirmError(message);
        notify({
          title: confirm === 'recall' ? '收回失敗' : '刪除失敗',
          message,
          tone: 'error',
        });
      })
      .finally(() => {
        setConfirming(false);
      });
  };

  return (
    <div className="et-own-actions">
      {edited ? <span className="et-own-actions__edited">已編輯</span> : null}
      {supported && !editing ? (
        <>
          <button
            type="button"
            className="et-own-actions__btn"
            aria-label="編輯此訊息"
            onClick={startEdit}
          >
            編輯
          </button>
          <button
            type="button"
            className="et-own-actions__btn"
            aria-label="收回此訊息"
            onClick={() => {
              setConfirmError('');
              setConfirm('recall');
            }}
          >
            收回
          </button>
          <button
            type="button"
            className="et-own-actions__btn"
            aria-label="僅刪除自己檢視的此訊息"
            onClick={() => {
              setConfirmError('');
              setConfirm('delete');
            }}
          >
            僅自己刪除
          </button>
        </>
      ) : null}
      {supported && editing ? (
        <div className="et-own-actions__editor">
          <label className="et-own-actions__editor-label" htmlFor={`edit-${messageId}`}>
            編輯訊息
          </label>
          <textarea
            id={`edit-${messageId}`}
            className="et-own-actions__editor-input"
            rows={3}
            value={draft}
            maxLength={MAX_EDIT_CONTENT_RUNES + 100}
            onChange={(event) => {
              setDraft(event.target.value);
              setEditError('');
            }}
          />
          {editError !== '' ? (
            <p className="et-field__error" role="alert">
              {editError}
            </p>
          ) : null}
          <div className="et-own-actions__editor-actions">
            <Button
              type="button"
              variant="secondary"
              size="sm"
              disabled={saving}
              onClick={() => {
                setEditing(false);
                setEditError('');
              }}
            >
              取消
            </Button>
            <Button type="button" size="sm" disabled={saving} loading={saving} onClick={saveEdit}>
              {saving ? '儲存中…' : '儲存編輯'}
            </Button>
          </div>
        </div>
      ) : null}
      <Dialog
        open={confirm !== null}
        onClose={() => {
          if (!confirming) {
            setConfirm(null);
          }
        }}
        title={confirm === 'delete' ? '僅自己刪除訊息' : '收回訊息'}
        description={
          confirm === 'delete'
            ? '只會從你的檢視中移除這則訊息，其他成員不受影響。'
            : '收回後所有成員都看不到這則訊息，且無法復原。'
        }
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => {
                setConfirm(null);
              }}
              disabled={confirming}
            >
              取消
            </Button>
            <Button disabled={confirming} loading={confirming} onClick={submitConfirm}>
              {confirming ? '處理中…' : confirm === 'delete' ? '僅刪除我的檢視' : '收回訊息'}
            </Button>
          </>
        }
      >
        {confirmError !== '' ? (
          <p className="et-field__error" role="alert">
            {confirmError}
          </p>
        ) : null}
      </Dialog>
    </div>
  );
}

/** 轉傳目標搜尋過濾（純函式）：依標題子字串比對，不分大小寫。 */
export function filterForwardTargets(targets: ForwardTarget[], query: string): ForwardTarget[] {
  const needle = query.trim().toLocaleLowerCase();
  if (needle === '') {
    return targets;
  }
  return targets.filter((target) => target.title.toLocaleLowerCase().includes(needle));
}

interface ForwardDialogProps {
  open: boolean;
  targets: ForwardTarget[];
  onClose: () => void;
  /** 轉傳成功時 resolve（父層負責後續處理）；失敗時 reject。 */
  onForward: (targetConversationId: string) => Promise<void>;
}

/**
 * 轉傳對話框（§2t，展示／API 雙模式共用）：目標對話由父層傳
 * 入（即對話清單），搜尋過濾、單選、失敗時對話框內顯示錯誤
 * 文字並 toast（可重試），成功時由父層關閉。開啟時重置查詢
 * 與選取。轉傳內容與原文相同，不帶附件與引用（後端語義）。
 */
export function ForwardDialog({
  open,
  targets,
  onClose,
  onForward,
}: ForwardDialogProps): ReactElement {
  const { notify } = useToast();
  const [query, setQuery] = useState('');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [error, setError] = useState('');
  const [forwarding, setForwarding] = useState(false);

  useEffect(() => {
    if (open) {
      setQuery('');
      setSelectedId(null);
      setError('');
      setForwarding(false);
    }
  }, [open]);

  const filtered = useMemo(() => filterForwardTargets(targets, query), [targets, query]);

  const submit = (): void => {
    if (selectedId === null) {
      setError('請先選擇一個對話。');
      return;
    }
    if (forwarding) {
      return;
    }
    const targetId = selectedId;
    setForwarding(true);
    setError('');
    void onForward(targetId)
      .then(() => {
        onClose();
      })
      .catch((forwardError: unknown) => {
        const message =
          forwardError instanceof Error ? forwardError.message : '轉傳失敗，請再試一次。';
        setError(message);
        notify({ title: '轉傳失敗', message, tone: 'error' });
      })
      .finally(() => {
        setForwarding(false);
      });
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="轉傳訊息"
      description="選擇一個對話。轉傳內容與原文相同，不帶附件與引用。"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={forwarding}>
            取消
          </Button>
          <Button
            disabled={forwarding || selectedId === null}
            loading={forwarding}
            onClick={submit}
          >
            {forwarding ? '轉傳中…' : '轉傳'}
          </Button>
        </>
      }
    >
      <SearchField
        value={query}
        onChange={(value) => {
          setQuery(value);
        }}
        placeholder="搜尋對話"
        label="搜尋對話"
      />
      {targets.length === 0 ? (
        <p className="et-chats__empty">目前沒有可轉傳的對話。</p>
      ) : filtered.length === 0 ? (
        <p className="et-chats__empty">沒有符合條件的對話。試試其他關鍵字。</p>
      ) : (
        <div role="listbox" aria-label="對話清單" className="et-friend-list">
          {filtered.map((target) => (
            <button
              key={target.id}
              type="button"
              role="option"
              aria-selected={selectedId === target.id}
              className="et-friend-row et-friend-row--pick"
              onClick={() => {
                setSelectedId(target.id);
                setError('');
              }}
            >
              <Avatar name={target.title} size="md" />
              <span className="et-friend-row__main">
                <span className="et-friend-row__name">{target.title}</span>
              </span>
              {selectedId === target.id ? <Badge tone="accent">已選擇</Badge> : null}
            </button>
          ))}
        </div>
      )}
      {error !== '' ? (
        <p className="et-field__error" role="alert">
          {error}
        </p>
      ) : null}
    </Dialog>
  );
}

export type AttachFilesState = 'loading' | 'error' | 'ready' | 'unsupported';

interface AttachFilesDialogProps {
  open: boolean;
  files: MediaFileItem[];
  filesState: AttachFilesState;
  onRetryFiles: () => void;
  onClose: () => void;
  /** 確認成功時 resolve（父層負責後續處理）；失敗時 reject。 */
  onConfirm: (mediaFileIds: string[]) => Promise<void>;
  confirmLabel: string;
}

/**
 * 附加檔案挑選對話框（§2t，展示／API 雙模式共用）：檔案清單
 * 由父層傳入（provider.listMediaFilesAsync）。僅掃描通過
 * （clean）的檔案可選；未通過掃描的檔案顯示狀態且停用——
 * 後端掃描閘門 fail-closed，UI 不得讓使用者選到不可關聯的
 * 檔案。單次最多 10 個（後端上限）。關聯的是既有媒體檔案，
 * 不會重新上傳檔案位元組。
 */
export function AttachFilesDialog({
  open,
  files,
  filesState,
  onRetryFiles,
  onClose,
  onConfirm,
  confirmLabel,
}: AttachFilesDialogProps): ReactElement {
  const { notify } = useToast();
  const [selected, setSelected] = useState<ReadonlySet<string>>(new Set());
  const [error, setError] = useState('');
  const [confirming, setConfirming] = useState(false);

  useEffect(() => {
    if (open) {
      setSelected(new Set());
      setError('');
      setConfirming(false);
    }
  }, [open]);

  const toggle = (fileId: string): void => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(fileId)) {
        next.delete(fileId);
      } else if (next.size >= 10) {
        setError('最多選擇 10 個檔案。');
        return prev;
      } else {
        next.add(fileId);
      }
      setError('');
      return next;
    });
  };

  const submit = (): void => {
    if (selected.size === 0) {
      setError('請先選擇至少一個檔案。');
      return;
    }
    if (confirming) {
      return;
    }
    const ids = [...selected];
    setConfirming(true);
    setError('');
    void onConfirm(ids)
      .then(() => {
        onClose();
      })
      .catch((confirmError: unknown) => {
        const message =
          confirmError instanceof Error ? confirmError.message : '附加檔案失敗，請再試一次。';
        setError(message);
        notify({ title: '附加檔案失敗', message, tone: 'error' });
      })
      .finally(() => {
        setConfirming(false);
      });
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="選擇附件"
      description="只列出已通過掃描的媒體檔案。附件關聯既有檔案，不會重新上傳檔案位元組。"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={confirming}>
            取消
          </Button>
          <Button
            disabled={confirming || selected.size === 0}
            loading={confirming}
            onClick={submit}
          >
            {confirming ? '處理中…' : confirmLabel}
          </Button>
        </>
      }
    >
      {filesState === 'loading' ? (
        <p className="et-chats__empty">載入媒體檔案中…</p>
      ) : filesState === 'error' ? (
        <div className="et-chats__empty">
          <p>載入媒體檔案失敗，請檢查連線後再試一次。</p>
          <Button type="button" variant="secondary" size="sm" onClick={onRetryFiles}>
            重新載入
          </Button>
        </div>
      ) : filesState === 'unsupported' ? (
        <p className="et-chats__empty">目前的資料模式不提供媒體檔案清單。</p>
      ) : files.length === 0 ? (
        <p className="et-chats__empty">還沒有媒體檔案。先到「媒體」建立檔案，再回來附加。</p>
      ) : (
        <div role="listbox" aria-label="媒體檔案清單" aria-multiselectable="true">
          {files.map((file) => {
            const attachable = file.scanStatus === 'clean';
            const isSelected = selected.has(file.id);
            return (
              <button
                key={file.id}
                type="button"
                role="option"
                aria-selected={isSelected}
                disabled={!attachable}
                className="et-friend-row et-friend-row--pick"
                onClick={() => {
                  toggle(file.id);
                }}
              >
                <span className="et-friend-row__main">
                  <span className="et-friend-row__name">{file.originalFilename}</span>
                  <span className="et-friend-row__meta">
                    {file.mimeType} · {formatAttachmentSize(file.sizeBytes)}
                    {attachable ? '' : ` · 掃描狀態：${file.scanStatus}，不可關聯`}
                  </span>
                </span>
                {isSelected ? <Badge tone="accent">已選擇</Badge> : null}
              </button>
            );
          })}
        </div>
      )}
      {error !== '' ? (
        <p className="et-field__error" role="alert">
          {error}
        </p>
      ) : null}
    </Dialog>
  );
}

export interface MessageSearchOption {
  value: string;
  label: string;
}

export interface MessageSearchPanelProps {
  conversations: MessageSearchOption[];
  senders: MessageSearchOption[];
  supported: boolean;
  sourceNote: string;
  onSearch: (input: MessageSearchInput) => Promise<MessageSearchPage>;
  onJump: (result: MessageSearchResultItem) => void;
  onClose: () => void;
}

type MessageSearchPanelState = 'idle' | 'loading' | 'error' | 'ready';

/**
 * 訊息搜尋面板（§2v，展示／API 雙模式同一元件）。
 *
 * 查詢語義以後端為準：關鍵字必填、AND、子字串、大小寫不敏感、
 * 建立時間降序、游標分頁；篩選僅以後端支援者為準（對話、發送
 * 者、建立時間區間，皆含邊界）。結果點選後由呼叫端跳至該訊息
 * 所在的對話。搜尋無即時性可言：索引由事件投影填充，編輯後的
 * 內容在事件協定定稿前不會更新索引（見 STATUS.md §2v）。
 */
export function MessageSearchPanel({
  conversations,
  senders,
  supported,
  sourceNote,
  onSearch,
  onJump,
  onClose,
}: MessageSearchPanelProps): ReactElement {
  const [keyword, setKeyword] = useState('');
  const [conversationId, setConversationId] = useState('');
  const [senderId, setSenderId] = useState('');
  const [sinceDate, setSinceDate] = useState('');
  const [untilDate, setUntilDate] = useState('');
  const [formError, setFormError] = useState('');
  const [panelState, setPanelState] = useState<MessageSearchPanelState>('idle');
  const [results, setResults] = useState<MessageSearchResultItem[]>([]);
  const [nextCursor, setNextCursor] = useState('');
  const [loadingMore, setLoadingMore] = useState(false);
  const [searchedKeyword, setSearchedKeyword] = useState('');
  const now = useMemo(() => new Date(), []);
  const conversationTitles = useMemo(
    () => new Map(conversations.map((option) => [option.value, option.label])),
    [conversations],
  );
  const senderLabels = useMemo(
    () => new Map(senders.map((option) => [option.value, option.label])),
    [senders],
  );

  const buildInput = (cursor?: string): MessageSearchInput | null => {
    const trimmed = keyword.trim();
    if (trimmed === '') {
      setFormError('請輸入關鍵字。');
      return null;
    }
    if (sinceDate !== '' && untilDate !== '' && sinceDate > untilDate) {
      setFormError('結束日期不得早於起始日期。');
      return null;
    }
    setFormError('');
    const since = searchDateToSince(sinceDate) ?? undefined;
    const until = searchDateToUntil(untilDate) ?? undefined;
    return {
      query: trimmed,
      ...(conversationId === '' ? {} : { conversationId }),
      ...(senderId === '' ? {} : { senderId }),
      ...(since === undefined ? {} : { since }),
      ...(until === undefined ? {} : { until }),
      limit: 20,
      ...(cursor === undefined ? {} : { cursor }),
    };
  };

  const runSearch = (cursor?: string): void => {
    const input = buildInput(cursor);
    if (input === null) {
      return;
    }
    if (cursor === undefined) {
      setPanelState('loading');
    } else {
      setLoadingMore(true);
    }
    void onSearch(input)
      .then((page) => {
        setResults((prev) => (cursor === undefined ? page.results : [...prev, ...page.results]));
        setNextCursor(page.nextCursor);
        setSearchedKeyword(input.query);
        setPanelState('ready');
      })
      .catch(() => {
        setPanelState('error');
      })
      .finally(() => {
        setLoadingMore(false);
      });
  };

  if (!supported) {
    return (
      <div className="et-search-panel" aria-label="搜尋訊息">
        <p className="et-chats__empty">目前的資料模式不提供訊息搜尋。</p>
        <Button type="button" variant="secondary" size="sm" onClick={onClose}>
          返回對話列表
        </Button>
      </div>
    );
  }

  return (
    <div className="et-search-panel" aria-label="搜尋訊息">
      <div className="et-search-panel__head">
        <h2 className="et-search-panel__title">搜尋訊息</h2>
        <Button type="button" variant="secondary" size="sm" onClick={onClose}>
          返回對話列表
        </Button>
      </div>
      <form
        className="et-search-panel__form"
        onSubmit={(event) => {
          event.preventDefault();
          runSearch();
        }}
      >
        <SearchField
          value={keyword}
          onChange={setKeyword}
          placeholder="輸入關鍵字"
          label="搜尋關鍵字"
        />
        <Select
          label="對話"
          value={conversationId}
          onChange={(event) => {
            setConversationId(event.target.value);
          }}
          options={[{ value: '', label: '全部對話' }, ...conversations]}
        />
        <Select
          label="發送者"
          value={senderId}
          onChange={(event) => {
            setSenderId(event.target.value);
          }}
          options={[{ value: '', label: '全部成員' }, ...senders]}
        />
        <Input
          type="date"
          label="起始日期"
          value={sinceDate}
          onChange={(event) => {
            setSinceDate(event.target.value);
          }}
        />
        <Input
          type="date"
          label="結束日期"
          value={untilDate}
          onChange={(event) => {
            setUntilDate(event.target.value);
          }}
        />
        {formError !== '' ? (
          <p className="et-field__error" role="alert">
            {formError}
          </p>
        ) : null}
        <Button type="submit" variant="primary" size="md" disabled={panelState === 'loading'}>
          {panelState === 'loading' ? '搜尋中…' : '搜尋'}
        </Button>
      </form>
      <p className="et-chats__source">{sourceNote}</p>
      {panelState === 'error' ? (
        <div className="et-chats__empty">
          <p>搜尋失敗，請檢查連線後再試一次。</p>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={() => {
              runSearch();
            }}
          >
            重新搜尋
          </Button>
        </div>
      ) : panelState === 'ready' && results.length === 0 ? (
        <p className="et-chats__empty">找不到符合「{searchedKeyword}」的訊息。</p>
      ) : null}
      {results.length > 0 ? (
        <div className="et-search-panel__results et-scroll" role="list" aria-label="搜尋結果">
          {results.map((result) => (
            <button
              type="button"
              key={result.id}
              className="et-search-result"
              onClick={() => {
                onJump(result);
              }}
            >
              <span className="et-search-result__top">
                <span className="et-search-result__conversation">
                  {conversationTitles.get(result.conversationId) ?? '對話'}
                </span>
                <time className="et-search-result__time" dateTime={result.createdAt}>
                  {formatMessageTime(result.createdAt, now)}
                </time>
              </span>
              <span className="et-search-result__sender">
                {senderLabels.get(result.senderId) ?? result.senderId.slice(0, 8)}
              </span>
              <span className="et-search-result__content">{result.content}</span>
            </button>
          ))}
        </div>
      ) : null}
      {panelState === 'ready' && nextCursor !== '' ? (
        <Button
          type="button"
          variant="secondary"
          size="sm"
          disabled={loadingMore}
          onClick={() => {
            runSearch(nextCursor);
          }}
        >
          {loadingMore ? '載入中…' : '載入更多結果'}
        </Button>
      ) : null}
    </div>
  );
}

export interface MessageAttachments {
  attachmentsById: Record<string, MessageAttachmentItem[]>;
  forwardsById: Record<string, ForwardRecordItem>;
  supported: boolean;
  refreshAll: () => void;
  addAttachments: (messageId: string, mediaFileIds: string[]) => Promise<void>;
}

/**
 * 訊息附件與轉傳紀錄的共用狀態（§2t，展示／API 雙模式同一邏輯）。
 *
 * 附件清單與轉傳紀錄以 REST 另查：歷史載入後，對尚未查過的
 * 訊息各查一次（失敗靜默，不阻擋歷史；轉傳紀錄查無時視為
 * 非轉傳產物——後端對兩者皆回 404）。寫入後以伺服器回傳為
 * 準更新本機狀態。反應／置頂同樣無即時推播：他人的附件與轉
 * 傳變更須以「重新載入」才會出現（realtime stub 不動）。
 * provider 未提供附件／轉傳方法時，supported 為 false，呼叫
 * 端不渲染附件區塊與轉傳按鈕。
 */
function useMessageAttachments(
  provider: ChatDataProvider,
  conversationId: string | null,
  messageIds: string[],
): MessageAttachments {
  const [attachmentsById, setAttachmentsById] = useState<Record<string, MessageAttachmentItem[]>>(
    {},
  );
  const [forwardsById, setForwardsById] = useState<Record<string, ForwardRecordItem>>({});
  const [epoch, setEpoch] = useState(0);
  const loadedRef = useRef<Set<string>>(new Set());
  const supported =
    provider.listAttachmentsAsync !== undefined &&
    provider.addAttachmentsAsync !== undefined &&
    provider.forwardMessageAsync !== undefined &&
    provider.getForwardAsync !== undefined;
  const idsKey = messageIds.join('\n');

  useEffect(() => {
    if (!supported || conversationId === null) {
      return;
    }
    const ids = idsKey === '' ? [] : idsKey.split('\n');
    const missing = ids.filter((id) => !loadedRef.current.has(`${conversationId}:${id}`));
    if (missing.length === 0) {
      return;
    }
    const listAttachments = provider.listAttachmentsAsync;
    const getForward = provider.getForwardAsync;
    if (listAttachments === undefined || getForward === undefined) {
      return;
    }
    let cancelled = false;
    for (const id of missing) {
      loadedRef.current.add(`${conversationId}:${id}`);
    }
    void Promise.allSettled(missing.map((id) => listAttachments(id))).then((results) => {
      if (cancelled) {
        return;
      }
      setAttachmentsById((prev) => {
        const next = { ...prev };
        results.forEach((result, index) => {
          const id = missing[index];
          if (result.status === 'fulfilled' && id !== undefined) {
            next[id] = result.value;
          }
        });
        return next;
      });
    });
    void Promise.allSettled(missing.map((id) => getForward(id))).then((results) => {
      if (cancelled) {
        return;
      }
      setForwardsById((prev) => {
        const next = { ...prev };
        results.forEach((result, index) => {
          const id = missing[index];
          if (result.status === 'fulfilled' && id !== undefined) {
            next[id] = result.value;
          }
        });
        return next;
      });
    });
    return () => {
      cancelled = true;
    };
  }, [supported, conversationId, idsKey, epoch, provider]);

  const refreshAll = useCallback(() => {
    loadedRef.current.clear();
    setAttachmentsById({});
    setForwardsById({});
    setEpoch((value) => value + 1);
  }, []);

  const addAttachments = useCallback(
    async (messageId: string, mediaFileIds: string[]): Promise<void> => {
      const updated = await provider.addAttachmentsAsync?.(messageId, mediaFileIds);
      if (updated !== undefined) {
        setAttachmentsById((prev) => ({ ...prev, [messageId]: updated }));
      }
    },
    [provider],
  );

  return { attachmentsById, forwardsById, supported, refreshAll, addAttachments };
}

/**
 * 附件下載回呼（§2aa，展示／API 雙模式共用）：經 provider 解析
 * 下載（授權檢查＋URL／位元組取得），再依解析結果開啟新分頁
 * 或儲存檔案。provider 未提供解析方法時回傳 null，呼叫端改
 * 顯示「不提供下載」註記。掃描未通過時 toast 說明，不下載。
 */
export function useMediaDownload(
  provider: ChatDataProvider,
): ((attachment: MessageAttachmentItem) => Promise<void>) | null {
  const { notify } = useToast();
  const resolve = provider.resolveMediaDownloadAsync;
  const download = useCallback(
    async (attachment: MessageAttachmentItem): Promise<void> => {
      if (resolve === undefined) {
        return;
      }
      try {
        const result = await resolve(attachment.mediaFileId);
        if (result.status === 'forbidden') {
          notify({
            title: '不可下載',
            message: `掃描狀態為 ${result.scanStatus}，檔案不可分發。`,
            tone: 'warning',
          });
          return;
        }
        if (result.status === 'unavailable') {
          notify({
            title: '無法下載',
            message: '已授權，但此部署未提供可下載的檔案位址。',
            tone: 'warning',
          });
          return;
        }
        if (result.status === 'ready-url') {
          openDownloadUrl(result.url);
          return;
        }
        downloadBlob(result.blob, attachment.fileName);
      } catch (error) {
        notify({
          title: '下載失敗',
          message: error instanceof Error ? error.message : '請再試一次。',
          tone: 'error',
        });
      }
    },
    [notify, resolve],
  );
  if (resolve === undefined) {
    return null;
  }
  return download;
}

export interface ReactionsAndPins {
  summaries: Record<string, ReactionSummaryItem>;
  pins: PinItem[];
  pinsState: PinsState;
  pickerFor: string | null;
  supported: boolean;
  setPickerFor: (messageId: string | null) => void;
  reloadPins: () => void;
  refreshAll: () => void;
  pickReaction: (messageId: string, emoji: string) => Promise<void>;
  removeReaction: (messageId: string) => Promise<void>;
  togglePin: (messageId: string) => Promise<void>;
}

/**
 * 反應與置頂的共用狀態（§2q，展示／API 雙模式同一邏輯）。
 *
 * 彙總以 REST 另查：歷史載入後，對尚未查過的訊息各查一次（失敗
 * 靜默、無彙總即不顯示 chip，不阻擋歷史）；寫入後以伺服器回傳
 * 的彙總為準更新本機狀態。反應／置頂無即時推播：他人的變更須
 * 以「重新載入」才會出現（realtime stub 不動）。provider 未提供
 * 反應／置頂方法時，supported 為 false，呼叫端不渲染操作列。
 */
function useReactionsAndPins(
  provider: ChatDataProvider,
  conversationId: string | null,
  messageIds: string[],
): ReactionsAndPins {
  const { notify } = useToast();
  const [summaries, setSummaries] = useState<Record<string, ReactionSummaryItem>>({});
  const [pins, setPins] = useState<PinItem[]>([]);
  const [pinsState, setPinsState] = useState<PinsState>('idle');
  const [pinsEpoch, setPinsEpoch] = useState(0);
  const [pickerFor, setPickerFor] = useState<string | null>(null);
  const loadedRef = useRef<Set<string>>(new Set());
  const supported =
    provider.listReactionsAsync !== undefined &&
    provider.setReactionAsync !== undefined &&
    provider.removeReactionAsync !== undefined &&
    provider.listPinsAsync !== undefined &&
    provider.pinMessageAsync !== undefined &&
    provider.unpinMessageAsync !== undefined;
  const idsKey = messageIds.join('\n');

  const fail = useCallback(
    (error: unknown, title: string) => {
      notify({
        title,
        message: error instanceof Error ? error.message : '操作失敗，請再試一次。',
        tone: 'error',
      });
    },
    [notify],
  );

  // 彙總載入：只查尚未查過的訊息（以「對話：訊息」為鍵去重）；
  // 單則查詢失敗不影響其他訊息，亦不阻擋歷史顯示。
  useEffect(() => {
    if (!supported || conversationId === null) {
      return;
    }
    const ids = idsKey === '' ? [] : idsKey.split('\n');
    const missing = ids.filter((id) => !loadedRef.current.has(`${conversationId}:${id}`));
    if (missing.length === 0) {
      return;
    }
    const listReactions = provider.listReactionsAsync;
    if (listReactions === undefined) {
      return;
    }
    let cancelled = false;
    for (const id of missing) {
      loadedRef.current.add(`${conversationId}:${id}`);
    }
    void Promise.allSettled(missing.map((id) => listReactions(id))).then((results) => {
      if (cancelled) {
        return;
      }
      setSummaries((prev) => {
        const next = { ...prev };
        results.forEach((result, index) => {
          const id = missing[index];
          if (result.status === 'fulfilled' && id !== undefined) {
            next[id] = result.value;
          }
        });
        return next;
      });
    });
    return () => {
      cancelled = true;
    };
  }, [supported, conversationId, idsKey, provider]);

  // 置頂清單載入：隨對話切換與手動重載重新查詢。
  useEffect(() => {
    if (conversationId === null) {
      setPins([]);
      setPinsState('idle');
      return;
    }
    if (!supported || provider.listPinsAsync === undefined) {
      setPins([]);
      setPinsState('unsupported');
      return;
    }
    const listPins = provider.listPinsAsync;
    let cancelled = false;
    setPinsState('loading');
    void listPins(conversationId)
      .then((fetched) => {
        if (!cancelled) {
          setPins(fetched);
          setPinsState('ready');
        }
      })
      .catch(() => {
        if (!cancelled) {
          setPinsState('error');
        }
      });
    return () => {
      cancelled = true;
    };
  }, [supported, conversationId, pinsEpoch, provider]);

  const reloadPins = useCallback(() => {
    setPinsEpoch((epoch) => epoch + 1);
  }, []);

  const refreshAll = useCallback(() => {
    loadedRef.current.clear();
    setSummaries({});
    setPickerFor(null);
    reloadPins();
  }, [reloadPins]);

  const pickReaction = useCallback(
    async (messageId: string, emoji: string): Promise<void> => {
      setPickerFor(null);
      const myReaction = summaries[messageId]?.reactions.find((item) => item.reactedByMe)?.reaction;
      try {
        const updated =
          myReaction === emoji
            ? await provider.removeReactionAsync?.(messageId)
            : await provider.setReactionAsync?.(messageId, emoji);
        if (updated !== undefined) {
          setSummaries((prev) => ({ ...prev, [messageId]: updated }));
        }
      } catch (error) {
        fail(error, '反應失敗');
      }
    },
    [provider, summaries, fail],
  );

  const removeReaction = useCallback(
    async (messageId: string): Promise<void> => {
      try {
        const updated = await provider.removeReactionAsync?.(messageId);
        if (updated !== undefined) {
          setSummaries((prev) => ({ ...prev, [messageId]: updated }));
        }
      } catch (error) {
        fail(error, '取消反應失敗');
      }
    },
    [provider, fail],
  );

  const togglePin = useCallback(
    async (messageId: string): Promise<void> => {
      if (conversationId === null) {
        return;
      }
      const pinned = pins.some((pin) => pin.messageId === messageId);
      try {
        if (pinned) {
          await provider.unpinMessageAsync?.(conversationId, messageId);
          setPins((prev) => prev.filter((pin) => pin.messageId !== messageId));
        } else {
          const created = await provider.pinMessageAsync?.(conversationId, messageId);
          if (created !== undefined) {
            setPins((prev) =>
              [...prev, created].sort((a, b) => a.pinnedAt.localeCompare(b.pinnedAt)),
            );
          }
        }
      } catch (error) {
        fail(error, pinned ? '取消置頂失敗' : '置頂失敗');
      }
    },
    [provider, conversationId, pins, fail],
  );

  return {
    summaries,
    pins,
    pinsState,
    pickerFor,
    supported,
    setPickerFor,
    reloadPins,
    refreshAll,
    pickReaction,
    removeReaction,
    togglePin,
  };
}

/**
 * 送達回條與回條列表（§2r，API 模式）：歷史載入完成時對當頁每則
 * 他人訊息各送一則 delivered（逐則，無批次端點；工作階段內去
 * 重，失敗靜默）；自己訊息的回條列表逐訊息查詢（失敗靜默），
 * 手動「重新載入」歷史（historyEpoch 變化）時重查。無即時推播，
 * 他人的回條須重新載入才會出現。回傳原始回條列，狀態推導由
 * deriveReceiptStatus 在渲染時依最新成員清單計算。
 */
function useMessageReceipts(
  provider: ChatDataProvider,
  conversationId: string | null,
  messages: MessageItem[],
  historyReady: boolean,
  myId: string | null,
  historyEpoch: number,
): { receiptsById: Record<string, ReceiptItem[]>; supported: boolean } {
  const [receiptsById, setReceiptsById] = useState<Record<string, ReceiptItem[]>>({});
  const deliveredRef = useRef<Set<string>>(new Set());
  const loadedRef = useRef<Set<string>>(new Set());
  const supported =
    provider.postReceiptAsync !== undefined && provider.listReceiptsAsync !== undefined;

  useEffect(() => {
    if (!supported || conversationId === null || !historyReady) {
      return;
    }
    const postReceipt = provider.postReceiptAsync;
    if (postReceipt === undefined) {
      return;
    }
    const targets = selectDeliveredReceiptTargets(messages, myId).filter(
      (id) => !deliveredRef.current.has(`${conversationId}:${id}`),
    );
    if (targets.length === 0) {
      return;
    }
    let cancelled = false;
    void Promise.allSettled(targets.map((id) => postReceipt(id, 'delivered'))).then((results) => {
      if (cancelled) {
        return;
      }
      results.forEach((result, index) => {
        const id = targets[index];
        if (result.status === 'fulfilled' && id !== undefined) {
          deliveredRef.current.add(`${conversationId}:${id}`);
        }
      });
    });
    return () => {
      cancelled = true;
    };
  }, [supported, conversationId, historyReady, messages, myId, provider]);

  useEffect(() => {
    if (!supported || conversationId === null) {
      return;
    }
    const listReceipts = provider.listReceiptsAsync;
    if (listReceipts === undefined) {
      return;
    }
    const ownIds = messages
      .filter(
        (message) =>
          message.senderId === myId && message.status !== 'recalled' && !message.deletedForSelf,
      )
      .map((message) => message.id);
    const missing = ownIds.filter(
      (id) => !loadedRef.current.has(`${conversationId}:${historyEpoch}:${id}`),
    );
    if (missing.length === 0) {
      return;
    }
    for (const id of missing) {
      loadedRef.current.add(`${conversationId}:${historyEpoch}:${id}`);
    }
    let cancelled = false;
    void Promise.allSettled(missing.map((id) => listReceipts(id))).then((results) => {
      if (cancelled) {
        return;
      }
      setReceiptsById((prev) => {
        const next = { ...prev };
        results.forEach((result, index) => {
          const id = missing[index];
          if (result.status === 'fulfilled' && id !== undefined) {
            next[id] = result.value;
          }
        });
        return next;
      });
    });
    return () => {
      cancelled = true;
    };
  }, [supported, conversationId, historyEpoch, messages, myId, provider]);

  return { receiptsById, supported };
}

function resolveConversationTitle(
  conversation: ConversationItemModel,
  members: ConversationMemberItem[],
  friendsById: ReadonlyMap<string, FriendItem>,
  myId: string | null,
): string {
  if (conversation.title !== '') {
    return conversation.title;
  }
  if (conversation.type === 'group') {
    return '未命名群組';
  }
  const other = members.find((member) => member.leftAt === null && member.userId !== myId);
  const name = other === undefined ? undefined : friendsById.get(other.userId)?.displayName;
  return name ?? '一對一對話';
}

function senderLabel(
  senderId: string,
  myId: string | null,
  friendsById: ReadonlyMap<string, FriendItem>,
): string {
  if (myId !== null && senderId === myId) {
    return '你';
  }
  return friendsById.get(senderId)?.displayName ?? `使用者 ${senderId.slice(0, 8)}`;
}

function toChatMessage(item: MessageItem): ChatMessage {
  const recalled = item.status === 'recalled';
  return {
    id: item.id,
    conversationId: item.conversationId,
    senderId: item.senderId,
    body: recalled ? '此訊息已收回' : item.deletedForSelf ? '此訊息已刪除' : item.content,
    sentAt: item.createdAt,
    status: 'sent',
    kind: 'text',
    ...(item.editedAt === null ? {} : { editedAt: item.editedAt }),
    recalled,
    deletedForSelf: item.deletedForSelf,
  };
}

function ApiChatsView({
  provider,
  selectedId,
  onSelect,
  onUnreadChange,
}: {
  provider: ChatDataProvider;
  selectedId: string | null;
  onSelect: (conversationId: string | null) => void;
  onUnreadChange: ((count: number) => void) | undefined;
}): ReactElement {
  const { notify } = useToast();
  const isMobile = useIsMobile();
  const [query, setQuery] = useState('');
  const [filter, setFilter] = useState<ApiListFilter>('all');
  const [conversations, setConversations] = useState<ConversationItemModel[]>([]);
  const [friends, setFriends] = useState<FriendItem[]>([]);
  const [listState, setListState] = useState<ApiListState>('loading');
  const [listStale, setListStale] = useState(false);
  const [histories, setHistories] = useState<Record<string, HistoryEntry>>({});
  const [historyState, setHistoryState] = useState<ApiHistoryState>('idle');
  const [historyEpoch, setHistoryEpoch] = useState(0);
  const [loadingMore, setLoadingMore] = useState(false);
  // 對話成員（§2x）：共用 hook 載入與重查；成員清單供標題解析
  // 與回條狀態推導，載入失敗時退回空清單（不阻擋歷史顯示）。
  const conversationMembers = useConversationMembers(provider, selectedId);
  const members = conversationMembers.members;
  const [createOpen, setCreateOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [highlightMessageId, setHighlightMessageId] = useState<string | null>(null);
  /** 醒目訊息所屬的對話：手動切換對話時清除醒目，跳轉時保留。 */
  const highlightForRef = useRef<string | null>(null);
  const messagesRef = useRef<HTMLDivElement | null>(null);
  const mountedRef = useRef(true);
  const historyRequestRef = useRef(0);
  const suppressScrollRef = useRef(false);
  /** 已送出回條的「對話：序號」鍵（工作階段內去重，無新訊息不重送）。 */
  const receiptedRef = useRef<Set<string>>(new Set());
  const conversationsRef = useRef<ConversationItemModel[]>([]);
  conversationsRef.current = conversations;
  const historiesRef = useRef<Record<string, HistoryEntry>>({});
  historiesRef.current = histories;
  const now = useMemo(() => new Date(), []);
  const myId = provider.getCurrentUserId?.() ?? null;
  const friendsById = useMemo(
    () => new Map(friends.map((friend) => [friend.id, friend])),
    [friends],
  );

  // 反應與置頂（§2q）：彙總與清單以 REST 另查，無即時推播。
  const selectedEntryForReactions = selectedId === null ? undefined : histories[selectedId];
  const apiMessageIds = useMemo(
    () => selectedEntryForReactions?.messages.map((message) => message.id) ?? [],
    [selectedEntryForReactions],
  );
  const reactions = useReactionsAndPins(provider, selectedId, apiMessageIds);

  // 回條（§2r）：送達回條逐則送出、回條列表逐訊息查詢，皆失敗靜默。
  const otherMemberIds = useMemo(
    () =>
      members
        .filter((member) => member.leftAt === null && member.userId !== myId)
        .map((member) => member.userId),
    [members, myId],
  );
  const { receiptsById } = useMessageReceipts(
    provider,
    selectedId,
    selectedEntryForReactions?.messages ?? [],
    historyState === 'ready',
    myId,
    historyEpoch,
  );

  // 附件與轉傳（§2t）：附件清單與轉傳紀錄以 REST 另查，無即時推播。
  const attachments = useMessageAttachments(provider, selectedId, apiMessageIds);
  const downloadAttachment = useMediaDownload(provider);
  const [forwardFor, setForwardFor] = useState<string | null>(null);
  const [attachPickerOpen, setAttachPickerOpen] = useState(false);
  const [attachPickerMode, setAttachPickerMode] = useState<'compose' | 'add'>('compose');
  const [attachTargetId, setAttachTargetId] = useState<string | null>(null);
  const [pendingAttachments, setPendingAttachments] = useState<MediaFileItem[]>([]);
  const [mediaFiles, setMediaFiles] = useState<MediaFileItem[]>([]);
  const [mediaFilesState, setMediaFilesState] = useState<AttachFilesState>('loading');

  const loadMediaFiles = useCallback(() => {
    const listFiles = provider.listMediaFilesAsync;
    if (listFiles === undefined) {
      setMediaFilesState('unsupported');
      return;
    }
    setMediaFilesState('loading');
    let cancelled = false;
    void listFiles()
      .then((fetched) => {
        if (!cancelled) {
          setMediaFiles(fetched);
          setMediaFilesState('ready');
        }
      })
      .catch(() => {
        if (!cancelled) {
          setMediaFilesState('error');
        }
      });
    return () => {
      cancelled = true;
    };
  }, [provider]);

  useEffect(() => {
    if (!attachPickerOpen) {
      return undefined;
    }
    return loadMediaFiles();
  }, [attachPickerOpen, loadMediaFiles]);

  const openAttachPicker = (mode: 'compose' | 'add', messageId: string | null): void => {
    if (provider.listMediaFilesAsync === undefined) {
      notify({
        title: '尚未接線',
        message: '目前的資料模式不提供媒體檔案清單。',
        tone: 'warning',
      });
      return;
    }
    setAttachPickerMode(mode);
    setAttachTargetId(messageId);
    setAttachPickerOpen(true);
  };

  const handleAttachConfirm = async (mediaFileIds: string[]): Promise<void> => {
    if (attachPickerMode === 'compose') {
      const picked = mediaFiles.filter((file) => mediaFileIds.includes(file.id));
      setPendingAttachments(picked);
      return;
    }
    if (attachTargetId === null || provider.addAttachmentsAsync === undefined) {
      throw new Error('目前的資料模式不提供新增附件。');
    }
    await attachments.addAttachments(attachTargetId, mediaFileIds);
    notify({ message: '已新增附件。', tone: 'success' });
  };

  const handleForward = async (targetConversationId: string): Promise<void> => {
    if (forwardFor === null || provider.forwardMessageAsync === undefined) {
      throw new Error('目前的資料模式不提供轉傳。');
    }
    await provider.forwardMessageAsync(forwardFor, targetConversationId);
    // 轉傳訊息落在目標對話：若正在看目標對話，重載歷史才會
    // 出現（無即時推播）；否則只報成功。
    if (targetConversationId === selectedId) {
      await reloadAll();
    }
    notify({ message: '已轉傳訊息。', tone: 'success' });
  };

  // 自己訊息的操作（編輯／收回／僅自己刪除，§2u）：以伺服器回應
  // 更新本機歷史（不本地拼裝），並重載清單取權威預覽與未讀數；
  // 他人變更無即時推播，須重載才會出現（見 §2m）。
  const ownActionsSupported =
    provider.editMessageAsync !== undefined &&
    provider.recallMessageAsync !== undefined &&
    provider.deleteMessageAsync !== undefined;

  const applyMessageUpdate = (updated: MessageItem): void => {
    if (selectedId === null) {
      return;
    }
    const conversationId = selectedId;
    setHistories((prev) => {
      const current = prev[conversationId];
      if (current === undefined) {
        return prev;
      }
      return {
        ...prev,
        [conversationId]: {
          ...current,
          messages: current.messages.map((entry) => (entry.id === updated.id ? updated : entry)),
        },
      };
    });
  };

  const handleEditMessage = async (messageId: string, content: string): Promise<void> => {
    if (provider.editMessageAsync === undefined) {
      throw new Error('目前的資料模式不提供編輯訊息。');
    }
    const updated = await provider.editMessageAsync(messageId, content);
    applyMessageUpdate(updated);
    void loadConversations();
  };

  const handleRecallMessage = async (messageId: string): Promise<void> => {
    if (provider.recallMessageAsync === undefined) {
      throw new Error('目前的資料模式不提供收回訊息。');
    }
    const updated = await provider.recallMessageAsync(messageId);
    applyMessageUpdate(updated);
    void loadConversations();
  };

  const handleDeleteMessage = async (messageId: string): Promise<void> => {
    if (provider.deleteMessageAsync === undefined) {
      throw new Error('目前的資料模式不提供刪除訊息。');
    }
    const updated = await provider.deleteMessageAsync(messageId);
    applyMessageUpdate(updated);
    void loadConversations();
  };

  // 訊息搜尋（§2v）：面板共用 MessageSearchPanel；跳轉時開啟對話
  // 並以 sinceSeq 錨定載入該訊息前後一頁（與最新一頁合併），再
  // 醒目提示。錨定載入失敗時只報提示，對話仍以最新歷史開啟。
  const handleSearchMessages = async (input: MessageSearchInput): Promise<MessageSearchPage> => {
    if (provider.searchMessagesAsync === undefined) {
      throw new Error('目前的資料模式不提供訊息搜尋。');
    }
    return provider.searchMessagesAsync(input);
  };

  const jumpToSearchResult = (result: MessageSearchResultItem): void => {
    highlightForRef.current = result.conversationId;
    setHighlightMessageId(result.id);
    setSearchOpen(false);
    onSelect(result.conversationId);
    const listMessages = provider.listMessagesAsync;
    if (listMessages === undefined) {
      return;
    }
    void listMessages(result.conversationId, {
      sinceSeq: result.sequenceNumber - 1,
      limit: HISTORY_PAGE_LIMIT,
    })
      .then((page) => {
        if (!mountedRef.current) {
          return;
        }
        setHistories((prev) => {
          const current = prev[result.conversationId];
          return {
            ...prev,
            [result.conversationId]: {
              messages: mergeMessages(current?.messages ?? [], page.messages),
              nextCursor: page.nextCursor,
              hasMore: page.nextCursor !== '' && page.messages.length > 0,
            },
          };
        });
        setHistoryState('ready');
      })
      .catch(() => {
        notify({
          title: '無法跳至該訊息',
          message: '已開啟對話並載入最新訊息，請用「載入更早的訊息」尋找。',
          tone: 'warning',
        });
      });
  };

  const searchConversationOptions = useMemo<MessageSearchOption[]>(
    () =>
      conversations.map((conversation) => ({
        value: conversation.id,
        label: resolveConversationTitle(conversation, [], friendsById, myId),
      })),
    [conversations, friendsById, myId],
  );

  const searchSenderOptions = useMemo<MessageSearchOption[]>(() => {
    const options: MessageSearchOption[] = [];
    if (myId !== null) {
      options.push({ value: myId, label: '我' });
    }
    for (const friend of friends) {
      if (!options.some((option) => option.value === friend.id)) {
        options.push({ value: friend.id, label: friend.displayName });
      }
    }
    return options;
  }, [friends, myId]);

  const forwardTargets = useMemo<ForwardTarget[]>(
    () =>
      conversations.map((conversation) => ({
        id: conversation.id,
        title: resolveConversationTitle(conversation, [], friendsById, myId),
      })),
    [conversations, friendsById, myId],
  );

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  // 搜尋跳轉的醒目提示（§2w）：手動切換對話時清除，跳轉時保留；
  // 歷史就緒後將目標訊息捲入視野（目標不在已載入歷史時不捲動）。
  useEffect(() => {
    if (selectedId === null || highlightForRef.current !== selectedId) {
      if (highlightMessageId !== null) {
        setHighlightMessageId(null);
      }
      return;
    }
    if (highlightMessageId === null || historyState !== 'ready') {
      return;
    }
    const target = messagesRef.current?.querySelector(`[data-message-id="${highlightMessageId}"]`);
    target?.scrollIntoView({ block: 'center' });
  }, [selectedId, highlightMessageId, historyState]);

  const loadConversations = useCallback(async (): Promise<void> => {
    const list = provider.listConversationsAsync;
    if (list === undefined) {
      if (mountedRef.current) {
        setListState('unsupported');
      }
      return;
    }
    if (conversationsRef.current.length === 0 && mountedRef.current) {
      setListState('loading');
    }
    try {
      const listFriends = provider.listFriendsAsync;
      const [fetched, fetchedFriends] = await Promise.all([
        list(),
        listFriends === undefined
          ? Promise.resolve<FriendItem[]>([])
          : listFriends(true).catch(() => [] as FriendItem[]),
      ]);
      if (!mountedRef.current) {
        return;
      }
      setConversations(fetched);
      setFriends(fetchedFriends);
      setListState('ready');
      setListStale(false);
    } catch {
      if (!mountedRef.current) {
        return;
      }
      if (conversationsRef.current.length === 0) {
        setListState('error');
      } else {
        // 重新載入失敗：保留既有清單，只標示內容已過時。
        setListStale(true);
      }
    }
  }, [provider]);

  useEffect(() => {
    void loadConversations();
  }, [loadConversations]);

  useEffect(() => {
    if (onUnreadChange === undefined) {
      return;
    }
    onUnreadChange(conversations.reduce((sum, conversation) => sum + conversation.unreadCount, 0));
  }, [conversations, onUnreadChange]);

  const loadHistory = useCallback(
    async (conversationId: string): Promise<void> => {
      const listMessages = provider.listMessagesAsync;
      if (listMessages === undefined) {
        if (mountedRef.current) {
          setHistoryState('unsupported');
        }
        return;
      }
      const requestId = ++historyRequestRef.current;
      if (mountedRef.current) {
        setHistoryState('loading');
      }
      try {
        const page = await listMessages(conversationId, { limit: HISTORY_PAGE_LIMIT });
        if (!mountedRef.current || requestId !== historyRequestRef.current) {
          return;
        }
        setHistories((prev) => ({
          ...prev,
          [conversationId]: {
            // 與既有快取合併：載入期間發送的訊息不得被回應覆蓋。
            messages: mergeMessages(prev[conversationId]?.messages ?? [], page.messages),
            nextCursor: page.nextCursor,
            hasMore: page.nextCursor !== '' && page.messages.length > 0,
          },
        }));
        setHistoryState('ready');
      } catch {
        if (!mountedRef.current || requestId !== historyRequestRef.current) {
          return;
        }
        setHistoryState('error');
      }
    },
    [provider],
  );

  useEffect(() => {
    if (selectedId === null) {
      setHistoryState('idle');
      return;
    }
    if (historiesRef.current[selectedId] !== undefined) {
      setHistoryState('ready');
      return;
    }
    void loadHistory(selectedId);
  }, [provider, selectedId, historyEpoch, loadHistory]);

  // 已讀回條：歷史載入完成且對話仍有未讀數時，送一則 read 回條；
  // 成功後重載清單取權威未讀數（不本地扣減，App 徽章經既有
  // onUnreadChange 同步）。失敗時靜默——不以通知干擾閱讀，未讀數
  // 維持後端舊值，下次開啟對話或手動「重新載入」時重試（失敗不記
  // 入去重集合）；401 由 provider 的 failUnauthorized 集中處理。
  // 載入更早訊息不觸發：舊訊息序號較小，水位已涵蓋。
  useEffect(() => {
    if (selectedId === null || historyState !== 'ready') {
      return;
    }
    const entry = histories[selectedId];
    const conversation = conversations.find((item) => item.id === selectedId);
    if (entry === undefined || conversation === undefined || conversation.unreadCount === 0) {
      return;
    }
    const postReceipt = provider.postReceiptAsync;
    if (postReceipt === undefined) {
      return;
    }
    const targetId = selectReadReceiptTarget(entry.messages, myId);
    if (targetId === null) {
      return;
    }
    const target = entry.messages.find((message) => message.id === targetId);
    if (target === undefined) {
      return;
    }
    const key = `${selectedId}:${target.sequenceNumber}`;
    if (receiptedRef.current.has(key)) {
      return;
    }
    let cancelled = false;
    void postReceipt(targetId, 'read')
      .then(() => {
        if (cancelled || !mountedRef.current) {
          return;
        }
        receiptedRef.current.add(key);
        void loadConversations();
      })
      .catch(() => {
        // 靜默失敗，見上方註解。
      });
    return () => {
      cancelled = true;
    };
  }, [provider, selectedId, historyState, histories, conversations, myId, loadConversations]);

  const loadMore = useCallback(async (): Promise<void> => {
    if (selectedId === null || loadingMore) {
      return;
    }
    const entry = historiesRef.current[selectedId];
    const listMessages = provider.listMessagesAsync;
    if (entry === undefined || !entry.hasMore || listMessages === undefined) {
      return;
    }
    const sequences = entry.messages.map((message) => message.sequenceNumber);
    if (sequences.length === 0) {
      return;
    }
    const cursor = Number(entry.nextCursor);
    const beforeSeq = Number.isFinite(cursor) && cursor > 0 ? cursor : Math.min(...sequences);
    setLoadingMore(true);
    try {
      const page = await listMessages(selectedId, {
        beforeSeq,
        limit: HISTORY_PAGE_LIMIT,
      });
      if (!mountedRef.current) {
        return;
      }
      suppressScrollRef.current = true;
      setHistories((prev) => {
        const current = prev[selectedId];
        if (current === undefined) {
          return prev;
        }
        return {
          ...prev,
          [selectedId]: {
            messages: mergeMessages(current.messages, page.messages),
            nextCursor: page.nextCursor,
            hasMore: page.nextCursor !== '' && page.messages.length > 0,
          },
        };
      });
    } catch {
      if (mountedRef.current) {
        notify({
          title: '載入失敗',
          message: '無法載入更早的訊息，請再試一次。',
          tone: 'error',
        });
      }
    } finally {
      if (mountedRef.current) {
        setLoadingMore(false);
      }
    }
  }, [provider, selectedId, loadingMore, notify]);

  const reloadAll = useCallback(async (): Promise<void> => {
    setHistories({});
    reactions.refreshAll();
    attachments.refreshAll();
    setHistoryEpoch((epoch) => epoch + 1);
    await loadConversations();
  }, [loadConversations, reactions, attachments]);

  const visible = useMemo(() => {
    const needle = query.trim().toLocaleLowerCase();
    return conversations.filter((conversation) => {
      if (filter === 'unread' && conversation.unreadCount === 0) {
        return false;
      }
      if (needle === '') {
        return true;
      }
      return resolveConversationTitle(conversation, [], friendsById, myId)
        .toLocaleLowerCase()
        .includes(needle);
    });
  }, [conversations, query, filter, friendsById, myId]);

  useEffect(() => {
    if (!isMobile && selectedId === null && visible.length > 0 && listState === 'ready') {
      const first = visible[0];
      if (first !== undefined) {
        onSelect(first.id);
      }
    }
  }, [isMobile, selectedId, visible, listState, onSelect]);

  const selected = conversations.find((conversation) => conversation.id === selectedId);
  const entry = selectedId === null ? undefined : histories[selectedId];
  const displayMessages = useMemo(() => {
    if (entry === undefined) {
      return [];
    }
    return entry.messages.map((item) => {
      const base = toChatMessage(item);
      if (item.senderId !== myId) {
        return base;
      }
      // 自己訊息的送達狀態：依回條列表與其他在線成員推導（§2r）；
      // 未達標或成員未知時維持「已傳送」，未知不顯示。
      const status = deriveReceiptStatus(receiptsById[item.id] ?? [], myId, otherMemberIds);
      return status === null ? base : { ...base, status };
    });
  }, [entry, myId, otherMemberIds, receiptsById]);
  const grouped = useMemo(() => groupMessagesByDay(displayMessages, now), [displayMessages, now]);
  const itemsById = useMemo(
    () => new Map((entry?.messages ?? []).map((item) => [item.id, item])),
    [entry],
  );
  const pinnedEntries = useMemo<PinnedMessageEntry[]>(() => {
    if (entry === undefined) {
      return [];
    }
    return reactions.pins.map((pin) => {
      const item = itemsById.get(pin.messageId);
      if (item === undefined) {
        return { pin, senderName: '', body: '置頂訊息尚未載入。' };
      }
      return {
        pin,
        senderName: senderLabel(item.senderId, myId, friendsById),
        body: toChatMessage(item).body,
      };
    });
  }, [reactions.pins, entry, itemsById, myId, friendsById]);
  const unreadTotal = conversations.reduce(
    (sum, conversation) => sum + conversation.unreadCount,
    0,
  );
  const title =
    selected === undefined ? '' : resolveConversationTitle(selected, members, friendsById, myId);

  useEffect(() => {
    const pane = messagesRef.current;
    if (pane === null) {
      return;
    }
    if (suppressScrollRef.current) {
      suppressScrollRef.current = false;
      return;
    }
    pane.scrollTop = pane.scrollHeight;
  }, [selectedId, displayMessages.length, historyState]);

  // 建對話人選：沿用清單載入時一併取回的好友（已含隱藏好友，挑選
  // 時排除）；未載入好友時對話框顯示空狀態，不得以使用者識別碼
  // 自由輸入（非好友建 DM 後端拒絕，見 CreateDirect）。
  const createCandidates = useMemo<CreateConversationCandidate[]>(
    () =>
      friends
        .filter((friend) => !friend.isHidden)
        .map((friend) => ({
          id: friend.id,
          displayName: friend.displayName,
          publicId: friend.publicId,
        })),
    [friends],
  );

  const handleCreateConversation = async (userId: string): Promise<void> => {
    const create = provider.createConversationAsync;
    if (create === undefined) {
      throw new Error('目前的資料模式不提供建立對話。');
    }
    // 既有 DM 後端回 200＋既有對話：直接選取，再重載清單取權威
    // 狀態（新建 201 時清單納入新對話；重載失敗時保留既有畫面，
    // 選取的對話仍可經歷史端點載入）。
    const created = await create({ type: 'direct', otherUserId: userId });
    onSelect(created.id);
    await loadConversations();
  };

  // 對話成員管理（§2x）：新增人選排除既有生效成員與自己；人選
  // 狀態沿用清單載入狀態（好友與清單一併取回）。
  const [addMembersOpen, setAddMembersOpen] = useState(false);
  const [leaveOpen, setLeaveOpen] = useState(false);

  const addMemberCandidates = useMemo<CreateConversationCandidate[]>(
    () =>
      friends
        .filter((friend) => !friend.isHidden)
        .filter((friend) => friend.id !== myId)
        .filter(
          (friend) =>
            !members.some((member) => member.leftAt === null && member.userId === friend.id),
        )
        .map((friend) => ({
          id: friend.id,
          displayName: friend.displayName,
          publicId: friend.publicId,
        })),
    [friends, members, myId],
  );

  const memberNames = useMemo(
    () => new Map(friends.map((friend) => [friend.id, friend.displayName])),
    [friends],
  );

  const handleAddMembers = async (memberIds: string[]): Promise<void> => {
    const addMembers = provider.addMembersAsync;
    if (addMembers === undefined || selectedId === null) {
      throw new Error('目前的資料模式不提供新增成員。');
    }
    await addMembers(selectedId, memberIds);
    conversationMembers.reload();
    notify({ message: `已新增 ${memberIds.length} 位成員。`, tone: 'success' });
  };

  const handleLeaveConversation = async (): Promise<void> => {
    const leave = provider.leaveConversationAsync;
    if (leave === undefined || selectedId === null) {
      throw new Error('目前的資料模式不提供離開對話。');
    }
    await leave(selectedId);
    onSelect(null);
    await loadConversations();
    notify({ message: '已離開對話。', tone: 'success' });
  };

  const handleSend = (body: string): void => {
    if (selectedId === null) {
      return;
    }
    const conversationId = selectedId;
    const send = provider.sendMessageAsync;
    if (send === undefined) {
      notify({
        title: '傳送失敗',
        message: '目前的資料模式不提供訊息傳送。',
        tone: 'error',
      });
      return;
    }
    const attachedIds = pendingAttachments.map((file) => file.id);
    void send(
      conversationId,
      body,
      attachedIds.length > 0 ? { attachmentFileIds: attachedIds } : {},
    )
      .then((sent) => {
        if (!mountedRef.current) {
          return;
        }
        setPendingAttachments([]);
        // 以伺服器回應為準：歷史追加回應訊息，清單更新最後活動時間。
        setHistories((prev) => {
          const current = prev[conversationId] ?? {
            messages: [],
            nextCursor: '',
            hasMore: false,
          };
          return {
            ...prev,
            [conversationId]: {
              ...current,
              messages: mergeMessages(current.messages, [sent]),
            },
          };
        });
        setConversations((prev) =>
          sortConversationsByUpdated(
            prev.map((conversation) =>
              conversation.id === conversationId
                ? {
                    ...conversation,
                    lastMessageAt: sent.createdAt,
                    updatedAt: sent.createdAt,
                  }
                : conversation,
            ),
          ),
        );
      })
      .catch((error: unknown) => {
        notify({
          title: '傳送失敗',
          message: error instanceof Error ? error.message : '訊息傳送失敗。',
          tone: 'error',
        });
      });
  };

  return (
    <div className={selectedId !== null ? 'et-chats et-chats--show-thread' : 'et-chats'}>
      <section className="et-chats__list" aria-label="對話列表">
        <div className="et-chats__list-tools">
          <SearchField value={query} onChange={setQuery} placeholder="搜尋對話" label="搜尋對話" />
          <Tabs
            label="對話篩選"
            value={filter}
            onChange={(id) => setFilter(id as ApiListFilter)}
            tabs={[
              { id: 'all', label: '全部' },
              {
                id: 'unread',
                label: '未讀',
                badge:
                  unreadTotal > 0 ? (
                    <CountBadge count={unreadTotal} label={`${unreadTotal} 則未讀`} />
                  ) : undefined,
              },
            ]}
          />
          <p className="et-chats__source">對話與訊息來自後端 API。</p>
        </div>
        {listStale ? (
          <p className="et-chats__empty">
            清單重新載入失敗，顯示上次載入的內容。
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={() => {
                void loadConversations();
              }}
            >
              重新載入
            </Button>
          </p>
        ) : null}
        {searchOpen ? (
          <MessageSearchPanel
            conversations={searchConversationOptions}
            senders={searchSenderOptions}
            supported={provider.searchMessagesAsync !== undefined}
            sourceNote="搜尋結果來自後端索引；索引填充迴圈尚未接線，結果可能不完整。"
            onSearch={handleSearchMessages}
            onJump={jumpToSearchResult}
            onClose={() => {
              setSearchOpen(false);
            }}
          />
        ) : (
          <div className="et-chats__items et-scroll" role="list" aria-label="對話">
            {listState === 'loading' ? (
              <p className="et-chats__empty">載入對話中…</p>
            ) : listState === 'error' ? (
              <div className="et-chats__empty">
                <p>載入對話失敗，請檢查連線後再試一次。</p>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => {
                    void loadConversations();
                  }}
                >
                  重新載入
                </Button>
              </div>
            ) : listState === 'unsupported' ? (
              <p className="et-chats__empty">目前的資料模式不提供對話清單。</p>
            ) : visible.length === 0 ? (
              <p className="et-chats__empty">
                {query !== '' || filter !== 'all'
                  ? '沒有符合條件的對話。試試其他關鍵字或篩選。'
                  : '目前沒有對話。'}
              </p>
            ) : (
              visible.map((conversation) => {
                const rowTitle = resolveConversationTitle(conversation, [], friendsById, myId);
                return (
                  <button
                    type="button"
                    key={conversation.id}
                    className={
                      conversation.id === selectedId
                        ? 'et-conv-item et-conv-item--active'
                        : 'et-conv-item'
                    }
                    aria-current={conversation.id === selectedId ? 'true' : undefined}
                    onClick={() => {
                      onSelect(conversation.id);
                    }}
                  >
                    <Avatar name={rowTitle} size="md" />
                    <span className="et-conv-item__main">
                      <span className="et-conv-item__top">
                        <span className="et-conv-item__name">{rowTitle}</span>
                        {conversation.lastMessageAt === null ? null : (
                          <time
                            className="et-conv-item__time"
                            dateTime={conversation.lastMessageAt}
                          >
                            {formatMessageTime(conversation.lastMessageAt, now)}
                          </time>
                        )}
                      </span>
                      <span className="et-conv-item__bottom">
                        <span className="et-conv-item__preview">
                          {conversation.lastMessageAt === null ? '尚無訊息' : ''}
                        </span>
                        <span className="et-conv-item__flags">
                          <CountBadge count={conversation.unreadCount} />
                        </span>
                      </span>
                    </span>
                  </button>
                );
              })
            )}
          </div>
        )}
        <div className="et-chats__list-tools">
          <button
            type="button"
            className="et-btn et-btn--secondary et-btn--md et-btn--full"
            onClick={() => {
              setSearchOpen((open) => !open);
            }}
            aria-expanded={searchOpen}
          >
            搜尋訊息
          </button>
          <button
            type="button"
            className="et-btn et-btn--secondary et-btn--md et-btn--full"
            onClick={() => {
              if (provider.createConversationAsync === undefined) {
                notify({
                  title: '尚未接線',
                  message: '目前的資料模式不提供建立對話。',
                  tone: 'warning',
                });
                return;
              }
              setCreateOpen(true);
            }}
          >
            <PlusIcon size={16} />
            新增對話
          </button>
          <button
            type="button"
            className="et-btn et-btn--secondary et-btn--md et-btn--full"
            onClick={() => {
              void reloadAll();
            }}
          >
            重新載入
          </button>
        </div>
      </section>

      <CreateConversationDialog
        open={createOpen}
        candidates={createCandidates}
        candidatesState={listState}
        onRetryCandidates={() => {
          void loadConversations();
        }}
        onClose={() => {
          setCreateOpen(false);
        }}
        onCreate={handleCreateConversation}
      />

      <AddMembersDialog
        open={addMembersOpen}
        candidates={addMemberCandidates}
        candidatesState={listState}
        onRetryCandidates={() => {
          void loadConversations();
        }}
        onClose={() => {
          setAddMembersOpen(false);
        }}
        onAdd={handleAddMembers}
      />

      {selected !== undefined ? (
        <LeaveConversationDialog
          open={leaveOpen}
          conversationTitle={title}
          conversationType={selected.type}
          onClose={() => {
            setLeaveOpen(false);
          }}
          onConfirm={handleLeaveConversation}
        />
      ) : null}

      <section className="et-chats__thread" aria-label="對話內容">
        {selected === undefined ? (
          <div className="et-thread__empty">
            <EmptyState
              title="選擇一個對話"
              description="從左側列表挑一個對話，開始閱讀與回覆訊息。"
            />
          </div>
        ) : (
          <>
            <header className="et-thread__header">
              <span className="et-thread__back">
                <IconButton
                  label="回到對話列表"
                  onClick={() => {
                    onSelect(null);
                  }}
                >
                  <BackIcon size={20} />
                </IconButton>
              </span>
              <span className="et-thread__identity">
                <Avatar name={title} size="md" />
                <span className="et-thread__identity-text">
                  <span className="et-thread__name">{title}</span>
                  <span className="et-thread__presence">
                    {selected.type === 'group' ? '群組對話' : '一對一對話'}
                  </span>
                </span>
              </span>
            </header>
            {provider.listMembersAsync !== undefined ? (
              <ConversationMembersBlock
                members={members}
                membersState={conversationMembers.state}
                onRetryMembers={conversationMembers.reload}
                memberNames={memberNames}
                myId={myId}
                canAddMembers={provider.addMembersAsync !== undefined}
                canLeave={provider.leaveConversationAsync !== undefined}
                onAddMembers={() => {
                  setAddMembersOpen(true);
                }}
                onLeave={() => {
                  setLeaveOpen(true);
                }}
              />
            ) : null}
            <PinnedMessagesBlock
              entries={pinnedEntries}
              state={reactions.pinsState}
              onRetry={reactions.reloadPins}
              onUnpin={(messageId) => {
                void reactions.togglePin(messageId);
              }}
            />
            <div className="et-thread__messages et-scroll" ref={messagesRef} aria-live="off">
              {historyState === 'loading' ? (
                <p className="et-chats__empty">載入訊息中…</p>
              ) : historyState === 'error' ? (
                <div className="et-chats__empty">
                  <p>載入訊息失敗，請檢查連線後再試一次。</p>
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    onClick={() => {
                      setHistoryEpoch((epoch) => epoch + 1);
                    }}
                  >
                    重新載入
                  </Button>
                </div>
              ) : historyState === 'unsupported' ? (
                <p className="et-chats__empty">目前的資料模式不提供訊息歷史。</p>
              ) : (
                <>
                  {entry !== undefined && entry.hasMore ? (
                    <div className="et-thread__load-more">
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        disabled={loadingMore}
                        onClick={() => {
                          void loadMore();
                        }}
                      >
                        {loadingMore ? '載入中…' : '載入更早的訊息'}
                      </Button>
                    </div>
                  ) : null}
                  {displayMessages.length === 0 ? (
                    <p className="et-chats__empty">還沒有訊息，傳送第一則訊息開始對話。</p>
                  ) : (
                    grouped.map((group) => (
                      <div key={group.label}>
                        <div className="et-day-divider">
                          <span className="et-day-divider__label">{group.label}</span>
                        </div>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
                          {group.messages.map((message) => {
                            const isOwn = myId !== null && message.senderId === myId;
                            const item = itemsById.get(message.id);
                            // 收回訊息與僅自己刪除的訊息不提供反應／置頂
                            // 操作（後端守衛，見 §2p）；彙總列出不在此限，
                            // 但收回訊息不顯示操作列。附件與轉傳（§2t）
                            // 同理：僅互動訊息顯示操作，附件清單另查。
                            const interactive =
                              item !== undefined &&
                              item.status !== 'recalled' &&
                              !item.deletedForSelf;
                            return (
                              <div
                                key={message.id}
                                data-message-id={message.id}
                                className={
                                  highlightMessageId === message.id
                                    ? 'et-message--jump-target'
                                    : undefined
                                }
                              >
                                <MessageBubble
                                  message={message}
                                  isOwn={isOwn}
                                  senderName={senderLabel(message.senderId, myId, friendsById)}
                                  showSenderName={!isOwn}
                                  now={now}
                                  actions={
                                    <>
                                      {attachments.supported ? (
                                        <MessageAttachmentBlock
                                          attachments={
                                            attachments.attachmentsById[message.id] ?? []
                                          }
                                          isForwarded={
                                            attachments.forwardsById[message.id] !== undefined
                                          }
                                          canAddAttachments={isOwn && interactive}
                                          onAddAttachments={() => {
                                            openAttachPicker('add', message.id);
                                          }}
                                          downloadSupported={downloadAttachment !== null}
                                          onDownloadAttachment={
                                            downloadAttachment === null
                                              ? undefined
                                              : downloadAttachment
                                          }
                                        />
                                      ) : null}
                                      {reactions.supported && interactive ? (
                                        <MessageReactionBar
                                          summary={reactions.summaries[message.id]}
                                          isPinned={reactions.pins.some(
                                            (pin) => pin.messageId === message.id,
                                          )}
                                          pickerOpen={reactions.pickerFor === message.id}
                                          onTogglePicker={() => {
                                            reactions.setPickerFor(
                                              reactions.pickerFor === message.id
                                                ? null
                                                : message.id,
                                            );
                                          }}
                                          onPickReaction={(emoji) => {
                                            void reactions.pickReaction(message.id, emoji);
                                          }}
                                          onRemoveReaction={() => {
                                            void reactions.removeReaction(message.id);
                                          }}
                                          onTogglePin={() => {
                                            void reactions.togglePin(message.id);
                                          }}
                                          onForward={
                                            attachments.supported
                                              ? () => {
                                                  setForwardFor(message.id);
                                                }
                                              : undefined
                                          }
                                        />
                                      ) : undefined}
                                      {isOwn && interactive ? (
                                        <MessageOwnActions
                                          messageId={message.id}
                                          content={item.content}
                                          edited={item.editCount > 0}
                                          supported={ownActionsSupported}
                                          onEdit={(content) =>
                                            handleEditMessage(message.id, content)
                                          }
                                          onRecall={() => handleRecallMessage(message.id)}
                                          onDelete={() => handleDeleteMessage(message.id)}
                                        />
                                      ) : null}
                                    </>
                                  }
                                />
                              </div>
                            );
                          })}
                        </div>
                      </div>
                    ))
                  )}
                </>
              )}
            </div>
            {pendingAttachments.length > 0 ? (
              <div className="et-pending-attachments" aria-label="已選擇的附件">
                <span className="et-pending-attachments__label">
                  已選擇 {pendingAttachments.length} 個附件（關聯既有檔案，不重新上傳）：
                </span>
                {pendingAttachments.map((file) => (
                  <span key={file.id} className="et-pending-attachments__chip">
                    {file.originalFilename}
                    <button
                      type="button"
                      aria-label={`移除附件 ${file.originalFilename}`}
                      onClick={() => {
                        setPendingAttachments((prev) => prev.filter((item) => item.id !== file.id));
                      }}
                    >
                      ×
                    </button>
                  </span>
                ))}
              </div>
            ) : null}
            <MessageComposer
              onSend={handleSend}
              onAttach={() => {
                openAttachPicker('compose', null);
              }}
              onEmoji={() => {
                notify({
                  title: '尚未接線',
                  message: '表情符號面板尚未接線：貼圖功能在 Phase 08 實作。',
                  tone: 'warning',
                });
              }}
            />
          </>
        )}
      </section>
      <ForwardDialog
        open={forwardFor !== null}
        targets={forwardTargets}
        onClose={() => {
          setForwardFor(null);
        }}
        onForward={handleForward}
      />
      <AttachFilesDialog
        open={attachPickerOpen}
        files={mediaFiles}
        filesState={mediaFilesState}
        onRetryFiles={loadMediaFiles}
        onClose={() => {
          setAttachPickerOpen(false);
        }}
        onConfirm={handleAttachConfirm}
        confirmLabel={attachPickerMode === 'compose' ? '選擇附件' : '新增附件'}
      />
    </div>
  );
}

export function ChatsView({
  provider,
  selectedId,
  onSelect,
  dataMode = 'demo',
  onUnreadChange,
}: ChatsViewProps): ReactElement {
  if (dataMode === 'api') {
    return (
      <ApiChatsView
        provider={provider}
        selectedId={selectedId}
        onSelect={onSelect}
        onUnreadChange={onUnreadChange}
      />
    );
  }
  return <DemoChatsView provider={provider} selectedId={selectedId} onSelect={onSelect} />;
}
