import { useId } from 'react';
import type { ReactElement } from 'react';
import { CloseIcon, SearchIcon } from './icons';

export interface SearchFieldProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  label?: string;
  autoFocus?: boolean;
}

export function SearchField({
  value,
  onChange,
  placeholder = '搜尋',
  label = '搜尋',
  autoFocus = false,
}: SearchFieldProps): ReactElement {
  const id = useId();
  return (
    <div className="et-search">
      <label className="et-sr-only" htmlFor={id}>
        {label}
      </label>
      <span className="et-search__icon" aria-hidden="true">
        <SearchIcon size={18} />
      </span>
      <input
        id={id}
        className="et-search__input"
        type="text"
        role="searchbox"
        value={value}
        placeholder={placeholder}
        autoFocus={autoFocus}
        onChange={(event) => {
          onChange(event.target.value);
        }}
      />
      {value !== '' ? (
        <button
          type="button"
          className="et-search__clear"
          aria-label="清除搜尋"
          onClick={() => {
            onChange('');
          }}
        >
          <CloseIcon size={16} />
        </button>
      ) : null}
    </div>
  );
}
