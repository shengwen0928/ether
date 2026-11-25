import type { ReactElement, ReactNode } from 'react';

/** Minimal stroke icon set. All icons use currentColor; decorative only. */

interface IconProps {
  size?: number | undefined;
}

function base(size: number): { width: number; height: number; viewBox: string } {
  return { width: size, height: size, viewBox: '0 0 24 24' };
}

function Svg({ size = 20, children }: IconProps & { children: ReactNode }): ReactElement {
  const attrs = base(size);
  return (
    <svg
      {...attrs}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      {children}
    </svg>
  );
}

export function SearchIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.5-3.5" />
    </Svg>
  );
}

export function CloseIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M6 6l12 12M18 6 6 18" />
    </Svg>
  );
}

export function CheckIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="m5 12 5 5L20 7" />
    </Svg>
  );
}

export function ChevronDownIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="m6 9 6 6 6-6" />
    </Svg>
  );
}

export function ChevronLeftIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M15 6l-6 6 6 6" />
    </Svg>
  );
}

export function SendIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M4 12 20 4l-4 8 4 8L4 12Z" />
      <path d="M20 4 12 12" />
    </Svg>
  );
}

export function PaperclipIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="m20 11-8.5 8.5a5.5 5.5 0 0 1-7.8-7.8L12 3.5a3.7 3.7 0 0 1 5.2 5.2l-8.2 8.2a1.85 1.85 0 0 1-2.6-2.6L14 6.7" />
    </Svg>
  );
}

export function SmileIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <circle cx="12" cy="12" r="9" />
      <path d="M8.5 14.5a4.5 4.5 0 0 0 7 0" />
      <circle cx="9" cy="9.5" r="0.6" fill="currentColor" />
      <circle cx="15" cy="9.5" r="0.6" fill="currentColor" />
    </Svg>
  );
}

export function MicIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <rect x="9" y="3" width="6" height="11" rx="3" />
      <path d="M5 11a7 7 0 0 0 14 0M12 18v3" />
    </Svg>
  );
}

export function PhoneIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M5 4h4l2 5-2.5 1.5a12 12 0 0 0 5 5L15 13l5 2v4a2 2 0 0 1-2 2A16 16 0 0 1 3 6a2 2 0 0 1 2-2Z" />
    </Svg>
  );
}

export function VideoIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <rect x="2.5" y="6" width="13" height="12" rx="2.5" />
      <path d="m15.5 10.5 6-3.5v10l-6-3.5" />
    </Svg>
  );
}

export function BellIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M6 9a6 6 0 0 1 12 0c0 5 2 6 2 6H4s2-1 2-6" />
      <path d="M10 19a2 2 0 0 0 4 0" />
    </Svg>
  );
}

export function SettingsIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <circle cx="12" cy="12" r="3" />
      <path d="M19 12a7 7 0 0 0-.1-1.2l2-1.6-2-3.4-2.4 1a7 7 0 0 0-2-1.2L14 3h-4l-.5 2.6a7 7 0 0 0-2 1.2l-2.4-1-2 3.4 2 1.6A7 7 0 0 0 5 12c0 .4 0 .8.1 1.2l-2 1.6 2 3.4 2.4-1a7 7 0 0 0 2 1.2L10 21h4l.5-2.6a7 7 0 0 0 2-1.2l2.4 1 2-3.4-2-1.6c.1-.4.1-.8.1-1.2Z" />
    </Svg>
  );
}

export function SunIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
    </Svg>
  );
}

export function MoonIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M20 13.5A8 8 0 0 1 10.5 4 8 8 0 1 0 20 13.5Z" />
    </Svg>
  );
}

export function MenuIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M4 7h16M4 12h16M4 17h16" />
    </Svg>
  );
}

export function BackIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M19 12H5M11 6l-6 6 6 6" />
    </Svg>
  );
}

export function PinIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M9 4h6l-1 7 3 3v2H7v-2l3-3-1-7Z" />
      <path d="M12 16v5" />
    </Svg>
  );
}

export function MuteIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M11 5 6 9H3v6h3l5 4V5Z" />
      <path d="m16 9 5 6M21 9l-5 6" />
    </Svg>
  );
}

export function ImageIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <circle cx="9" cy="10" r="1.6" />
      <path d="m5 18 5-5 3 3 3-3 3 3" />
    </Svg>
  );
}

export function FileIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M6 3h8l4 4v14H6V3Z" />
      <path d="M14 3v4h4M9 12h6M9 16h6" />
    </Svg>
  );
}

export function PlayIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M7 4.5v15l12-7.5-12-7.5Z" />
    </Svg>
  );
}

export function PlusIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M12 5v14M5 12h14" />
    </Svg>
  );
}

export function DotsIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <circle cx="5" cy="12" r="1" fill="currentColor" />
      <circle cx="12" cy="12" r="1" fill="currentColor" />
      <circle cx="19" cy="12" r="1" fill="currentColor" />
    </Svg>
  );
}

export function ChatIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M21 12a8 8 0 0 1-8 8H4l2-3.5A8 8 0 1 1 21 12Z" />
      <path d="M8.5 12h7" />
    </Svg>
  );
}

export function UsersIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <circle cx="9" cy="8" r="3.5" />
      <path d="M2.5 20a6.5 6.5 0 0 1 13 0" />
      <circle cx="17" cy="9" r="2.6" />
      <path d="M15.5 14.6a5.5 5.5 0 0 1 6 4.4" />
    </Svg>
  );
}

export function UserIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <circle cx="12" cy="8" r="4" />
      <path d="M4.5 20a7.5 7.5 0 0 1 15 0" />
    </Svg>
  );
}

export function PaletteIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M12 3a9 9 0 1 0 0 18c1.5 0 2-.8 2-1.8 0-1.4-1.2-1.7-1.2-2.7 0-1 .8-1.7 2-1.7H17a4.5 4.5 0 0 0 4.5-4.5C21.5 6 17.5 3 12 3Z" />
      <circle cx="7.5" cy="11.5" r="1" fill="currentColor" />
      <circle cx="12" cy="7.5" r="1" fill="currentColor" />
      <circle cx="16.5" cy="11.5" r="1" fill="currentColor" />
    </Svg>
  );
}

export function LogoutIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" />
    </Svg>
  );
}

/** Feed glyph for the timeline nav item (§2ac): stacked post rows. */
export function FeedIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <circle cx="8" cy="8.5" r="1.6" />
      <path d="M12 8.5h6M5.5 14h13M5.5 17.5h8" />
    </Svg>
  );
}

/** Sticker tile icon (Sticker, §2ah): a square tile with a smile. Pending design review. */
export function StickerIcon({ size }: IconProps): ReactElement {
  return (
    <Svg size={size}>
      <rect x="3.5" y="3.5" width="17" height="17" rx="3.5" />
      <path d="M8.5 14.5a4.5 4.5 0 0 0 7 0" />
      <circle cx="9" cy="9.5" r="0.6" fill="currentColor" />
      <circle cx="15" cy="9.5" r="0.6" fill="currentColor" />
    </Svg>
  );
}
