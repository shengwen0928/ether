import { useState } from 'react';
import type { ReactElement } from 'react';
import { initialsOf, presenceLabel } from '../data/logic';
import type { PresenceStatus } from '../data/types';

export type AvatarSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl';

export interface AvatarProps {
  name: string;
  size?: AvatarSize;
  imageUrl?: string;
  presence?: PresenceStatus;
}

const SIZE_PX: Record<AvatarSize, number> = {
  xs: 24,
  sm: 32,
  md: 40,
  lg: 48,
  xl: 64,
};

export function Avatar({ name, size = 'md', imageUrl, presence }: AvatarProps): ReactElement {
  const [imageFailed, setImageFailed] = useState(false);
  const px = SIZE_PX[size];
  return (
    <span
      className={`et-avatar et-avatar--${size}`}
      style={{ width: px, height: px }}
      role="img"
      aria-label={presence !== undefined ? `${name}，${presenceLabel(presence)}` : name}
    >
      {imageUrl !== undefined && !imageFailed ? (
        <img
          className="et-avatar__img"
          src={imageUrl}
          alt=""
          width={px}
          height={px}
          onError={() => {
            setImageFailed(true);
          }}
        />
      ) : (
        <span className="et-avatar__initials" aria-hidden="true">
          {initialsOf(name)}
        </span>
      )}
      {presence !== undefined ? (
        <span
          className={`et-avatar__presence et-avatar__presence--${presence}`}
          aria-hidden="true"
        />
      ) : null}
    </span>
  );
}
