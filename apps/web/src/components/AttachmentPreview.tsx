import type { ReactElement } from 'react';
import { FileIcon, ImageIcon } from './icons';

export interface AttachmentPreviewProps {
  fileName: string;
  fileSize?: string | undefined;
  isImage?: boolean;
  onDownload?: (() => void) | undefined;
}

export function AttachmentPreview({
  fileName,
  fileSize,
  isImage = false,
  onDownload,
}: AttachmentPreviewProps): ReactElement {
  return (
    <span className="et-attachment">
      <span className="et-attachment__icon" aria-hidden="true">
        {isImage ? <ImageIcon size={20} /> : <FileIcon size={20} />}
      </span>
      <span className="et-attachment__meta">
        <span className="et-attachment__name">{fileName}</span>
        {fileSize !== undefined ? <span className="et-attachment__size">{fileSize}</span> : null}
      </span>
      {onDownload !== undefined ? (
        <button type="button" className="et-attachment__download" onClick={onDownload}>
          下載
        </button>
      ) : null}
    </span>
  );
}
