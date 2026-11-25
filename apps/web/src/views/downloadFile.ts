/**
 * Browser download helpers shared by the media view and the
 * message attachment block. Both functions only run from user
 * gestures (click handlers), never during render.
 */

/** Open an absolute URL in a new tab (presigned download URLs). */
export function openDownloadUrl(url: string): void {
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.target = '_blank';
  anchor.rel = 'noopener noreferrer';
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
}

/** Save fetched bytes as a file download under the given name. */
export function downloadBlob(blob: Blob, filename: string): void {
  const objectUrl = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = objectUrl;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(objectUrl);
}
