import type { ReactElement } from 'react';
import { formatMessageTime } from '../data/logic';
import type { Contact, Conversation } from '../data/types';
import { Avatar } from './Avatar';
import { CountBadge } from './Badge';
import { MuteIcon, PinIcon } from './icons';
import { TypingIndicator } from './TypingIndicator';

export interface ConversationItemProps {
  conversation: Conversation;
  contact: Contact;
  active: boolean;
  now: Date;
  onSelect: (conversationId: string) => void;
}

export function ConversationItem({
  conversation,
  contact,
  active,
  now,
  onSelect,
}: ConversationItemProps): ReactElement {
  return (
    <button
      type="button"
      className={active ? 'et-conv-item et-conv-item--active' : 'et-conv-item'}
      aria-current={active ? 'true' : undefined}
      onClick={() => {
        onSelect(conversation.id);
      }}
    >
      <Avatar name={contact.name} size="md" presence={contact.presence} />
      <span className="et-conv-item__main">
        <span className="et-conv-item__top">
          <span className="et-conv-item__name">{contact.name}</span>
          <time className="et-conv-item__time" dateTime={conversation.lastMessageAt}>
            {formatMessageTime(conversation.lastMessageAt, now)}
          </time>
        </span>
        <span className="et-conv-item__bottom">
          {conversation.isTyping ? (
            <span className="et-conv-item__typing">
              正在輸入
              <TypingIndicator name={contact.name} />
            </span>
          ) : (
            <span className="et-conv-item__preview">{conversation.lastMessagePreview}</span>
          )}
          <span className="et-conv-item__flags">
            {conversation.isMuted ? (
              <span aria-label="已靜音">
                <MuteIcon size={14} />
              </span>
            ) : null}
            {conversation.isPinned ? (
              <span aria-label="已置頂">
                <PinIcon size={14} />
              </span>
            ) : null}
            <CountBadge count={conversation.unreadCount} />
          </span>
        </span>
      </span>
    </button>
  );
}
