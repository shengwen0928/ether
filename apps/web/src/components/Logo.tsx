import type { ReactElement } from 'react';

/**
 * Provisional ETHER logomark (Phase 02 placeholder implementation).
 *
 * Direction: an abstract E whose strokes are crossed by one continuous
 * thread line — connection and memory. This SVG is a replaceable stand-in
 * built from the brand brief; it is NOT the approved final logo and no
 * trademark claim is made for it. Swap the paths when the final artwork
 * is approved; the component API stays the same.
 */

interface LogoMarkProps {
  size?: number;
  monochrome?: boolean;
  label?: string;
}

export function LogoMark({ size = 32, monochrome = false }: LogoMarkProps): ReactElement {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 48 48"
      fill="none"
      role="img"
      aria-label="ETHER 標誌"
      focusable="false"
    >
      <rect width="48" height="48" rx="11" fill={monochrome ? 'currentColor' : '#101923'} />
      <path
        d="M15 13h18M15 24h14.5M15 35h18"
        stroke={monochrome ? '#101923' : '#86b9d2'}
        strokeWidth="3"
        strokeLinecap="round"
      />
      <path
        d="M15 13v22"
        stroke={monochrome ? '#101923' : '#f4f7f9'}
        strokeWidth="3"
        strokeLinecap="round"
      />
      <path
        d="M12 30c6-8 12 2 18-4s8 4 12-2"
        stroke={monochrome ? '#101923' : '#65d4d6'}
        strokeWidth="2"
        strokeLinecap="round"
      />
      <circle cx="34" cy="24" r="2.4" fill={monochrome ? '#101923' : '#d9b98c'} />
    </svg>
  );
}

interface LogoLockupProps {
  size?: number;
  direction?: 'horizontal' | 'vertical';
  showWordmark?: boolean;
}

export function LogoLockup({
  size = 32,
  direction = 'horizontal',
  showWordmark = true,
}: LogoLockupProps): ReactElement {
  return (
    <span
      className={direction === 'vertical' ? 'et-logo et-logo--vertical' : 'et-logo'}
      aria-label="ETHER"
    >
      <LogoMark size={size} />
      {showWordmark ? (
        <span className="et-logo__word">
          ETHER
          <span className="et-logo__tagline">One Connection. Endless Possibilities.</span>
        </span>
      ) : null}
    </span>
  );
}
