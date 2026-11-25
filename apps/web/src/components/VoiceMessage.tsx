import { useEffect, useRef, useState } from 'react';
import type { ReactElement } from 'react';
import { PlayIcon } from './icons';

export interface VoiceMessageProps {
  durationSeconds: number;
  label: string;
}

/**
 * Voice message playback control (UI prototype).
 * Phase 02 has no audio files or backend: the control animates playback
 * progress locally and is labelled as demonstration data in the UI.
 */
export function VoiceMessage({ durationSeconds, label }: VoiceMessageProps): ReactElement {
  const [playing, setPlaying] = useState(false);
  const [elapsed, setElapsed] = useState(0);
  const timer = useRef<number | null>(null);

  useEffect(() => {
    if (!playing) {
      return undefined;
    }
    timer.current = window.setInterval(() => {
      setElapsed((current) => {
        if (current + 1 >= durationSeconds) {
          setPlaying(false);
          return durationSeconds;
        }
        return current + 1;
      });
    }, 1000);
    return () => {
      if (timer.current !== null) {
        window.clearInterval(timer.current);
      }
    };
  }, [playing, durationSeconds]);

  const toggle = (): void => {
    if (playing) {
      setPlaying(false);
      return;
    }
    if (elapsed >= durationSeconds) {
      setElapsed(0);
    }
    setPlaying(true);
  };

  const format = (seconds: number): string => {
    const minutes = Math.floor(seconds / 60);
    const rest = seconds % 60;
    return `${minutes}:${rest.toString().padStart(2, '0')}`;
  };

  const progress = durationSeconds === 0 ? 0 : Math.min(100, (elapsed / durationSeconds) * 100);

  return (
    <span className="et-voice">
      <button
        type="button"
        className="et-voice__play"
        aria-label={playing ? `暫停語音訊息，${label}` : `播放語音訊息，${label}`}
        aria-pressed={playing}
        onClick={toggle}
      >
        {playing ? <span className="et-voice__pause" aria-hidden="true" /> : <PlayIcon size={16} />}
      </button>
      <span className="et-voice__track" aria-hidden="true">
        <span className="et-voice__progress" style={{ width: `${progress}%` }} />
      </span>
      <span className="et-voice__time">
        {format(playing || elapsed > 0 ? elapsed : durationSeconds)}
      </span>
    </span>
  );
}
