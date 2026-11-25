import { useState } from 'react';
import type { FormEvent, ReactElement } from 'react';
import { validateLogin, validateRegister, validateEmail, validatePassword } from '../data/logic';
import { Badge } from '../components/Badge';
import { Button } from '../components/Button';
import { Checkbox } from '../components/Checkbox';
import { Input, TextArea } from '../components/Input';
import { LogoLockup } from '../components/Logo';
import { useToast } from '../components/Toast';

type AuthMode = 'login' | 'register' | 'forgot' | 'reset' | 'expired';

export interface AuthViewProps {
  onAuthenticated: () => void;
  /** 'demo' (default): local showcase auth. 'api': backend identity via Keycloak OIDC or developer token. */
  variant?: 'demo' | 'api';
  /** API variant only: validates the pasted token; returns an error message or null on success. */
  onTokenSubmit?: (token: string) => Promise<string | null>;
  /** API variant only: leave API mode and fall back to the demo provider. */
  onSwitchToDemo?: () => void;
  /** API variant only: Keycloak OIDC is configured (VITE_KEYCLOAK_* all set). */
  oidcAvailable?: boolean;
  /** API variant only: starts the Keycloak login redirect; returns an error message or null. */
  onKeycloakLogin?: () => Promise<string | null>;
  /**
   * API variant only: show the developer token form. App enables it
   * only behind VITE_ALLOW_DEV_TOKEN_LOGIN when OIDC is available,
   * and as the fallback when OIDC is not configured.
   */
  devTokenEnabled?: boolean;
  /** API variant only: OIDC callback failure message from App, shown once. */
  oidcError?: string | undefined;
}

function fieldError(
  errors: Array<{ field: string; message: string }>,
  field: string,
): string | undefined {
  return errors.find((error) => error.field === field)?.message;
}

export function AuthView({
  onAuthenticated,
  variant = 'demo',
  onTokenSubmit,
  onSwitchToDemo,
  oidcAvailable = false,
  onKeycloakLogin,
  devTokenEnabled = true,
  oidcError,
}: AuthViewProps): ReactElement {
  const { notify } = useToast();
  const [mode, setMode] = useState<AuthMode>('login');
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [remember, setRemember] = useState(false);
  const [agreed, setAgreed] = useState(false);
  const [errors, setErrors] = useState<Array<{ field: string; message: string }>>([]);
  const [resetSent, setResetSent] = useState(false);
  const [resetDone, setResetDone] = useState(false);
  const [token, setToken] = useState('');
  const [tokenError, setTokenError] = useState<string | undefined>(undefined);
  const [tokenSubmitting, setTokenSubmitting] = useState(false);
  const [keycloakError, setKeycloakError] = useState<string | undefined>(undefined);
  const [keycloakSubmitting, setKeycloakSubmitting] = useState(false);

  const submitKeycloakLogin = async (): Promise<void> => {
    if (onKeycloakLogin === undefined) {
      return;
    }
    setKeycloakSubmitting(true);
    setKeycloakError(undefined);
    const failure = await onKeycloakLogin();
    setKeycloakSubmitting(false);
    if (failure !== null) {
      setKeycloakError(failure);
    }
    // 成功時瀏覽器已導向 Keycloak，本頁不會繼續停留。
  };

  const submitToken = async (event: FormEvent): Promise<void> => {
    event.preventDefault();
    const trimmed = token.trim();
    if (trimmed === '') {
      setTokenError('請貼上 Access Token。');
      return;
    }
    if (onTokenSubmit === undefined) {
      return;
    }
    setTokenSubmitting(true);
    setTokenError(undefined);
    const failure = await onTokenSubmit(trimmed);
    setTokenSubmitting(false);
    if (failure !== null) {
      setTokenError(failure);
      return;
    }
    onAuthenticated();
  };

  const switchMode = (next: AuthMode): void => {
    setMode(next);
    setErrors([]);
    setResetSent(false);
    setResetDone(false);
  };

  const submitLogin = (event: FormEvent): void => {
    event.preventDefault();
    const found = validateLogin(email, password);
    setErrors(found);
    if (found.length === 0) {
      notify({
        title: '展示模式',
        message: '尚未連接後端驗證，登入僅為介面展示。',
        tone: 'warning',
      });
      onAuthenticated();
    }
  };

  const submitRegister = (event: FormEvent): void => {
    event.preventDefault();
    const found = validateRegister(name, email, password, confirm);
    if (!agreed) {
      found.push({ field: 'agreed', message: '請先閱讀並同意服務條款與隱私權政策。' });
    }
    setErrors(found);
    if (found.length === 0) {
      notify({
        title: '展示模式',
        message: '尚未連接後端驗證，註冊僅為介面展示。',
        tone: 'warning',
      });
      onAuthenticated();
    }
  };

  const submitForgot = (event: FormEvent): void => {
    event.preventDefault();
    const emailError = validateEmail(email);
    if (emailError !== null) {
      setErrors([{ field: 'email', message: emailError }]);
      return;
    }
    setErrors([]);
    setResetSent(true);
  };

  const submitReset = (event: FormEvent): void => {
    event.preventDefault();
    const passwordError = validatePassword(password);
    const found: Array<{ field: string; message: string }> = [];
    if (passwordError !== null) {
      found.push({ field: 'password', message: passwordError });
    }
    if (confirm !== password) {
      found.push({ field: 'confirm', message: '兩次輸入的密碼不一致。' });
    }
    setErrors(found);
    if (found.length === 0) {
      setResetDone(true);
    }
  };

  return (
    <div className="et-auth">
      <section className="et-auth__brand" aria-label="ETHER 品牌介紹">
        <LogoLockup size={36} />
        <div className="et-auth__brand-copy">
          <h1 className="et-auth__brand-title">每一段連結，都值得被記住。</h1>
          <p className="et-auth__brand-text">
            ETHER 是一個以人為本的連結平台：對話、語音與回憶，都在同一個安靜、清晰的地方。
          </p>
          <ul className="et-auth__brand-points">
            <li>端對端加密的私人對話，內容只屬於你與對方。</li>
            <li>跨裝置的連續體驗，手機、平板與桌面無縫接續。</li>
            <li>把重要時刻存成回憶，時間過去，連結還在。</li>
          </ul>
        </div>
        <p className="et-auth__brand-foot">One Connection. Endless Possibilities.</p>
      </section>
      <main className="et-auth__main">
        <div className="et-auth__card">
          <div className="et-auth__head">
            {variant === 'api' ? (
              <Badge tone="success">後端身分模式</Badge>
            ) : (
              <Badge tone="warning">展示模式</Badge>
            )}
            {variant === 'api' ? (
              <>
                {oidcAvailable ? (
                  <>
                    <h2 className="et-auth__title">使用 Keycloak 登入</h2>
                    <p className="et-auth__subtitle">
                      將前往身分服務完成登入（授權碼流程＋PKCE），再回到這裡。密碼不會經過 ETHER
                      前端。
                    </p>
                  </>
                ) : (
                  <>
                    <h2 className="et-auth__title">以 Access Token 連接後端</h2>
                    <p className="et-auth__subtitle">
                      開發階段專用：貼上 Keycloak 核發的 Access Token，驗證你的身分。未設定 Keycloak
                      登入（VITE_KEYCLOAK_*），故顯示此表單。
                    </p>
                  </>
                )}
              </>
            ) : null}
            {variant === 'demo' && mode === 'login' ? (
              <>
                <h2 className="et-auth__title">歡迎回來</h2>
                <p className="et-auth__subtitle">登入你的 ETHER 帳號，繼續未完的對話。</p>
              </>
            ) : null}
            {variant === 'demo' && mode === 'register' ? (
              <>
                <h2 className="et-auth__title">建立帳號</h2>
                <p className="et-auth__subtitle">幾分鐘內開始你的第一段連結。</p>
              </>
            ) : null}
            {variant === 'demo' && mode === 'forgot' ? (
              <>
                <h2 className="et-auth__title">忘記密碼</h2>
                <p className="et-auth__subtitle">輸入註冊 Email，我們會寄出重設連結。</p>
              </>
            ) : null}
            {variant === 'demo' && mode === 'reset' ? (
              <>
                <h2 className="et-auth__title">重設密碼</h2>
                <p className="et-auth__subtitle">為你的帳號設定一組新密碼。</p>
              </>
            ) : null}
            {variant === 'demo' && mode === 'expired' ? (
              <>
                <h2 className="et-auth__title">工作階段已過期</h2>
                <p className="et-auth__subtitle">為了保護你的帳號，請重新登入。</p>
              </>
            ) : null}
          </div>

          {variant === 'api' ? (
            <div className="et-auth__form">
              {oidcError !== undefined ? (
                <p className="et-field__error" role="alert">
                  {oidcError}
                </p>
              ) : null}
              {keycloakError !== undefined ? (
                <p className="et-field__error" role="alert">
                  {keycloakError}
                </p>
              ) : null}
              {oidcAvailable ? (
                <Button
                  type="button"
                  size="lg"
                  fullWidth
                  loading={keycloakSubmitting}
                  onClick={() => {
                    void submitKeycloakLogin();
                  }}
                >
                  使用 Keycloak 登入
                </Button>
              ) : null}
              {devTokenEnabled ? (
                <form className="et-auth__form" onSubmit={submitToken} noValidate>
                  <TextArea
                    label="Access Token"
                    value={token}
                    rows={4}
                    placeholder="eyJhbGciOiJSUzI1NiIsInR5cCIg…"
                    error={tokenError}
                    onChange={(event) => setToken(event.target.value)}
                  />
                  <Button type="submit" size="lg" fullWidth loading={tokenSubmitting}>
                    驗證並進入
                  </Button>
                </form>
              ) : null}
            </div>
          ) : null}

          {variant === 'demo' && mode === 'login' ? (
            <form className="et-auth__form" onSubmit={submitLogin} noValidate>
              <Input
                label="Email"
                type="email"
                autoComplete="username"
                value={email}
                placeholder="you@example.com"
                error={fieldError(errors, 'email')}
                onChange={(event) => setEmail(event.target.value)}
              />
              <Input
                label="密碼"
                type="password"
                autoComplete="current-password"
                value={password}
                placeholder="至少 8 個字元"
                error={fieldError(errors, 'password')}
                onChange={(event) => setPassword(event.target.value)}
              />
              <div className="et-auth__row-between">
                <Checkbox
                  label="記住我"
                  checked={remember}
                  onChange={(event) => setRemember(event.target.checked)}
                />
                <button
                  type="button"
                  className="et-auth__link"
                  onClick={() => switchMode('forgot')}
                >
                  忘記密碼？
                </button>
              </div>
              <Button type="submit" size="lg" fullWidth>
                登入
              </Button>
            </form>
          ) : null}

          {variant === 'demo' && mode === 'register' ? (
            <form className="et-auth__form" onSubmit={submitRegister} noValidate>
              <Input
                label="顯示名稱"
                type="text"
                autoComplete="nickname"
                value={name}
                placeholder="例如：林曉彤"
                error={fieldError(errors, 'name')}
                onChange={(event) => setName(event.target.value)}
              />
              <Input
                label="Email"
                type="email"
                autoComplete="username"
                value={email}
                placeholder="you@example.com"
                error={fieldError(errors, 'email')}
                onChange={(event) => setEmail(event.target.value)}
              />
              <Input
                label="密碼"
                type="password"
                autoComplete="new-password"
                value={password}
                placeholder="至少 8 個字元"
                error={fieldError(errors, 'password')}
                onChange={(event) => setPassword(event.target.value)}
              />
              <Input
                label="確認密碼"
                type="password"
                autoComplete="new-password"
                value={confirm}
                error={fieldError(errors, 'confirm')}
                onChange={(event) => setConfirm(event.target.value)}
              />
              <div>
                <Checkbox
                  label="我已閱讀並同意服務條款與隱私權政策"
                  checked={agreed}
                  onChange={(event) => setAgreed(event.target.checked)}
                />
                {fieldError(errors, 'agreed') !== undefined ? (
                  <p className="et-field__error" role="alert">
                    {fieldError(errors, 'agreed')}
                  </p>
                ) : null}
              </div>
              <Button type="submit" size="lg" fullWidth>
                建立帳號
              </Button>
            </form>
          ) : null}

          {variant === 'demo' && mode === 'forgot' && !resetSent ? (
            <form className="et-auth__form" onSubmit={submitForgot} noValidate>
              <Input
                label="Email"
                type="email"
                autoComplete="username"
                value={email}
                placeholder="you@example.com"
                error={fieldError(errors, 'email')}
                onChange={(event) => setEmail(event.target.value)}
              />
              <Button type="submit" size="lg" fullWidth>
                寄出重設連結
              </Button>
            </form>
          ) : null}

          {variant === 'demo' && mode === 'forgot' && resetSent ? (
            <div className="et-auth__success" role="status">
              <strong>重設連結已寄出（展示）</strong>
              <p>
                如果 {email}{' '}
                已註冊，重設連結將寄到該信箱。展示模式下不會實際寄信，請直接前往重設密碼畫面查看表單。
              </p>
              <Button variant="secondary" onClick={() => switchMode('reset')}>
                前往重設密碼
              </Button>
            </div>
          ) : null}

          {variant === 'demo' && mode === 'reset' && !resetDone ? (
            <form className="et-auth__form" onSubmit={submitReset} noValidate>
              <Input
                label="新密碼"
                type="password"
                autoComplete="new-password"
                value={password}
                placeholder="至少 8 個字元"
                error={fieldError(errors, 'password')}
                onChange={(event) => setPassword(event.target.value)}
              />
              <Input
                label="確認新密碼"
                type="password"
                autoComplete="new-password"
                value={confirm}
                error={fieldError(errors, 'confirm')}
                onChange={(event) => setConfirm(event.target.value)}
              />
              <Button type="submit" size="lg" fullWidth>
                重設密碼
              </Button>
            </form>
          ) : null}

          {variant === 'demo' && mode === 'reset' && resetDone ? (
            <div className="et-auth__success" role="status">
              <strong>密碼已重設（展示）</strong>
              <p>你的密碼已更新。請使用新密碼登入。</p>
              <Button variant="secondary" onClick={() => switchMode('login')}>
                回到登入
              </Button>
            </div>
          ) : null}

          {variant === 'demo' && mode === 'expired' ? (
            <div className="et-auth__form">
              <div className="et-auth__success" role="status">
                <strong>你已安全登出</strong>
                <p>工作階段已過期，之前的對話內容不受影響。重新登入即可繼續。</p>
              </div>
              <Button size="lg" fullWidth onClick={() => switchMode('login')}>
                重新登入
              </Button>
              <Button variant="ghost" fullWidth onClick={() => switchMode('expired')}>
                模擬工作階段過期狀態
              </Button>
            </div>
          ) : null}

          <div className="et-auth__switch">
            {variant === 'demo' && mode === 'login' ? (
              <p>
                還沒有帳號？
                <button
                  type="button"
                  className="et-auth__link"
                  onClick={() => switchMode('register')}
                >
                  建立帳號
                </button>
              </p>
            ) : null}
            {variant === 'demo' &&
            (mode === 'register' || mode === 'forgot' || mode === 'reset') ? (
              <p>
                已經有帳號？
                <button type="button" className="et-auth__link" onClick={() => switchMode('login')}>
                  回到登入
                </button>
              </p>
            ) : null}
            {variant === 'demo' && mode === 'login' ? (
              <p>
                想要查看過期狀態？
                <button
                  type="button"
                  className="et-auth__link"
                  onClick={() => switchMode('expired')}
                >
                  模擬工作階段過期
                </button>
              </p>
            ) : null}
            {variant === 'api' ? (
              <>
                <p>
                  登入狀態只保留在這個分頁的記憶體中；關閉分頁或重新整理後需要重新登入。Token
                  不會寫入硬碟，也不會傳送到 ETHER 後端與身分服務以外的伺服器。
                </p>
                <p>
                  沒有帳號或無法登入？
                  <button
                    type="button"
                    className="et-auth__link"
                    onClick={() => {
                      onSwitchToDemo?.();
                    }}
                  >
                    改用展示模式
                  </button>
                </p>
              </>
            ) : (
              <p>展示模式：尚未連接後端驗證，輸入的資料不會傳送到任何伺服器。</p>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}
