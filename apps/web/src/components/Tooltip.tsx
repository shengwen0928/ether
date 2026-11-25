import type { ReactElement, ReactNode } from 'react';

export interface TooltipProps {
  tip: string;
  children: ReactNode;
}

/** CSS-driven tooltip: appears on hover and on keyboard focus. */
export function Tooltip({ tip, children }: TooltipProps): ReactElement {
  return (
    <span className="et-tooltip">
      {children}
      <span className="et-tooltip__bubble" role="tooltip">
        {tip}
      </span>
    </span>
  );
}
