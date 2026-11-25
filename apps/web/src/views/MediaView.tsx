import { useCallback, useEffect, useRef, useState } from 'react';
import type { ChangeEvent, ReactElement } from 'react';
import type { ChatDataProvider } from '../data/provider';
import type {
  InitiateUploadInput,
  MediaAuthorization,
  MediaFileItem,
  MediaThumbnailItem,
  MediaUploadItem,
} from '../data/types';
import { Badge } from '../components/Badge';
import { Button } from '../components/Button';
import { Dialog } from '../components/Dialog';
import { EmptyState } from '../components/EmptyState';
import { useToast } from '../components/Toast';
import { downloadBlob, openDownloadUrl } from './downloadFile';

export interface MediaViewProps {
  provider: ChatDataProvider;
  dataMode?: 'demo' | 'api';
}

type LoadState = 'loading' | 'ready' | 'error' | 'unsupported';

interface PendingConfirm {
  title: string;
  description: string;
  confirmLabel: string;
  danger: boolean;
  action: () => Promise<void>;
}

/**
 * 後端 MIME 白名單（見 services/api/internal/media/media.go）。
 * 檔案選擇器以此過濾；瀏覽器未回報 MIME 類型時拒絕，不得以
 * 副檔名代替 MIME 判斷（與後端規則一致）。
 */
const ALLOWED_MIME_TYPES = new Set([
  'image/jpeg',
  'image/png',
  'image/webp',
  'image/gif',
  'video/mp4',
  'video/quicktime',
  'video/webm',
  'audio/mpeg',
  'audio/mp4',
  'audio/ogg',
  'audio/webm',
  'application/pdf',
]);

/** 單檔上限 100 MiB（後端實作預設，見 STATUS.md 待決定 18）。 */
const MAX_FILE_SIZE_BYTES = 100 * 1024 * 1024;

function scanLabel(scanStatus: string): { label: string; tone: 'success' | 'warning' | 'error' } {
  if (scanStatus === 'clean') {
    return { label: '掃描通過', tone: 'success' };
  }
  if (scanStatus === 'rejected') {
    return { label: '已拒絕，不可分發', tone: 'error' };
  }
  if (scanStatus === 'pending') {
    return { label: '掃描中，不可分發', tone: 'warning' };
  }
  return { label: `掃描狀態：${scanStatus}，不可分發`, tone: 'warning' };
}

function uploadLabel(status: string): {
  label: string;
  tone: 'accent' | 'success' | 'warning' | 'error' | 'neutral';
} {
  if (status === 'uploading') {
    return { label: '上傳中', tone: 'accent' };
  }
  if (status === 'completed') {
    return { label: '已完成', tone: 'success' };
  }
  if (status === 'failed') {
    return { label: '失敗', tone: 'error' };
  }
  if (status === 'expired') {
    return { label: '已逾期', tone: 'warning' };
  }
  if (status === 'cancelled') {
    return { label: '已取消', tone: 'neutral' };
  }
  return { label: status, tone: 'neutral' };
}

function formatBytes(sizeBytes: number): string {
  if (sizeBytes < 1024) {
    return `${sizeBytes} B`;
  }
  if (sizeBytes < 1024 * 1024) {
    return `${(sizeBytes / 1024).toFixed(1)} KB`;
  }
  return `${(sizeBytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatDateTime(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString('zh-Hant');
}

function dimensionsLabel(file: MediaFileItem): string {
  if (file.width > 0 && file.height > 0) {
    return `${file.width}×${file.height}`;
  }
  if (file.durationSeconds > 0) {
    return `${file.durationSeconds} 秒`;
  }
  return '';
}

/**
 * Media view (Phase 06 media, web wiring): file list, upload
 * sessions, thumbnail records and the file-side authorization
 * check. The same component serves demo mode (local fixtures,
 * labelled as such) and API mode (backend data); the provider
 * methods are optional, so absence renders the unsupported state
 * instead of an error.
 *
 * Honesty rules: uploading sends the file bytes through the
 * issued upload URL and only then completes the session, so a
 * completed session really holds bytes (the backend verifies
 * this with a Stat check). The scan gate decides
 * distributability: pending and rejected files are always shown
 * as not distributable. Thumbnails are metadata records only;
 * the backend returns no distribution URL, so they render as
 * placeholders, never as openable images.
 */
export function MediaView({ provider, dataMode = 'demo' }: MediaViewProps): ReactElement {
  const { notify } = useToast();
  const [loadState, setLoadState] = useState<LoadState>('loading');
  const [files, setFiles] = useState<MediaFileItem[]>([]);
  const [uploads, setUploads] = useState<MediaUploadItem[]>([]);
  const [thumbnails, setThumbnails] = useState<Record<string, MediaThumbnailItem[]>>({});
  const [authorizations, setAuthorizations] = useState<Record<string, MediaAuthorization>>({});
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState('');
  const [acting, setActing] = useState(false);
  // 進行中的上傳流程：位元組傳輸階段文字與失敗後重試所需的
  // 會話＋檔案（重試時重新核發上傳 URL，不重建會話）。
  const [uploadPhase, setUploadPhase] = useState<string | null>(null);
  const [uploadError, setUploadError] = useState('');
  const [pendingUpload, setPendingUpload] = useState<{
    upload: MediaUploadItem;
    file: File;
  } | null>(null);
  const [pendingConfirm, setPendingConfirm] = useState<PendingConfirm | null>(null);
  const mountedRef = useRef(true);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const load = useCallback(async (): Promise<void> => {
    const listFiles = provider.listMediaFilesAsync;
    const listUploads = provider.listMediaUploadsAsync;
    const listThumbnails = provider.listMediaThumbnailsAsync;
    if (listFiles === undefined || listUploads === undefined) {
      if (mountedRef.current) {
        setLoadState('unsupported');
      }
      return;
    }
    if (mountedRef.current) {
      setLoadState('loading');
    }
    try {
      const [fetchedFiles, fetchedUploads] = await Promise.all([listFiles(), listUploads()]);
      let fetchedThumbnails: Record<string, MediaThumbnailItem[]> = {};
      if (listThumbnails !== undefined) {
        // 縮圖載入失敗不拖垮整個清單：單一檔案失敗時以空清單計。
        const entries = await Promise.all(
          fetchedFiles.map(async (file) => {
            try {
              const rows = await listThumbnails(file.id);
              return [file.id, rows] as const;
            } catch {
              return [file.id, []] as const;
            }
          }),
        );
        fetchedThumbnails = Object.fromEntries(entries);
      }
      if (!mountedRef.current) {
        return;
      }
      setFiles(fetchedFiles);
      setUploads(fetchedUploads);
      setThumbnails(fetchedThumbnails);
      setLoadState('ready');
    } catch {
      if (mountedRef.current) {
        setLoadState('error');
      }
    }
  }, [provider]);

  useEffect(() => {
    void load();
  }, [load]);

  const confirmPending = async (): Promise<void> => {
    if (pendingConfirm === null) {
      return;
    }
    const target = pendingConfirm;
    setPendingConfirm(null);
    setActing(true);
    try {
      await target.action();
      notify({ title: '已完成', message: target.confirmLabel, tone: 'success' });
      await load();
    } catch (error) {
      notify({
        title: '操作失敗',
        message: error instanceof Error ? error.message : '請再試一次。',
        tone: 'error',
      });
    } finally {
      if (mountedRef.current) {
        setActing(false);
      }
    }
  };

  const onSelectFile = (event: ChangeEvent<HTMLInputElement>): void => {
    const picked = event.target.files?.[0] ?? null;
    setFileError('');
    if (picked === null) {
      setSelectedFile(null);
      return;
    }
    if (picked.type === '' || !ALLOWED_MIME_TYPES.has(picked.type)) {
      setSelectedFile(null);
      setFileError('不支援的檔案類型：僅接受圖片、影片、音訊與 PDF，且須由瀏覽器判斷 MIME 類型。');
      return;
    }
    if (picked.size > MAX_FILE_SIZE_BYTES) {
      setSelectedFile(null);
      setFileError('檔案超過 100 MiB 上限。');
      return;
    }
    if ([...picked.name].length > 255) {
      setSelectedFile(null);
      setFileError('檔名超過 255 字。');
      return;
    }
    setSelectedFile(picked);
  };

  /**
   * Byte upload flow (§2aa): PUT the bytes to the issued upload
   * URL, report progress (the backend refuses completion until
   * the reported progress equals the declared total), then
   * complete with the same metadata. A 403 from the PUT (expired
   * presigned signature) earns exactly one re-issue and retry;
   * any other failure keeps the session so the user can retry.
   */
  const runUploadFlow = async (
    session: MediaUploadItem,
    file: File,
    firstUploadUrl: string | null,
  ): Promise<void> => {
    const putBytes = provider.putMediaUploadBytesAsync;
    const reportProgress = provider.reportMediaUploadProgressAsync;
    const complete = provider.completeMediaUploadAsync;
    const issueUrl = provider.issueMediaUploadUrlAsync;
    if (
      putBytes === undefined ||
      reportProgress === undefined ||
      complete === undefined ||
      issueUrl === undefined
    ) {
      notify({ title: '不支援', message: '目前的資料模式不提供位元組上傳。', tone: 'warning' });
      return;
    }
    const input: InitiateUploadInput = {
      mimeType: file.type,
      totalBytes: file.size,
      originalFilename: file.name,
      // 寬高與秒數未知時以 0 表示（後端以 0 為未知）；校驗值待定稿，不提供。
      width: 0,
      height: 0,
      durationSeconds: 0,
      checksum: '',
    };
    setActing(true);
    setUploadError('');
    try {
      let uploadUrl = firstUploadUrl;
      if (uploadUrl === null || uploadUrl === '') {
        const issued = await issueUrl(session.id);
        uploadUrl = issued.uploadUrl;
      }
      if (mountedRef.current) {
        setUploadPhase('正在上傳檔案位元組…');
      }
      try {
        await putBytes(uploadUrl, file);
      } catch (error) {
        const status = (error as { status?: unknown }).status;
        if (status !== 403) {
          throw error;
        }
        const issued = await issueUrl(session.id);
        await putBytes(issued.uploadUrl, file);
      }
      if (mountedRef.current) {
        setUploadPhase('正在回報上傳進度…');
      }
      // 單次 PUT 語義：分段數為 1（實作預設，待定稿覆核）。
      await reportProgress(session.id, file.size, 1);
      if (mountedRef.current) {
        setUploadPhase('正在完成上傳…');
      }
      await complete(session.id, input);
      notify({
        title: '上傳完成',
        message: '檔案位元組已送達，等待掃描。',
        tone: 'success',
      });
      if (mountedRef.current) {
        setPendingUpload(null);
        setSelectedFile(null);
      }
      if (fileInputRef.current !== null) {
        fileInputRef.current.value = '';
      }
      await load();
    } catch (error) {
      const message = error instanceof Error ? error.message : '請再試一次。';
      if (mountedRef.current) {
        setUploadError(message);
      }
      notify({ title: '上傳失敗', message, tone: 'error' });
    } finally {
      if (mountedRef.current) {
        setActing(false);
        setUploadPhase(null);
      }
    }
  };

  const createUploadSession = async (): Promise<void> => {
    if (selectedFile === null) {
      return;
    }
    const initiate = provider.initiateMediaUploadAsync;
    if (
      initiate === undefined ||
      provider.putMediaUploadBytesAsync === undefined ||
      provider.reportMediaUploadProgressAsync === undefined ||
      provider.completeMediaUploadAsync === undefined ||
      provider.issueMediaUploadUrlAsync === undefined
    ) {
      notify({ title: '不支援', message: '目前的資料模式不提供上傳。', tone: 'warning' });
      return;
    }
    const input: InitiateUploadInput = {
      mimeType: selectedFile.type,
      totalBytes: selectedFile.size,
      originalFilename: selectedFile.name,
      // 寬高與秒數未知時以 0 表示（後端以 0 為未知）；校驗值待定稿，不提供。
      width: 0,
      height: 0,
      durationSeconds: 0,
      checksum: '',
    };
    setActing(true);
    try {
      const session = await initiate(input);
      if (mountedRef.current) {
        setPendingUpload({ upload: session, file: selectedFile });
      }
      await runUploadFlow(session, selectedFile, session.uploadUrl);
    } catch (error) {
      notify({
        title: '建立失敗',
        message: error instanceof Error ? error.message : '請再試一次。',
        tone: 'error',
      });
      if (mountedRef.current) {
        setActing(false);
      }
    }
  };

  const retryUpload = async (): Promise<void> => {
    if (pendingUpload === null) {
      return;
    }
    await runUploadFlow(pendingUpload.upload, pendingUpload.file, null);
  };

  const downloadFile = async (file: MediaFileItem): Promise<void> => {
    const resolve = provider.resolveMediaDownloadAsync;
    if (resolve === undefined) {
      notify({ title: '不支援', message: '目前的資料模式不提供下載。', tone: 'warning' });
      return;
    }
    try {
      const result = await resolve(file.id);
      if (mountedRef.current) {
        setAuthorizations((previous) => ({
          ...previous,
          [file.id]: {
            allowed: result.status !== 'forbidden',
            scanStatus: result.status === 'forbidden' ? result.scanStatus : file.scanStatus,
            downloadUrl: result.status === 'ready-url' ? result.url : null,
            downloadUrlExpiresAt: null,
          },
        }));
      }
      if (result.status === 'forbidden') {
        notify({
          title: '不可下載',
          message: `掃描狀態為 ${result.scanStatus}，檔案不可分發。`,
          tone: 'warning',
        });
        return;
      }
      if (result.status === 'unavailable') {
        notify({
          title: '無法下載',
          message: '已授權，但此部署未提供可下載的檔案位址。',
          tone: 'warning',
        });
        return;
      }
      if (result.status === 'ready-url') {
        openDownloadUrl(result.url);
        return;
      }
      downloadBlob(result.blob, file.originalFilename);
    } catch (error) {
      notify({
        title: '下載失敗',
        message: error instanceof Error ? error.message : '請再試一次。',
        tone: 'error',
      });
    }
  };

  const sourceLabel = dataMode === 'api' ? '檔案與上傳會話來自後端 API' : '展示資料 · 尚未連接後端';

  return (
    <div className="et-page et-scroll">
      <div className="et-page__narrow">
        <header>
          <h2 className="et-page__title">媒體</h2>
          <p className="et-page__description">{sourceLabel}</p>
        </header>
        {loadState === 'loading' ? <p className="et-page__description">載入媒體檔案中…</p> : null}
        {loadState === 'error' ? (
          <>
            <p className="et-page__description">載入失敗，請檢查連線後再試一次。</p>
            <Button
              onClick={() => {
                void load();
              }}
            >
              重新載入
            </Button>
          </>
        ) : null}
        {loadState === 'unsupported' ? (
          <p className="et-page__description">目前的資料模式不提供媒體檔案。</p>
        ) : null}
        {loadState === 'ready' ? (
          <>
            <section className="et-card" aria-labelledby="et-media-upload-heading">
              <h3 className="et-card__title" id="et-media-upload-heading">
                建立上傳
              </h3>
              <p className="et-card__subtitle">
                選擇檔案後會建立上傳會話、傳送檔案位元組、回報進度並完成上傳；完成後檔案進入掃描，掃描未通過的檔案一律不可分發。
              </p>
              <div className="et-form-grid">
                <div className="et-field">
                  <label className="et-field__label" htmlFor="et-media-file-input">
                    選擇檔案
                  </label>
                  <input
                    id="et-media-file-input"
                    ref={fileInputRef}
                    type="file"
                    accept={[...ALLOWED_MIME_TYPES].join(',')}
                    onChange={onSelectFile}
                  />
                </div>
              </div>
              {selectedFile !== null ? (
                <p className="et-page__description">
                  已選擇：{selectedFile.name}（{selectedFile.type}，{formatBytes(selectedFile.size)}
                  ）
                </p>
              ) : null}
              {fileError !== '' ? (
                <p className="et-page__description" role="alert">
                  {fileError}
                </p>
              ) : null}
              {uploadPhase !== null ? (
                <p className="et-page__description" role="status">
                  {uploadPhase}
                </p>
              ) : null}
              {uploadError !== '' ? (
                <p className="et-page__description" role="alert">
                  上傳失敗：{uploadError}
                </p>
              ) : null}
              <Button
                disabled={selectedFile === null || acting}
                loading={acting}
                onClick={() => {
                  void createUploadSession();
                }}
              >
                上傳檔案
              </Button>
              {pendingUpload !== null && uploadError !== '' ? (
                <Button
                  variant="secondary"
                  disabled={acting}
                  onClick={() => {
                    void retryUpload();
                  }}
                >
                  重試上傳
                </Button>
              ) : null}
            </section>

            <section className="et-card" aria-labelledby="et-media-uploads-heading">
              <h3 className="et-card__title" id="et-media-uploads-heading">
                上傳會話
              </h3>
              {uploads.length === 0 ? (
                <EmptyState title="沒有上傳會話" description="建立上傳會話後會列在這裡。" />
              ) : (
                <ul className="et-device-list">
                  {uploads.map((upload) => {
                    const status = uploadLabel(upload.status);
                    return (
                      <li className="et-device-row" key={upload.id}>
                        <div className="et-device-row__main">
                          <span className="et-device-row__name">
                            {upload.mediaFileId !== '' ? upload.mediaFileId : '未關聯檔案'}
                          </span>
                          <span className="et-device-row__meta">
                            {formatBytes(upload.uploadedBytes)} / {formatBytes(upload.totalBytes)}
                            {' · '}
                            {upload.partCount} 個分段
                            {' · '}
                            物件鍵：{upload.storageKey}
                            {' · '}
                            {formatDateTime(upload.createdAt)}
                          </span>
                        </div>
                        <Badge tone={status.tone}>{status.label}</Badge>
                        {upload.status === 'uploading' &&
                        provider.cancelMediaUploadAsync !== undefined ? (
                          <Button
                            size="sm"
                            variant="secondary"
                            disabled={acting}
                            onClick={() => {
                              setPendingConfirm({
                                title: '取消上傳會話',
                                description: '取消後此會話無法繼續，檔案位元組不會送達。',
                                confirmLabel: '已取消上傳會話',
                                danger: true,
                                action: async () => {
                                  await provider.cancelMediaUploadAsync?.(upload.id);
                                },
                              });
                            }}
                          >
                            取消
                          </Button>
                        ) : null}
                      </li>
                    );
                  })}
                </ul>
              )}
            </section>

            <section className="et-card" aria-labelledby="et-media-files-heading">
              <h3 className="et-card__title" id="et-media-files-heading">
                我的檔案
              </h3>
              {files.length === 0 ? (
                <EmptyState title="沒有檔案" description="完成掃描的檔案會列在這裡。" />
              ) : (
                <ul className="et-device-list">
                  {files.map((file) => {
                    const scan = scanLabel(file.scanStatus);
                    const authorization = authorizations[file.id];
                    const rows = thumbnails[file.id] ?? [];
                    const dimensions = dimensionsLabel(file);
                    return (
                      <li className="et-device-row" key={file.id}>
                        <div className="et-device-row__main">
                          <span className="et-device-row__name">{file.originalFilename}</span>
                          <span className="et-device-row__meta">
                            {file.mimeType}
                            {' · '}
                            {formatBytes(file.sizeBytes)}
                            {dimensions !== '' ? ` · ${dimensions}` : ''}
                            {' · '}
                            {formatDateTime(file.createdAt)}
                          </span>
                          {rows.length > 0 ? (
                            <span className="et-device-row__meta">
                              縮圖（僅詮釋資料，尚無分發位址）：
                              {rows.map((row) => (
                                <Badge key={row.id} tone="neutral">
                                  {row.variantType} {row.width}×{row.height}
                                </Badge>
                              ))}
                            </span>
                          ) : (
                            <span className="et-device-row__meta">尚無縮圖紀錄</span>
                          )}
                          {authorization !== undefined ? (
                            <span className="et-device-row__meta">
                              授權檢查：{authorization.allowed ? '可存取' : '不可存取'}
                              （掃描狀態：{authorization.scanStatus}）
                            </span>
                          ) : null}
                        </div>
                        <Badge tone={scan.tone}>{scan.label}</Badge>
                        <Button
                          size="sm"
                          variant="secondary"
                          onClick={() => {
                            void downloadFile(file);
                          }}
                        >
                          下載
                        </Button>
                        {provider.deleteMediaFileAsync !== undefined ? (
                          <Button
                            size="sm"
                            variant="danger"
                            disabled={acting}
                            onClick={() => {
                              setPendingConfirm({
                                title: '刪除檔案',
                                description: `確定要刪除「${file.originalFilename}」嗎？此操作會將檔案標記為已刪除。`,
                                confirmLabel: '已刪除檔案',
                                danger: true,
                                action: async () => {
                                  await provider.deleteMediaFileAsync?.(file.id);
                                },
                              });
                            }}
                          >
                            刪除
                          </Button>
                        ) : null}
                      </li>
                    );
                  })}
                </ul>
              )}
            </section>
          </>
        ) : null}
        {pendingConfirm !== null ? (
          <Dialog
            open={pendingConfirm !== null}
            onClose={() => {
              setPendingConfirm(null);
            }}
            title={pendingConfirm.title}
            description={pendingConfirm.description}
            footer={
              <>
                <Button
                  variant="secondary"
                  onClick={() => {
                    setPendingConfirm(null);
                  }}
                >
                  取消
                </Button>
                <Button
                  variant={pendingConfirm.danger ? 'danger' : 'primary'}
                  disabled={acting}
                  onClick={() => {
                    void confirmPending();
                  }}
                >
                  確認
                </Button>
              </>
            }
          >
            <p className="et-page__description">此操作會立即生效。</p>
          </Dialog>
        ) : null}
      </div>
    </div>
  );
}
