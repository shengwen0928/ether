import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ReactElement, ReactNode } from 'react';
import { totalUnread } from './data/logic';
import type { ChatDataProvider } from './data/provider';
import { createDemoProvider } from './data/provider';
import { useChatData } from './data/provider';
import { EtherApiClient, MemoryTokenStore } from './data/api';
import type { ApiChatDataProvider } from './data/api-provider';
import { createApiProvider } from './data/api-provider';
import type { OidcSession } from './data/oidc';
import { createOidcSession, resolveOidcConfig } from './data/oidc';
import { AppShell } from './components/AppShell';
import type { NavItem } from './components/AppShell';
import { Avatar } from './components/Avatar';
import { Badge, CountBadge } from './components/Badge';
import { Button } from './components/Button';
import { EmptyState, ErrorState } from './components/EmptyState';
import { Spinner } from './components/Spinner';
import {
  BellIcon,
  ChatIcon,
  FeedIcon,
  ImageIcon,
  MoonIcon,
  PaletteIcon,
  PhoneIcon,
  StickerIcon,
  SunIcon,
  UserIcon,
  UsersIcon,
} from './components/icons';
import { IconButton } from './components/IconButton';
import { ThemeProvider, useTheme } from './theme/theme';
import { ToastProvider, useToast } from './components/Toast';
import { Tooltip } from './components/Tooltip';
import { AuthView } from './views/AuthView';
import { CallsView } from './views/CallsView';
import { ChatsView } from './views/ChatsView';
import { ContactsView } from './views/ContactsView';
import { GroupsView } from './views/GroupsView';
import { MediaView } from './views/MediaView';
import { ProfileView } from './views/ProfileView';
import { ShowcaseView } from './views/ShowcaseView';
import { StickersView } from './views/StickersView';
import { TimelineView } from './views/TimelineView';

type MainView =
  | 'calls'
  | 'chats'
  | 'contacts'
  | 'groups'
  | 'media'
  | 'profile'
  | 'showcase'
  | 'stickers'
  | 'timeline';

type DataMode = 'demo' | 'api';

/**
 * Data mode resolution (M2a, extended M2d and the list migration).
 * Default is demo: nothing leaves the browser and every screen says
 * so. API mode is opt-in via VITE_DATA_MODE=api and connects
 * identity/profile, friends, conversations/messages and groups to
 * services/api. VITE_API_BASE_URL sets the API origin (empty =
 * same origin).
 *
 * OIDC (M2d): VITE_KEYCLOAK_ISSUER, VITE_KEYCLOAK_CLIENT_ID and
 * VITE_KEYCLOAK_REDIRECT_URI must all be set for the Keycloak login
 * path; otherwise the developer token form remains the fallback.
 * VITE_ALLOW_DEV_TOKEN_LOGIN=true keeps the token form visible even
 * when OIDC is configured (development only; never set in a
 * production build serving real accounts).
 */
function resolveDataMode(): {
  mode: DataMode;
  baseUrl: string;
  oidc: { issuer: string; clientId: string; redirectUri: string } | null;
  allowDevToken: boolean;
} {
  const env = import.meta.env as Record<string, string | undefined>;
  return {
    mode: env['VITE_DATA_MODE'] === 'api' ? 'api' : 'demo',
    baseUrl: env['VITE_API_BASE_URL'] ?? '',
    oidc: resolveOidcConfig(env),
    allowDevToken: env['VITE_ALLOW_DEV_TOKEN_LOGIN'] === 'true',
  };
}

const VIEW_TITLES: Record<MainView, string> = {
  calls: '通話',
  chats: '對話',
  contacts: '聯絡人',
  groups: '群組',
  media: '媒體',
  profile: '個人檔案與設定',
  showcase: '設計系統展示',
  stickers: '貼圖',
  timeline: '社群動態',
};

function EtherApp(): ReactElement {
  const { notify } = useToast();
  const { resolved, toggle } = useTheme();
  const [envMode] = useState(() => resolveDataMode());
  const [modeOverride, setModeOverride] = useState<DataMode | null>(null);
  const dataMode = modeOverride ?? envMode.mode;
  const apiSetup = useMemo<{
    provider: ApiChatDataProvider;
    session: OidcSession | null;
  } | null>(() => {
    if (dataMode !== 'api') {
      return null;
    }
    const session = envMode.oidc === null ? null : createOidcSession(envMode.oidc);
    // Tokens live in memory only (M2d): the OIDC session owns the
    // access-token store; without OIDC a bare memory store is used
    // for the developer token fallback.
    const tokenStore = session?.accessStore ?? new MemoryTokenStore();
    const client = new EtherApiClient({
      baseUrl: envMode.baseUrl,
      tokenStore,
      ...(session === null
        ? {}
        : {
            shouldRefreshTokens: () => session.isExpiringSoon(),
            refreshTokens: () => session.refresh(),
          }),
    });
    const provider = createApiProvider(
      client,
      tokenStore,
      session === null
        ? undefined
        : {
            session,
            pendingStorage: (globalThis as { sessionStorage?: Storage }).sessionStorage,
            navigate: (url: string) => {
              (globalThis as { location?: { assign: (url: string) => void } }).location?.assign(
                url,
              );
            },
          },
    );
    return { provider, session };
  }, [dataMode, envMode]);
  const demoProvider = useMemo(() => createDemoProvider(), []);
  const provider: ChatDataProvider = apiSetup === null ? demoProvider : apiSetup.provider;
  const apiProvider: ApiChatDataProvider | null = apiSetup?.provider ?? null;
  const snapshot = useChatData(provider);
  const [authenticated, setAuthenticated] = useState(false);
  // API 模式的未讀總數取後端清單（ChatsView 回報）；展示模式
  // 沿用展示快照的未讀數。
  const [apiUnread, setApiUnread] = useState(0);
  const [view, setView] = useState<MainView>('chats');
  const [selectedConversationId, setSelectedConversationId] = useState<string | null>(null);
  const [oidcCallback, setOidcCallback] = useState<
    { status: 'processing' } | { status: 'failed'; message: string | undefined } | null
  >(null);
  // StrictMode double-invokes effects in dev: an authorization code
  // is single-use, so the callback must be handled exactly once per
  // distinct query string.
  const handledCallbackRef = useRef<string | null>(null);

  useEffect(() => {
    if (apiSetup === null) {
      return;
    }
    const search = (globalThis as { location?: { search?: string } }).location?.search ?? '';
    const hasCallback = /(?:^|[?&])(?:code|error)=/.test(search);
    if (!hasCallback || apiSetup.session === null || handledCallbackRef.current === search) {
      void apiSetup.provider.initialize();
      return;
    }
    handledCallbackRef.current = search;
    setOidcCallback({ status: 'processing' });
    void apiSetup.provider.handleOidcCallback(search).then((result) => {
      const globals = globalThis as {
        history?: History;
        location?: { pathname?: string; hash?: string };
      };
      globals.history?.replaceState(
        null,
        '',
        `${globals.location?.pathname ?? '/'}${globals.location?.hash ?? ''}`,
      );
      if (result.status === 'signed-in' || result.status === 'ignored') {
        setOidcCallback(null);
      } else {
        setOidcCallback({ status: 'failed', message: result.message });
      }
    });
  }, [apiSetup]);

  const unread = dataMode === 'api' ? apiUnread : totalUnread(snapshot.conversations);

  const navigate = useCallback((id: string) => {
    setView(id as MainView);
  }, []);

  const startConversation = useCallback(
    (contactId: string) => {
      const conversation = snapshot.conversations.find((item) => item.contactId === contactId);
      if (conversation !== undefined) {
        setSelectedConversationId(conversation.id);
        setView('chats');
      }
    },
    [snapshot.conversations],
  );

  const navItems: NavItem[] = useMemo(
    () => [
      {
        id: 'chats',
        label: '對話',
        icon: <ChatIcon size={20} />,
        badge: unread > 0 ? <CountBadge count={unread} label={`${unread} 則未讀`} /> : undefined,
      },
      { id: 'contacts', label: '聯絡人', icon: <UsersIcon size={20} /> },
      { id: 'groups', label: '群組', icon: <UsersIcon size={20} /> },
      { id: 'media', label: '媒體', icon: <ImageIcon size={20} /> },
      { id: 'timeline', label: '社群動態', icon: <FeedIcon size={20} /> },
      { id: 'calls', label: '通話', icon: <PhoneIcon size={20} /> },
      { id: 'stickers', label: '貼圖', icon: <StickerIcon size={20} /> },
      { id: 'profile', label: '我的檔案', icon: <UserIcon size={20} /> },
      { id: 'showcase', label: '設計系統', icon: <PaletteIcon size={20} /> },
    ],
    [unread],
  );

  const connection = snapshot.apiConnection;
  const isAuthenticated = dataMode === 'demo' ? authenticated : connection?.state === 'ready';

  if (!isAuthenticated) {
    if (dataMode === 'api' && oidcCallback?.status === 'processing') {
      return (
        <div className="et-auth">
          <main className="et-auth__main">
            <div className="et-auth__card">
              <EmptyState
                icon={<Spinner label="正在完成登入" size={28} />}
                title="正在完成登入…"
                description="正在向身分服務換取 Token。"
              />
            </div>
          </main>
        </div>
      );
    }
    if (dataMode === 'api' && connection?.state === 'connecting') {
      return (
        <div className="et-auth">
          <main className="et-auth__main">
            <div className="et-auth__card">
              <EmptyState
                icon={<Spinner label="正在連接後端" size={28} />}
                title="正在連接後端…"
                description="正在向 API 驗證你的身分。"
              />
            </div>
          </main>
        </div>
      );
    }
    if (dataMode === 'api' && connection?.state === 'unreachable') {
      return (
        <div className="et-auth">
          <main className="et-auth__main">
            <div className="et-auth__card">
              <ErrorState
                title="無法連接後端服務"
                description={connection.message ?? '請確認 API 服務正在執行，然後再試一次。'}
                action={
                  <>
                    <Button
                      onClick={() => {
                        void apiProvider?.initialize();
                      }}
                    >
                      再試一次
                    </Button>{' '}
                    <Button
                      variant="secondary"
                      onClick={() => {
                        setModeOverride('demo');
                      }}
                    >
                      改用展示模式
                    </Button>
                  </>
                }
              />
            </div>
          </main>
        </div>
      );
    }
    return (
      <AuthView
        variant={dataMode}
        onAuthenticated={() => {
          setAuthenticated(true);
        }}
        oidcAvailable={apiSetup !== null && apiSetup.session !== null}
        oidcError={oidcCallback?.status === 'failed' ? oidcCallback.message : undefined}
        devTokenEnabled={apiSetup === null || envMode.allowDevToken || apiSetup.session === null}
        onKeycloakLogin={async (): Promise<string | null> => {
          try {
            await apiProvider?.signInWithKeycloak();
            return null;
          } catch (error) {
            return error instanceof Error ? error.message : '登入失敗。';
          }
        }}
        onTokenSubmit={async (token: string): Promise<string | null> => {
          try {
            await apiProvider?.signInWithToken(token);
            return null;
          } catch (error) {
            return error instanceof Error ? error.message : '驗證失敗。';
          }
        }}
        onSwitchToDemo={() => {
          setModeOverride('demo');
        }}
      />
    );
  }

  const headerActions: ReactNode = (
    <>
      <Tooltip tip="通知（展示）">
        <span>
          <IconButton
            label="通知"
            onClick={() => {
              notify({
                title: dataMode === 'api' ? '後端身分模式' : '展示模式',
                message: '通知中心尚未接線。',
                tone: 'warning',
              });
            }}
          >
            <BellIcon size={20} />
          </IconButton>
        </span>
      </Tooltip>
      <Tooltip tip={resolved === 'dark' ? '切換為淺色主題' : '切換為深色主題'}>
        <span>
          <IconButton
            label={resolved === 'dark' ? '切換為淺色主題' : '切換為深色主題'}
            onClick={toggle}
          >
            {resolved === 'dark' ? <SunIcon size={20} /> : <MoonIcon size={20} />}
          </IconButton>
        </span>
      </Tooltip>
      <Tooltip tip="前往個人檔案">
        <span>
          <IconButton
            label="前往個人檔案"
            onClick={() => {
              setView('profile');
            }}
          >
            <Avatar name={snapshot.profile.name} size="sm" presence={snapshot.profile.presence} />
          </IconButton>
        </span>
      </Tooltip>
    </>
  );

  const sidebarFooter: ReactNode = (
    <>
      <button
        type="button"
        className="et-shell__profile-mini"
        onClick={() => {
          setView('profile');
        }}
      >
        <Avatar name={snapshot.profile.name} size="md" presence={snapshot.profile.presence} />
        <span className="et-shell__profile-mini-text">
          <span className="et-shell__profile-mini-name">{snapshot.profile.name}</span>
          <span className="et-shell__profile-mini-status">
            {snapshot.profile.statusMessage !== ''
              ? snapshot.profile.statusMessage
              : snapshot.profile.handle}
          </span>
        </span>
      </button>
      {dataMode === 'api' ? (
        <Badge tone="success">後端身分 · 對話來自後端 API</Badge>
      ) : (
        <Badge tone="warning">展示資料 · 尚未連接後端</Badge>
      )}
    </>
  );

  return (
    <AppShell
      navItems={navItems}
      activeNav={view}
      onNavigate={navigate}
      headerTitle={VIEW_TITLES[view]}
      headerActions={headerActions}
      sidebarFooter={sidebarFooter}
    >
      {view === 'chats' ? (
        <ChatsView
          provider={provider}
          selectedId={selectedConversationId}
          onSelect={setSelectedConversationId}
          dataMode={dataMode}
          onUnreadChange={setApiUnread}
        />
      ) : null}
      {view === 'contacts' ? (
        <ContactsView
          provider={provider}
          dataMode={dataMode}
          onStartConversation={startConversation}
        />
      ) : null}
      {view === 'groups' ? <GroupsView provider={provider} dataMode={dataMode} /> : null}
      {view === 'media' ? <MediaView provider={provider} dataMode={dataMode} /> : null}
      {view === 'timeline' ? <TimelineView provider={provider} dataMode={dataMode} /> : null}
      {view === 'calls' ? <CallsView provider={provider} dataMode={dataMode} /> : null}
      {view === 'stickers' ? <StickersView provider={provider} dataMode={dataMode} /> : null}
      {view === 'profile' ? (
        <ProfileView
          provider={provider}
          dataMode={dataMode}
          onSignOut={() => {
            if (apiProvider !== null) {
              void apiProvider.signOut();
            }
            setAuthenticated(false);
            setView('chats');
            setSelectedConversationId(null);
          }}
        />
      ) : null}
      {view === 'showcase' ? <ShowcaseView /> : null}
    </AppShell>
  );
}

export default function App(): ReactElement {
  return (
    <ThemeProvider>
      <ToastProvider>
        <EtherApp />
      </ToastProvider>
    </ThemeProvider>
  );
}
