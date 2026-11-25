import { useEffect, useRef, useState } from 'react';
import type { KeyboardEvent, ReactElement, ReactNode } from 'react';
import { ChevronDownIcon } from './icons';

export interface DropdownItem {
  id: string;
  label: string;
  danger?: boolean;
  disabled?: boolean;
}

export interface DropdownProps {
  label: ReactNode;
  items: DropdownItem[];
  onSelect: (id: string) => void;
  ariaLabel?: string;
  align?: 'start' | 'end';
}

export function Dropdown({
  label,
  items,
  onSelect,
  ariaLabel,
  align = 'end',
}: DropdownProps): ReactElement {
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const buttonRef = useRef<HTMLButtonElement | null>(null);
  const itemRefs = useRef<Array<HTMLButtonElement | null>>([]);

  const enabledIndexes = items
    .map((item, index) => (item.disabled === true ? -1 : index))
    .filter((index) => index >= 0);

  useEffect(() => {
    if (!open) {
      return undefined;
    }
    const first = enabledIndexes[0];
    if (first !== undefined) {
      setActiveIndex(first);
      itemRefs.current[first]?.focus();
    }
    return undefined;
  }, [open]);

  const close = (): void => {
    setOpen(false);
    buttonRef.current?.focus();
  };

  const choose = (item: DropdownItem): void => {
    if (item.disabled === true) {
      return;
    }
    setOpen(false);
    onSelect(item.id);
  };

  const onMenuKeyDown = (event: KeyboardEvent<HTMLDivElement>): void => {
    if (event.key === 'Escape') {
      event.preventDefault();
      close();
      return;
    }
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      if (enabledIndexes.length === 0) {
        return;
      }
      const current = enabledIndexes.indexOf(activeIndex);
      const next =
        event.key === 'ArrowDown'
          ? (enabledIndexes[(current + 1) % enabledIndexes.length] ?? activeIndex)
          : (enabledIndexes[(current - 1 + enabledIndexes.length) % enabledIndexes.length] ??
            activeIndex);
      setActiveIndex(next);
      itemRefs.current[next]?.focus();
      return;
    }
    if (event.key === 'Home' || event.key === 'End') {
      event.preventDefault();
      const target =
        event.key === 'Home'
          ? (enabledIndexes[0] ?? 0)
          : (enabledIndexes[enabledIndexes.length - 1] ?? 0);
      setActiveIndex(target);
      itemRefs.current[target]?.focus();
    }
  };

  return (
    <div className="et-dropdown">
      <button
        ref={buttonRef}
        type="button"
        className="et-dropdown__trigger"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={ariaLabel}
        onClick={() => {
          setOpen((wasOpen) => !wasOpen);
        }}
      >
        {label}
        <span className="et-dropdown__chevron" aria-hidden="true">
          <ChevronDownIcon size={16} />
        </span>
      </button>
      {open ? (
        <>
          <button
            type="button"
            className="et-dropdown__backdrop"
            aria-label="關閉選單"
            tabIndex={-1}
            onClick={() => {
              setOpen(false);
            }}
          />
          <div
            className={
              align === 'end' ? 'et-dropdown__menu et-dropdown__menu--end' : 'et-dropdown__menu'
            }
            role="menu"
            onKeyDown={onMenuKeyDown}
          >
            {items.map((item, index) => (
              <button
                key={item.id}
                ref={(element) => {
                  itemRefs.current[index] = element;
                }}
                type="button"
                role="menuitem"
                className={
                  item.danger === true
                    ? 'et-dropdown__item et-dropdown__item--danger'
                    : 'et-dropdown__item'
                }
                disabled={item.disabled === true}
                tabIndex={index === activeIndex ? 0 : -1}
                onClick={() => {
                  choose(item);
                }}
                onMouseEnter={() => {
                  if (item.disabled !== true) {
                    setActiveIndex(index);
                  }
                }}
              >
                {item.label}
              </button>
            ))}
          </div>
        </>
      ) : null}
    </div>
  );
}
