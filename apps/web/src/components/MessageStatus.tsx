import type { ReactElement } from 'react';
import { CheckIcon } from './icons';
import { Spinner } from './Spinner';
import type { MessageStatus } from '../data/types';

const STATUS_TEXT: Record<MessageStatus, string> = {
  sending: '傳送中',
  sent: '已傳送',
  delivered: '已送達',
  read: '已讀',
};

export function MessageStatus({ status }: { status: MessageStatus }): ReactElement {
  if (status === 'sending') {
    return (
      <span className="et-msg-status" role="img" aria-label={STATUS_TEXT[status]}>
        <Spinner size={12} label={STATUS_TEXT[status]} />
      </span>
    );
  }
  const checks =
    status === 'sent' ? (
      <CheckIcon size={14} />
    ) : (
      <span className="et-msg-status__double" aria-hidden="true">
        <CheckIcon size={14} />
        <CheckIcon size={14} />
      </span>
    );
  return (
    <span
      className={status === 'read' ? 'et-msg-status et-msg-status--read' : 'et-msg-status'}
      role="img"
      aria-label={STATUS_TEXT[status]}
    >
      {checks}
    </span>
  );
}
