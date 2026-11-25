/**
 * OIDC 授權碼＋PKCE 登入流程（Web 端，M2d）。
 *
 * 依據：IDENTITY_AND_PERMISSIONS.md §3（授權碼流程須使用 PKCE，不得
 * 使用隱式流程；Client Secret 不得寫入前端可讀取的設定）與 §4
 * （Access Token 不得存於第三方腳本可讀取的位置；Refresh Token
 * Rotation 強制）；SECURITY_ARCHITECTURE.md §5（Web 端 Token 儲存
 * 方式未定稿——HttpOnly Cookie 或記憶體二選一）。
 *
 * 本里程碑的暫行選擇（待定稿覆核，見 STATUS.md §3 待決定 12）：
 * Access Token 與 Refresh Token 一律只存記憶體（MemoryTokenStore
 * ＋ OidcSession 行程內狀態），不寫入 localStorage／sessionStorage；
 * 分頁關閉或重新整理後須重新登入（Keycloak 的 SSO Cookie 可能讓
 * 再次授權不需重打帳密，屬身分服務行為，本端不依賴）。localStorage
 * 暫行作法（M2a）已退場，見 api.ts 的 LocalStorageTokenStore 註解。
 *
 * 實機邊界：本檔案只以 stub fetch 測試，未對真實 Keycloak 發出
 * 任何請求（見 STATUS.md §3 未驗證 2）。Keycloak 的
 * end_session_endpoint 登出（RP-Initiated Logout）未實作，列為
 * 後續事項；登出目前只呼叫 ETHER 後端 logout 並清除本地 Token。
 *
 * 安全規則：Token 永不記錄、不進 URL；PKCE 一律 S256，無 WebCrypto
 * 時明確報錯、不得降級為 plain；state 不符或授權錯誤時清除暫存
 * 並拒絕，不得以舊的暫存完成登入。
 */

import type { FetchFn, TokenStore } from './api';
import { MemoryTokenStore } from './api';

/** OIDC 設定：全部經 VITE_ 環境變數讀入，不得寫死任何 URL 或密鑰。 */
export interface OidcConfig {
  /** 發行者位址，如 http://localhost:8080/realms/ether（結尾斜線會去掉）。 */
  issuer: string;
  /** Public client 的 client_id，如 ether-web（無 secret）。 */
  clientId: string;
  /** 須與 Keycloak 註冊的 redirect URI 完全一致。 */
  redirectUri: string;
}

/**
 * 由環境變數表解析 OIDC 設定。三者缺一即回傳 null，PKCE 路徑
 * 停用、行為退回現況（展示模式／開發者 Token 登入）。
 */
export function resolveOidcConfig(env: Record<string, string | undefined>): OidcConfig | null {
  const issuer = env['VITE_KEYCLOAK_ISSUER']?.trim().replace(/\/+$/, '');
  const clientId = env['VITE_KEYCLOAK_CLIENT_ID']?.trim();
  const redirectUri = env['VITE_KEYCLOAK_REDIRECT_URI']?.trim();
  if (issuer === undefined || issuer === '' || clientId === undefined || clientId === '') {
    return null;
  }
  if (redirectUri === undefined || redirectUri === '') {
    return null;
  }
  return { issuer, clientId, redirectUri };
}

export interface OidcEndpoints {
  authorizationEndpoint: string;
  tokenEndpoint: string;
  endSessionEndpoint?: string | undefined;
}

/** 使用者可讀的 OIDC 失敗（不含 Token 內容）。 */
export class OidcError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'OidcError';
  }
}

export type StorageLike = {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
};

/** 摘要函式（SHA-256）：預設用 WebCrypto，可注入替身供測試。 */
export type DigestFn = (data: Uint8Array) => Promise<ArrayBuffer>;

function getCrypto(): {
  getRandomValues: (array: Uint8Array) => Uint8Array;
  subtle?: { digest: (algorithm: string, data: BufferSource) => Promise<ArrayBuffer> } | undefined;
} | null {
  const maybe = (globalThis as { crypto?: unknown }).crypto;
  if (maybe === null || typeof maybe !== 'object') {
    return null;
  }
  const crypto = maybe as {
    getRandomValues?: unknown;
    subtle?: unknown;
  };
  if (typeof crypto.getRandomValues !== 'function') {
    return null;
  }
  return crypto as {
    getRandomValues: (array: Uint8Array) => Uint8Array;
    subtle?:
      | { digest: (algorithm: string, data: BufferSource) => Promise<ArrayBuffer> }
      | undefined;
  };
}

function randomBytes(size: number): Uint8Array {
  const crypto = getCrypto();
  if (crypto === null) {
    throw new OidcError('此環境不支援安全的隨機數產生，無法開始登入。');
  }
  return crypto.getRandomValues(new Uint8Array(size));
}

function base64UrlEncode(bytes: Uint8Array): string {
  let binary = '';
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

/**
 * PKCE code_verifier：64 個隨機位元組的 base64url 編碼（86 字元，
 * 符合 RFC 7636 的 43–128 字元要求）。
 */
export function createCodeVerifier(): string {
  return base64UrlEncode(randomBytes(64));
}

/** state／nonce：32 個隨機位元組的 base64url 編碼。 */
export function randomOAuthString(): string {
  return base64UrlEncode(randomBytes(32));
}

async function sha256(data: Uint8Array, digest?: DigestFn | undefined): Promise<ArrayBuffer> {
  if (digest !== undefined) {
    return digest(data);
  }
  const subtle = getCrypto()?.subtle;
  if (subtle === undefined) {
    throw new OidcError(
      '此環境不支援 WebCrypto，無法建立 PKCE 驗證，登入已停止（不得降級為 plain）。',
    );
  }
  const copy = data.buffer.slice(data.byteOffset, data.byteOffset + data.byteLength);
  return subtle.digest('SHA-256', copy);
}

/**
 * PKCE S256 code_challenge：SHA-256(verifier) 的 base64url 編碼。
 * 無 WebCrypto 時拋錯，不得降級為 plain。
 */
export async function computeCodeChallenge(
  verifier: string,
  digest?: DigestFn | undefined,
): Promise<string> {
  const encoded = new TextEncoder().encode(verifier);
  const hash = await sha256(encoded, digest);
  return base64UrlEncode(new Uint8Array(hash));
}

/** 登入暫存（state／nonce／verifier）：只放 sessionStorage，完成即清。 */
export interface PendingLogin {
  state: string;
  nonce: string;
  codeVerifier: string;
}

export const PENDING_LOGIN_KEY = 'ether-oidc-pending-login';

export function savePendingLogin(storage: StorageLike, pending: PendingLogin): void {
  storage.setItem(PENDING_LOGIN_KEY, JSON.stringify(pending));
}

export function loadPendingLogin(storage: StorageLike): PendingLogin | null {
  const raw = storage.getItem(PENDING_LOGIN_KEY);
  if (raw === null) {
    return null;
  }
  try {
    const parsed: unknown = JSON.parse(raw) as unknown;
    if (parsed === null || typeof parsed !== 'object') {
      return null;
    }
    const record = parsed as { state?: unknown; nonce?: unknown; codeVerifier?: unknown };
    if (
      typeof record.state !== 'string' ||
      record.state === '' ||
      typeof record.nonce !== 'string' ||
      record.nonce === '' ||
      typeof record.codeVerifier !== 'string' ||
      record.codeVerifier === ''
    ) {
      return null;
    }
    return { state: record.state, nonce: record.nonce, codeVerifier: record.codeVerifier };
  } catch {
    return null;
  }
}

export function clearPendingLogin(storage: StorageLike): void {
  storage.removeItem(PENDING_LOGIN_KEY);
}

/** OIDC 授權回呼的解析結果。 */
export type CallbackParse =
  | { kind: 'code'; code: string; state: string }
  | { kind: 'error'; error: string; errorDescription?: string | undefined }
  | { kind: 'none' };

/** 解析授權回呼的查詢字串（?code=… 或 ?error=…；兩者皆無即 none）。 */
export function parseCallback(search: string): CallbackParse {
  const params = new URLSearchParams(search.startsWith('?') ? search : `?${search}`);
  const error = params.get('error');
  if (error !== null && error !== '') {
    const description = params.get('error_description');
    return {
      kind: 'error',
      error,
      errorDescription: description !== null && description !== '' ? description : undefined,
    };
  }
  const code = params.get('code');
  const state = params.get('state');
  if (code !== null && code !== '' && state !== null && state !== '') {
    return { kind: 'code', code, state };
  }
  return { kind: 'none' };
}

/** 授權錯誤碼轉可讀訊息（不含 Token，不洩漏身分服務細節）。 */
export function oidcErrorMessage(error: string, description?: string | undefined): string {
  switch (error) {
    case 'access_denied':
      return '你已拒絕授權，登入未完成。';
    case 'login_required':
    case 'consent_required':
    case 'interaction_required':
      return '身分服務要求重新登入，請再試一次。';
    case 'invalid_request':
    case 'invalid_grant':
    case 'unauthorized_client':
    case 'invalid_client':
      return '授權請求無效或授權碼已失效，請重新登入。';
    case 'server_error':
    case 'temporarily_unavailable':
      return '身分服務暫時無法回應，請稍後再試。';
    default:
      return description !== undefined ? `登入失敗：${description}` : '登入失敗，請重新再試。';
  }
}

function normalizeIssuer(issuer: string): string {
  return issuer.trim().replace(/\/+$/, '');
}

/**
 * OIDC discovery：由 .well-known 文件取得端點。issuer 須與文件所載
 * 一致（與後端 DiscoverJWKSURL 相同規則）；端點缺失即報錯，不得
 * 猜測 Keycloak 的預設路徑。
 */
export async function discoverOidcEndpoints(
  fetchFn: FetchFn,
  issuer: string,
): Promise<OidcEndpoints> {
  const discoveryUrl = `${normalizeIssuer(issuer)}/.well-known/openid-configuration`;
  let response: Response;
  try {
    response = await fetchFn(discoveryUrl, {
      method: 'GET',
      headers: { Accept: 'application/json' },
    });
  } catch {
    throw new OidcError('無法連接身分服務，請確認身分服務正在執行。');
  }
  if (!response.ok) {
    throw new OidcError('身分服務的探索文件無法取得，請確認發行者位址正確。');
  }
  const text = await response.text();
  let doc: unknown = null;
  try {
    doc = (text === '' ? null : JSON.parse(text)) as unknown;
  } catch {
    doc = null;
  }
  if (doc === null || typeof doc !== 'object') {
    throw new OidcError('身分服務的探索文件格式不正確。');
  }
  const record = doc as {
    issuer?: unknown;
    authorization_endpoint?: unknown;
    token_endpoint?: unknown;
    end_session_endpoint?: unknown;
  };
  if (
    typeof record.issuer === 'string' &&
    normalizeIssuer(record.issuer) !== normalizeIssuer(issuer)
  ) {
    throw new OidcError('身分服務的發行者與設定不符，登入已停止。');
  }
  if (
    typeof record.authorization_endpoint !== 'string' ||
    record.authorization_endpoint === '' ||
    typeof record.token_endpoint !== 'string' ||
    record.token_endpoint === ''
  ) {
    throw new OidcError('身分服務的探索文件缺少授權或 Token 端點。');
  }
  return {
    authorizationEndpoint: record.authorization_endpoint,
    tokenEndpoint: record.token_endpoint,
    endSessionEndpoint:
      typeof record.end_session_endpoint === 'string' && record.end_session_endpoint !== ''
        ? record.end_session_endpoint
        : undefined,
  };
}

/** 授權請求 URL（response_type=code、S256、openid profile email）。 */
export function buildAuthorizationUrl(
  endpoints: OidcEndpoints,
  config: OidcConfig,
  input: { state: string; nonce: string; codeChallenge: string },
): string {
  const url = new URL(endpoints.authorizationEndpoint);
  url.searchParams.set('response_type', 'code');
  url.searchParams.set('client_id', config.clientId);
  url.searchParams.set('redirect_uri', config.redirectUri);
  url.searchParams.set('scope', 'openid profile email');
  url.searchParams.set('state', input.state);
  url.searchParams.set('nonce', input.nonce);
  url.searchParams.set('code_challenge', input.codeChallenge);
  url.searchParams.set('code_challenge_method', 'S256');
  return url.toString();
}

/** Token 端點核發的 Token 組（expiry 由 expires_in 換算，未提供即未知）。 */
export interface TokenSet {
  accessToken: string;
  refreshToken: string | null;
  /** Access Token 到期時間（epoch ms）；null 表示未提供 expires_in。 */
  expiresAt: number | null;
}

function parseTokenResponse(data: unknown, nowMs: number): TokenSet {
  if (data === null || typeof data !== 'object') {
    throw new OidcError('身分服務的 Token 回應格式不正確。');
  }
  const record = data as {
    access_token?: unknown;
    refresh_token?: unknown;
    token_type?: unknown;
    expires_in?: unknown;
  };
  if (typeof record.access_token !== 'string' || record.access_token === '') {
    throw new OidcError('身分服務未核發有效的 Access Token。');
  }
  if (typeof record.token_type === 'string' && record.token_type.toLowerCase() !== 'bearer') {
    throw new OidcError('身分服務核發的 Token 型別不受支援。');
  }
  const refreshToken =
    typeof record.refresh_token === 'string' && record.refresh_token !== ''
      ? record.refresh_token
      : null;
  const expiresAt =
    typeof record.expires_in === 'number' &&
    Number.isFinite(record.expires_in) &&
    record.expires_in > 0
      ? nowMs + Math.floor(record.expires_in) * 1000
      : null;
  return { accessToken: record.access_token, refreshToken, expiresAt };
}

async function postTokenEndpoint(
  fetchFn: FetchFn,
  tokenEndpoint: string,
  body: URLSearchParams,
  nowMs: number,
): Promise<TokenSet> {
  let response: Response;
  try {
    response = await fetchFn(tokenEndpoint, {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body: body.toString(),
    });
  } catch {
    throw new OidcError('無法連接身分服務，請確認身分服務正在執行。');
  }
  const text = await response.text();
  let data: unknown = null;
  if (text !== '') {
    try {
      data = JSON.parse(text) as unknown;
    } catch {
      data = null;
    }
  }
  if (!response.ok) {
    if (data !== null && typeof data === 'object') {
      const record = data as { error?: unknown; error_description?: unknown };
      if (typeof record.error === 'string' && record.error !== '') {
        const description =
          typeof record.error_description === 'string' ? record.error_description : undefined;
        throw new OidcError(oidcErrorMessage(record.error, description));
      }
    }
    throw new OidcError('身分服務拒絕了 Token 請求，請重新登入。');
  }
  return parseTokenResponse(data, nowMs);
}

/** 以授權碼＋verifier 向 Token 端點換取 Token（public client，無 secret）。 */
export async function exchangeAuthorizationCode(
  fetchFn: FetchFn,
  tokenEndpoint: string,
  input: { code: string; codeVerifier: string; clientId: string; redirectUri: string },
  nowMs: number = Date.now(),
): Promise<TokenSet> {
  const body = new URLSearchParams();
  body.set('grant_type', 'authorization_code');
  body.set('code', input.code);
  body.set('redirect_uri', input.redirectUri);
  body.set('client_id', input.clientId);
  body.set('code_verifier', input.codeVerifier);
  return postTokenEndpoint(fetchFn, tokenEndpoint, body, nowMs);
}

/** 以 Refresh Token 換發（Rotation：新 Refresh Token 取代舊的）。 */
export async function refreshAccessToken(
  fetchFn: FetchFn,
  tokenEndpoint: string,
  input: { refreshToken: string; clientId: string },
  nowMs: number = Date.now(),
): Promise<TokenSet> {
  const body = new URLSearchParams();
  body.set('grant_type', 'refresh_token');
  body.set('refresh_token', input.refreshToken);
  body.set('client_id', input.clientId);
  return postTokenEndpoint(fetchFn, tokenEndpoint, body, nowMs);
}

/**
 * OIDC 工作階段：持有設定、端點快取與記憶體內的 Token 組。
 * Access Token 經 accessStore 供 EtherApiClient 讀取；Refresh Token
 * 與到期時間只存於此物件內，不得寫入任何持久化儲存。
 */
export interface OidcSession {
  readonly config: OidcConfig;
  /** 供 EtherApiClient 使用的 Access Token 儲存視圖（記憶體）。 */
  readonly accessStore: TokenStore;
  getEndpoints(): Promise<OidcEndpoints>;
  getRefreshToken(): string | null;
  /** 以授權碼＋verifier 換取 Token 組（不寫入工作階段，呼叫方決定）。 */
  exchangeCode(code: string, codeVerifier: string): Promise<TokenSet>;
  /** Access Token 是否將在 skewMs 內到期（到期時間未知時回 false）。 */
  isExpiringSoon(skewMs?: number): boolean;
  setTokens(tokens: TokenSet): void;
  clear(): void;
  /**
   * 以 Refresh Token 換發。成功回 true；無 Refresh Token 或任何
   * 失敗時清除本地 Token 並回 false（呼叫方導向重新登入）。
   */
  refresh(): Promise<boolean>;
}

export function createOidcSession(
  config: OidcConfig,
  options?: { fetchFn?: FetchFn | undefined; now?: (() => number) | undefined },
): OidcSession {
  const fetchFn = options?.fetchFn ?? fetch;
  const now = options?.now ?? Date.now;
  const accessStore = new MemoryTokenStore();
  let refreshToken: string | null = null;
  let expiresAt: number | null = null;
  let endpoints: OidcEndpoints | null = null;

  const getEndpoints = async (): Promise<OidcEndpoints> => {
    if (endpoints === null) {
      endpoints = await discoverOidcEndpoints(fetchFn, config.issuer);
    }
    return endpoints;
  };

  const clear = (): void => {
    accessStore.setToken(null);
    refreshToken = null;
    expiresAt = null;
  };

  return {
    config,
    accessStore,
    getEndpoints,
    getRefreshToken: (): string | null => refreshToken,
    exchangeCode: async (code: string, codeVerifier: string): Promise<TokenSet> => {
      const resolved = await getEndpoints();
      return exchangeAuthorizationCode(
        fetchFn,
        resolved.tokenEndpoint,
        {
          code,
          codeVerifier,
          clientId: config.clientId,
          redirectUri: config.redirectUri,
        },
        now(),
      );
    },
    isExpiringSoon: (skewMs: number = 60_000): boolean => {
      if (expiresAt === null || accessStore.getToken() === null || refreshToken === null) {
        return false;
      }
      return now() + skewMs >= expiresAt;
    },
    setTokens: (tokens: TokenSet): void => {
      accessStore.setToken(tokens.accessToken);
      refreshToken = tokens.refreshToken;
      expiresAt = tokens.expiresAt;
    },
    clear,
    refresh: async (): Promise<boolean> => {
      if (refreshToken === null) {
        return false;
      }
      try {
        const resolved = await getEndpoints();
        const next = await refreshAccessToken(
          fetchFn,
          resolved.tokenEndpoint,
          { refreshToken, clientId: config.clientId },
          now(),
        );
        accessStore.setToken(next.accessToken);
        refreshToken = next.refreshToken;
        expiresAt = next.expiresAt;
        return true;
      } catch {
        clear();
        return false;
      }
    },
  };
}

/** 開始 OIDC 登入：產生 PKCE 參數、存暫存、導向授權端點。 */
export async function beginOidcLogin(
  session: OidcSession,
  deps: { navigate: (url: string) => void; storage: StorageLike | undefined },
): Promise<void> {
  if (deps.storage === undefined) {
    throw new OidcError('此瀏覽器不支援工作階段儲存，無法開始登入。');
  }
  const endpoints = await session.getEndpoints();
  const codeVerifier = createCodeVerifier();
  const codeChallenge = await computeCodeChallenge(codeVerifier);
  const pending: PendingLogin = {
    state: randomOAuthString(),
    nonce: randomOAuthString(),
    codeVerifier,
  };
  savePendingLogin(deps.storage, pending);
  deps.navigate(
    buildAuthorizationUrl(endpoints, session.config, {
      state: pending.state,
      nonce: pending.nonce,
      codeChallenge,
    }),
  );
}

/**
 * 完成授權回呼：驗 state、換 Token、寫入工作階段、清暫存。
 * 回傳 null 表示查詢字串沒有回呼參數（不是一次登入回呼）。
 * 任何失敗皆拋 OidcError（暫存已清除，不得重試同一授權碼）。
 */
export async function completeOidcCallback(
  session: OidcSession,
  search: string,
  storage: StorageLike | undefined,
): Promise<TokenSet | null> {
  const parsed = parseCallback(search);
  if (parsed.kind === 'none') {
    return null;
  }
  if (parsed.kind === 'error') {
    if (storage !== undefined) {
      clearPendingLogin(storage);
    }
    throw new OidcError(oidcErrorMessage(parsed.error, parsed.errorDescription));
  }
  if (storage === undefined) {
    throw new OidcError('此瀏覽器不支援工作階段儲存，無法完成登入。');
  }
  const pending = loadPendingLogin(storage);
  if (pending === null) {
    clearPendingLogin(storage);
    throw new OidcError('登入狀態遺失（可能已逾時或在其他分頁完成），請重新登入。');
  }
  if (parsed.state !== pending.state) {
    clearPendingLogin(storage);
    throw new OidcError('登入狀態不符（state 驗證失敗），請重新登入。');
  }
  // 暫存已完成使命：授權碼只可用一次，交換前先清，避免失敗後
  // 以同一授權碼重試造成混淆。
  clearPendingLogin(storage);
  const tokens = await session.exchangeCode(parsed.code, pending.codeVerifier);
  session.setTokens(tokens);
  return tokens;
}
