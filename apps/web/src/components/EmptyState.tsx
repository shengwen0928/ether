import type { ReactElement, ReactNode } from 'react';

export interface StateBlockProps {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
}

export function EmptyState({ icon, title, description, action }: StateBlockProps): ReactElement {
  return (
    <div className="et-state" role="status">
      {icon !== undefined ? (
        <span className="et-state__icon" aria-hidden="true">
          {icon}
        </span>
      ) : null}
      <h2 className="et-state__title">{title}</h2>
      {description !== undefined ? <p className="et-state__description">{description}</p> : null}
      {action !== undefined ? <div className="et-state__action">{action}</div> : null}
    </div>
  );
}

export function ErrorState({ icon, title, description, action }: StateBlockProps): ReactElement {
  return (
    <div className="et-state et-state--error" role="alert">
      {icon !== undefined ? (
        <span className="et-state__icon" aria-hidden="true">
          {icon}
        </span>
      ) : null}
      <h2 className="et-state__title">{title}</h2>
      {description !== undefined ? <p className="et-state__description">{description}</p> : null}
      {action !== undefined ? <div className="et-state__action">{action}</div> : null}
    </div>
  );
}
