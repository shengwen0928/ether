import type { CSSProperties, ReactElement } from 'react';

export function Skeleton({
  width = '100%',
  height = 14,
  circle = false,
  label = '載入中',
}: {
  width?: number | string;
  height?: number | string;
  circle?: boolean;
  label?: string;
}): ReactElement {
  const style: CSSProperties = {
    width,
    height,
    borderRadius: circle ? '50%' : undefined,
  };
  return <span className="et-skeleton" style={style} role="status" aria-label={label} />;
}
