import { useCallback, useEffect, useRef, useState } from 'react';
import type { ChangeEvent, ReactElement } from 'react';
import type { ChatDataProvider } from '../data/provider';
import type { MediaFileItem, PostCommentItem, PostItem, PostVisibility } from '../data/types';
import { Avatar } from '../components/Avatar';
import { Badge } from '../components/Badge';
import { Button } from '../components/Button';
import { Dialog } from '../components/Dialog';
import { EmptyState } from '../components/EmptyState';
import { Select } from '../components/Select';
import { TextArea } from '../components/Input';
import { useToast } from '../components/Toast';

export interface TimelineViewProps {
  provider: ChatDataProvider;
  dataMode?: 'demo' | 'api';
}

type LoadState = 'loading' | 'ready' | 'error' | 'unsupported';
type FeedTab = 'feed' | 'mine';
type FilesState = 'loading' | 'ready' | 'error' | 'unsupported';

const POST_PAGE_LIMIT = 20;
const COMMENT_PAGE_LIMIT = 30;
const MAX_POST_MEDIA = 9;

const VISIBILITY_OPTIONS: { value: PostVisibility; label: string }[] = [
  { value: 'friends', label: '僅好友' },
  { value: 'public', label: '公開' },
  { value: 'only_me', label: '僅自己' },
];

export function visibilityLabel(visibility: string): string {
  if (visibility === 'public') {
    return '公開';
  }
  if (visibility === 'only_me') {
    return '僅自己';
  }
  return '僅好友';
}

/** 貼文內容檢查（與後端同規則）：至多 2000 runes；空白內容須附媒體。 */
export function validatePostContent(content: string, mediaCount: number): string {
  if ([...content].length > 2000) {
    return '貼文內容不得超過 2000 字。';
  }
  if (content.trim() === '' && mediaCount === 0) {
    return '貼文內容與媒體不得皆為空。';
  }
  return '';
}

/** 留言內容檢查（與後端同規則）：不得為空，至多 1000 runes。 */
export function validatePostComment(content: string): string {
  if ([...content].length > 1000) {
    return '留言內容不得超過 1000 字。';
  }
  if (content.trim() === '') {
    return '留言內容不得為空。';
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

interface MediaPickerProps {
  open: boolean;
  files: MediaFileItem[];
  filesState: FilesState;
  selectedIds: string[];
  onRetryFiles: () => void;
  onClose: () => void;
  onConfirm: (mediaFileIds: string[]) => void;
}

/**
 * 貼文媒體挑選對話框（§2ac，展示／API 雙模式共用）：檔案清單
 * 由父層傳入（provider.listMediaFilesAsync）。僅掃描通過
 * （clean）的檔案可選；未通過掃描的檔案顯示狀態且停用——
 * 後端掃描閘門 fail-closed，UI 不得讓使用者選到不可關聯的
 * 檔案。單則貼文最多 9 個（後端上限）。關聯的是既有媒體檔案，
 * 不會重新上傳檔案位元組；貼文媒體只作詮釋資料顯示，不提供下載。
 */
export function PostMediaPickerDialog({
  open,
  files,
  filesState,
  selectedIds,
  onRetryFiles,
  onClose,
  onConfirm,
}: MediaPickerProps): ReactElement {
  const [selected, setSelected] = useState<ReadonlySet<string>>(new Set(selectedIds));
  const [error, setError] = useState('');

  useEffect(() => {
    if (open) {
      setSelected(new Set(selectedIds));
      setError('');
    }
  }, [open, selectedIds]);

  const toggle = (fileId: string): void => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(fileId)) {
        next.delete(fileId);
      } else if (next.size >= MAX_POST_MEDIA) {
        setError(`單則貼文最多關聯 ${MAX_POST_MEDIA} 個媒體檔案。`);
        return prev;
      } else {
        next.add(fileId);
      }
      setError('');
      return next;
    });
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="選擇貼文媒體"
      description="只列出媒體檔案清單。僅掃描通過的檔案可關聯；關聯既有檔案，不會重新上傳檔案位元組。"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            取消
          </Button>
          <Button
            onClick={() => {
              onConfirm([...selected]);
            }}
          >
            確認選擇（{selected.size}）
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
        <>
          {files.length === 0 ? (
            <p className="et-page__description">沒有媒體檔案，請先到媒體頁上傳。</p>
          ) : (
            <ul className="et-timeline__picker-list">
              {files.map((file) => {
                const selectable = file.scanStatus === 'clean';
                const checked = selected.has(file.id);
                return (
                  <li key={file.id} className="et-timeline__picker-row">
                    <label className="et-timeline__picker-label">
                      <input
                        type="checkbox"
                        checked={checked}
                        disabled={!selectable}
                        onChange={() => {
                          toggle(file.id);
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
          )}
          {error !== '' ? (
            <p className="et-page__description" role="alert">
              {error}
            </p>
          ) : null}
        </>
      ) : null}
    </Dialog>
  );
}

interface PostCardProps {
  post: PostItem;
  myId: string | null;
  authorName: string;
  mediaById: Map<string, MediaFileItem>;
  nameOf: (userId: string) => string;
  provider: ChatDataProvider;
  onChanged: (post: PostItem) => void;
  onRemoved: (postId: string) => void;
}

/**
 * 單則貼文卡片：內容、能見度、媒體詮釋資料、按讚、留言區、
 * 作者專屬的編輯／刪除。留言以 REST 查詢同步（動態無即時推播）：
 * 展開時載入，留言與按讚皆以伺服器回應更新計數。
 */
export function PostCard({
  post,
  myId,
  authorName,
  mediaById,
  nameOf,
  provider,
  onChanged,
  onRemoved,
}: PostCardProps): ReactElement {
  const { notify } = useToast();
  const [liking, setLiking] = useState(false);
  const [commentsOpen, setCommentsOpen] = useState(false);
  const [commentsState, setCommentsState] = useState<'idle' | 'loading' | 'ready' | 'error'>(
    'idle',
  );
  const [comments, setComments] = useState<PostCommentItem[]>([]);
  const [commentsCursor, setCommentsCursor] = useState('');
  const [loadingMoreComments, setLoadingMoreComments] = useState(false);
  const [commentDraft, setCommentDraft] = useState('');
  const [commentError, setCommentError] = useState('');
  const [submittingComment, setSubmittingComment] = useState(false);
  const [replyTo, setReplyTo] = useState<PostCommentItem | null>(null);
  const [editing, setEditing] = useState(false);
  const [editContent, setEditContent] = useState(post.content);
  const [editVisibility, setEditVisibility] = useState<PostVisibility>(post.visibility);
  const [editError, setEditError] = useState('');
  const [savingEdit, setSavingEdit] = useState(false);
  const [pendingDelete, setPendingDelete] = useState<'post' | PostCommentItem | null>(null);
  const [acting, setActing] = useState(false);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const isMine = myId !== null && post.authorId === myId;

  const loadComments = useCallback(
    async (cursor: string, append: boolean): Promise<void> => {
      const listComments = provider.listPostCommentsAsync;
      if (listComments === undefined) {
        return;
      }
      if (append) {
        setLoadingMoreComments(true);
      } else {
        setCommentsState('loading');
      }
      try {
        const page = await listComments(
          post.id,
          cursor === '' ? { limit: COMMENT_PAGE_LIMIT } : { cursor, limit: COMMENT_PAGE_LIMIT },
        );
        if (!mountedRef.current) {
          return;
        }
        setComments((prev) => (append ? [...prev, ...page.comments] : page.comments));
        setCommentsCursor(page.nextCursor);
        setCommentsState('ready');
      } catch {
        if (mountedRef.current) {
          setCommentsState('error');
        }
      } finally {
        if (mountedRef.current) {
          setLoadingMoreComments(false);
        }
      }
    },
    [provider, post.id],
  );

  const toggleComments = (): void => {
    const next = !commentsOpen;
    setCommentsOpen(next);
    if (next && commentsState === 'idle') {
      void loadComments('', false);
    }
  };

  const toggleLike = async (): Promise<void> => {
    const like = provider.likePostAsync;
    const unlike = provider.unlikePostAsync;
    if (like === undefined || unlike === undefined || liking) {
      return;
    }
    setLiking(true);
    try {
      const result = post.likedByMe ? await unlike(post.id) : await like(post.id);
      onChanged({ ...post, likedByMe: result.liked, likeCount: result.likeCount });
    } catch (error) {
      notify({
        title: '按讚失敗',
        message: error instanceof Error ? error.message : '請再試一次。',
        tone: 'error',
      });
    } finally {
      if (mountedRef.current) {
        setLiking(false);
      }
    }
  };

  const submitComment = async (): Promise<void> => {
    const createComment = provider.createPostCommentAsync;
    if (createComment === undefined || submittingComment) {
      return;
    }
    const problem = validatePostComment(commentDraft);
    if (problem !== '') {
      setCommentError(problem);
      return;
    }
    setCommentError('');
    setSubmittingComment(true);
    try {
      const created = await createComment(
        post.id,
        commentDraft,
        replyTo === null ? undefined : replyTo.id,
      );
      if (!mountedRef.current) {
        return;
      }
      setComments((prev) => [...prev, created]);
      setCommentDraft('');
      setReplyTo(null);
      onChanged({ ...post, commentCount: post.commentCount + 1 });
    } catch (error) {
      const message = error instanceof Error ? error.message : '請再試一次。';
      if (mountedRef.current) {
        setCommentError(message);
      }
      notify({ title: '留言失敗', message, tone: 'error' });
    } finally {
      if (mountedRef.current) {
        setSubmittingComment(false);
      }
    }
  };

  const saveEdit = async (): Promise<void> => {
    const updatePost = provider.updatePostAsync;
    if (updatePost === undefined || savingEdit) {
      return;
    }
    const problem = validatePostContent(editContent, post.media.length);
    if (problem !== '') {
      setEditError(problem);
      return;
    }
    setEditError('');
    setSavingEdit(true);
    try {
      const updated = await updatePost(post.id, {
        content: editContent,
        visibility: editVisibility,
      });
      if (!mountedRef.current) {
        return;
      }
      onChanged(updated);
      setEditing(false);
      notify({ title: '已更新貼文', message: '貼文內容已更新。', tone: 'success' });
    } catch (error) {
      const message = error instanceof Error ? error.message : '請再試一次。';
      if (mountedRef.current) {
        setEditError(message);
      }
      notify({ title: '編輯失敗', message, tone: 'error' });
    } finally {
      if (mountedRef.current) {
        setSavingEdit(false);
      }
    }
  };

  const confirmPendingDelete = async (): Promise<void> => {
    if (pendingDelete === null || acting) {
      return;
    }
    const target = pendingDelete;
    setPendingDelete(null);
    setActing(true);
    try {
      if (target === 'post') {
        await provider.deletePostAsync?.(post.id);
        onRemoved(post.id);
        notify({ title: '已刪除貼文', message: '貼文已刪除，無法復原。', tone: 'success' });
      } else {
        await provider.deletePostCommentAsync?.(target.id);
        if (mountedRef.current) {
          setComments((prev) =>
            prev.map((comment) =>
              comment.id === target.id ? { ...comment, isDeleted: true, content: '' } : comment,
            ),
          );
        }
        onChanged({ ...post, commentCount: Math.max(0, post.commentCount - 1) });
        notify({ title: '已刪除留言', message: '留言已刪除。', tone: 'success' });
      }
    } catch (error) {
      notify({
        title: '刪除失敗',
        message: error instanceof Error ? error.message : '請再試一次。',
        tone: 'error',
      });
    } finally {
      if (mountedRef.current) {
        setActing(false);
      }
    }
  };

  const parentOf = (comment: PostCommentItem): PostCommentItem | undefined => {
    if (comment.parentCommentId === '') {
      return undefined;
    }
    return comments.find((item) => item.id === comment.parentCommentId);
  };

  return (
    <article className="et-post" aria-label={`作者 ${authorName} 的貼文`}>
      <header className="et-post__header">
        <Avatar name={authorName} size="sm" />
        <div className="et-post__author-block">
          <span className="et-post__author">{authorName}</span>
          <span className="et-post__meta">
            {formatDateTime(post.createdAt)}
            {' · '}
            {visibilityLabel(post.visibility)}
            {post.editedAt !== null ? ' · 已編輯' : ''}
          </span>
        </div>
        <Badge tone="neutral">{visibilityLabel(post.visibility)}</Badge>
      </header>
      {editing ? (
        <div className="et-post__edit">
          <TextArea
            label="編輯貼文內容"
            value={editContent}
            rows={4}
            onChange={(event: ChangeEvent<HTMLTextAreaElement>) => {
              setEditContent(event.target.value);
            }}
          />
          <Select
            label="能見度"
            value={editVisibility}
            options={VISIBILITY_OPTIONS}
            onChange={(event: ChangeEvent<HTMLSelectElement>) => {
              setEditVisibility(event.target.value as PostVisibility);
            }}
          />
          {editError !== '' ? (
            <p className="et-page__description" role="alert">
              {editError}
            </p>
          ) : null}
          <div className="et-post__edit-actions">
            <Button
              variant="secondary"
              disabled={savingEdit}
              onClick={() => {
                setEditing(false);
                setEditContent(post.content);
                setEditVisibility(post.visibility);
                setEditError('');
              }}
            >
              取消
            </Button>
            <Button disabled={savingEdit} loading={savingEdit} onClick={() => void saveEdit()}>
              儲存
            </Button>
          </div>
        </div>
      ) : (
        <p className="et-post__content">{post.content}</p>
      )}
      {post.media.length > 0 ? (
        <ul className="et-post__media" aria-label="貼文媒體">
          {post.media.map((media) => {
            const file = mediaById.get(media.mediaFileId);
            return (
              <li key={media.mediaFileId} className="et-post__media-chip">
                <span className="et-post__media-name">
                  {file?.originalFilename ?? media.mediaFileId}
                </span>
                <span className="et-post__media-meta">
                  {file?.mimeType ?? '媒體檔案'}
                  {' · '}
                  僅詮釋資料顯示，尚未開放下載
                </span>
              </li>
            );
          })}
        </ul>
      ) : null}
      <div className="et-post__actions">
        <Button
          size="sm"
          variant={post.likedByMe ? 'primary' : 'secondary'}
          disabled={liking || provider.likePostAsync === undefined}
          aria-pressed={post.likedByMe}
          onClick={() => void toggleLike()}
        >
          {post.likedByMe ? '已按讚' : '按讚'}（{post.likeCount}）
        </Button>
        <Button size="sm" variant="secondary" aria-expanded={commentsOpen} onClick={toggleComments}>
          留言（{post.commentCount}）
        </Button>
        {isMine && !editing ? (
          <>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => {
                setEditing(true);
                setEditContent(post.content);
                setEditVisibility(post.visibility);
                setEditError('');
              }}
            >
              編輯
            </Button>
            <Button
              size="sm"
              variant="danger"
              disabled={acting}
              onClick={() => {
                setPendingDelete('post');
              }}
            >
              刪除
            </Button>
          </>
        ) : null}
      </div>
      {commentsOpen ? (
        <section className="et-post__comments" aria-label="留言區">
          {commentsState === 'loading' ? <p className="et-page__description">載入留言中…</p> : null}
          {commentsState === 'error' ? (
            <>
              <p className="et-page__description">載入留言失敗，請再試一次。</p>
              <Button
                size="sm"
                variant="secondary"
                onClick={() => {
                  void loadComments('', false);
                }}
              >
                重新載入
              </Button>
            </>
          ) : null}
          {commentsState === 'ready' ? (
            <>
              {comments.length === 0 ? (
                <p className="et-page__description">還沒有留言。</p>
              ) : (
                <ul className="et-comment__list">
                  {comments.map((comment) => {
                    const parent = parentOf(comment);
                    const replyLabel =
                      parent === undefined
                        ? null
                        : parent.isDeleted
                          ? '回覆已刪除的留言'
                          : `回覆 ${nameOf(parent.authorId)}`;
                    return (
                      <li
                        key={comment.id}
                        className={
                          comment.parentCommentId === ''
                            ? 'et-comment'
                            : 'et-comment et-comment--reply'
                        }
                      >
                        <span className="et-comment__author">{nameOf(comment.authorId)}</span>
                        <span className="et-comment__meta">
                          {formatDateTime(comment.createdAt)}
                        </span>
                        {replyLabel !== null ? (
                          <span className="et-comment__reply-label">{replyLabel}</span>
                        ) : null}
                        <p className="et-comment__body">
                          {comment.isDeleted ? '此留言已刪除。' : comment.content}
                        </p>
                        {!comment.isDeleted && provider.createPostCommentAsync !== undefined ? (
                          <Button
                            size="sm"
                            variant="secondary"
                            onClick={() => {
                              setReplyTo(comment);
                            }}
                          >
                            回覆
                          </Button>
                        ) : null}
                        {!comment.isDeleted &&
                        myId !== null &&
                        comment.authorId === myId &&
                        provider.deletePostCommentAsync !== undefined ? (
                          <Button
                            size="sm"
                            variant="danger"
                            disabled={acting}
                            onClick={() => {
                              setPendingDelete(comment);
                            }}
                          >
                            刪除留言
                          </Button>
                        ) : null}
                      </li>
                    );
                  })}
                </ul>
              )}
              {commentsCursor !== '' ? (
                <Button
                  size="sm"
                  variant="secondary"
                  disabled={loadingMoreComments}
                  loading={loadingMoreComments}
                  onClick={() => {
                    void loadComments(commentsCursor, true);
                  }}
                >
                  載入更多留言
                </Button>
              ) : null}
              {provider.createPostCommentAsync !== undefined ? (
                <div className="et-post__comment-form">
                  {replyTo !== null ? (
                    <p className="et-page__description">
                      回覆 {replyTo.isDeleted ? '已刪除的留言' : nameOf(replyTo.authorId)}
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => {
                          setReplyTo(null);
                        }}
                      >
                        取消回覆
                      </Button>
                    </p>
                  ) : null}
                  <TextArea
                    label="寫留言"
                    value={commentDraft}
                    rows={2}
                    onChange={(event: ChangeEvent<HTMLTextAreaElement>) => {
                      setCommentDraft(event.target.value);
                    }}
                  />
                  {commentError !== '' ? (
                    <p className="et-page__description" role="alert">
                      {commentError}
                    </p>
                  ) : null}
                  <Button
                    size="sm"
                    disabled={submittingComment || commentDraft.trim() === ''}
                    loading={submittingComment}
                    onClick={() => void submitComment()}
                  >
                    送出留言
                  </Button>
                </div>
              ) : null}
            </>
          ) : null}
        </section>
      ) : null}
      {pendingDelete !== null ? (
        <Dialog
          open={pendingDelete !== null}
          onClose={() => {
            setPendingDelete(null);
          }}
          title={pendingDelete === 'post' ? '刪除貼文' : '刪除留言'}
          description={
            pendingDelete === 'post'
              ? '刪除後此貼文將不再顯示，且無法復原。按讚與留言紀錄保留於後端，但不再顯示。'
              : '刪除後此留言將顯示為已刪除，且無法復原。'
          }
          footer={
            <>
              <Button
                variant="secondary"
                onClick={() => {
                  setPendingDelete(null);
                }}
              >
                取消
              </Button>
              <Button
                variant="danger"
                disabled={acting}
                onClick={() => {
                  void confirmPendingDelete();
                }}
              >
                確認刪除
              </Button>
            </>
          }
        >
          <p className="et-page__description">此操作會立即生效。</p>
        </Dialog>
      ) : null}
    </article>
  );
}

/**
 * Timeline view (Phase 08, web wiring): friend feed and the
 * caller's own posts, post creation with visibility and media
 * associations, likes, comments and replies, and author-only
 * edit/delete. The same component serves demo mode (local
 * fixtures, labelled as such) and API mode (backend data); the
 * provider methods are optional, so absence renders the
 * unsupported state instead of an error.
 *
 * Honesty rules: there is no realtime push for posts, so the view
 * says new posts appear only after reloading; post media renders
 * as metadata only (no distribution URL exists in the post
 * response), never as a downloadable file.
 */
export function TimelineView({ provider, dataMode = 'demo' }: TimelineViewProps): ReactElement {
  const { notify } = useToast();
  const [tab, setTab] = useState<FeedTab>('feed');
  const [loadState, setLoadState] = useState<LoadState>('loading');
  const [posts, setPosts] = useState<PostItem[]>([]);
  const [nextCursor, setNextCursor] = useState('');
  const [loadingMore, setLoadingMore] = useState(false);
  const [composerContent, setComposerContent] = useState('');
  const [composerVisibility, setComposerVisibility] = useState<PostVisibility>('friends');
  const [composerMediaIds, setComposerMediaIds] = useState<string[]>([]);
  const [composerError, setComposerError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [mediaFiles, setMediaFiles] = useState<MediaFileItem[]>([]);
  const [mediaFilesState, setMediaFilesState] = useState<FilesState>('loading');
  const [names, setNames] = useState<Record<string, string>>({});
  const mountedRef = useRef(true);
  const myId = provider.getCurrentUserId?.() ?? null;

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const loadMediaFiles = useCallback(async (): Promise<void> => {
    const listFiles = provider.listMediaFilesAsync;
    if (listFiles === undefined) {
      if (mountedRef.current) {
        setMediaFilesState('unsupported');
      }
      return;
    }
    if (mountedRef.current) {
      setMediaFilesState('loading');
    }
    try {
      const files = await listFiles();
      if (mountedRef.current) {
        setMediaFiles(files);
        setMediaFilesState('ready');
      }
    } catch {
      if (mountedRef.current) {
        setMediaFilesState('error');
      }
    }
  }, [provider]);

  useEffect(() => {
    void loadMediaFiles();
  }, [loadMediaFiles]);

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
      // 名稱解析失敗不阻擋時間線：未知作者顯示識別碼縮寫。
    }
  }, [provider]);

  useEffect(() => {
    void loadNames();
  }, [loadNames]);

  const loadPosts = useCallback(
    async (nextTab: FeedTab, cursor: string, append: boolean): Promise<void> => {
      const listFeed = provider.listFeedAsync;
      const listAuthorPosts = provider.listAuthorPostsAsync;
      const currentUserId = provider.getCurrentUserId?.() ?? null;
      if (listFeed === undefined || listAuthorPosts === undefined) {
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
        const page =
          nextTab === 'feed'
            ? await listFeed(
                cursor === '' ? { limit: POST_PAGE_LIMIT } : { cursor, limit: POST_PAGE_LIMIT },
              )
            : currentUserId === null
              ? { posts: [], nextCursor: '' }
              : await listAuthorPosts(
                  currentUserId,
                  cursor === '' ? { limit: POST_PAGE_LIMIT } : { cursor, limit: POST_PAGE_LIMIT },
                );
        if (!mountedRef.current) {
          return;
        }
        setPosts((prev) => (append ? [...prev, ...page.posts] : page.posts));
        setNextCursor(page.nextCursor);
        setLoadState('ready');
      } catch {
        if (mountedRef.current) {
          setLoadState(append ? 'ready' : 'error');
          if (append) {
            notify({ title: '載入失敗', message: '載入更多貼文失敗，請再試一次。', tone: 'error' });
          }
        }
      } finally {
        if (mountedRef.current) {
          setLoadingMore(false);
        }
      }
    },
    [provider, notify],
  );

  useEffect(() => {
    void loadPosts(tab, '', false);
  }, [loadPosts, tab]);

  const nameOf = useCallback(
    (userId: string): string => {
      if (myId !== null && userId === myId) {
        return provider.getSnapshot().profile.name;
      }
      return names[userId] ?? userId.slice(0, 8);
    },
    [myId, names, provider],
  );

  const mediaById = new Map(mediaFiles.map((file) => [file.id, file]));

  const submitPost = async (): Promise<void> => {
    const createPost = provider.createPostAsync;
    if (createPost === undefined || submitting) {
      return;
    }
    const problem = validatePostContent(composerContent, composerMediaIds.length);
    if (problem !== '') {
      setComposerError(problem);
      return;
    }
    setComposerError('');
    setSubmitting(true);
    try {
      const created = await createPost({
        content: composerContent,
        visibility: composerVisibility,
        mediaFileIds: composerMediaIds,
      });
      if (!mountedRef.current) {
        return;
      }
      setPosts((prev) => [created, ...prev]);
      setComposerContent('');
      setComposerVisibility('friends');
      setComposerMediaIds([]);
      notify({ title: '已發佈貼文', message: '貼文已發佈到時間線。', tone: 'success' });
    } catch (error) {
      const message = error instanceof Error ? error.message : '請再試一次。';
      if (mountedRef.current) {
        setComposerError(message);
      }
      notify({ title: '發佈失敗', message, tone: 'error' });
    } finally {
      if (mountedRef.current) {
        setSubmitting(false);
      }
    }
  };

  const sourceLabel = dataMode === 'api' ? '貼文來自後端 API' : '展示資料 · 尚未連接後端';
  const canCompose = provider.createPostAsync !== undefined;

  return (
    <div className="et-page et-scroll">
      <div className="et-page__narrow">
        <header>
          <h2 className="et-page__title">社群動態</h2>
          <p className="et-page__description">
            {sourceLabel} · 沒有即時推播，重新載入後才會顯示新貼文。
          </p>
        </header>
        <div className="et-timeline__tabs" role="tablist" aria-label="時間線範圍">
          <Button
            size="sm"
            variant={tab === 'feed' ? 'primary' : 'secondary'}
            role="tab"
            aria-selected={tab === 'feed'}
            onClick={() => {
              setTab('feed');
            }}
          >
            好友動態
          </Button>
          <Button
            size="sm"
            variant={tab === 'mine' ? 'primary' : 'secondary'}
            role="tab"
            aria-selected={tab === 'mine'}
            onClick={() => {
              setTab('mine');
            }}
          >
            我的貼文
          </Button>
          <Button
            size="sm"
            variant="secondary"
            disabled={loadState !== 'ready' && loadState !== 'error'}
            onClick={() => {
              void loadPosts(tab, '', false);
            }}
          >
            重新載入
          </Button>
        </div>
        {loadState === 'loading' ? <p className="et-page__description">載入動態中…</p> : null}
        {loadState === 'error' ? (
          <>
            <p className="et-page__description">載入失敗，請檢查連線後再試一次。</p>
            <Button
              onClick={() => {
                void loadPosts(tab, '', false);
              }}
            >
              重新載入
            </Button>
          </>
        ) : null}
        {loadState === 'unsupported' ? (
          <p className="et-page__description">目前的資料模式不提供社群動態。</p>
        ) : null}
        {loadState === 'ready' ? (
          <>
            {canCompose ? (
              <section className="et-card" aria-labelledby="et-timeline-composer-heading">
                <h3 className="et-card__title" id="et-timeline-composer-heading">
                  發佈貼文
                </h3>
                <TextArea
                  label="貼文內容"
                  value={composerContent}
                  rows={4}
                  onChange={(event: ChangeEvent<HTMLTextAreaElement>) => {
                    setComposerContent(event.target.value);
                  }}
                />
                <Select
                  label="能見度"
                  value={composerVisibility}
                  options={VISIBILITY_OPTIONS}
                  onChange={(event: ChangeEvent<HTMLSelectElement>) => {
                    setComposerVisibility(event.target.value as PostVisibility);
                  }}
                />
                {provider.listMediaFilesAsync !== undefined ? (
                  <div className="et-timeline__composer-media">
                    <Button
                      size="sm"
                      variant="secondary"
                      onClick={() => {
                        setPickerOpen(true);
                      }}
                    >
                      選擇媒體（{composerMediaIds.length}/{MAX_POST_MEDIA}）
                    </Button>
                    {composerMediaIds.length > 0 ? (
                      <ul className="et-post__media" aria-label="已選媒體">
                        {composerMediaIds.map((fileId) => {
                          const file = mediaById.get(fileId);
                          return (
                            <li key={fileId} className="et-post__media-chip">
                              <span className="et-post__media-name">
                                {file?.originalFilename ?? fileId}
                              </span>
                              <Button
                                size="sm"
                                variant="secondary"
                                onClick={() => {
                                  setComposerMediaIds((prev) => prev.filter((id) => id !== fileId));
                                }}
                              >
                                移除
                              </Button>
                            </li>
                          );
                        })}
                      </ul>
                    ) : null}
                  </div>
                ) : null}
                {composerError !== '' ? (
                  <p className="et-page__description" role="alert">
                    {composerError}
                  </p>
                ) : null}
                <Button
                  disabled={submitting}
                  loading={submitting}
                  onClick={() => void submitPost()}
                >
                  發佈
                </Button>
              </section>
            ) : null}
            {posts.length === 0 ? (
              <EmptyState
                title={tab === 'feed' ? '還沒有動態' : '你還沒有發佈貼文'}
                description={
                  tab === 'feed'
                    ? '好友發佈貼文後會顯示在這裡。'
                    : '發佈第一則貼文，讓好友看到你的近況。'
                }
              />
            ) : (
              <div className="et-timeline__list">
                {posts.map((post) => (
                  <PostCard
                    key={post.id}
                    post={post}
                    myId={myId}
                    authorName={nameOf(post.authorId)}
                    mediaById={mediaById}
                    nameOf={nameOf}
                    provider={provider}
                    onChanged={(updated) => {
                      setPosts((prev) =>
                        prev.map((item) => (item.id === updated.id ? updated : item)),
                      );
                    }}
                    onRemoved={(postId) => {
                      setPosts((prev) => prev.filter((item) => item.id !== postId));
                    }}
                  />
                ))}
              </div>
            )}
            {nextCursor !== '' ? (
              <Button
                disabled={loadingMore}
                loading={loadingMore}
                onClick={() => {
                  void loadPosts(tab, nextCursor, true);
                }}
              >
                載入更多
              </Button>
            ) : null}
          </>
        ) : null}
        <PostMediaPickerDialog
          open={pickerOpen}
          files={mediaFiles}
          filesState={mediaFilesState}
          selectedIds={composerMediaIds}
          onRetryFiles={() => {
            void loadMediaFiles();
          }}
          onClose={() => {
            setPickerOpen(false);
          }}
          onConfirm={(ids) => {
            setComposerMediaIds(ids);
            setPickerOpen(false);
          }}
        />
      </div>
    </div>
  );
}
