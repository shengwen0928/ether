import type { ReactElement } from 'react';

export function Spinner({
  label = '載入中',
  size = 16,
}: {
  label?: string;
  size?: number;
}): ReactElement {
  return (
    <span className="et-spinner" role="status" aria-label={label}>
      <span className="et-spinner__ring" style={{ width: size, height: size }} aria-hidden="true" />
    </span>
  );
}
