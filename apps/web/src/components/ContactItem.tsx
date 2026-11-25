import type { ReactElement } from 'react';
import { presenceLabel } from '../data/logic';
import type { Contact } from '../data/types';
import { Avatar } from './Avatar';
import { Badge } from './Badge';
import { StatusIndicator } from './StatusIndicator';

export interface ContactItemProps {
  contact: Contact;
  onSelect: (contactId: string) => void;
}

export function ContactItem({ contact, onSelect }: ContactItemProps): ReactElement {
  return (
    <button
      type="button"
      className="et-contact-item"
      onClick={() => {
        onSelect(contact.id);
      }}
    >
      <Avatar name={contact.name} size="md" presence={contact.presence} />
      <span className="et-contact-item__main">
        <span className="et-contact-item__top">
          <span className="et-contact-item__name">{contact.name}</span>
          <StatusIndicator presence={contact.presence} />
        </span>
        <span className="et-contact-item__handle">{contact.handle}</span>
        <span className="et-contact-item__status">
          {contact.statusMessage !== '' ? contact.statusMessage : presenceLabel(contact.presence)}
        </span>
      </span>
      {contact.isFavorite ? <Badge tone="accent">最愛</Badge> : null}
    </button>
  );
}
