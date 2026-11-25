import { useId } from 'react';
import type { InputHTMLAttributes, ReactElement, TextareaHTMLAttributes } from 'react';

export interface FieldMeta {
  label?: string | undefined;
  hint?: string | undefined;
  error?: string | undefined;
}

function useFieldIds(
  id: string | undefined,
  meta: FieldMeta,
): {
  fieldId: string;
  hintId: string | undefined;
  errorId: string | undefined;
  describedBy: string | undefined;
} {
  const generated = useId();
  const fieldId = id ?? generated;
  const hintId = meta.hint !== undefined ? `${fieldId}-hint` : undefined;
  const errorId = meta.error !== undefined ? `${fieldId}-error` : undefined;
  const describedBy = [hintId, errorId].filter((part) => part !== undefined).join(' ');
  return { fieldId, hintId, errorId, describedBy: describedBy === '' ? undefined : describedBy };
}

export interface InputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'className' | 'id'>,
    FieldMeta {
  id?: string;
}

export function Input({ label, hint, error, id, ...rest }: InputProps): ReactElement {
  const { fieldId, hintId, errorId, describedBy } = useFieldIds(id, { label, hint, error });
  return (
    <div className="et-field">
      {label !== undefined ? (
        <label className="et-field__label" htmlFor={fieldId}>
          {label}
        </label>
      ) : null}
      <input
        id={fieldId}
        className={error !== undefined ? 'et-input et-input--invalid' : 'et-input'}
        aria-invalid={error !== undefined || undefined}
        aria-describedby={describedBy}
        {...rest}
      />
      {hint !== undefined && error === undefined ? (
        <p className="et-field__hint" id={hintId}>
          {hint}
        </p>
      ) : null}
      {error !== undefined ? (
        <p className="et-field__error" id={errorId} role="alert">
          {error}
        </p>
      ) : null}
    </div>
  );
}

export interface TextAreaProps
  extends Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, 'className' | 'id'>,
    FieldMeta {
  id?: string;
}

export function TextArea({ label, hint, error, id, ...rest }: TextAreaProps): ReactElement {
  const { fieldId, hintId, errorId, describedBy } = useFieldIds(id, { label, hint, error });
  return (
    <div className="et-field">
      {label !== undefined ? (
        <label className="et-field__label" htmlFor={fieldId}>
          {label}
        </label>
      ) : null}
      <textarea
        id={fieldId}
        className={error !== undefined ? 'et-input et-input--invalid' : 'et-input'}
        aria-invalid={error !== undefined || undefined}
        aria-describedby={describedBy}
        {...rest}
      />
      {hint !== undefined && error === undefined ? (
        <p className="et-field__hint" id={hintId}>
          {hint}
        </p>
      ) : null}
      {error !== undefined ? (
        <p className="et-field__error" id={errorId} role="alert">
          {error}
        </p>
      ) : null}
    </div>
  );
}
