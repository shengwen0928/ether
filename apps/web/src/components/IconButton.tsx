import type { ButtonHTMLAttributes, ReactElement, ReactNode } from 'react';

export type IconButtonVariant = 'ghost' | 'filled' | 'outline';
export type IconButtonSize = 'sm' | 'md' | 'lg';

export interface IconButtonProps
  extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'className' | 'aria-label'> {
  label: string;
  variant?: IconButtonVariant;
  size?: IconButtonSize;
  active?: boolean;
  children: ReactNode;
}

export function IconButton({
  label,
  variant = 'ghost',
  size = 'md',
  active = false,
  children,
  type = 'button',
  ...rest
}: IconButtonProps): ReactElement {
  const classes = [
    'et-icon-btn',
    `et-icon-btn--${variant}`,
    `et-icon-btn--${size}`,
    active ? 'et-icon-btn--active' : '',
  ]
    .filter(Boolean)
    .join(' ');
  return (
    <button
      type={type}
      className={classes}
      aria-label={label}
      aria-pressed={active || undefined}
      {...rest}
    >
      <span className="et-icon-btn__icon" aria-hidden="true">
        {children}
      </span>
    </button>
  );
}
