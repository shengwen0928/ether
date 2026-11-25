import { renderToString } from 'react-dom/server';
import { Avatar } from '../src/components/Avatar';
import { Badge, CountBadge } from '../src/components/Badge';
import { Button } from '../src/components/Button';
import { ConversationItem } from '../src/components/ConversationItem';
import { Dialog } from '../src/components/Dialog';
import { EmptyState } from '../src/components/EmptyState';
import { IconButton } from '../src/components/IconButton';
import { SearchIcon } from '../src/components/icons';
import { Input } from '../src/components/Input';
import { MessageBubble } from '../src/components/MessageBubble';
import { Tabs } from '../src/components/Tabs';
import { ToastProvider } from '../src/components/Toast';
import { TypingIndicator } from '../src/components/TypingIndicator';
import { createDemoProvider } from '../src/data/provider';
import { ThemeProvider } from '../src/theme/theme';
import { AuthView } from '../src/views/AuthView';
import {
  ChatsView,
  AddMembersDialog,
  ConversationMembersBlock,
  CreateConversationDialog,
  deriveReceiptStatus,
  filterCreateCandidates,
  filterForwardTargets,
  formatAttachmentSize,
  ForwardDialog,
  AttachFilesDialog,
  LeaveConversationDialog,
  MessageAttachmentBlock,
  MessageReactionBar,
  MessageOwnActions,
  MessageSearchPanel,
  PinnedMessagesBlock,
  selectDeliveredReceiptTargets,
  selectReadReceiptTarget,
  validateEditContent,
} from '../src/views/ChatsView';
import { ContactsView } from '../src/views/ContactsView';
import { GroupsView } from '../src/views/GroupsView';
import { MediaView } from '../src/views/MediaView';
import { ProfileView } from '../src/views/ProfileView';
import { ShowcaseView } from '../src/views/ShowcaseView';
import {
  PostCard,
  PostMediaPickerDialog,
  TimelineView,
  validatePostComment,
  validatePostContent,
  visibilityLabel,
} from '../src/views/TimelineView';
import {
  CallCard,
  CallDetailPanel,
  CallsView,
  InitiateCallDialog,
  callActionState,
  callEventLabel,
  callStatusLabel,
  callTypeLabel,
  elapsedSeconds,
  endReasonLabel,
  formatCallDuration,
  isCallTerminal,
  participantResultLabel,
} from '../src/views/CallsView';
import {
  AddStickerDialog,
  StickerMediaPickerDialog,
  StickerPackCard,
  StickerPackDetailPanel,
  StickerPackFormDialog,
  StickerTile,
  StickersView,
  canManageStickerPack,
  stickerPackCategoryLabel,
  stickerPackStatusLabel,
  validateStickerName,
  validateStickerPackDescription,
  validateStickerPackName,
} from '../src/views/StickersView';
import type { ChatMessage } from '../src/data/types';

const noop = (): void => undefined;

export function renderButton(): string {
  return renderToString(
    <Button variant="danger" loading>
      刪除
    </Button>,
  );
}

export function renderIconButton(): string {
  return renderToString(
    <IconButton label="搜尋">
      <SearchIcon size={20} />
    </IconButton>,
  );
}

export function renderInputError(): string {
  return renderToString(<Input label="Email" defaultValue="bad" error="Email 格式不正確。" />);
}

export function renderAvatar(): string {
  return renderToString(<Avatar name="林曉彤" size="md" presence="online" />);
}

export function renderBadges(): string {
  return renderToString(
    <span>
      <Badge tone="accent">最愛</Badge>
      <CountBadge count={128} />
    </span>,
  );
}

export function renderDialogOpen(): string {
  return renderToString(
    <Dialog open onClose={noop} title="展示對話框" description="說明文字">
      <p>內容</p>
    </Dialog>,
  );
}

export function renderDialogClosed(): string {
  return renderToString(
    <Dialog open={false} onClose={noop} title="展示對話框">
      <p>內容</p>
    </Dialog>,
  );
}

export function renderTabs(): string {
  return renderToString(
    <Tabs
      label="展示頁籤"
      value="b"
      onChange={noop}
      tabs={[
        { id: 'a', label: '甲' },
        { id: 'b', label: '乙' },
      ]}
    />,
  );
}

const SAMPLE_MESSAGE: ChatMessage = {
  id: 't-1',
  conversationId: 'v-hao',
  senderId: 'me',
  body: '測試訊息',
  sentAt: '2026-11-25T12:00:00+08:00',
  status: 'read',
  kind: 'text',
};

export function renderMessageBubble(): string {
  return renderToString(
    <MessageBubble
      message={SAMPLE_MESSAGE}
      isOwn
      senderName="你"
      showSenderName={false}
      now={new Date('2026-11-25T13:00:00+08:00')}
    />,
  );
}

export function renderConversationItem(): string {
  const provider = createDemoProvider();
  const snapshot = provider.getSnapshot();
  // v-mei is not typing, so the last-message preview is visible.
  const conversation = snapshot.conversations.find((item) => item.id === 'v-mei');
  const contact = snapshot.contacts.find((item) => item.id === conversation?.contactId);
  if (conversation === undefined || contact === undefined) {
    return '';
  }
  return renderToString(
    <ConversationItem
      conversation={conversation}
      contact={contact}
      active
      now={new Date('2026-11-25T13:00:00+08:00')}
      onSelect={noop}
    />,
  );
}

export function renderTyping(): string {
  return renderToString(<TypingIndicator name="陳玫" />);
}

export function renderEmptyState(): string {
  return renderToString(<EmptyState title="選擇一個對話" description="從列表挑一位聯絡人。" />);
}

export function renderChats(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <ChatsView provider={provider} selectedId="v-hao" onSelect={noop} />
    </ToastProvider>,
  );
}

export function renderChatsApi(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <ChatsView provider={provider} dataMode="api" selectedId={null} onSelect={noop} />
    </ToastProvider>,
  );
}

export function renderMessageSearchPanel(): string {
  return renderToString(
    <MessageSearchPanel
      conversations={[{ value: 'v-hao', label: '陳建豪' }]}
      senders={[
        { value: 'me', label: '我' },
        { value: 'c-hao', label: '陳建豪' },
      ]}
      supported
      sourceNote="搜尋結果來自展示資料（本機比對）。"
      onSearch={async () => ({ results: [], nextCursor: '' })}
      onJump={noop}
      onClose={noop}
    />,
  );
}

export function renderMessageSearchPanelUnsupported(): string {
  return renderToString(
    <MessageSearchPanel
      conversations={[]}
      senders={[]}
      supported={false}
      sourceNote=""
      onSearch={async () => ({ results: [], nextCursor: '' })}
      onJump={noop}
      onClose={noop}
    />,
  );
}

export {
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
};

export function renderCreateConversationDialog(): string {
  return renderToString(
    <ToastProvider>
      <CreateConversationDialog
        open
        candidates={[
          { id: 'u-hao', displayName: '王梓豪', publicId: 'zihao' },
          { id: 'u-mei', displayName: '陳玫', publicId: 'mei' },
        ]}
        candidatesState="ready"
        onRetryCandidates={noop}
        onClose={noop}
        onCreate={async () => undefined}
      />
    </ToastProvider>,
  );
}

export function renderReactionBar(): string {
  return renderToString(
    <MessageReactionBar
      summary={{
        messageId: 'm-1',
        reactions: [
          { reaction: '👍', count: 2, reactedByMe: true },
          { reaction: '❤️', count: 1, reactedByMe: false },
        ],
      }}
      isPinned={false}
      pickerOpen
      onTogglePicker={noop}
      onPickReaction={noop}
      onRemoveReaction={noop}
      onTogglePin={noop}
      onForward={noop}
    />,
  );
}

export function renderPinsBlock(): string {
  return renderToString(
    <PinnedMessagesBlock
      entries={[
        {
          pin: {
            id: 'p-1',
            conversationId: 'c-1',
            messageId: 'm-1',
            pinnedBy: 'u-1',
            pinnedAt: '2026-11-25T00:02:00Z',
            unpinnedAt: null,
          },
          senderName: '陳玫',
          body: '週六下午三點見。',
        },
      ]}
      state="ready"
      onRetry={noop}
      onUnpin={noop}
    />,
  );
}

export function renderAttachmentBlock(): string {
  return renderToString(
    <MessageAttachmentBlock
      attachments={[
        {
          id: 'a-1',
          messageId: 'm-1',
          mediaFileId: 'f-1',
          mimeType: 'image/jpeg',
          sizeBytes: 184320,
          fileName: '陽明山雲海.jpg',
          createdAt: '2026-11-25T00:03:00Z',
        },
      ]}
      isForwarded
      canAddAttachments
      onAddAttachments={noop}
      downloadSupported
      onDownloadAttachment={noop}
    />,
  );
}

export function renderAttachmentBlockNoDownload(): string {
  return renderToString(
    <MessageAttachmentBlock
      attachments={[
        {
          id: 'a-1',
          messageId: 'm-1',
          mediaFileId: 'f-1',
          mimeType: 'image/jpeg',
          sizeBytes: 184320,
          fileName: '陽明山雲海.jpg',
          createdAt: '2026-11-25T00:03:00Z',
        },
      ]}
      isForwarded
      canAddAttachments
      onAddAttachments={noop}
      downloadSupported={false}
    />,
  );
}

export function renderMembersBlock(): string {
  return renderToString(
    <ConversationMembersBlock
      members={[
        {
          userId: 'me',
          role: 'admin',
          joinedAt: '2026-11-25T00:00:00Z',
          leftAt: null,
          lastReadSequence: 0,
        },
        {
          userId: 'u-2',
          role: 'member',
          joinedAt: '2026-11-25T00:00:00Z',
          leftAt: null,
          lastReadSequence: 0,
        },
        {
          userId: 'u-3',
          role: 'member',
          joinedAt: '2026-11-25T00:00:00Z',
          leftAt: '2026-11-25T01:00:00Z',
          lastReadSequence: 0,
        },
      ]}
      membersState="ready"
      onRetryMembers={noop}
      memberNames={
        new Map([
          ['u-2', '陳玫'],
          ['u-3', '王梓豪'],
        ])
      }
      myId="me"
      canAddMembers
      canLeave
      onAddMembers={noop}
      onLeave={noop}
    />,
  );
}

export function renderMembersBlockUnsupported(): string {
  return renderToString(
    <ConversationMembersBlock
      members={[]}
      membersState="unsupported"
      onRetryMembers={noop}
      memberNames={new Map()}
      myId="me"
      canAddMembers={false}
      canLeave={false}
      onAddMembers={noop}
      onLeave={noop}
    />,
  );
}

export function renderAddMembersDialog(): string {
  return renderToString(
    <ToastProvider>
      <AddMembersDialog
        open
        candidates={[
          { id: 'u-2', displayName: '陳玫', publicId: 'mei' },
          { id: 'u-3', displayName: '王梓豪', publicId: 'zihao' },
        ]}
        candidatesState="ready"
        onRetryCandidates={noop}
        onClose={noop}
        onAdd={async () => undefined}
      />
    </ToastProvider>,
  );
}

export function renderLeaveConversationDialog(): string {
  return renderToString(
    <ToastProvider>
      <LeaveConversationDialog
        open
        conversationTitle="讀書會"
        conversationType="group"
        onClose={noop}
        onConfirm={async () => undefined}
      />
    </ToastProvider>,
  );
}

export function renderForwardDialog(): string {
  return renderToString(
    <ToastProvider>
      <ForwardDialog
        open
        targets={[
          { id: 'c-1', title: '陳玫' },
          { id: 'c-2', title: '王梓豪' },
        ]}
        onClose={noop}
        onForward={async () => undefined}
      />
    </ToastProvider>,
  );
}

export function renderMessageOwnActions(): string {
  return renderToString(
    <ToastProvider>
      <MessageOwnActions
        messageId="m-1"
        content="你好"
        edited
        supported
        onEdit={async () => undefined}
        onRecall={async () => undefined}
        onDelete={async () => undefined}
      />
    </ToastProvider>,
  );
}

export function renderMessageOwnActionsUnsupported(): string {
  return renderToString(
    <ToastProvider>
      <MessageOwnActions
        messageId="m-1"
        content="你好"
        edited={false}
        supported={false}
        onEdit={async () => undefined}
        onRecall={async () => undefined}
        onDelete={async () => undefined}
      />
    </ToastProvider>,
  );
}

export function renderAttachFilesDialog(): string {
  return renderToString(
    <ToastProvider>
      <AttachFilesDialog
        open
        files={[
          {
            id: 'f-1',
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
            id: 'f-2',
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
        ]}
        filesState="ready"
        onRetryFiles={noop}
        onClose={noop}
        onConfirm={async () => undefined}
        confirmLabel="選擇附件"
      />
    </ToastProvider>,
  );
}

export function renderAuth(): string {
  return renderToString(
    <ToastProvider>
      <AuthView onAuthenticated={noop} />
    </ToastProvider>,
  );
}

export function renderContacts(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <ContactsView provider={provider} onStartConversation={noop} />
    </ToastProvider>,
  );
}

export function renderContactsApi(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <ContactsView provider={provider} dataMode="api" onStartConversation={noop} />
    </ToastProvider>,
  );
}

export function renderGroups(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <GroupsView provider={provider} />
    </ToastProvider>,
  );
}

export function renderGroupsApi(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <GroupsView provider={provider} dataMode="api" />
    </ToastProvider>,
  );
}

export function renderMedia(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <MediaView provider={provider} />
    </ToastProvider>,
  );
}

export function renderMediaApi(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <MediaView provider={provider} dataMode="api" />
    </ToastProvider>,
  );
}

export function renderProfile(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ThemeProvider>
      <ToastProvider>
        <ProfileView provider={provider} onSignOut={noop} />
      </ToastProvider>
    </ThemeProvider>,
  );
}

export function renderShowcase(): string {
  return renderToString(
    <ToastProvider>
      <ShowcaseView />
    </ToastProvider>,
  );
}

export function renderTimeline(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <TimelineView provider={provider} />
    </ToastProvider>,
  );
}

export function renderTimelineApi(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <TimelineView provider={provider} dataMode="api" />
    </ToastProvider>,
  );
}

export function renderPostCard(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <PostCard
        post={{
          id: 'demo-post-1',
          authorId: 'c-mei',
          content: '週六下午三點，媽媽家吃飯，別遲到。',
          visibility: 'public',
          editedAt: null,
          media: [{ mediaFileId: 'demo-file-photo', position: 0, caption: '' }],
          likeCount: 2,
          commentCount: 3,
          likedByMe: false,
          createdAt: '2026-11-25T10:30:00+08:00',
          updatedAt: '2026-11-25T10:30:00+08:00',
        }}
        myId="me"
        authorName="陳玫"
        mediaById={new Map()}
        nameOf={(userId) => userId}
        provider={provider}
        onChanged={noop}
        onRemoved={noop}
      />
    </ToastProvider>,
  );
}

export function renderPostMediaPicker(): string {
  return renderToString(
    <ToastProvider>
      <PostMediaPickerDialog
        open
        files={[
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
        ]}
        filesState="ready"
        selectedIds={[]}
        onRetryFiles={noop}
        onClose={noop}
        onConfirm={noop}
      />
    </ToastProvider>,
  );
}

export function renderCalls(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <CallsView provider={provider} />
    </ToastProvider>,
  );
}

export function renderCallsApi(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <CallsView provider={provider} dataMode="api" />
    </ToastProvider>,
  );
}

export function renderCallCard(): string {
  return renderToString(
    <ToastProvider>
      <CallCard
        call={{
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
        }}
        counterpartName="王梓豪"
        initiatorName="林曉彤"
        selected={false}
        onSelect={noop}
      />
    </ToastProvider>,
  );
}

export function renderCallDetailRinging(): string {
  return renderToString(
    <ToastProvider>
      <CallDetailPanel
        call={{
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
        }}
        participants={[
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
        ]}
        participantsState="ready"
        events={[
          {
            id: 'demo-call-event-11',
            userId: 'c-mei',
            eventType: 'initiated',
            occurredAt: '2026-11-25T12:50:00+08:00',
          },
        ]}
        eventsState="ready"
        myId="me"
        nameOf={(userId) => (userId === 'me' ? '林曉彤' : userId === 'c-mei' ? '陳玫' : userId)}
        counterpartName="陳玫"
        acting={false}
        nowMs={Date.parse('2026-11-25T12:51:00+08:00')}
        onAnswer={noop}
        onDecline={noop}
        onBusy={noop}
        onEnd={noop}
        onReload={noop}
        onClose={noop}
      />
    </ToastProvider>,
  );
}

export function renderCallDetailEnded(): string {
  return renderToString(
    <ToastProvider>
      <CallDetailPanel
        call={{
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
        }}
        participants={[
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
        ]}
        participantsState="ready"
        events={[
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
        ]}
        eventsState="ready"
        myId="me"
        nameOf={(userId) => (userId === 'me' ? '林曉彤' : userId === 'c-hao' ? '王梓豪' : userId)}
        counterpartName="王梓豪"
        acting={false}
        nowMs={Date.parse('2026-11-25T11:15:00+08:00')}
        onAnswer={noop}
        onDecline={noop}
        onBusy={noop}
        onEnd={noop}
        onReload={noop}
        onClose={noop}
      />
    </ToastProvider>,
  );
}

export function renderInitiateCallDialog(): string {
  return renderToString(
    <ToastProvider>
      <InitiateCallDialog
        open
        conversations={[
          {
            id: 'v-mei',
            type: 'direct',
            title: '陳玫',
            createdBy: 'me',
            lastMessageId: '',
            lastMessageAt: null,
            unreadCount: 0,
            createdAt: '2026-11-25T11:05:00+08:00',
            updatedAt: '2026-11-25T11:05:00+08:00',
          },
        ]}
        groups={[
          {
            id: 'demo-group-hiking',
            name: '週末爬山小隊',
            description: '',
            avatarMediaId: '',
            conversationId: '',
            createdBy: 'me',
            isDisbanded: false,
            memberCount: 4,
            createdAt: '2026-11-20T09:00:00+08:00',
            updatedAt: '2026-11-24T18:30:00+08:00',
          },
        ]}
        candidatesState="ready"
        onRetryCandidates={noop}
        onClose={noop}
        onConfirm={async () => undefined}
      />
    </ToastProvider>,
  );
}

export {
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
};

export function renderStickers(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <StickersView provider={provider} />
    </ToastProvider>,
  );
}

export function renderStickersApi(): string {
  const provider = createDemoProvider();
  return renderToString(
    <ToastProvider>
      <StickersView provider={provider} dataMode="api" />
    </ToastProvider>,
  );
}

export function renderStickerPackCard(): string {
  return renderToString(
    <ToastProvider>
      <StickerPackCard
        pack={{
          id: 'demo-pack-3',
          creatorId: 'c-mei',
          name: '山間小日子（付費）',
          description: '付費貼圖包，已授權發放才看得到。',
          category: 'general',
          coverMediaId: '',
          status: 'published',
          isFree: false,
          stickerCount: 4,
          favoritedByMe: true,
          createdAt: '2026-11-20T09:00:00+08:00',
          updatedAt: '2026-11-24T18:30:00+08:00',
        }}
        creatorName="陳玫"
        selected={false}
        favoriteBusy={false}
        onSelect={noop}
        onToggleFavorite={noop}
      />
    </ToastProvider>,
  );
}

export function renderStickerPackDetail(): string {
  return renderToString(
    <ToastProvider>
      <StickerPackDetailPanel
        pack={{
          id: 'demo-pack-1',
          creatorId: 'me',
          name: '山間小日子',
          description: '作者本人的貼圖包。',
          category: 'general',
          coverMediaId: '',
          status: 'published',
          isFree: true,
          stickerCount: 2,
          favoritedByMe: true,
          createdAt: '2026-11-20T09:00:00+08:00',
          updatedAt: '2026-11-24T18:30:00+08:00',
        }}
        stickers={[
          {
            id: 'demo-sticker-1',
            packId: 'demo-pack-1',
            packName: '山間小日子',
            mediaFileId: 'demo-media-1',
            name: '咖啡時間',
            position: 0,
            isAnimated: false,
            status: 'active',
            createdAt: '2026-11-20T09:05:00+08:00',
            updatedAt: '2026-11-20T09:05:00+08:00',
          },
          {
            id: 'demo-sticker-2',
            packId: 'demo-pack-1',
            packName: '山間小日子',
            mediaFileId: 'demo-media-2',
            name: '山徑漫步',
            position: 1,
            isAnimated: true,
            status: 'active',
            createdAt: '2026-11-20T09:06:00+08:00',
            updatedAt: '2026-11-20T09:06:00+08:00',
          },
        ]}
        stickersState="ready"
        creatorName="林曉彤"
        myId="me"
        acting={false}
        onToggleFavorite={noop}
        onEdit={noop}
        onDelete={noop}
        onAddSticker={noop}
        onRemoveSticker={noop}
        onRecordUse={noop}
        onReload={noop}
        onClose={noop}
      />
    </ToastProvider>,
  );
}

export function renderStickerTile(): string {
  return renderToString(
    <ToastProvider>
      <ul>
        <StickerTile
          sticker={{
            id: 'demo-sticker-1',
            packId: 'demo-pack-1',
            packName: '山間小日子',
            mediaFileId: 'demo-media-1',
            name: '咖啡時間',
            position: 0,
            isAnimated: false,
            status: 'active',
            createdAt: '2026-11-20T09:05:00+08:00',
            updatedAt: '2026-11-20T09:05:00+08:00',
          }}
          onRecordUse={noop}
        />
      </ul>
    </ToastProvider>,
  );
}

export function renderStickerPackFormCreate(): string {
  return renderToString(
    <ToastProvider>
      <StickerPackFormDialog
        open
        initial={null}
        files={[]}
        filesState="ready"
        onRetryFiles={noop}
        onClose={noop}
        onSubmit={async () => undefined}
      />
    </ToastProvider>,
  );
}

export function renderStickerMediaPicker(): string {
  return renderToString(
    <ToastProvider>
      <StickerMediaPickerDialog
        open
        files={[
          {
            id: 'demo-media-1',
            ownerId: 'me',
            objectKey: 'media/me/demo-media-1',
            originalFilename: 'morning-coffee.png',
            mimeType: 'image/png',
            sizeBytes: 1024,
            scanStatus: 'clean',
            status: 'ready',
            createdAt: '2026-11-20T09:00:00+08:00',
            updatedAt: '2026-11-20T09:00:00+08:00',
          },
        ]}
        filesState="ready"
        selectedId=""
        onRetryFiles={noop}
        onClose={noop}
        onConfirm={noop}
      />
    </ToastProvider>,
  );
}

export function renderAddStickerDialog(): string {
  return renderToString(
    <ToastProvider>
      <AddStickerDialog
        open
        packName="山間小日子"
        files={[
          {
            id: 'demo-media-1',
            ownerId: 'me',
            objectKey: 'media/me/demo-media-1',
            originalFilename: 'morning-coffee.png',
            mimeType: 'image/png',
            sizeBytes: 1024,
            scanStatus: 'clean',
            status: 'ready',
            createdAt: '2026-11-20T09:00:00+08:00',
            updatedAt: '2026-11-20T09:00:00+08:00',
          },
        ]}
        filesState="ready"
        onRetryFiles={noop}
        onClose={noop}
        onConfirm={async () => undefined}
      />
    </ToastProvider>,
  );
}
