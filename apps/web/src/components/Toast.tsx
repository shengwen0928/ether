import { createContext, useCallback, useContext, useMemo, useRef, useState } from 'react';
import type { ReactElement, ReactNode } from 'react';
import { CloseIcon } from './icons';

export type ToastTone = 'info' | 'success' | 'warning' | 'error';

export interface ToastInput {
  title?: string;
  message: string;
  tone?: ToastTone;
}

interface ToastItem extends Required<Pick<ToastInput, 'message'>> {
  id: number;
  title?: string | undefined;
  tone: ToastTone;
}

interface ToastContextValue {
  notify: (input: ToastInput) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast(): ToastContextValue {
  const context = useContext(ToastContext);
  if (context === null) {
    throw new Error('useToast must be used within ToastProvider.');
  }
  return context;
}

export function ToastProvider({ children }: { children: ReactNode }): ReactElement {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const nextId = useRef(1);

  const dismiss = useCallback((id: number) => {
    setToasts((current) => current.filter((toast) => toast.id !== id));
  }, []);

  const notify = useCallback(
    (input: ToastInput) => {
      const id = nextId.current;
      nextId.current += 1;
      setToasts((current) => [
        ...current,
        { id, message: input.message, title: input.title, tone: input.tone ?? 'info' },
      ]);
      window.setTimeout(() => {
        dismiss(id);
      }, 4200);
    },
    [dismiss],
  );

  const value = useMemo(() => ({ notify }), [notify]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="et-toast-region" aria-live="polite" aria-label="通知">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={`et-toast et-toast--${toast.tone}`}
            role={toast.tone === 'error' ? 'alert' : 'status'}
          >
            <div className="et-toast__body">
              {toast.title !== undefined ? (
                <strong className="et-toast__title">{toast.title}</strong>
              ) : null}
              <span className="et-toast__message">{toast.message}</span>
            </div>
            <button
              type="button"
              className="et-toast__close"
              aria-label="關閉通知"
              onClick={() => {
                dismiss(toast.id);
              }}
            >
              <CloseIcon size={16} />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
