import { useEffect, useId, useRef } from 'react';
import type { ReactElement, ReactNode } from 'react';
import { CloseIcon } from './icons';

export interface DialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children: ReactNode;
  footer?: ReactNode;
  dismissLabel?: string;
}

export function Dialog({
  open,
  onClose,
  title,
  description,
  children,
  footer,
  dismissLabel = '關閉對話框',
}: DialogProps): ReactElement | null {
  const titleId = useId();
  const descriptionId = useId();
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
    <div
      className="et-dialog-overlay"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      <div
        ref={panelRef}
        className="et-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description !== undefined ? descriptionId : undefined}
        tabIndex={-1}
      >
        <div className="et-dialog__header">
          <div>
            <h2 className="et-dialog__title" id={titleId}>
              {title}
            </h2>
            {description !== undefined ? (
              <p className="et-dialog__description" id={descriptionId}>
                {description}
              </p>
            ) : null}
          </div>
          <button
            type="button"
            className="et-dialog__close"
            aria-label={dismissLabel}
            onClick={onClose}
          >
            <CloseIcon size={18} />
          </button>
        </div>
        <div className="et-dialog__body">{children}</div>
        {footer !== undefined ? <div className="et-dialog__footer">{footer}</div> : null}
      </div>
    </div>
  );
}
