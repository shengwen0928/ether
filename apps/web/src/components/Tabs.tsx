import { useRef } from 'react';
import type { KeyboardEvent, ReactElement, ReactNode } from 'react';

export interface TabItem {
  id: string;
  label: string;
  icon?: ReactNode;
  badge?: ReactNode;
}

export interface TabsProps {
  label: string;
  tabs: TabItem[];
  value: string;
  onChange: (id: string) => void;
}

export function Tabs({ label, tabs, value, onChange }: TabsProps): ReactElement {
  const tabRefs = useRef<Array<HTMLButtonElement | null>>([]);

  const focusTab = (index: number): void => {
    const target = tabRefs.current[index];
    if (target !== undefined && target !== null) {
      target.focus();
    }
  };

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>): void => {
    const current = tabs.findIndex((tab) => tab.id === value);
    if (event.key === 'ArrowRight' || event.key === 'ArrowLeft') {
      event.preventDefault();
      const delta = event.key === 'ArrowRight' ? 1 : -1;
      const next = (current + delta + tabs.length) % tabs.length;
      const nextTab = tabs[next];
      if (nextTab !== undefined) {
        onChange(nextTab.id);
        focusTab(next);
      }
      return;
    }
    if (event.key === 'Home' || event.key === 'End') {
      event.preventDefault();
      const next = event.key === 'Home' ? 0 : tabs.length - 1;
      const nextTab = tabs[next];
      if (nextTab !== undefined) {
        onChange(nextTab.id);
        focusTab(next);
      }
    }
  };

  return (
    <div className="et-tabs" role="tablist" aria-label={label} onKeyDown={onKeyDown}>
      {tabs.map((tab, index) => {
        const selected = tab.id === value;
        return (
          <button
            key={tab.id}
            ref={(element) => {
              tabRefs.current[index] = element;
            }}
            type="button"
            role="tab"
            id={`et-tab-${tab.id}`}
            aria-selected={selected}
            aria-controls={`et-panel-${tab.id}`}
            tabIndex={selected ? 0 : -1}
            className={selected ? 'et-tabs__tab et-tabs__tab--active' : 'et-tabs__tab'}
            onClick={() => {
              onChange(tab.id);
            }}
          >
            {tab.icon !== undefined ? (
              <span className="et-tabs__icon" aria-hidden="true">
                {tab.icon}
              </span>
            ) : null}
            <span>{tab.label}</span>
            {tab.badge !== undefined ? <span className="et-tabs__badge">{tab.badge}</span> : null}
          </button>
        );
      })}
    </div>
  );
}

export function TabPanel({
  id,
  active,
  children,
}: {
  id: string;
  active: boolean;
  children: ReactNode;
}): ReactElement {
  return (
    <div
      role="tabpanel"
      id={`et-panel-${id}`}
      aria-labelledby={`et-tab-${id}`}
      hidden={!active}
      className="et-tabs__panel"
    >
      {children}
    </div>
  );
}
