import { useCallback, useEffect, useRef, useState } from 'react';
import type { ChangeEvent, ReactElement } from 'react';
import type { ChatDataProvider } from '../data/provider';
import type {
  CallEventItem,
  CallItem,
  CallParticipantItem,
  CallStatus,
  CallType,
  ConversationItem,
  GroupItem,
  InitiateCallInput,
} from '../data/types';
import { Avatar } from '../components/Avatar';
import { Badge } from '../components/Badge';
import { Button } from '../components/Button';
import { Dialog } from '../components/Dialog';
import { EmptyState } from '../components/EmptyState';
import { Select } from '../components/Select';
import { useToast } from '../components/Toast';
import { PhoneIcon, VideoIcon } from '../components/icons';

export interface CallsViewProps {
  provider: ChatDataProvider;
  dataMode?: 'demo' | 'api';
}

type LoadState = 'loading' | 'ready' | 'error' | 'unsupported';
type DetailState = 'loading' | 'ready' | 'error' | 'unsupported';
type ScopeTab = 'direct' | 'group';

const CALL_PAGE_LIMIT = 20;

const STATUS_LABELS: Record<CallStatus, string> = {
  ringing: '響鈴中',
  active: '通話中',
  declined: '全員拒接',
  missed: '未接',
  busy: '全員忙線',
  ended: '已結束',
};

const TYPE_LABELS: Record<CallType, string> = {
  voice: '語音通話',
  video: '視訊通話',
};

const RESULT_LABELS: Record<CallParticipantItem['result'], string> = {
  pending: '待回應',
  joined: '已加入',
  declined: '已拒接',
  busy: '忙線',
  missed: '未接',
};

const EVENT_LABELS: Record<CallEventItem['eventType'], string> = {
  initiated: '發起通話',
  answered: '接聽',
  declined: '拒接',
  busy: '忙線',
  ended: '結束通話',
};

const END_REASON_LABELS: Record<string, string> = {
  hangup: '掛斷',
  declined: '拒接',
  missed: '未接',
  busy: '忙線',
};

/** 通話狀態中文標籤（與後端狀態詞彙對應，待定稿覆核）。 */
export function callStatusLabel(status: CallStatus): string {
  return STATUS_LABELS[status];
}

/** 通話類型中文標籤。 */
export function callTypeLabel(callType: CallType): string {
  return TYPE_LABELS[callType];
}

/** 參與者結果中文標籤。 */
export function participantResultLabel(result: CallParticipantItem['result']): string {
  return RESULT_LABELS[result];
}

/** 通話事件中文標籤。事件不帶 payload，只顯示類型與時間。 */
export function callEventLabel(eventType: CallEventItem['eventType']): string {
  return EVENT_LABELS[eventType];
}

/** 結束原因中文標籤；空值回空字串。 */
export function endReasonLabel(endReason: string): string {
  return END_REASON_LABELS[endReason] ?? '';
}

/** 終端狀態判定：結束後不得再轉移。 */
export function isCallTerminal(status: CallStatus): boolean {
  return status === 'declined' || status === 'missed' || status === 'busy' || status === 'ended';
}

/** 通話時長格式化：null 回破折號，不捏造數字。 */
export function formatCallDuration(durationSeconds: number | null): string {
  if (durationSeconds === null) {
    return '—';
  }
  const total = Math.max(0, Math.floor(durationSeconds));
  if (total < 60) {
    return `${total} 秒`;
  }
  const minutes = Math.floor(total / 60);
  const seconds = total % 60;
  if (minutes < 60) {
    return seconds === 0 ? `${minutes} 分` : `${minutes} 分 ${seconds} 秒`;
  }
  const hours = Math.floor(minutes / 60);
  const restMinutes = minutes % 60;
  return restMinutes === 0 ? `${hours} 小時` : `${hours} 小時 ${restMinutes} 分`;
}

/** 自 startedAt 起算的已通話秒數（僅以 startedAt 計算，不含其他時間來源）。 */
export function elapsedSeconds(startedAt: string | null, nowMs: number): number {
  if (startedAt === null) {
    return 0;
  }
  const startedMs = Date.parse(startedAt);
  if (Number.isNaN(startedMs)) {
    return 0;
  }
  return Math.max(0, Math.floor((nowMs - startedMs) / 1000));
}

export type CallActionState = 'answer' | 'end' | 'none';

/**
 * 依通話狀態與本人的參與者紀錄，判定可用的操作（與後端守衛同規則）：
 * 響鈴中且本人待回應 → 接聽；本人已加入 → 結束；其餘皆無操作。
 * 終端狀態一律無操作。
 */
export function callActionState(
  call: CallItem,
  myParticipant: CallParticipantItem | undefined,
): CallActionState {
  if (isCallTerminal(call.status)) {
    return 'none';
  }
  if (call.status === 'ringing' && myParticipant?.result === 'pending') {
    return 'answer';
  }
  if (myParticipant?.result === 'joined') {
    return 'end';
  }
  return 'none';
}

function formatDateTime(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString('zh-Hant');
}

function statusTone(status: CallStatus): 'neutral' | 'success' | 'warning' | 'error' {
  if (status === 'active') {
    return 'success';
  }
  if (status === 'ringing') {
    return 'warning';
  }
  if (status === 'missed' || status === 'declined' || status === 'busy') {
    return 'error';
  }
  return 'neutral';
}

interface CallCardProps {
  call: CallItem;
  counterpartName: string;
  initiatorName: string;
  selected: boolean;
  onSelect: (callId: string) => void;
}

/**
 * 通話紀錄卡片（§2ae）：狀態徽章、類型、對象、時間與時長摘要。
 * 不顯示任何媒體狀態：媒體尚未接通，卡片只呈現會話紀錄。
 */
export function CallCard({
  call,
  counterpartName,
  initiatorName,
  selected,
  onSelect,
}: CallCardProps): ReactElement {
  return (
    <article className="et-call" aria-label={`${callTypeLabel(call.callType)}，${counterpartName}`}>
      <div className="et-call__header">
        <Avatar name={counterpartName} size="sm" />
        <div className="et-call__title-block">
          <p className="et-call__title">{counterpartName}</p>
          <p className="et-call__meta">
            {callTypeLabel(call.callType)} · 由{initiatorName}發起 ·{' '}
            {formatDateTime(call.createdAt)}
          </p>
        </div>
        <Badge tone={statusTone(call.status)}>{callStatusLabel(call.status)}</Badge>
      </div>
      <div className="et-call__summary">
        <span>
          {call.participantCount} 位參與者
          {call.status === 'ended' && call.durationSeconds !== null
            ? ` · 通話時長 ${formatCallDuration(call.durationSeconds)}`
            : ''}
          {call.endReason !== '' ? ` · ${endReasonLabel(call.endReason)}` : ''}
        </span>
        <Button
          size="sm"
          variant={selected ? 'primary' : 'secondary'}
          onClick={() => {
            onSelect(call.id);
          }}
        >
          {selected ? '已選取' : '查看詳情'}
        </Button>
      </div>
    </article>
  );
}

interface CallDetailProps {
  call: CallItem;
  participants: CallParticipantItem[];
  participantsState: DetailState;
  events: CallEventItem[];
  eventsState: DetailState;
  myId: string | null;
  nameOf: (userId: string) => string;
  counterpartName: string;
  acting: boolean;
  nowMs: number;
  onAnswer: (callId: string) => void;
  onDecline: (callId: string) => void;
  onBusy: (callId: string) => void;
  onEnd: (callId: string) => void;
  onReload: (callId: string) => void;
  onClose: () => void;
}

/**
 * 通話詳情（§2ae）：參與者狀態列、狀態事件列、依身分與狀態顯示的
 * 操作按鈕。通話中或響鈴中皆明載「媒體尚未接通，僅會話狀態」，
 * 不得假裝音訊或視訊已接通。
 */
export function CallDetailPanel({
  call,
  participants,
  participantsState,
  events,
  eventsState,
  myId,
  nameOf,
  counterpartName,
  acting,
  nowMs,
  onAnswer,
  onDecline,
  onBusy,
  onEnd,
  onReload,
  onClose,
}: CallDetailProps): ReactElement {
  const [endConfirmOpen, setEndConfirmOpen] = useState(false);
  const myParticipant = participants.find((participant) => participant.userId === myId);
  const actions = callActionState(call, myParticipant);
  const TypeIcon = call.callType === 'video' ? VideoIcon : PhoneIcon;

  return (
    <article className="et-call-detail" aria-label={`通話詳情，${counterpartName}`}>
      <div className="et-call-detail__header">
        <span aria-hidden="true">
          <TypeIcon size={20} />
        </span>
        <div className="et-call__title-block">
          <h3 className="et-call__title">{counterpartName}</h3>
          <p className="et-call__meta">
            {callTypeLabel(call.callType)} · 由{nameOf(call.initiatorId)}發起 ·{' '}
            {formatDateTime(call.createdAt)}
          </p>
        </div>
        <Badge tone={statusTone(call.status)}>{callStatusLabel(call.status)}</Badge>
      </div>
      {call.status === 'ringing' || call.status === 'active' ? (
        <p className="et-page__description">媒體尚未接通，僅會話狀態。</p>
      ) : null}
      {call.status === 'active' && call.startedAt !== null ? (
        <p className="et-call-detail__elapsed">
          已通話 {formatCallDuration(elapsedSeconds(call.startedAt, nowMs))}
        </p>
      ) : null}
      {isCallTerminal(call.status) ? (
        <p className="et-call__meta">
          {call.endedAt !== null ? `結束於 ${formatDateTime(call.endedAt)}` : ''}
          {call.endReason !== '' ? ` · ${endReasonLabel(call.endReason)}` : ''}
          {call.status === 'ended' ? ` · 通話時長 ${formatCallDuration(call.durationSeconds)}` : ''}
        </p>
      ) : null}
      <div className="et-call-detail__actions">
        {actions === 'answer' ? (
          <>
            <Button
              disabled={acting}
              onClick={() => {
                onAnswer(call.id);
              }}
            >
              接聽
            </Button>
            <Button
              variant="secondary"
              disabled={acting}
              onClick={() => {
                onDecline(call.id);
              }}
            >
              拒接
            </Button>
            <Button
              variant="secondary"
              disabled={acting}
              onClick={() => {
                onBusy(call.id);
              }}
            >
              忙線
            </Button>
          </>
        ) : null}
        {actions === 'end' ? (
          <Button
            variant="danger"
            disabled={acting}
            onClick={() => {
              setEndConfirmOpen(true);
            }}
          >
            結束通話
          </Button>
        ) : null}
        <Button
          size="sm"
          variant="secondary"
          disabled={acting}
          onClick={() => {
            onReload(call.id);
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
      <section aria-label="參與者">
        <h4 className="et-call-detail__section-title">參與者（{participants.length} 位）</h4>
        {participantsState === 'loading' ? (
          <p className="et-page__description">載入參與者中…</p>
        ) : participantsState === 'error' ? (
          <p className="et-page__description">參與者載入失敗，請重新載入。</p>
        ) : participantsState === 'unsupported' ? (
          <p className="et-page__description">此資料來源不提供參與者清單。</p>
        ) : participants.length === 0 ? (
          <p className="et-page__description">尚無參與者紀錄。</p>
        ) : (
          <ul className="et-calls__list">
            {participants.map((participant) => (
              <li key={participant.userId} className="et-calls__row">
                <span>{nameOf(participant.userId)}</span>
                <Badge>{participantResultLabel(participant.result)}</Badge>
              </li>
            ))}
          </ul>
        )}
      </section>
      <section aria-label="通話事件">
        <h4 className="et-call-detail__section-title">通話事件</h4>
        {eventsState === 'loading' ? (
          <p className="et-page__description">載入事件中…</p>
        ) : eventsState === 'error' ? (
          <p className="et-page__description">事件載入失敗，請重新載入。</p>
        ) : eventsState === 'unsupported' ? (
          <p className="et-page__description">此資料來源不提供通話事件。</p>
        ) : events.length === 0 ? (
          <p className="et-page__description">尚無通話事件。</p>
        ) : (
          <ul className="et-calls__list">
            {events.map((event) => (
              <li key={event.id} className="et-calls__row">
                <span>
                  {callEventLabel(event.eventType)}
                  {event.userId !== '' ? ` · ${nameOf(event.userId)}` : ''}
                </span>
                <span className="et-call__meta">{formatDateTime(event.occurredAt)}</span>
              </li>
            ))}
          </ul>
        )}
      </section>
      <Dialog
        open={endConfirmOpen}
        onClose={() => {
          setEndConfirmOpen(false);
        }}
        title="結束這通通話？"
        description="結束後通話即為終端狀態，不得重開；通話紀錄會保留。"
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => {
                setEndConfirmOpen(false);
              }}
            >
              取消
            </Button>
            <Button
              variant="danger"
              disabled={acting}
              onClick={() => {
                setEndConfirmOpen(false);
                onEnd(call.id);
              }}
            >
              結束通話
            </Button>
          </>
        }
      >
        <p className="et-page__description">此操作會立即生效。</p>
      </Dialog>
    </article>
  );
}

interface InitiateCallDialogProps {
  open: boolean;
  conversations: ConversationItem[];
  groups: GroupItem[];
  candidatesState: LoadState;
  onRetryCandidates: () => void;
  onClose: () => void;
  onConfirm: (input: InitiateCallInput) => Promise<void>;
}

/**
 * 發起通話對話框（§2ae）：一對一對話與群組雙頁籤、名稱搜尋、
 * 語音／視訊選擇。只送出範圍識別碼與類型；同一範圍已有進行中
 * 通話時後端拒絕，錯誤顯示於對話框內，不另開新通話。
 */
export function InitiateCallDialog({
  open,
  conversations,
  groups,
  candidatesState,
  onRetryCandidates,
  onClose,
  onConfirm,
}: InitiateCallDialogProps): ReactElement {
  const [scope, setScope] = useState<ScopeTab>('direct');
  const [selectedId, setSelectedId] = useState('');
  const [callType, setCallType] = useState<CallType>('voice');
  const [query, setQuery] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (open) {
      setScope('direct');
      setSelectedId('');
      setCallType('voice');
      setQuery('');
      setError('');
      setSubmitting(false);
    }
  }, [open]);

  const candidates =
    scope === 'direct'
      ? conversations
          .filter((conversation) => conversation.type === 'direct')
          .map((conversation) => ({ id: conversation.id, name: conversation.title }))
      : groups.map((group) => ({ id: group.id, name: group.name }));
  const visible = candidates.filter((candidate) =>
    query.trim() === '' ? true : candidate.name.includes(query.trim()),
  );

  const submit = async (): Promise<void> => {
    if (selectedId === '' || submitting) {
      return;
    }
    setError('');
    setSubmitting(true);
    try {
      await onConfirm(
        scope === 'direct'
          ? { conversationId: selectedId, callType }
          : { groupId: selectedId, callType },
      );
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : '發起通話失敗，請再試一次。');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="發起通話"
      description="選擇一對一對話或群組，以及通話類型。媒體尚未接通，僅建立通話會話。"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            取消
          </Button>
          <Button disabled={selectedId === '' || submitting} onClick={() => void submit()}>
            {submitting ? '發起中…' : '發起通話'}
          </Button>
        </>
      }
    >
      <div className="et-calls__dialog-body">
        <div className="et-timeline__tabs" role="tablist" aria-label="通話範圍">
          <Button
            size="sm"
            variant={scope === 'direct' ? 'primary' : 'secondary'}
            role="tab"
            aria-selected={scope === 'direct'}
            onClick={() => {
              setScope('direct');
              setSelectedId('');
            }}
          >
            一對一對話
          </Button>
          <Button
            size="sm"
            variant={scope === 'group' ? 'primary' : 'secondary'}
            role="tab"
            aria-selected={scope === 'group'}
            onClick={() => {
              setScope('group');
              setSelectedId('');
            }}
          >
            群組
          </Button>
        </div>
        <label className="et-calls__field" htmlFor="initiate-call-search">
          搜尋對象
          <input
            id="initiate-call-search"
            type="search"
            value={query}
            onChange={(event: ChangeEvent<HTMLInputElement>) => {
              setQuery(event.target.value);
            }}
            placeholder="輸入名稱搜尋"
          />
        </label>
        {candidatesState === 'loading' ? (
          <p className="et-page__description">載入對象中…</p>
        ) : candidatesState === 'error' ? (
          <>
            <p className="et-page__description">對象載入失敗，請再試一次。</p>
            <Button size="sm" variant="secondary" onClick={onRetryCandidates}>
              重新載入
            </Button>
          </>
        ) : candidatesState === 'unsupported' ? (
          <p className="et-page__description">此資料來源不提供發起通話的對象清單。</p>
        ) : visible.length === 0 ? (
          <p className="et-page__description">
            {query.trim() === '' ? '尚無可發起通話的對象。' : '沒有符合搜尋的對象。'}
          </p>
        ) : (
          <ul className="et-calls__list">
            {visible.map((candidate) => (
              <li key={candidate.id} className="et-calls__row">
                <label className="et-calls__pick">
                  <input
                    type="radio"
                    name="initiate-call-target"
                    value={candidate.id}
                    checked={selectedId === candidate.id}
                    onChange={() => {
                      setSelectedId(candidate.id);
                    }}
                  />
                  {candidate.name === '' ? candidate.id.slice(0, 8) : candidate.name}
                </label>
              </li>
            ))}
          </ul>
        )}
        <Select
          label="通話類型"
          value={callType}
          options={[
            { value: 'voice', label: '語音通話' },
            { value: 'video', label: '視訊通話' },
          ]}
          onChange={(event: ChangeEvent<HTMLSelectElement>) => {
            setCallType(event.target.value as CallType);
          }}
        />
        {error !== '' ? (
          <p className="et-page__description" role="alert">
            {error}
          </p>
        ) : null}
      </div>
    </Dialog>
  );
}

/**
 * Calls view (Phase 07, web wiring): call history with cursor
 * pagination, call initiation from direct conversations and groups,
 * incoming-call responses and in-call actions, participants and
 * state events. The same component serves demo mode (local
 * fixtures, labelled as such) and API mode (backend data); the
 * provider methods are optional, so absence renders the
 * unsupported state instead of an error.
 *
 * Honesty rules: there is no realtime signaling or incoming-call
 * push, so the view says the state only syncs on reload; no media
 * is ever involved at this layer, and the in-call panel states
 * plainly that media is not connected and only the session state
 * is shown. Call records are append-only: no delete affordance
 * exists, matching the backend.
 */
export function CallsView({ provider, dataMode = 'demo' }: CallsViewProps): ReactElement {
  const { notify } = useToast();
  const [loadState, setLoadState] = useState<LoadState>('loading');
  const [calls, setCalls] = useState<CallItem[]>([]);
  const [nextCursor, setNextCursor] = useState('');
  const [loadingMore, setLoadingMore] = useState(false);
  const [selectedCallId, setSelectedCallId] = useState<string | null>(null);
  const [detail, setDetail] = useState<CallItem | null>(null);
  const [detailState, setDetailState] = useState<DetailState>('loading');
  const [participants, setParticipants] = useState<CallParticipantItem[]>([]);
  const [participantsState, setParticipantsState] = useState<DetailState>('loading');
  const [events, setEvents] = useState<CallEventItem[]>([]);
  const [eventsState, setEventsState] = useState<DetailState>('loading');
  const [names, setNames] = useState<Record<string, string>>({});
  const [groupNames, setGroupNames] = useState<Record<string, string>>({});
  const [conversationTitles, setConversationTitles] = useState<Record<string, string>>({});
  const [conversations, setConversations] = useState<ConversationItem[]>([]);
  const [groups, setGroups] = useState<GroupItem[]>([]);
  const [candidatesState, setCandidatesState] = useState<LoadState>('loading');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [acting, setActing] = useState(false);
  const [nowMs, setNowMs] = useState(() => Date.now());
  // 本人在各通話中的參與者結果（僅響鈴中來電需要）：來電橫幅只
  // 顯示本人尚未回應的來電；查不到時顯示通話（操作仍受後端守衛）。
  const [myResults, setMyResults] = useState<Record<string, string>>({});
  const mountedRef = useRef(true);
  const myId = provider.getCurrentUserId?.() ?? null;

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const loadCalls = useCallback(
    async (cursor: string, append: boolean): Promise<void> => {
      const listCalls = provider.listCallsAsync;
      if (listCalls === undefined) {
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
        const page = await listCalls(
          cursor === '' ? { limit: CALL_PAGE_LIMIT } : { cursor, limit: CALL_PAGE_LIMIT },
        );
        if (!mountedRef.current) {
          return;
        }
        setCalls((prev) => (append ? [...prev, ...page.calls] : page.calls));
        setNextCursor(page.nextCursor);
        setLoadState('ready');
        const listParticipants = provider.listCallParticipantsAsync;
        if (listParticipants !== undefined) {
          const ringing = page.calls.filter(
            (call) => call.status === 'ringing' && call.initiatorId !== myId,
          );
          const settled = await Promise.allSettled(
            ringing.map(async (call) => {
              const found = await listParticipants(call.id);
              const mine = found.find((participant) => participant.userId === myId);
              return { callId: call.id, result: mine?.result ?? '' };
            }),
          );
          if (mountedRef.current) {
            const next: Record<string, string> = {};
            settled.forEach((entry) => {
              if (entry.status === 'fulfilled' && entry.value.result !== '') {
                next[entry.value.callId] = entry.value.result;
              }
            });
            setMyResults((prev) => (append ? { ...prev, ...next } : next));
          }
        }
      } catch {
        if (mountedRef.current) {
          setLoadState(append ? 'ready' : 'error');
          if (append) {
            notify({ title: '載入失敗', message: '載入更多通話失敗，請再試一次。', tone: 'error' });
          }
        }
      } finally {
        if (mountedRef.current) {
          setLoadingMore(false);
        }
      }
    },
    [myId, provider, notify],
  );

  useEffect(() => {
    void loadCalls('', false);
  }, [loadCalls]);

  const loadCandidates = useCallback(async (): Promise<void> => {
    const listConversations = provider.listConversationsAsync;
    const listGroups = provider.listGroupsAsync;
    if (listConversations === undefined || listGroups === undefined) {
      if (mountedRef.current) {
        setCandidatesState('unsupported');
      }
      return;
    }
    if (mountedRef.current) {
      setCandidatesState('loading');
    }
    try {
      const [foundConversations, foundGroups] = await Promise.all([
        listConversations(),
        listGroups(),
      ]);
      if (!mountedRef.current) {
        return;
      }
      setConversations(foundConversations);
      setGroups(foundGroups);
      setConversationTitles(
        Object.fromEntries(foundConversations.map((item) => [item.id, item.title])),
      );
      setGroupNames(Object.fromEntries(foundGroups.map((item) => [item.id, item.name])));
      setCandidatesState('ready');
    } catch {
      if (mountedRef.current) {
        setCandidatesState('error');
      }
    }
  }, [provider]);

  useEffect(() => {
    void loadCandidates();
  }, [loadCandidates]);

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
      // 名稱解析失敗不阻擋通話清單：未知使用者顯示識別碼縮寫。
    }
  }, [provider]);

  useEffect(() => {
    void loadNames();
  }, [loadNames]);

  const loadDetail = useCallback(
    async (callId: string): Promise<void> => {
      const getCall = provider.getCallAsync;
      const listParticipants = provider.listCallParticipantsAsync;
      const listEvents = provider.listCallEventsAsync;
      if (getCall === undefined) {
        if (mountedRef.current) {
          setDetailState('unsupported');
        }
        return;
      }
      if (mountedRef.current) {
        setDetailState('loading');
        setParticipantsState(listParticipants === undefined ? 'unsupported' : 'loading');
        setEventsState(listEvents === undefined ? 'unsupported' : 'loading');
      }
      try {
        const found = await getCall(callId);
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
      if (listParticipants !== undefined) {
        try {
          const foundParticipants = await listParticipants(callId);
          if (mountedRef.current) {
            setParticipants(foundParticipants);
            setParticipantsState('ready');
          }
        } catch {
          if (mountedRef.current) {
            setParticipantsState('error');
          }
        }
      }
      if (listEvents !== undefined) {
        try {
          const foundEvents = await listEvents(callId);
          if (mountedRef.current) {
            setEvents(foundEvents);
            setEventsState('ready');
          }
        } catch {
          if (mountedRef.current) {
            setEventsState('error');
          }
        }
      }
    },
    [provider],
  );

  useEffect(() => {
    if (selectedCallId !== null) {
      void loadDetail(selectedCallId);
    }
  }, [loadDetail, selectedCallId]);

  // 通話計時只以 startedAt 計算：僅在有通話中的詳情開啟時每秒
  // 更新一次畫面，不建立任何媒體連線。
  useEffect(() => {
    if (detail === null || detail.status !== 'active') {
      return undefined;
    }
    const timer = setInterval(() => {
      setNowMs(Date.now());
    }, 1000);
    return () => {
      clearInterval(timer);
    };
  }, [detail]);

  const nameOf = useCallback(
    (userId: string): string => {
      if (myId !== null && userId === myId) {
        return provider.getSnapshot().profile.name;
      }
      return names[userId] ?? userId.slice(0, 8);
    },
    [myId, names, provider],
  );

  const counterpartName = useCallback(
    (call: CallItem): string => {
      if (call.groupId !== '') {
        return groupNames[call.groupId] ?? call.groupId.slice(0, 8);
      }
      if (call.conversationId !== '') {
        const title = conversationTitles[call.conversationId];
        if (title !== undefined && title !== '') {
          return title;
        }
      }
      // 無標題時不猜測對方身分：非發起人的來電顯示發起人名稱，
      // 本人發起的通話只顯示範圍種類，避免把發起人誤寫為對方。
      if (call.initiatorId !== myId) {
        return nameOf(call.initiatorId);
      }
      return '一對一通話';
    },
    [conversationTitles, groupNames, myId, nameOf],
  );

  const refreshAfterAction = useCallback(
    async (callId: string): Promise<void> => {
      await loadCalls('', false);
      await loadDetail(callId);
    },
    [loadCalls, loadDetail],
  );

  const respond = useCallback(
    async (callId: string, action: 'answer' | 'decline' | 'busy' | 'end'): Promise<void> => {
      const method =
        action === 'answer'
          ? provider.answerCallAsync
          : action === 'decline'
            ? provider.declineCallAsync
            : action === 'busy'
              ? provider.busyCallAsync
              : provider.endCallAsync;
      if (method === undefined || acting) {
        return;
      }
      setActing(true);
      try {
        await method(callId);
        if (!mountedRef.current) {
          return;
        }
        await refreshAfterAction(callId);
        notify({
          title:
            action === 'answer'
              ? '已接聽'
              : action === 'end'
                ? '已結束通話'
                : action === 'decline'
                  ? '已拒接'
                  : '已回報忙線',
          message: '通話狀態已更新。',
          tone: 'success',
        });
      } catch (error) {
        const message = error instanceof Error ? error.message : '請再試一次。';
        notify({ title: '操作失敗', message, tone: 'error' });
      } finally {
        if (mountedRef.current) {
          setActing(false);
        }
      }
    },
    [acting, notify, provider, refreshAfterAction],
  );

  const initiate = useCallback(
    async (input: InitiateCallInput): Promise<void> => {
      const initiateCall = provider.initiateCallAsync;
      if (initiateCall === undefined) {
        throw new Error('此資料來源不支援發起通話。');
      }
      const created = await initiateCall(input);
      if (!mountedRef.current) {
        return;
      }
      setDialogOpen(false);
      setSelectedCallId(created.id);
      await loadCalls('', false);
      await loadDetail(created.id);
      notify({ title: '已發起通話', message: '對方尚未接聽，僅建立通話會話。', tone: 'success' });
    },
    [loadCalls, loadDetail, notify, provider],
  );

  const incoming = calls.filter((call) => {
    if (call.status !== 'ringing' || call.initiatorId === myId) {
      return false;
    }
    const result = myResults[call.id];
    return result === undefined || result === 'pending';
  });

  const sourceLabel = dataMode === 'api' ? '通話紀錄來自後端 API' : '展示資料 · 尚未連接後端';
  const canInitiate = provider.initiateCallAsync !== undefined;

  return (
    <div className="et-page et-scroll">
      <div className="et-page__narrow">
        <header>
          <h2 className="et-page__title">通話</h2>
          <p className="et-page__description">
            {sourceLabel} · 沒有即時來電推播與信令，重新載入後才會同步最新狀態。
          </p>
        </header>
        <div className="et-calls__toolbar">
          <Button
            disabled={!canInitiate}
            onClick={() => {
              setDialogOpen(true);
            }}
          >
            發起通話
          </Button>
          <Button
            size="sm"
            variant="secondary"
            disabled={loadState !== 'ready' && loadState !== 'error'}
            onClick={() => {
              void loadCalls('', false);
            }}
          >
            重新載入
          </Button>
        </div>
        {incoming.length > 0 ? (
          <section aria-label="來電">
            <h3 className="et-call-detail__section-title">來電</h3>
            <div className="et-timeline__list">
              {incoming.map((call) => (
                <article
                  key={call.id}
                  className="et-call"
                  aria-label={`來電，${counterpartName(call)}`}
                >
                  <div className="et-call__header">
                    <Avatar name={counterpartName(call)} size="sm" />
                    <div className="et-call__title-block">
                      <p className="et-call__title">{counterpartName(call)}</p>
                      <p className="et-call__meta">
                        {callTypeLabel(call.callType)} · 由{nameOf(call.initiatorId)}發起 ·{' '}
                        {formatDateTime(call.createdAt)}
                      </p>
                    </div>
                    <Badge tone="warning">來電響鈴中</Badge>
                  </div>
                  <p className="et-page__description">媒體尚未接通，僅會話狀態。</p>
                  <div className="et-call-detail__actions">
                    <Button
                      disabled={acting}
                      onClick={() => {
                        void respond(call.id, 'answer');
                      }}
                    >
                      接聽
                    </Button>
                    <Button
                      variant="secondary"
                      disabled={acting}
                      onClick={() => {
                        void respond(call.id, 'decline');
                      }}
                    >
                      拒接
                    </Button>
                    <Button
                      variant="secondary"
                      disabled={acting}
                      onClick={() => {
                        void respond(call.id, 'busy');
                      }}
                    >
                      忙線
                    </Button>
                    <Button
                      size="sm"
                      variant="secondary"
                      onClick={() => {
                        setSelectedCallId(call.id);
                      }}
                    >
                      查看詳情
                    </Button>
                  </div>
                </article>
              ))}
            </div>
          </section>
        ) : null}
        {loadState === 'loading' ? <p className="et-page__description">載入通話中…</p> : null}
        {loadState === 'error' ? (
          <>
            <p className="et-page__description">載入失敗，請檢查連線後再試一次。</p>
            <Button
              onClick={() => {
                void loadCalls('', false);
              }}
            >
              重新載入
            </Button>
          </>
        ) : null}
        {loadState === 'unsupported' ? (
          <EmptyState
            icon={<PhoneIcon size={28} />}
            title="不支援通話紀錄"
            description="目前的資料來源不提供通話會話查詢。"
          />
        ) : null}
        {loadState === 'ready' && calls.length === 0 ? (
          <EmptyState
            icon={<PhoneIcon size={28} />}
            title="尚無通話紀錄"
            description="發起第一通通話後，紀錄會顯示在這裡。"
          />
        ) : null}
        {loadState === 'ready' && calls.length > 0 ? (
          <div className="et-timeline__list">
            {calls.map((call) => (
              <CallCard
                key={call.id}
                call={call}
                counterpartName={counterpartName(call)}
                initiatorName={nameOf(call.initiatorId)}
                selected={selectedCallId === call.id}
                onSelect={(callId) => {
                  setSelectedCallId(callId);
                }}
              />
            ))}
          </div>
        ) : null}
        {loadState === 'ready' && nextCursor !== '' ? (
          <div className="et-calls__toolbar">
            <Button
              variant="secondary"
              disabled={loadingMore}
              onClick={() => {
                void loadCalls(nextCursor, true);
              }}
            >
              {loadingMore ? '載入中…' : '載入更多'}
            </Button>
          </div>
        ) : null}
        {selectedCallId !== null && detailState === 'loading' ? (
          <p className="et-page__description">載入通話詳情中…</p>
        ) : null}
        {selectedCallId !== null && detailState === 'error' ? (
          <>
            <p className="et-page__description">詳情載入失敗，請再試一次。</p>
            <Button
              onClick={() => {
                void loadDetail(selectedCallId);
              }}
            >
              重新載入
            </Button>
          </>
        ) : null}
        {selectedCallId !== null && detailState === 'unsupported' ? (
          <EmptyState
            icon={<PhoneIcon size={28} />}
            title="不支援通話詳情"
            description="目前的資料來源不提供單則通話查詢。"
          />
        ) : null}
        {selectedCallId !== null && detailState === 'ready' && detail !== null ? (
          <CallDetailPanel
            call={detail}
            participants={participants}
            participantsState={participantsState}
            events={events}
            eventsState={eventsState}
            myId={myId}
            nameOf={nameOf}
            counterpartName={counterpartName(detail)}
            acting={acting}
            nowMs={nowMs}
            onAnswer={(callId) => {
              void respond(callId, 'answer');
            }}
            onDecline={(callId) => {
              void respond(callId, 'decline');
            }}
            onBusy={(callId) => {
              void respond(callId, 'busy');
            }}
            onEnd={(callId) => {
              void respond(callId, 'end');
            }}
            onReload={(callId) => {
              void loadDetail(callId);
            }}
            onClose={() => {
              setSelectedCallId(null);
              setDetail(null);
            }}
          />
        ) : null}
        <InitiateCallDialog
          open={dialogOpen}
          conversations={conversations}
          groups={groups}
          candidatesState={candidatesState}
          onRetryCandidates={() => {
            void loadCandidates();
          }}
          onClose={() => {
            setDialogOpen(false);
          }}
          onConfirm={initiate}
        />
      </div>
    </div>
  );
}
