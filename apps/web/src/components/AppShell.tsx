import type { ReactElement, ReactNode } from 'react';
import { useState } from 'react';
import { MenuIcon } from './icons';
import { Drawer } from './Drawer';
import { IconButton } from './IconButton';
import { LogoLockup } from './Logo';

export interface NavItem {
  id: string;
  label: string;
  icon: ReactNode;
  badge?: ReactNode;
}

export interface AppShellProps {
  navItems: NavItem[];
  activeNav: string;
  onNavigate: (id: string) => void;
  headerTitle: string;
  headerActions?: ReactNode;
  sidebarFooter?: ReactNode;
  bottomNavLabel?: string;
  children: ReactNode;
}

export function AppShell({
  navItems,
  activeNav,
  onNavigate,
  headerTitle,
  headerActions,
  sidebarFooter,
  children,
}: AppShellProps): ReactElement {
  const [drawerOpen, setDrawerOpen] = useState(false);

  const nav = (
    <nav className="et-shell__nav" aria-label="主要導覽">
      <ul>
        {navItems.map((item) => (
          <li key={item.id}>
            <button
              type="button"
              className={
                item.id === activeNav
                  ? 'et-shell__nav-btn et-shell__nav-btn--active'
                  : 'et-shell__nav-btn'
              }
              aria-current={item.id === activeNav ? 'page' : undefined}
              onClick={() => {
                onNavigate(item.id);
                setDrawerOpen(false);
              }}
            >
              <span className="et-shell__nav-icon" aria-hidden="true">
                {item.icon}
              </span>
              <span className="et-shell__nav-label">{item.label}</span>
              {item.badge !== undefined ? (
                <span className="et-shell__nav-badge">{item.badge}</span>
              ) : null}
            </button>
          </li>
        ))}
      </ul>
    </nav>
  );

  return (
    <div className="et-shell">
      <a className="et-sr-only" href="#et-main">
        跳至主要內容
      </a>
      <aside className="et-shell__sidebar" aria-label="側邊欄">
        <div className="et-shell__brand">
          <LogoLockup size={32} />
        </div>
        {nav}
        {sidebarFooter !== undefined ? (
          <div className="et-shell__sidebar-footer">{sidebarFooter}</div>
        ) : null}
      </aside>
      <div className="et-shell__main">
        <header className="et-shell__header">
          <span className="et-shell__menu-btn">
            <IconButton
              label="開啟導覽"
              onClick={() => {
                setDrawerOpen(true);
              }}
            >
              <MenuIcon size={20} />
            </IconButton>
          </span>
          <h1 className="et-shell__title">{headerTitle}</h1>
          {headerActions !== undefined ? (
            <div className="et-shell__actions">{headerActions}</div>
          ) : null}
        </header>
        <div id="et-main" className="et-shell__content" tabIndex={-1}>
          {children}
        </div>
      </div>
      <nav className="et-shell__bottomnav" aria-label="主要導覽（手機）">
        {navItems.map((item) => (
          <button
            key={item.id}
            type="button"
            className={
              item.id === activeNav
                ? 'et-shell__bottom-btn et-shell__bottom-btn--active'
                : 'et-shell__bottom-btn'
            }
            aria-current={item.id === activeNav ? 'page' : undefined}
            onClick={() => {
              onNavigate(item.id);
            }}
          >
            <span className="et-shell__nav-icon" aria-hidden="true">
              {item.icon}
            </span>
            <span className="et-shell__bottom-label">{item.label}</span>
            {item.badge !== undefined ? (
              <span className="et-shell__nav-badge">{item.badge}</span>
            ) : null}
          </button>
        ))}
      </nav>
      <Drawer open={drawerOpen} title="ETHER 導覽" onClose={() => setDrawerOpen(false)}>
        <div className="et-shell__drawer-nav">{nav}</div>
      </Drawer>
    </div>
  );
}
