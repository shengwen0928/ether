import { useCallback, useEffect, useRef, useState } from 'react';
import type { FormEvent, ReactElement } from 'react';
import type { ChatDataProvider } from '../data/provider';
import { useChatData } from '../data/provider';
import type { DeviceItem, PresenceStatus, SessionItem } from '../data/types';
import { Avatar } from '../components/Avatar';
import { Badge } from '../components/Badge';
import { Button } from '../components/Button';
import { Dialog } from '../components/Dialog';
import { Input, TextArea } from '../components/Input';
import { RadioGroup } from '../components/Checkbox';
import { Select } from '../components/Select';
import { Switch } from '../components/Switch';
import { useToast } from '../components/Toast';
import { Divider } from '../components/StatusIndicator';
import { useTheme } from '../theme/theme';
import type { ThemePreference } from '../theme/theme';

export interface ProfileViewProps {
  provider: ChatDataProvider;
  dataMode?: 'demo' | 'api';
  onSignOut: () => void;
}

type DevicesState = 'loading' | 'ready' | 'error' | 'unsupported';

interface PendingRevoke {
  kind: 'session' | 'device';
  id: string;
  label: string;
}

function shortId(id: string): string {
  return id.length > 12 ? `${id.slice(0, 8)}…` : id;
}

function formatDateTime(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return iso;
  }
  return date.toLocaleString('zh-Hant', { dateStyle: 'medium', timeStyle: 'short' });
}

export function ProfileView({
  provider,
  dataMode = 'demo',
  onSignOut,
}: ProfileViewProps): ReactElement {
  const { notify } = useToast();
  const snapshot = useChatData(provider);
  const { preference, setPreference } = useTheme();
  const [name, setName] = useState(snapshot.profile.name);
  const [statusMessage, setStatusMessage] = useState(snapshot.profile.statusMessage);
  const [presence, setPresence] = useState<PresenceStatus>(snapshot.profile.presence);
  const [nameError, setNameError] = useState<string | undefined>(undefined);
  const [notifyMessages, setNotifyMessages] = useState(true);
  const [notifySound, setNotifySound] = useState(true);
  const [notifyPreview, setNotifyPreview] = useState(false);
  const [signOutOpen, setSignOutOpen] = useState(false);
  const [devicesState, setDevicesState] = useState<DevicesState>('loading');
  const [sessions, setSessions] = useState<SessionItem[]>([]);
  const [devices, setDevices] = useState<DeviceItem[]>([]);
  const [pendingRevoke, setPendingRevoke] = useState<PendingRevoke | null>(null);
  const [revoking, setRevoking] = useState(false);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const loadDevicesAndSessions = useCallback(async (): Promise<void> => {
    const listSessions = provider.listSessionsAsync;
    const listDevices = provider.listDevicesAsync;
    if (listSessions === undefined || listDevices === undefined) {
      if (mountedRef.current) {
        setDevicesState('unsupported');
      }
      return;
    }
    if (mountedRef.current) {
      setDevicesState('loading');
    }
    try {
      const [fetchedSessions, fetchedDevices] = await Promise.all([listSessions(), listDevices()]);
      if (!mountedRef.current) {
        return;
      }
      setSessions(fetchedSessions);
      setDevices(fetchedDevices);
      setDevicesState('ready');
    } catch {
      if (mountedRef.current) {
        setDevicesState('error');
      }
    }
  }, [provider]);

  useEffect(() => {
    void loadDevicesAndSessions();
  }, [loadDevicesAndSessions]);

  const confirmRevoke = async (): Promise<void> => {
    if (pendingRevoke === null) {
      return;
    }
    const target = pendingRevoke;
    setPendingRevoke(null);
    setRevoking(true);
    try {
      if (target.kind === 'session') {
        await provider.revokeSessionAsync?.(target.id);
      } else {
        await provider.revokeDeviceAsync?.(target.id);
      }
      notify({ title: '已撤銷', message: `已撤銷「${target.label}」。`, tone: 'success' });
      await loadDevicesAndSessions();
    } catch {
      notify({ title: '撤銷失敗', message: '請檢查連線後再試一次。', tone: 'error' });
    } finally {
      if (mountedRef.current) {
        setRevoking(false);
      }
    }
  };

  const save = async (event: FormEvent): Promise<void> => {
    event.preventDefault();
    if (name.trim() === '') {
      setNameError('請輸入顯示名稱。');
      return;
    }
    setNameError(undefined);
    const patch = { name: name.trim(), statusMessage: statusMessage.trim(), presence };
    if (provider.updateProfileAsync !== undefined) {
      try {
        await provider.updateProfileAsync(patch);
        notify({ title: '已儲存', message: '個人資料已同步至後端。', tone: 'success' });
      } catch {
        notify({
          title: '同步失敗',
          message: '後端拒絕了這次更新，畫面已還原為後端資料。',
          tone: 'error',
        });
      }
      return;
    }
    provider.updateProfile(patch);
    notify({ title: '已儲存', message: '個人資料已更新（本機展示）。', tone: 'success' });
  };

  const resetForm = (): void => {
    setName(snapshot.profile.name);
    setStatusMessage(snapshot.profile.statusMessage);
    setPresence(snapshot.profile.presence);
    setNameError(undefined);
  };

  return (
    <div className="et-page et-scroll">
      <div className="et-page__narrow">
        <header>
          <h2 className="et-page__title">個人檔案與設定</h2>
          <p className="et-page__description">
            {dataMode === 'api'
              ? '編輯後端帳號的公開檔案，變更會同步至 API。'
              : '編輯本機展示資料，變更不會傳送到任何伺服器。'}
          </p>
        </header>

        <div className="et-demo-note" role="note">
          {dataMode === 'api' ? (
            <>
              <Badge tone="success">後端身分</Badge>
              <span>這裡的個人資料來自後端 API。對話與聯絡人仍為展示資料（Phase 05 接線）。</span>
            </>
          ) : (
            <>
              <Badge tone="warning">展示資料</Badge>
              <span>這裡的個人資料來自展示資料層。後端帳號系統在 Phase 03 實作。</span>
            </>
          )}
        </div>

        <form className="et-card" onSubmit={save}>
          <div className="et-profile__head">
            <Avatar
              name={name === '' ? snapshot.profile.name : name}
              size="xl"
              presence={presence}
            />
            <div className="et-profile__head-text">
              <span className="et-profile__name">{name === '' ? snapshot.profile.name : name}</span>
              <span className="et-profile__handle">{snapshot.profile.handle}</span>
            </div>
          </div>
          <div className="et-form-grid">
            <Input
              label="顯示名稱"
              value={name}
              error={nameError}
              onChange={(event) => setName(event.target.value)}
            />
            <Select
              label="上線狀態"
              value={presence}
              options={[
                { value: 'online', label: '線上' },
                { value: 'away', label: '離開' },
                { value: 'busy', label: '忙碌中' },
                { value: 'offline', label: '離線' },
              ]}
              onChange={(event) => {
                setPresence(event.target.value as PresenceStatus);
              }}
            />
            <div className="et-field--full">
              <TextArea
                label="狀態訊息"
                value={statusMessage}
                rows={3}
                onChange={(event) => setStatusMessage(event.target.value)}
              />
            </div>
          </div>
          <div>
            <Button type="submit">儲存變更</Button>{' '}
            <Button type="button" variant="secondary" onClick={resetForm}>
              還原
            </Button>
          </div>
        </form>

        <section className="et-card" aria-labelledby="et-appearance-heading">
          <h3 className="et-card__title" id="et-appearance-heading">
            外觀
          </h3>
          <RadioGroup
            legend="主題"
            name="theme-preference"
            value={preference}
            options={[
              {
                value: 'system',
                label: '跟隨系統',
                description: '依裝置的深色／淺色設定自動切換。',
              },
              { value: 'dark', label: '深色', description: 'Ether Midnight 深色介面。' },
              { value: 'light', label: '淺色', description: 'Pure Light 淺色介面。' },
            ]}
            onChange={(value) => {
              setPreference(value as ThemePreference);
              notify({ message: '主題已切換（偏好存在本機）。', tone: 'success' });
            }}
          />
        </section>

        <section className="et-card" aria-labelledby="et-notify-heading">
          <h3 className="et-card__title" id="et-notify-heading">
            通知
          </h3>
          <div className="et-settings-group">
            <Switch
              label="訊息通知"
              description="收到新訊息時顯示通知。"
              checked={notifyMessages}
              onChange={setNotifyMessages}
            />
            <Switch
              label="通知音效"
              description="收到通知時播放提示音。"
              checked={notifySound}
              onChange={setNotifySound}
            />
            <Switch
              label="在通知中顯示訊息預覽"
              description="關閉後，通知只會顯示寄件者名稱。"
              checked={notifyPreview}
              onChange={setNotifyPreview}
            />
          </div>
          <p className="et-page__description">
            通知偏好目前只儲存在這個瀏覽器工作階段。推播接線在 Phase 03 之後實作。
          </p>
        </section>

        <section className="et-card" aria-labelledby="et-devices-heading">
          <h3 className="et-card__title" id="et-devices-heading">
            裝置與工作階段
          </h3>
          <p className="et-page__description">
            {dataMode === 'api'
              ? '以下清單來自後端 API。撤銷工作階段或裝置後，該裝置需要重新登入。'
              : '以下為展示資料，撤銷只會改變本機展示狀態，不會影響任何真實帳號。'}
          </p>
          {devicesState === 'loading' ? (
            <p className="et-page__description">載入裝置與工作階段中…</p>
          ) : devicesState === 'error' ? (
            <div>
              <p className="et-page__description">載入失敗，請檢查連線後再試一次。</p>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                onClick={() => {
                  void loadDevicesAndSessions();
                }}
              >
                重新載入
              </Button>
            </div>
          ) : devicesState === 'unsupported' ? (
            <p className="et-page__description">目前的資料模式不提供裝置與工作階段清單。</p>
          ) : (
            <>
              <h4 className="et-card__subtitle">已連結裝置</h4>
              {devices.length === 0 ? (
                <p className="et-page__description">沒有已連結的裝置。</p>
              ) : (
                <ul className="et-device-list">
                  {devices.map((device) => (
                    <li key={device.id} className="et-device-row">
                      <div className="et-device-row__main">
                        <span className="et-device-row__name">{device.name}</span>
                        <span className="et-device-row__meta">
                          {device.platform} · 上次活動{' '}
                          {device.lastActiveAt === null
                            ? '未知'
                            : formatDateTime(device.lastActiveAt)}
                        </span>
                      </div>
                      {device.isTrusted ? <Badge tone="success">受信任</Badge> : null}
                      {device.isRevoked ? <Badge tone="warning">已撤銷</Badge> : null}
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        disabled={device.isRevoked}
                        onClick={() => {
                          setPendingRevoke({
                            kind: 'device',
                            id: device.id,
                            label: device.name,
                          });
                        }}
                      >
                        撤銷
                      </Button>
                    </li>
                  ))}
                </ul>
              )}
              <h4 className="et-card__subtitle">工作階段</h4>
              {sessions.length === 0 ? (
                <p className="et-page__description">沒有進行中的工作階段。</p>
              ) : (
                <ul className="et-device-list">
                  {sessions.map((session) => (
                    <li key={session.id} className="et-device-row">
                      <div className="et-device-row__main">
                        <span className="et-device-row__name">工作階段 {shortId(session.id)}</span>
                        <span className="et-device-row__meta">
                          上次活動 {formatDateTime(session.lastActiveAt)} · 到期{' '}
                          {formatDateTime(session.expiresAt)}
                        </span>
                      </div>
                      {session.isRevoked ? <Badge tone="warning">已撤銷</Badge> : null}
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        disabled={session.isRevoked}
                        onClick={() => {
                          setPendingRevoke({
                            kind: 'session',
                            id: session.id,
                            label: `工作階段 ${shortId(session.id)}`,
                          });
                        }}
                      >
                        撤銷
                      </Button>
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}
          <p className="et-page__description">封鎖名單與隱私設定將在後續階段提供。</p>
          <Divider label="工作階段" />
          <div>
            <Button
              type="button"
              variant="danger"
              onClick={() => {
                setSignOutOpen(true);
              }}
            >
              {dataMode === 'api' ? '登出（撤銷後端工作階段）' : '登出展示工作階段'}
            </Button>
          </div>
        </section>
      </div>

      <Dialog
        open={signOutOpen}
        onClose={() => {
          setSignOutOpen(false);
        }}
        title="要登出嗎？"
        description={
          dataMode === 'api'
            ? '將撤銷後端工作階段並清除本機 Token，回到 Token 登入畫面。'
            : '展示工作階段將結束，回到登入畫面。本機展示資料會重置。'
        }
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => {
                setSignOutOpen(false);
              }}
            >
              取消
            </Button>
            <Button
              variant="danger"
              onClick={() => {
                setSignOutOpen(false);
                onSignOut();
              }}
            >
              登出
            </Button>
          </>
        }
      >
        <p>
          {dataMode === 'api'
            ? '登出後需要重新貼上 Access Token 才能進入。不會影響 Keycloak 帳號本身。'
            : '登出後需要重新進入展示模式，不會影響任何真實帳號（目前沒有連接後端）。'}
        </p>
      </Dialog>

      <Dialog
        open={pendingRevoke !== null}
        onClose={() => {
          setPendingRevoke(null);
        }}
        title="要撤銷嗎？"
        description={
          pendingRevoke === null
            ? ''
            : pendingRevoke.kind === 'session'
              ? `將撤銷「${pendingRevoke.label}」，該工作階段會立即失效。`
              : `將撤銷裝置「${pendingRevoke.label}」，該裝置需要重新登入。`
        }
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => {
                setPendingRevoke(null);
              }}
            >
              取消
            </Button>
            <Button
              variant="danger"
              disabled={revoking}
              onClick={() => {
                void confirmRevoke();
              }}
            >
              撤銷
            </Button>
          </>
        }
      >
        <p>撤銷後無法復原。如需繼續使用，請重新登入。</p>
      </Dialog>
    </div>
  );
}
