import { useId } from 'react';
import type { ReactElement, SelectHTMLAttributes } from 'react';
import { ChevronDownIcon } from './icons';

export interface SelectOption {
  value: string;
  label: string;
}

export interface SelectProps
  extends Omit<SelectHTMLAttributes<HTMLSelectElement>, 'className' | 'id'> {
  id?: string;
  label?: string;
  hint?: string | undefined;
  error?: string | undefined;
  options: SelectOption[];
}

export function Select({ label, hint, error, id, options, ...rest }: SelectProps): ReactElement {
  const generated = useId();
  const fieldId = id ?? generated;
  const hintId = hint !== undefined ? `${fieldId}-hint` : undefined;
  const errorId = error !== undefined ? `${fieldId}-error` : undefined;
  const describedBy = [hintId, errorId].filter((part) => part !== undefined).join(' ');
  return (
    <div className="et-field">
      {label !== undefined ? (
        <label className="et-field__label" htmlFor={fieldId}>
          {label}
        </label>
      ) : null}
      <div className="et-select-wrap">
        <select
          id={fieldId}
          className={error !== undefined ? 'et-select et-select--invalid' : 'et-select'}
          aria-invalid={error !== undefined || undefined}
          aria-describedby={describedBy === '' ? undefined : describedBy}
          {...rest}
        >
          {options.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        <span className="et-select__chevron" aria-hidden="true">
          <ChevronDownIcon size={18} />
        </span>
      </div>
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
