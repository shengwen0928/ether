import { useEffect, useRef, useState } from 'react';
import type { FormEvent, KeyboardEvent, ReactElement } from 'react';
import { PaperclipIcon, SendIcon, SmileIcon } from './icons';
import { IconButton } from './IconButton';
import { Tooltip } from './Tooltip';

export interface MessageComposerProps {
  placeholder?: string;
  disabled?: boolean;
  onSend: (body: string) => void;
  onAttach?: () => void;
  onEmoji?: () => void;
}

export function MessageComposer({
  placeholder = '輸入訊息…',
  disabled = false,
  onSend,
  onAttach,
  onEmoji,
}: MessageComposerProps): ReactElement {
  const [value, setValue] = useState('');
  const [rows, setRows] = useState(1);
  const areaRef = useRef<HTMLTextAreaElement | null>(null);

  useEffect(() => {
    const area = areaRef.current;
    if (area === null) {
      return;
    }
    area.style.height = 'auto';
    const nextRows = Math.min(5, Math.max(1, Math.ceil(area.scrollHeight / 24)));
    setRows(nextRows);
  }, [value]);

  const submit = (): void => {
    const trimmed = value.trim();
    if (trimmed === '' || disabled) {
      return;
    }
    onSend(trimmed);
    setValue('');
  };

  const onSubmit = (event: FormEvent): void => {
    event.preventDefault();
    submit();
  };

  const onKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>): void => {
    if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault();
      submit();
    }
  };

  const canSend = value.trim() !== '' && !disabled;

  return (
    <form className="et-composer" onSubmit={onSubmit} aria-label="訊息輸入區">
      {onAttach !== undefined ? (
        <Tooltip tip="附加檔案（展示）">
          <IconButton label="附加檔案" onClick={onAttach} disabled={disabled}>
            <PaperclipIcon size={20} />
          </IconButton>
        </Tooltip>
      ) : null}
      <label className="et-sr-only" htmlFor="et-composer-input">
        輸入訊息
      </label>
      <textarea
        id="et-composer-input"
        ref={areaRef}
        className="et-composer__input et-scroll"
        value={value}
        rows={rows}
        placeholder={placeholder}
        disabled={disabled}
        aria-label="輸入訊息，Enter 傳送，Shift 加 Enter 換行"
        onChange={(event) => {
          setValue(event.target.value);
        }}
        onKeyDown={onKeyDown}
      />
      {onEmoji !== undefined ? (
        <Tooltip tip="表情符號（展示）">
          <IconButton label="插入表情符號" onClick={onEmoji} disabled={disabled}>
            <SmileIcon size={20} />
          </IconButton>
        </Tooltip>
      ) : null}
      <button type="submit" className="et-composer__send" disabled={!canSend} aria-label="傳送訊息">
        <SendIcon size={18} />
      </button>
    </form>
  );
}
