import type { ReactElement } from 'react';

export function TypingIndicator({ name }: { name: string }): ReactElement {
  return (
    <span className="et-typing" role="status" aria-label={`${name} 正在輸入`}>
      <span className="et-typing__dot" aria-hidden="true" />
      <span className="et-typing__dot" aria-hidden="true" />
      <span className="et-typing__dot" aria-hidden="true" />
    </span>
  );
}
