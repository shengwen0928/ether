import { useCallback, useEffect, useRef, useState } from 'react';
import type { ChangeEvent, ReactElement } from 'react';
import type { ChatDataProvider } from '../data/provider';
import type {
  AddStickerInput,
  CreateStickerPackInput,
  MediaFileItem,
  StickerItem,
  StickerPackCategory,
  StickerPackItem,
  StickerPackStatus,
  UpdateStickerPackInput,
} from '../data/types';
import { Badge } from '../components/Badge';
import { Button } from '../components/Button';
import { Checkbox } from '../components/Checkbox';
import { Dialog } from '../components/Dialog';
import { EmptyState } from '../components/EmptyState';
import { Input, TextArea } from '../components/Input';
import { Select } from '../components/Select';
import { useToast } from '../components/Toast';
import { StickerIcon } from '../components/icons';

export interface StickersViewProps {
  provider: ChatDataProvider;
  dataMode?: 'demo' | 'api';
}

type LoadState = 'loading' | 'ready' | 'error' | 'unsupported';
type DetailState = 'loading' | 'ready' | 'error' | 'unsupported';
type FilesState = 'loading' | 'ready' | 'error' | 'unsupported';
type StickerTab = 'browse' | 'favorites' | 'recent';

const PACK_PAGE_LIMIT = 20;

const CATEGORY_LABELS: Record<StickerPackCategory, string> = {
  '': '未分類',
  general: '一般',
  animals: '動物',
  food: '食物',
  emotions: '情緒',
  festivals: '節慶',
  characters: '角色',
  other: '其他',
};

const STATUS_LABELS: Record<StickerPackStatus, string> = {
  draft: '草稿',
  published: '已上架',
  unlisted: '未上架',
  archived: '已封存',
};

const CATEGORY_OPTIONS: { value: string; label: string }[] = [
  { value: 'all', label: '全部分類' },
  { value: 'general', label: '一般' },
  { value: 'animals', label: '動物' },
  { value: 'food', label: '食物' },
  { value: 'emotions', label: '情緒' },
  { value: 'festivals', label: '節慶' },
  { value: 'characters', label: '角色' },
  { value: 'other', label: '其他' },
];

const PACK_FORM_CATEGORY_OPTIONS: { value: string; label: string }[] = [
  { value: '', label: '未分類' },
  { value: 'general', label: '一般' },
  { value: 'animals', label: '動物' },
  { value: 'food', label: '食物' },
  { value: 'emotions', label: '情緒' },
  { value: 'festivals', label: '節慶' },
  { value: 'characters', label: '角色' },
  { value: 'other', label: '其他' },
];

const STATUS_OPTIONS: { value: string; label: string }[] = [
  { value: 'draft', label: '草稿' },
  { value: 'published', label: '已上架' },
  { value: 'unlisted', label: '未上架' },
];

/** 貼圖包分類中文標籤（與後端分類詞彙對應，待定稿覆核）。 */
export function stickerPackCategoryLabel(category: string): string {
  return CATEGORY_LABELS[category as StickerPackCategory] ?? '未分類';
}

/** 貼圖包狀態中文標籤（與後端狀態詞彙對應，待定稿覆核）。 */
export function stickerPackStatusLabel(status: StickerPackStatus): string {
  return STATUS_LABELS[status];
}

/** 作者管理權判定：只有建立者可編輯、刪除與增減貼圖。 */
export function canManageStickerPack(pack: StickerPackItem, myId: string | null): boolean {
  return myId !== null && pack.creatorId === myId;
}

/** 貼圖包名稱檢查（與後端上限同規則：不得為空、64 runes）。 */
export function validateStickerPackName(name: string): string {
  if (name.trim() === '') {
    return '貼圖包名稱不得為空。';
  }
  if ([...name].length > 64) {
    return '貼圖包名稱過長。';
  }
  return '';
}

/** 貼圖包描述檢查（與後端上限同規則：500 runes）。 */
export function validateStickerPackDescription(description: string): string {
  if ([...description].length > 500) {
    return '貼圖包描述過長。';
  }
  return '';
}

/** 貼圖名稱檢查（與後端上限同規則：64 runes；空名稱後端未拒絕，沿用）。 */
export function validateStickerName(name: string): string {
  if ([...name].length > 64) {
    return '貼圖名稱過長。';
  }
  return '';
}

function formatDateTime(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString('zh-Hant');
}

function scanNote(scanStatus: string): string {
  if (scanStatus === 'clean') {
    return '掃描通過';
  }
  if (scanStatus === 'rejected') {
    return '已拒絕，不可關聯';
  }
  return '掃描中，不可關聯';
}

interface StickerPackCardProps {
  pack: StickerPackItem;
  creatorName: string;
  selected: boolean;
  favoriteBusy: boolean;
  onSelect: (packId: string) => void;
  onToggleFavorite: (pack: StickerPackItem) => void;
}

/**
 * 貼圖包卡片（§2ah）：名稱、作者、分類、免費／付費、狀態、張數
 * 與收藏切換。付費包只在持有授權發放紀錄時才會出現在清單中；
 * 這裡沒有任何購買流程與購買按鈕。
 */
export function StickerPackCard({
  pack,
  creatorName,
  selected,
  favoriteBusy,
  onSelect,
  onToggleFavorite,
}: StickerPackCardProps): ReactElement {
  return (
    <article className="et-sticker-pack" aria-label={`貼圖包，${pack.name}`}>
      <div className="et-sticker-pack__header">
        <span className="et-sticker-pack__cover" aria-hidden="true">
          <StickerIcon size={24} />
        </span>
        <div className="et-call__title-block">
          <p className="et-call__title">{pack.name}</p>
          <p className="et-call__meta">
            {creatorName} · {stickerPackCategoryLabel(pack.category)} ·{' '}
            {pack.isFree ? '免費' : '付費 · 已授權'} · {`${pack.stickerCount} 張貼圖`}
          </p>
        </div>
        <Badge tone={pack.status === 'published' ? 'success' : 'neutral'}>
          {stickerPackStatusLabel(pack.status)}
        </Badge>
      </div>
      {pack.description !== '' ? (
        <p className="et-sticker-pack__description">{pack.description}</p>
      ) : null}
      <div className="et-sticker-pack__actions">
        <Button
          size="sm"
          variant={selected ? 'primary' : 'secondary'}
          onClick={() => {
            onSelect(pack.id);
          }}
        >
          {selected ? '已選取' : '查看貼圖'}
        </Button>
        <Button
          size="sm"
          variant="secondary"
          disabled={favoriteBusy}
          onClick={() => {
            onToggleFavorite(pack);
          }}
        >
          {pack.favoritedByMe ? '取消收藏' : '收藏'}
        </Button>
      </div>
    </article>
  );
}

interface StickerTileProps {
  sticker: StickerItem;
  onRecordUse?: (sticker: StickerItem) => void;
  recordBusy?: boolean;
}

/**
 * 貼圖方塊（§2ah）：只顯示名稱與詮釋資料的佔位方塊。貼圖本體
 * 只有 media_file_id，沒有檔案位元組與分發位址，不得假裝成可
 * 開啟或下載的圖片。
 */
export function StickerTile({
  sticker,
  onRecordUse,
  recordBusy = false,
}: StickerTileProps): ReactElement {
  return (
    <li className="et-sticker-tile" aria-label={`貼圖，${sticker.name}`}>
      <span className="et-sticker-tile__art" aria-hidden="true">
        <StickerIcon size={28} />
      </span>
      <span className="et-sticker-tile__name">
        {sticker.name === '' ? '（未命名）' : sticker.name}
      </span>
      <span className="et-call__meta">
        {sticker.packName}
        {sticker.isAnimated ? ' · 動態' : ''}
      </span>
      {onRecordUse !== undefined ? (
        <Button
          size="sm"
          variant="secondary"
          disabled={recordBusy}
          onClick={() => {
            onRecordUse(sticker);
          }}
        >
          記錄使用
        </Button>
      ) : null}
    </li>
  );
}

interface StickerPackDetailProps {
  pack: StickerPackItem;
  stickers: StickerItem[];
  stickersState: DetailState;
  creatorName: string;
  myId: string | null;
  acting: boolean;
  onToggleFavorite: (pack: StickerPackItem) => void;
  onEdit: (pack: StickerPackItem) => void;
  onDelete: (pack: StickerPackItem) => void;
  onAddSticker: (pack: StickerPackItem) => void;
  onRemoveSticker: (sticker: StickerItem) => void;
  onRecordUse: (sticker: StickerItem) => void;
  onReload: (packId: string) => void;
  onClose: () => void;
}

/**
 * 貼圖包詳情（§2ah）：貼圖格、收藏切換、作者專屬管理操作。
 * 貼圖只顯示名稱與詮釋資料，不提供檔案下載。
 */
export function StickerPackDetailPanel({
  pack,
  stickers,
  stickersState,
  creatorName,
  myId,
  acting,
  onToggleFavorite,
  onEdit,
  onDelete,
  onAddSticker,
  onRemoveSticker,
  onRecordUse,
  onReload,
  onClose,
}: StickerPackDetailProps): ReactElement {
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const manageable = canManageStickerPack(pack, myId);

  return (
    <article className="et-sticker-detail" aria-label={`貼圖包詳情，${pack.name}`}>
      <div className="et-sticker-pack__header">
        <span className="et-sticker-pack__cover" aria-hidden="true">
          <StickerIcon size={24} />
        </span>
        <div className="et-call__title-block">
          <h3 className="et-call__title">{pack.name}</h3>
          <p className="et-call__meta">
            {creatorName} · {stickerPackCategoryLabel(pack.category)} ·{' '}
            {pack.isFree ? '免費' : '付費 · 已授權'} · 建立於 {formatDateTime(pack.createdAt)}
          </p>
        </div>
        <Badge tone={pack.status === 'published' ? 'success' : 'neutral'}>
          {stickerPackStatusLabel(pack.status)}
        </Badge>
      </div>
      {pack.description !== '' ? (
        <p className="et-sticker-pack__description">{pack.description}</p>
      ) : null}
      <p className="et-page__description">
        貼圖只顯示名稱與詮釋資料，不提供檔案下載；沒有即時推播，重新載入後才會同步。
      </p>
      <div className="et-call-detail__actions">
        <Button
          size="sm"
          variant="secondary"
          disabled={acting}
          onClick={() => {
            onToggleFavorite(pack);
          }}
        >
          {pack.favoritedByMe ? '取消收藏' : '收藏'}
        </Button>
        {manageable ? (
          <>
            <Button
              size="sm"
              variant="secondary"
              disabled={acting}
              onClick={() => {
                onEdit(pack);
              }}
            >
              編輯貼圖包
            </Button>
            <Button
              size="sm"
              variant="secondary"
              disabled={acting}
              onClick={() => {
                onAddSticker(pack);
              }}
            >
              新增貼圖
            </Button>
            <Button
              size="sm"
              variant="danger"
              disabled={acting}
              onClick={() => {
                setDeleteConfirmOpen(true);
              }}
            >
              刪除貼圖包
            </Button>
          </>
        ) : null}
        <Button
          size="sm"
          variant="secondary"
          disabled={acting}
          onClick={() => {
            onReload(pack.id);
          }}
        >
          重新載入
        </Button>
        <Button
          size="sm"
          variant="secondary"
          onClick={() => {
            onClose();
          }}
        >
          關閉詳情
        </Button>
      </div>
      <section aria-label="貼圖清單">
        <h4 className="et-call-detail__section-title">{`貼圖（${stickers.length} 張）`}</h4>
        {stickersState === 'loading' ? (
          <p className="et-page__description">載入貼圖中…</p>
        ) : stickersState === 'error' ? (
          <p className="et-page__description">貼圖載入失敗，請重新載入。</p>
        ) : stickersState === 'unsupported' ? (
          <p className="et-page__description">此資料來源不提供貼圖清單。</p>
        ) : stickers.length === 0 ? (
          <p className="et-page__description">這個貼圖包還沒有貼圖。</p>
        ) : (
          <ul className="et-sticker-grid">
            {stickers.map((sticker) => (
              <StickerTile
                key={sticker.id}
                sticker={sticker}
                onRecordUse={(item) => {
                  onRecordUse(item);
                }}
                recordBusy={acting}
              />
            ))}
          </ul>
        )}
        {manageable && stickersState === 'ready' && stickers.length > 0 ? (
          <ul className="et-calls__list">
            {stickers.map((sticker) => (
              <li key={sticker.id} className="et-calls__row">
                <span>
                  {sticker.name === '' ? '（未命名）' : sticker.name} · 第 {sticker.position + 1} 張
                </span>
                <Button
                  size="sm"
                  variant="secondary"
                  disabled={acting}
                  onClick={() => {
                    onRemoveSticker(sticker);
                  }}
                >
                  移除
                </Button>
              </li>
            ))}
          </ul>
        ) : null}
      </section>
      <Dialog
        open={deleteConfirmOpen}
        onClose={() => {
          setDeleteConfirmOpen(false);
        }}
        title="刪除這個貼圖包？"
        description="刪除為封存狀態標記，刪除後一切讀取皆以不存在回應；收藏與授權紀錄會保留。"
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => {
                setDeleteConfirmOpen(false);
              }}
            >
              取消
            </Button>
            <Button
              variant="danger"
              disabled={acting}
              onClick={() => {
                setDeleteConfirmOpen(false);
                onDelete(pack);
              }}
            >
              刪除貼圖包
            </Button>
          </>
        }
      >
        <p className="et-page__description">此操作會立即生效。</p>
      </Dialog>
    </article>
  );
}

interface StickerMediaPickerDialogProps {
  open: boolean;
  files: MediaFileItem[];
  filesState: FilesState;
  selectedId: string;
  onRetryFiles: () => void;
  onClose: () => void;
  onConfirm: (mediaFileId: string) => void;
}

/**
 * 單一媒體檔案挑選對話框（§2ah）：僅掃描通過（clean）的檔案可
 * 選；關聯的是既有媒體檔案，不會重新上傳檔案位元組。
 */
export function StickerMediaPickerDialog({
  open,
  files,
  filesState,
  selectedId,
  onRetryFiles,
  onClose,
  onConfirm,
}: StickerMediaPickerDialogProps): ReactElement {
  const [selected, setSelected] = useState(selectedId);

  useEffect(() => {
    if (open) {
      setSelected(selectedId);
    }
  }, [open, selectedId]);

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="選擇媒體檔案"
      description="只列出媒體檔案清單。僅掃描通過的檔案可關聯；關聯既有檔案，不會重新上傳檔案位元組。"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            取消
          </Button>
          <Button
            disabled={selected === ''}
            onClick={() => {
              onConfirm(selected);
            }}
          >
            確認選擇
          </Button>
        </>
      }
    >
      {filesState === 'loading' ? <p className="et-page__description">載入媒體檔案中…</p> : null}
      {filesState === 'error' ? (
        <>
          <p className="et-page__description">載入媒體檔案失敗，請再試一次。</p>
          <Button variant="secondary" onClick={onRetryFiles}>
            重新載入
          </Button>
        </>
      ) : null}
      {filesState === 'unsupported' ? (
        <p className="et-page__description">目前的資料模式不提供媒體檔案清單。</p>
      ) : null}
      {filesState === 'ready' ? (
        files.length === 0 ? (
          <p className="et-page__description">沒有媒體檔案，請先到媒體頁上傳。</p>
        ) : (
          <ul className="et-timeline__picker-list">
            {files.map((file) => {
              const selectable = file.scanStatus === 'clean';
              return (
                <li key={file.id} className="et-timeline__picker-row">
                  <label className="et-timeline__picker-label">
                    <input
                      type="radio"
                      name="sticker-media-file"
                      value={file.id}
                      checked={selected === file.id}
                      disabled={!selectable}
                      onChange={() => {
                        setSelected(file.id);
                      }}
                    />
                    <span>
                      {file.originalFilename}
                      <span className="et-timeline__picker-meta">
                        {file.mimeType} · {scanNote(file.scanStatus)}
                      </span>
                    </span>
                  </label>
                </li>
              );
            })}
          </ul>
        )
      ) : null}
    </Dialog>
  );
}

export interface StickerPackFormValues {
  name: string;
  description: string;
  category: StickerPackCategory;
  coverMediaId: string;
  isFree: boolean;
  status: 'draft' | 'published' | 'unlisted';
}

interface StickerPackFormDialogProps {
  open: boolean;
  initial: StickerPackItem | null;
  files: MediaFileItem[];
  filesState: FilesState;
  onRetryFiles: () => void;
  onClose: () => void;
  onSubmit: (values: StickerPackFormValues) => Promise<void>;
}

/**
 * 貼圖包建立／編輯對話框（§2ah）：新建一律為草稿（狀態欄位只
 * 在編輯時出現）；封面以既有媒體檔案挑選，留空表示無封面。
 * 沒有購買、定價以外的商業欄位。
 */
export function StickerPackFormDialog({
  open,
  initial,
  files,
  filesState,
  onRetryFiles,
  onClose,
  onSubmit,
}: StickerPackFormDialogProps): ReactElement {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [category, setCategory] = useState<StickerPackCategory>('');
  const [coverMediaId, setCoverMediaId] = useState('');
  const [isFree, setIsFree] = useState(true);
  const [status, setStatus] = useState<'draft' | 'published' | 'unlisted'>('draft');
  const [pickerOpen, setPickerOpen] = useState(false);
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (open) {
      setName(initial?.name ?? '');
      setDescription(initial?.description ?? '');
      setCategory(initial?.category ?? '');
      setCoverMediaId(initial?.coverMediaId ?? '');
      setIsFree(initial?.isFree ?? true);
      setStatus(
        initial?.status === 'published' || initial?.status === 'unlisted'
          ? initial.status
          : 'draft',
      );
      setPickerOpen(false);
      setError('');
      setSubmitting(false);
    }
  }, [open, initial]);

  const coverFile = files.find((file) => file.id === coverMediaId);

  const submit = async (): Promise<void> => {
    const nameError = validateStickerPackName(name);
    const descriptionError = validateStickerPackDescription(description);
    if (nameError !== '' || descriptionError !== '') {
      setError(nameError !== '' ? nameError : descriptionError);
      return;
    }
    if (submitting) {
      return;
    }
    setError('');
    setSubmitting(true);
    try {
      await onSubmit({ name, description, category, coverMediaId, isFree, status });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : '儲存貼圖包失敗，請再試一次。');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <Dialog
        open={open}
        onClose={onClose}
        title={initial === null ? '建立貼圖包' : '編輯貼圖包'}
        description={
          initial === null
            ? '新建的貼圖包一律為草稿，上架後其他人才看得到。'
            : '只有建立者可以編輯；已封存的貼圖包無法以編輯復活。'
        }
        footer={
          <>
            <Button variant="secondary" onClick={onClose}>
              取消
            </Button>
            <Button disabled={submitting} onClick={() => void submit()}>
              {submitting ? '儲存中…' : initial === null ? '建立貼圖包' : '儲存變更'}
            </Button>
          </>
        }
      >
        <div className="et-calls__dialog-body">
          <Input
            label="貼圖包名稱"
            value={name}
            onChange={(event: ChangeEvent<HTMLInputElement>) => {
              setName(event.target.value);
            }}
            placeholder="例如：山間小日子"
          />
          <TextArea
            label="描述"
            value={description}
            rows={3}
            onChange={(event: ChangeEvent<HTMLTextAreaElement>) => {
              setDescription(event.target.value);
            }}
            placeholder="簡單介紹這個貼圖包"
          />
          <Select
            label="分類"
            value={category}
            options={PACK_FORM_CATEGORY_OPTIONS}
            onChange={(event: ChangeEvent<HTMLSelectElement>) => {
              setCategory(event.target.value as StickerPackCategory);
            }}
          />
          <div>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => {
                setPickerOpen(true);
              }}
            >
              {coverMediaId === '' ? '選擇封面' : '更換封面'}
            </Button>{' '}
            {coverMediaId !== '' ? (
              <>
                <span className="et-call__meta">{coverFile?.originalFilename ?? coverMediaId}</span>{' '}
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => {
                    setCoverMediaId('');
                  }}
                >
                  清除封面
                </Button>
              </>
            ) : (
              <span className="et-call__meta">未設定封面</span>
            )}
          </div>
          <Checkbox
            label="免費貼圖包"
            description="取消勾選即為付費貼圖包；付費包只在授權發放後才對他人可見，這裡沒有購買流程。"
            checked={isFree}
            onChange={(event: ChangeEvent<HTMLInputElement>) => {
              setIsFree(event.target.checked);
            }}
          />
          {initial !== null ? (
            <Select
              label="狀態"
              value={status}
              options={STATUS_OPTIONS}
              onChange={(event: ChangeEvent<HTMLSelectElement>) => {
                setStatus(event.target.value as 'draft' | 'published' | 'unlisted');
              }}
            />
          ) : null}
          {error !== '' ? (
            <p className="et-page__description" role="alert">
              {error}
            </p>
          ) : null}
        </div>
      </Dialog>
      <StickerMediaPickerDialog
        open={pickerOpen}
        files={files}
        filesState={filesState}
        selectedId={coverMediaId}
        onRetryFiles={onRetryFiles}
        onClose={() => {
          setPickerOpen(false);
        }}
        onConfirm={(mediaFileId) => {
          setCoverMediaId(mediaFileId);
          setPickerOpen(false);
        }}
      />
    </>
  );
}

interface AddStickerDialogProps {
  open: boolean;
  packName: string;
  files: MediaFileItem[];
  filesState: FilesState;
  onRetryFiles: () => void;
  onClose: () => void;
  onConfirm: (input: AddStickerInput) => Promise<void>;
}

/**
 * 新增貼圖對話框（§2ah）：名稱、動態旗標與單一媒體檔案挑選。
 * 序號由後端接在最大序號之後，單包生效貼圖上限 100 張。
 */
export function AddStickerDialog({
  open,
  packName,
  files,
  filesState,
  onRetryFiles,
  onClose,
  onConfirm,
}: AddStickerDialogProps): ReactElement {
  const [name, setName] = useState('');
  const [isAnimated, setIsAnimated] = useState(false);
  const [mediaFileId, setMediaFileId] = useState('');
  const [pickerOpen, setPickerOpen] = useState(false);
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (open) {
      setName('');
      setIsAnimated(false);
      setMediaFileId('');
      setPickerOpen(false);
      setError('');
      setSubmitting(false);
    }
  }, [open]);

  const pickedFile = files.find((file) => file.id === mediaFileId);

  const submit = async (): Promise<void> => {
    const nameError = validateStickerName(name);
    if (nameError !== '') {
      setError(nameError);
      return;
    }
    if (mediaFileId === '') {
      setError('請先選擇一個媒體檔案作為貼圖本體。');
      return;
    }
    if (submitting) {
      return;
    }
    setError('');
    setSubmitting(true);
    try {
      await onConfirm({ mediaFileId, name, isAnimated });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : '新增貼圖失敗，請再試一次。');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <Dialog
        open={open}
        onClose={onClose}
        title={`新增貼圖到「${packName}」`}
        description="貼圖本體為既有媒體檔案的參照，不會重新上傳檔案位元組。"
        footer={
          <>
            <Button variant="secondary" onClick={onClose}>
              取消
            </Button>
            <Button disabled={submitting} onClick={() => void submit()}>
              {submitting ? '新增中…' : '新增貼圖'}
            </Button>
          </>
        }
      >
        <div className="et-calls__dialog-body">
          <Input
            label="貼圖名稱"
            value={name}
            onChange={(event: ChangeEvent<HTMLInputElement>) => {
              setName(event.target.value);
            }}
            placeholder="例如：咖啡時間"
          />
          <Checkbox
            label="動態貼圖"
            description="勾選表示此貼圖為動態格式，僅為詮釋資料標記。"
            checked={isAnimated}
            onChange={(event: ChangeEvent<HTMLInputElement>) => {
              setIsAnimated(event.target.checked);
            }}
          />
          <div>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => {
                setPickerOpen(true);
              }}
            >
              {mediaFileId === '' ? '選擇媒體檔案' : '更換媒體檔案'}
            </Button>{' '}
            <span className="et-call__meta">
              {mediaFileId === '' ? '尚未選擇' : (pickedFile?.originalFilename ?? mediaFileId)}
            </span>
          </div>
          {error !== '' ? (
            <p className="et-page__description" role="alert">
              {error}
            </p>
          ) : null}
        </div>
      </Dialog>
      <StickerMediaPickerDialog
        open={pickerOpen}
        files={files}
        filesState={filesState}
        selectedId={mediaFileId}
        onRetryFiles={onRetryFiles}
        onClose={() => {
          setPickerOpen(false);
        }}
        onConfirm={(fileId) => {
          setMediaFileId(fileId);
          setPickerOpen(false);
        }}
      />
    </>
  );
}

/**
 * Stickers view (Phase 08, web wiring): pack browsing with
 * category filter and keyword search, favorites, recent use,
 * pack detail with a sticker grid, and author-only pack and
 * sticker management. The same component serves demo mode
 * (local fixtures, labelled as such) and API mode (backend
 * data); the provider methods are optional, so absence renders
 * the unsupported state instead of an error.
 *
 * Honesty rules: there is no purchase flow and no store
 * ranking, so paid packs without a grant simply do not appear
 * and the view says so; sticker tiles render names and metadata
 * only, never bytes or download affordances; there is no
 * realtime push, so the view says state only syncs on reload.
 * Deleting a pack archives it; removing a sticker marks it
 * removed; neither can be undone from this view.
 */
export function StickersView({ provider, dataMode = 'demo' }: StickersViewProps): ReactElement {
  const { notify } = useToast();
  const [tab, setTab] = useState<StickerTab>('browse');
  const [loadState, setLoadState] = useState<LoadState>('loading');
  const [packs, setPacks] = useState<StickerPackItem[]>([]);
  const [nextCursor, setNextCursor] = useState('');
  const [loadingMore, setLoadingMore] = useState(false);
  const [categoryFilter, setCategoryFilter] = useState('all');
  const [searchInput, setSearchInput] = useState('');
  const [submittedQuery, setSubmittedQuery] = useState('');
  const [searchState, setSearchState] = useState<LoadState>('loading');
  const [searchResults, setSearchResults] = useState<StickerItem[]>([]);
  const [searchCursor, setSearchCursor] = useState('');
  const [searchLoadingMore, setSearchLoadingMore] = useState(false);
  const [selectedPackId, setSelectedPackId] = useState<string | null>(null);
  const [detail, setDetail] = useState<StickerPackItem | null>(null);
  const [detailState, setDetailState] = useState<DetailState>('loading');
  const [stickers, setStickers] = useState<StickerItem[]>([]);
  const [stickersState, setStickersState] = useState<DetailState>('loading');
  const [names, setNames] = useState<Record<string, string>>({});
  const [formOpen, setFormOpen] = useState(false);
  const [formInitial, setFormInitial] = useState<StickerPackItem | null>(null);
  const [addStickerOpen, setAddStickerOpen] = useState(false);
  const [addStickerPack, setAddStickerPack] = useState<StickerPackItem | null>(null);
  const [files, setFiles] = useState<MediaFileItem[]>([]);
  const [filesState, setFilesState] = useState<FilesState>('loading');
  const [acting, setActing] = useState(false);
  const [favoriteBusy, setFavoriteBusy] = useState(false);
  const mountedRef = useRef(true);
  const myId = provider.getCurrentUserId?.() ?? null;

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const loadPacks = useCallback(
    async (
      options: { cursor?: string; limit?: number; category?: string; creatorId?: string },
      append: boolean,
    ): Promise<void> => {
      const listPacks = provider.listStickerPacksAsync;
      if (listPacks === undefined) {
        if (mountedRef.current) {
          setLoadState('unsupported');
        }
        return;
      }
      if (append) {
        setLoadingMore(true);
      } else if (mountedRef.current) {
        setLoadState('loading');
      }
      try {
        const page = await listPacks({
          ...(options.cursor !== undefined && options.cursor !== ''
            ? { cursor: options.cursor }
            : {}),
          ...(options.limit !== undefined ? { limit: options.limit } : {}),
          ...(options.category !== undefined && options.category !== ''
            ? { category: options.category }
            : {}),
          ...(options.creatorId !== undefined && options.creatorId !== ''
            ? { creatorId: options.creatorId }
            : {}),
        });
        if (!mountedRef.current) {
          return;
        }
        setPacks((prev) => (append ? [...prev, ...page.packs] : page.packs));
        setNextCursor(page.nextCursor);
        setLoadState('ready');
      } catch {
        if (mountedRef.current) {
          setLoadState(append ? 'ready' : 'error');
          if (append) {
            notify({
              title: '載入失敗',
              message: '載入更多貼圖包失敗，請再試一次。',
              tone: 'error',
            });
          }
        }
      } finally {
        if (mountedRef.current) {
          setLoadingMore(false);
        }
      }
    },
    [notify, provider],
  );

  const loadFavorites = useCallback(
    async (cursor: string, append: boolean): Promise<void> => {
      const listFavorites = provider.listFavoriteStickerPacksAsync;
      if (listFavorites === undefined) {
        if (mountedRef.current) {
          setLoadState('unsupported');
        }
        return;
      }
      if (append) {
        setLoadingMore(true);
      } else if (mountedRef.current) {
        setLoadState('loading');
      }
      try {
        const page = await listFavorites(
          cursor === '' ? { limit: PACK_PAGE_LIMIT } : { cursor, limit: PACK_PAGE_LIMIT },
        );
        if (!mountedRef.current) {
          return;
        }
        setPacks((prev) => (append ? [...prev, ...page.packs] : page.packs));
        setNextCursor(page.nextCursor);
        setLoadState('ready');
      } catch {
        if (mountedRef.current) {
          setLoadState(append ? 'ready' : 'error');
          if (append) {
            notify({ title: '載入失敗', message: '載入更多收藏失敗，請再試一次。', tone: 'error' });
          }
        }
      } finally {
        if (mountedRef.current) {
          setLoadingMore(false);
        }
      }
    },
    [notify, provider],
  );

  const loadRecent = useCallback(
    async (cursor: string, append: boolean): Promise<void> => {
      const listRecent = provider.listRecentStickersAsync;
      if (listRecent === undefined) {
        if (mountedRef.current) {
          setLoadState('unsupported');
        }
        return;
      }
      if (append) {
        setLoadingMore(true);
      } else if (mountedRef.current) {
        setLoadState('loading');
      }
      try {
        const page = await listRecent(
          cursor === '' ? { limit: PACK_PAGE_LIMIT } : { cursor, limit: PACK_PAGE_LIMIT },
        );
        if (!mountedRef.current) {
          return;
        }
        setSearchResults((prev) => (append ? [...prev, ...page.stickers] : page.stickers));
        setNextCursor(page.nextCursor);
        setLoadState('ready');
      } catch {
        if (mountedRef.current) {
          setLoadState(append ? 'ready' : 'error');
          if (append) {
            notify({
              title: '載入失敗',
              message: '載入更多最近使用失敗，請再試一次。',
              tone: 'error',
            });
          }
        }
      } finally {
        if (mountedRef.current) {
          setLoadingMore(false);
        }
      }
    },
    [notify, provider],
  );

  const runSearch = useCallback(
    async (query: string, cursor: string, append: boolean): Promise<void> => {
      const search = provider.searchStickersAsync;
      if (search === undefined) {
        if (mountedRef.current) {
          setSearchState('unsupported');
        }
        return;
      }
      if (append) {
        setSearchLoadingMore(true);
      } else if (mountedRef.current) {
        setSearchState('loading');
      }
      try {
        const page = await search(query, {
          ...(cursor !== '' ? { cursor } : {}),
          limit: PACK_PAGE_LIMIT,
          ...(categoryFilter !== 'all' ? { category: categoryFilter } : {}),
        });
        if (!mountedRef.current) {
          return;
        }
        setSearchResults((prev) => (append ? [...prev, ...page.stickers] : page.stickers));
        setSearchCursor(page.nextCursor);
        setSearchState('ready');
      } catch (error) {
        if (mountedRef.current) {
          setSearchState(append ? 'ready' : 'error');
          if (append) {
            notify({
              title: '搜尋失敗',
              message: '載入更多搜尋結果失敗，請再試一次。',
              tone: 'error',
            });
          } else {
            notify({
              title: '搜尋失敗',
              message: error instanceof Error ? error.message : '請再試一次。',
              tone: 'error',
            });
          }
        }
      } finally {
        if (mountedRef.current) {
          setSearchLoadingMore(false);
        }
      }
    },
    [categoryFilter, notify, provider],
  );

  const loadCurrentTab = useCallback(
    (cursor: string, append: boolean): void => {
      if (tab === 'browse') {
        void loadPacks(
          {
            ...(cursor !== '' ? { cursor } : {}),
            limit: PACK_PAGE_LIMIT,
            ...(categoryFilter !== 'all' ? { category: categoryFilter } : {}),
          },
          append,
        );
      } else if (tab === 'favorites') {
        void loadFavorites(cursor, append);
      } else {
        void loadRecent(cursor, append);
      }
    },
    [categoryFilter, loadFavorites, loadPacks, loadRecent, tab],
  );

  useEffect(() => {
    setSelectedPackId(null);
    setDetail(null);
    setSubmittedQuery('');
    loadCurrentTab('', false);
  }, [loadCurrentTab]);

  const loadNames = useCallback(async (): Promise<void> => {
    const listFriends = provider.listFriendsAsync;
    if (listFriends === undefined) {
      return;
    }
    try {
      const friends = await listFriends();
      if (!mountedRef.current) {
        return;
      }
      setNames(Object.fromEntries(friends.map((friend) => [friend.id, friend.displayName])));
    } catch {
      // 名稱解析失敗不阻擋貼圖包清單：未知作者顯示識別碼縮寫。
    }
  }, [provider]);

  useEffect(() => {
    void loadNames();
  }, [loadNames]);

  const loadFiles = useCallback(async (): Promise<void> => {
    const listFiles = provider.listMediaFilesAsync;
    if (listFiles === undefined) {
      if (mountedRef.current) {
        setFilesState('unsupported');
      }
      return;
    }
    if (mountedRef.current) {
      setFilesState('loading');
    }
    try {
      const found = await listFiles();
      if (!mountedRef.current) {
        return;
      }
      setFiles(found);
      setFilesState('ready');
    } catch {
      if (mountedRef.current) {
        setFilesState('error');
      }
    }
  }, [provider]);

  const loadDetail = useCallback(
    async (packId: string): Promise<void> => {
      const getPack = provider.getStickerPackAsync;
      const listStickers = provider.listPackStickersAsync;
      if (getPack === undefined) {
        if (mountedRef.current) {
          setDetailState('unsupported');
        }
        return;
      }
      if (mountedRef.current) {
        setDetailState('loading');
        setStickersState(listStickers === undefined ? 'unsupported' : 'loading');
      }
      try {
        const found = await getPack(packId);
        if (!mountedRef.current) {
          return;
        }
        setDetail(found);
        setDetailState('ready');
      } catch {
        if (mountedRef.current) {
          setDetailState('error');
        }
        return;
      }
      if (listStickers !== undefined) {
        try {
          const foundStickers = await listStickers(packId);
          if (mountedRef.current) {
            setStickers(foundStickers);
            setStickersState('ready');
          }
        } catch {
          if (mountedRef.current) {
            setStickersState('error');
          }
        }
      }
    },
    [provider],
  );

  useEffect(() => {
    if (selectedPackId !== null) {
      void loadDetail(selectedPackId);
    }
  }, [loadDetail, selectedPackId]);

  const nameOf = useCallback(
    (userId: string): string => {
      if (myId !== null && userId === myId) {
        return provider.getSnapshot().profile.name;
      }
      return names[userId] ?? userId.slice(0, 8);
    },
    [myId, names, provider],
  );

  const refreshAfterAction = useCallback(
    async (packId: string | null): Promise<void> => {
      loadCurrentTab('', false);
      if (packId !== null) {
        await loadDetail(packId);
      }
    },
    [loadCurrentTab, loadDetail],
  );

  const toggleFavorite = useCallback(
    async (pack: StickerPackItem): Promise<void> => {
      const favorite = provider.favoriteStickerPackAsync;
      const unfavorite = provider.unfavoriteStickerPackAsync;
      if (favorite === undefined || unfavorite === undefined || favoriteBusy) {
        return;
      }
      setFavoriteBusy(true);
      try {
        const updated = pack.favoritedByMe ? await unfavorite(pack.id) : await favorite(pack.id);
        if (!mountedRef.current) {
          return;
        }
        setPacks((prev) => prev.map((item) => (item.id === updated.id ? updated : item)));
        if (detail?.id === updated.id) {
          setDetail(updated);
        }
        notify({
          title: updated.favoritedByMe ? '已收藏' : '已取消收藏',
          message: `貼圖包「${updated.name}」${updated.favoritedByMe ? '已加入收藏。' : '已移出收藏。'}`,
          tone: 'success',
        });
      } catch (error) {
        notify({
          title: '操作失敗',
          message: error instanceof Error ? error.message : '請再試一次。',
          tone: 'error',
        });
      } finally {
        if (mountedRef.current) {
          setFavoriteBusy(false);
        }
      }
    },
    [detail, favoriteBusy, notify, provider],
  );

  const recordUse = useCallback(
    async (sticker: StickerItem): Promise<void> => {
      const record = provider.recordStickerUseAsync;
      if (record === undefined || acting) {
        return;
      }
      setActing(true);
      try {
        await record(sticker.id);
        if (!mountedRef.current) {
          return;
        }
        notify({
          title: '已記錄使用',
          message: `貼圖「${sticker.name}」已記入最近使用。`,
          tone: 'success',
        });
        if (tab === 'recent') {
          loadCurrentTab('', false);
        }
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
    },
    [acting, loadCurrentTab, notify, provider, tab],
  );

  const submitPackForm = useCallback(
    async (values: StickerPackFormValues): Promise<void> => {
      if (formInitial === null) {
        const create = provider.createStickerPackAsync;
        if (create === undefined) {
          throw new Error('此資料來源不支援建立貼圖包。');
        }
        const created = await create({
          name: values.name,
          description: values.description,
          category: values.category,
          coverMediaId: values.coverMediaId,
          isFree: values.isFree,
        } satisfies CreateStickerPackInput);
        if (!mountedRef.current) {
          return;
        }
        setFormOpen(false);
        setSelectedPackId(created.id);
        loadCurrentTab('', false);
        await loadDetail(created.id);
        notify({
          title: '已建立貼圖包',
          message: '新貼圖包為草稿，上架後其他人才看得到。',
          tone: 'success',
        });
        return;
      }
      const update = provider.updateStickerPackAsync;
      if (update === undefined) {
        throw new Error('此資料來源不支援編輯貼圖包。');
      }
      const updated = await update(formInitial.id, {
        name: values.name,
        description: values.description,
        category: values.category,
        coverMediaId: values.coverMediaId,
        status: values.status,
        isFree: values.isFree,
      } satisfies UpdateStickerPackInput);
      if (!mountedRef.current) {
        return;
      }
      setFormOpen(false);
      setPacks((prev) => prev.map((item) => (item.id === updated.id ? updated : item)));
      setDetail(updated);
      notify({
        title: '已儲存變更',
        message: `貼圖包「${updated.name}」已更新。`,
        tone: 'success',
      });
    },
    [formInitial, loadCurrentTab, loadDetail, notify, provider],
  );

  const deletePack = useCallback(
    async (pack: StickerPackItem): Promise<void> => {
      const remove = provider.deleteStickerPackAsync;
      if (remove === undefined || acting) {
        return;
      }
      setActing(true);
      try {
        await remove(pack.id);
        if (!mountedRef.current) {
          return;
        }
        setSelectedPackId(null);
        setDetail(null);
        loadCurrentTab('', false);
        notify({
          title: '已刪除貼圖包',
          message: `貼圖包「${pack.name}」已封存。`,
          tone: 'success',
        });
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
    },
    [acting, loadCurrentTab, notify, provider],
  );

  const confirmAddSticker = useCallback(
    async (input: AddStickerInput): Promise<void> => {
      const add = provider.addStickerAsync;
      if (add === undefined || addStickerPack === null) {
        throw new Error('此資料來源不支援新增貼圖。');
      }
      await add(addStickerPack.id, input);
      if (!mountedRef.current) {
        return;
      }
      setAddStickerOpen(false);
      await refreshAfterAction(addStickerPack.id);
      notify({
        title: '已新增貼圖',
        message: `貼圖「${input.name}」已加入貼圖包。`,
        tone: 'success',
      });
    },
    [addStickerPack, notify, provider, refreshAfterAction],
  );

  const removeSticker = useCallback(
    async (sticker: StickerItem): Promise<void> => {
      const remove = provider.removeStickerAsync;
      if (remove === undefined || acting) {
        return;
      }
      setActing(true);
      try {
        await remove(sticker.id);
        if (!mountedRef.current) {
          return;
        }
        await refreshAfterAction(sticker.packId);
        notify({
          title: '已移除貼圖',
          message: `貼圖「${sticker.name}」已標記為移除。`,
          tone: 'success',
        });
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
    },
    [acting, notify, provider, refreshAfterAction],
  );

  const submitSearch = (): void => {
    const query = searchInput.trim();
    if (query === '') {
      setSubmittedQuery('');
      setSearchResults([]);
      setSearchCursor('');
      return;
    }
    setSubmittedQuery(query);
    void runSearch(query, '', false);
  };

  const sourceLabel = dataMode === 'api' ? '貼圖來自後端 API' : '展示資料 · 尚未連接後端';
  const canCreate = provider.createStickerPackAsync !== undefined;
  const searching = submittedQuery !== '';
  const listEmptyText =
    tab === 'browse'
      ? '還沒有可見的貼圖包。'
      : tab === 'favorites'
        ? '還沒有收藏的貼圖包。'
        : '還沒有最近使用過的貼圖。';

  return (
    <div className="et-page et-scroll">
      <div className="et-page__narrow">
        <header>
          <h2 className="et-page__title">貼圖</h2>
          <p className="et-page__description">
            {sourceLabel} ·
            付費貼圖包須經授權發放才會顯示，這裡沒有購買流程；沒有即時推播，重新載入後才會同步。
          </p>
        </header>
        <div className="et-timeline__tabs" role="tablist" aria-label="貼圖範圍">
          <Button
            size="sm"
            variant={tab === 'browse' ? 'primary' : 'secondary'}
            role="tab"
            aria-selected={tab === 'browse'}
            onClick={() => {
              setTab('browse');
            }}
          >
            瀏覽貼圖包
          </Button>
          <Button
            size="sm"
            variant={tab === 'favorites' ? 'primary' : 'secondary'}
            role="tab"
            aria-selected={tab === 'favorites'}
            onClick={() => {
              setTab('favorites');
            }}
          >
            我的收藏
          </Button>
          <Button
            size="sm"
            variant={tab === 'recent' ? 'primary' : 'secondary'}
            role="tab"
            aria-selected={tab === 'recent'}
            onClick={() => {
              setTab('recent');
            }}
          >
            最近使用
          </Button>
          <Button
            size="sm"
            variant="secondary"
            disabled={loadState !== 'ready' && loadState !== 'error'}
            onClick={() => {
              loadCurrentTab('', false);
            }}
          >
            重新載入
          </Button>
        </div>
        {tab === 'browse' ? (
          <div className="et-stickers__toolbar">
            <Select
              label="分類篩選"
              value={categoryFilter}
              options={CATEGORY_OPTIONS}
              onChange={(event: ChangeEvent<HTMLSelectElement>) => {
                setCategoryFilter(event.target.value);
              }}
            />
            <label className="et-stickers__search" htmlFor="sticker-search">
              搜尋貼圖
              <span className="et-stickers__search-row">
                <input
                  id="sticker-search"
                  type="search"
                  value={searchInput}
                  onChange={(event: ChangeEvent<HTMLInputElement>) => {
                    setSearchInput(event.target.value);
                  }}
                  placeholder="輸入貼圖或貼圖包名稱"
                />
                <Button
                  size="sm"
                  onClick={() => {
                    submitSearch();
                  }}
                >
                  搜尋
                </Button>
                {searching ? (
                  <Button
                    size="sm"
                    variant="secondary"
                    onClick={() => {
                      setSearchInput('');
                      setSubmittedQuery('');
                      setSearchResults([]);
                      setSearchCursor('');
                    }}
                  >
                    清除搜尋
                  </Button>
                ) : null}
              </span>
            </label>
            <Button
              disabled={!canCreate}
              onClick={() => {
                setFormInitial(null);
                setFormOpen(true);
                void loadFiles();
              }}
            >
              建立貼圖包
            </Button>
          </div>
        ) : null}
        {searching ? (
          <>
            {searchState === 'loading' ? <p className="et-page__description">搜尋貼圖中…</p> : null}
            {searchState === 'error' ? (
              <>
                <p className="et-page__description">搜尋失敗，請檢查連線後再試一次。</p>
                <Button
                  onClick={() => {
                    void runSearch(submittedQuery, '', false);
                  }}
                >
                  重新搜尋
                </Button>
              </>
            ) : null}
            {searchState === 'unsupported' ? (
              <p className="et-page__description">目前的資料模式不提供貼圖搜尋。</p>
            ) : null}
            {searchState === 'ready' && searchResults.length === 0 ? (
              <EmptyState
                icon={<StickerIcon size={28} />}
                title="沒有符合的貼圖"
                description={`沒有名稱包含「${submittedQuery}」的貼圖。`}
              />
            ) : null}
            {searchState === 'ready' && searchResults.length > 0 ? (
              <ul className="et-sticker-grid">
                {searchResults.map((sticker) => (
                  <StickerTile
                    key={sticker.id}
                    sticker={sticker}
                    onRecordUse={(item) => {
                      void recordUse(item);
                    }}
                    recordBusy={acting}
                  />
                ))}
              </ul>
            ) : null}
            {searchState === 'ready' && searchCursor !== '' ? (
              <div className="et-stickers__toolbar">
                <Button
                  variant="secondary"
                  disabled={searchLoadingMore}
                  onClick={() => {
                    void runSearch(submittedQuery, searchCursor, true);
                  }}
                >
                  {searchLoadingMore ? '載入中…' : '載入更多'}
                </Button>
              </div>
            ) : null}
          </>
        ) : (
          <>
            {loadState === 'loading' ? <p className="et-page__description">載入貼圖中…</p> : null}
            {loadState === 'error' ? (
              <>
                <p className="et-page__description">載入失敗，請檢查連線後再試一次。</p>
                <Button
                  onClick={() => {
                    loadCurrentTab('', false);
                  }}
                >
                  重新載入
                </Button>
              </>
            ) : null}
            {loadState === 'unsupported' ? (
              <EmptyState
                icon={<StickerIcon size={28} />}
                title="不支援貼圖"
                description="目前的資料來源不提供貼圖包查詢。"
              />
            ) : null}
            {loadState === 'ready' && tab !== 'recent' && packs.length === 0 ? (
              <EmptyState
                icon={<StickerIcon size={28} />}
                title={listEmptyText}
                description={
                  tab === 'browse'
                    ? '建立第一個貼圖包，或等候他人上架新的貼圖包。'
                    : '收藏喜歡的貼圖包後，會顯示在這裡。'
                }
              />
            ) : null}
            {loadState === 'ready' && tab !== 'recent' && packs.length > 0 ? (
              <div className="et-timeline__list">
                {packs.map((pack) => (
                  <StickerPackCard
                    key={pack.id}
                    pack={pack}
                    creatorName={nameOf(pack.creatorId)}
                    selected={selectedPackId === pack.id}
                    favoriteBusy={favoriteBusy}
                    onSelect={(packId) => {
                      setSelectedPackId(packId);
                    }}
                    onToggleFavorite={(item) => {
                      void toggleFavorite(item);
                    }}
                  />
                ))}
              </div>
            ) : null}
            {loadState === 'ready' && tab === 'recent' && searchResults.length === 0 ? (
              <EmptyState
                icon={<StickerIcon size={28} />}
                title={listEmptyText}
                description="在貼圖包詳情或搜尋結果中記錄使用後，會顯示在這裡（最多保留 30 筆）。"
              />
            ) : null}
            {loadState === 'ready' && tab === 'recent' && searchResults.length > 0 ? (
              <ul className="et-sticker-grid">
                {searchResults.map((sticker) => (
                  <StickerTile key={sticker.id} sticker={sticker} />
                ))}
              </ul>
            ) : null}
            {loadState === 'ready' && nextCursor !== '' ? (
              <div className="et-stickers__toolbar">
                <Button
                  variant="secondary"
                  disabled={loadingMore}
                  onClick={() => {
                    loadCurrentTab(nextCursor, true);
                  }}
                >
                  {loadingMore ? '載入中…' : '載入更多'}
                </Button>
              </div>
            ) : null}
          </>
        )}
        {selectedPackId !== null && detailState === 'loading' ? (
          <p className="et-page__description">載入貼圖包詳情中…</p>
        ) : null}
        {selectedPackId !== null && detailState === 'error' ? (
          <>
            <p className="et-page__description">詳情載入失敗，請再試一次。</p>
            <Button
              onClick={() => {
                void loadDetail(selectedPackId);
              }}
            >
              重新載入
            </Button>
          </>
        ) : null}
        {selectedPackId !== null && detailState === 'unsupported' ? (
          <EmptyState
            icon={<StickerIcon size={28} />}
            title="不支援貼圖包詳情"
            description="目前的資料來源不提供單一貼圖包查詢。"
          />
        ) : null}
        {selectedPackId !== null && detailState === 'ready' && detail !== null ? (
          <StickerPackDetailPanel
            pack={detail}
            stickers={stickers}
            stickersState={stickersState}
            creatorName={nameOf(detail.creatorId)}
            myId={myId}
            acting={acting}
            onToggleFavorite={(pack) => {
              void toggleFavorite(pack);
            }}
            onEdit={(pack) => {
              setFormInitial(pack);
              setFormOpen(true);
              void loadFiles();
            }}
            onDelete={(pack) => {
              void deletePack(pack);
            }}
            onAddSticker={(pack) => {
              setAddStickerPack(pack);
              setAddStickerOpen(true);
              void loadFiles();
            }}
            onRemoveSticker={(sticker) => {
              void removeSticker(sticker);
            }}
            onRecordUse={(sticker) => {
              void recordUse(sticker);
            }}
            onReload={(packId) => {
              void loadDetail(packId);
            }}
            onClose={() => {
              setSelectedPackId(null);
              setDetail(null);
            }}
          />
        ) : null}
        <StickerPackFormDialog
          open={formOpen}
          initial={formInitial}
          files={files}
          filesState={filesState}
          onRetryFiles={() => {
            void loadFiles();
          }}
          onClose={() => {
            setFormOpen(false);
          }}
          onSubmit={submitPackForm}
        />
        <AddStickerDialog
          open={addStickerOpen}
          packName={addStickerPack?.name ?? ''}
          files={files}
          filesState={filesState}
          onRetryFiles={() => {
            void loadFiles();
          }}
          onClose={() => {
            setAddStickerOpen(false);
          }}
          onConfirm={confirmAddSticker}
        />
      </div>
    </div>
  );
}
