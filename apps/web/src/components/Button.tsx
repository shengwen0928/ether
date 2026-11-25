import type { ButtonHTMLAttributes, ReactElement, ReactNode } from 'react';
import { Spinner } from './Spinner';

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger';
export type ButtonSize = 'sm' | 'md' | 'lg';

export interface ButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'className'> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
  fullWidth?: boolean;
  iconStart?: ReactNode;
  iconEnd?: ReactNode;
}

export function Button({
  variant = 'primary',
  size = 'md',
  loading = false,
  fullWidth = false,
  iconStart,
  iconEnd,
  disabled,
  children,
  type = 'button',
  ...rest
}: ButtonProps): ReactElement {
  const classes = [
    'et-btn',
    `et-btn--${variant}`,
    `et-btn--${size}`,
    fullWidth ? 'et-btn--full' : '',
    loading ? 'et-btn--loading' : '',
  ]
    .filter(Boolean)
    .join(' ');
  return (
    <button
      type={type}
      className={classes}
      disabled={disabled === true || loading}
      aria-busy={loading || undefined}
      {...rest}
    >
      {loading ? <Spinner size={size === 'lg' ? 18 : 14} /> : null}
      {iconStart !== undefined && !loading ? (
        <span className="et-btn__icon" aria-hidden="true">
          {iconStart}
        </span>
      ) : null}
      <span className="et-btn__label">{children}</span>
      {iconEnd !== undefined ? (
        <span className="et-btn__icon" aria-hidden="true">
          {iconEnd}
        </span>
      ) : null}
    </button>
  );
}
