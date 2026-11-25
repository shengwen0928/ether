import { useEffect, useId, useRef } from 'react';
import type { ReactElement, ReactNode } from 'react';
import { CloseIcon } from './icons';

export interface DrawerProps {
  open: boolean;
  onClose: () => void;
  title: string;
  side?: 'left' | 'right';
  children: ReactNode;
}

export function Drawer({
  open,
  onClose,
  title,
  side = 'left',
  children,
}: DrawerProps): ReactElement | null {
  const titleId = useId();
  const panelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!open) {
      return undefined;
    }
    const onKeyDown = (event: globalThis.KeyboardEvent): void => {
      if (event.key === 'Escape') {
        onClose();
      }
    };
    document.addEventListener('keydown', onKeyDown);
    panelRef.current?.focus();
    return () => {
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [open, onClose]);

  if (!open) {
    return null;
  }

  return (
    <div className="et-drawer-overlay">
      <button
        type="button"
        className="et-drawer__backdrop"
        aria-label="關閉側邊欄"
        onClick={onClose}
      />
      <div
        ref={panelRef}
        className={side === 'right' ? 'et-drawer et-drawer--right' : 'et-drawer'}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
      >
        <div className="et-drawer__header">
          <h2 className="et-drawer__title" id={titleId}>
            {title}
          </h2>
          <button
            type="button"
            className="et-drawer__close"
            aria-label="關閉側邊欄"
            onClick={onClose}
          >
            <CloseIcon size={18} />
          </button>
        </div>
        <div className="et-drawer__body">{children}</div>
      </div>
    </div>
  );
}
