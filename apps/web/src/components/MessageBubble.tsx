import type { ReactElement, ReactNode } from 'react';
import { formatMessageTime } from '../data/logic';
import type { ChatMessage } from '../data/types';
import { AttachmentPreview } from './AttachmentPreview';
import { Avatar } from './Avatar';
import { MessageStatus } from './MessageStatus';
import { VoiceMessage } from './VoiceMessage';

export interface MessageBubbleProps {
  message: ChatMessage;
  isOwn: boolean;
  senderName: string;
  showSenderName: boolean;
  now: Date;
  onAttachmentDownload?: (message: ChatMessage) => void;
  /**
   * 訊息下方操作列（反應／置頂 §2q、附件／轉傳 §2t、自己訊息的
   * 編輯／收回／刪除 §2u）：由呼叫端傳入，未傳入時不渲染，
   * 氣泡外觀與既有行為不變。
   */
  actions?: ReactNode;
}

export function MessageBubble({
  message,
  isOwn,
  senderName,
  showSenderName,
  now,
  onAttachmentDownload,
  actions,
}: MessageBubbleProps): ReactElement {
  const time = formatMessageTime(message.sentAt, now);
  return (
    <div className={isOwn ? 'et-message et-message--own' : 'et-message'}>
      {!isOwn ? (
        <Avatar name={senderName} size="sm" />
      ) : (
        <span className="et-message__spacer" aria-hidden="true" />
      )}
      <div className="et-message__body">
        {showSenderName && !isOwn ? <span className="et-message__sender">{senderName}</span> : null}
        <div className="et-message__bubble">
          {message.kind === 'text' ? (
            <span className="et-message__text">{message.body}</span>
          ) : null}
          {message.kind === 'voice' ? (
            <VoiceMessage
              durationSeconds={message.durationSeconds ?? 0}
              label={`來自 ${senderName} 的語音訊息`}
            />
          ) : null}
          {message.kind === 'attachment' ? (
            <AttachmentPreview
              fileName={message.fileName ?? message.body}
              fileSize={message.fileSize}
              isImage={(message.fileName ?? '').toLowerCase().endsWith('.jpg')}
              onDownload={
                onAttachmentDownload !== undefined
                  ? () => {
                      onAttachmentDownload(message);
                    }
                  : undefined
              }
            />
          ) : null}
        </div>
        <span className="et-message__meta">
          <time dateTime={message.sentAt}>{time}</time>
          {isOwn ? <MessageStatus status={message.status} /> : null}
        </span>
        {actions}
      </div>
    </div>
  );
}
