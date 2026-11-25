import type { ReactElement, ReactNode } from 'react';

export type BadgeTone = 'neutral' | 'accent' | 'success' | 'warning' | 'error';

export interface BadgeProps {
  tone?: BadgeTone;
  children: ReactNode;
}

export function Badge({ tone = 'neutral', children }: BadgeProps): ReactElement {
  return <span className={`et-badge et-badge--${tone}`}>{children}</span>;
}

export function CountBadge({
  count,
  label,
}: {
  count: number;
  label?: string;
}): ReactElement | null {
  if (count <= 0) {
    return null;
  }
  const text = count > 99 ? '99+' : String(count);
  return (
    <span
      className="et-badge et-badge--error et-badge--count"
      aria-label={label ?? `${count} 則未讀`}
    >
      {text}
    </span>
  );
}

export function DotBadge({ label }: { label: string }): ReactElement {
  return <span className="et-badge et-badge--dot" aria-label={label} />;
}
