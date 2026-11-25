import type { ReactElement } from 'react';

export interface SwitchProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: string;
  description?: string;
  disabled?: boolean;
}

export function Switch({
  checked,
  onChange,
  label,
  description,
  disabled = false,
}: SwitchProps): ReactElement {
  return (
    <div className="et-switch-row">
      <div className="et-switch-row__text">
        <span className="et-switch-row__label">{label}</span>
        {description !== undefined ? (
          <span className="et-switch-row__description">{description}</span>
        ) : null}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        className={checked ? 'et-switch et-switch--on' : 'et-switch'}
        disabled={disabled}
        onClick={() => {
          onChange(!checked);
        }}
      >
        <span className="et-switch__thumb" aria-hidden="true" />
      </button>
    </div>
  );
}
