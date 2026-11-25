import type { ReactElement } from 'react';
import { presenceLabel } from '../data/logic';
import type { PresenceStatus } from '../data/types';

export function StatusIndicator({
  presence,
  showLabel = false,
}: {
  presence: PresenceStatus;
  showLabel?: boolean;
}): ReactElement {
  return (
    <span className="et-status">
      <span className={`et-status__dot et-status__dot--${presence}`} aria-hidden="true" />
      {showLabel ? <span className="et-status__label">{presenceLabel(presence)}</span> : null}
    </span>
  );
}

export function Divider({ label }: { label?: string }): ReactElement {
  if (label === undefined) {
    return <hr className="et-divider" />;
  }
  return (
    <div className="et-divider et-divider--labeled" role="separator">
      <span className="et-divider__label">{label}</span>
    </div>
  );
}
