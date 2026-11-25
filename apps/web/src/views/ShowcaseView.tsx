import { useState } from 'react';
import type { ReactElement } from 'react';
import { Avatar } from '../components/Avatar';
import { Badge, CountBadge } from '../components/Badge';
import { Button } from '../components/Button';
import { Checkbox, RadioGroup } from '../components/Checkbox';
import { ConversationItem } from '../components/ConversationItem';
import { Dialog } from '../components/Dialog';
import { Divider, StatusIndicator } from '../components/StatusIndicator';
import { Dropdown } from '../components/Dropdown';
import { EmptyState, ErrorState } from '../components/EmptyState';
import { BellIcon, SearchIcon, SettingsIcon } from '../components/icons';
import { IconButton } from '../components/IconButton';
import { Input, TextArea } from '../components/Input';
import { MessageBubble } from '../components/MessageBubble';
import { MessageComposer } from '../components/MessageComposer';
import { SearchField } from '../components/SearchField';
import { Select } from '../components/Select';
import { Skeleton } from '../components/Skeleton';
import { Spinner } from '../components/Spinner';
import { Switch } from '../components/Switch';
import { Tabs } from '../components/Tabs';
import { useToast } from '../components/Toast';
import { Tooltip } from '../components/Tooltip';
import { TypingIndicator } from '../components/TypingIndicator';
import type { ChatMessage, Contact, Conversation } from '../data/types';

const SHOWCASE_NOW = new Date('2026-11-25T13:00:00+08:00');

const SAMPLE_CONTACT: Contact = {
  id: 'c-show',
  name: '展示聯絡人',
  handle: '@showcase',
  statusMessage: '設計系統展示用聯絡人',
  presence: 'online',
  lastSeen: '線上',
  isFavorite: true,
  group: '朋友',
};

const SAMPLE_CONVERSATION: Conversation = {
  id: 'v-show',
  contactId: 'c-show',
  lastMessagePreview: '這是一則展示訊息預覽。',
  lastMessageAt: '2026-11-25T12:30:00+08:00',
  unreadCount: 3,
  isPinned: true,
  isMuted: false,
  isTyping: false,
};

const SAMPLE_MESSAGES: ChatMessage[] = [
  {
    id: 's-1',
    conversationId: 'v-show',
    senderId: 'c-show',
    body: '這是對方傳來的訊息氣泡，背景使用 Elevated Surface。',
    sentAt: '2026-11-25T12:28:00+08:00',
    status: 'read',
    kind: 'text',
  },
  {
    id: 's-2',
    conversationId: 'v-show',
    senderId: 'me',
    body: '這是自己傳出的訊息，使用品牌強調色，文字為深色。',
    sentAt: '2026-11-25T12:29:00+08:00',
    status: 'read',
    kind: 'text',
  },
  {
    id: 's-3',
    conversationId: 'v-show',
    senderId: 'c-show',
    body: '',
    sentAt: '2026-11-25T12:30:00+08:00',
    status: 'read',
    kind: 'voice',
    durationSeconds: 18,
  },
];

const SWATCHES: Array<{ name: string; value: string; variable: string }> = [
  { name: 'Ether Midnight', value: '#101923', variable: '--brand-midnight' },
  { name: 'Deep Nebula', value: '#182A3A', variable: '--brand-nebula' },
  { name: 'Ethereal Blue', value: '#86B9D2', variable: '--brand-ethereal' },
  { name: 'Silver Mist', value: '#C7D5DE', variable: '--brand-silver' },
  { name: 'Pure Light', value: '#F4F7F9', variable: '--brand-light' },
  { name: 'Aurora Cyan', value: '#65D4D6', variable: '--brand-aurora' },
  { name: 'Soft Violet', value: '#A99ACB', variable: '--brand-violet' },
  { name: 'Warm Horizon', value: '#D9B98C', variable: '--brand-horizon' },
];

const SEMANTIC_SWATCHES: Array<{ name: string; value: string; variable: string }> = [
  { name: '背景 / Base', value: 'var(--bg-base)', variable: '--bg-base' },
  { name: '表面 / Surface', value: 'var(--bg-surface)', variable: '--bg-surface' },
  { name: '主要文字', value: 'var(--text-primary)', variable: '--text-primary' },
  { name: '次要文字', value: 'var(--text-secondary)', variable: '--text-secondary' },
  { name: '強調色', value: 'var(--accent)', variable: '--accent' },
  { name: '成功', value: 'var(--success)', variable: '--success' },
  { name: '警告', value: 'var(--warning)', variable: '--warning' },
  { name: '錯誤', value: 'var(--error)', variable: '--error' },
];

function Section({
  title,
  children,
}: {
  title: string;
  children: ReactElement | ReactElement[];
}): ReactElement {
  return (
    <section className="et-showcase__section" aria-label={title}>
      <h3 className="et-showcase__section-title">{title}</h3>
      {children}
    </section>
  );
}

export function ShowcaseView(): ReactElement {
  const { notify } = useToast();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [tab, setTab] = useState('buttons');
  const [checked, setChecked] = useState(true);
  const [radioValue, setRadioValue] = useState('system');
  const [switchOn, setSwitchOn] = useState(true);
  const [search, setSearch] = useState('');

  return (
    <div className="et-page et-scroll">
      <div className="et-page__narrow et-showcase">
        <header>
          <h2 className="et-page__title">設計系統展示</h2>
          <p className="et-page__description">
            ETHER Design
            System：Token、元件與互動範例。所有元件皆可實際操作；展示資料皆標示為展示用途。
          </p>
        </header>

        <Section title="品牌色彩">
          <div className="et-showcase__grid">
            {SWATCHES.map((swatch) => (
              <div className="et-swatch" key={swatch.variable}>
                <span
                  className="et-swatch__chip"
                  style={{ background: `var(${swatch.variable})` }}
                />
                <span className="et-swatch__name">{swatch.name}</span>
                <span className="et-swatch__value">{swatch.value}</span>
              </div>
            ))}
          </div>
        </Section>

        <Section title="語意 Token（隨主題切換）">
          <div className="et-showcase__grid">
            {SEMANTIC_SWATCHES.map((swatch) => (
              <div className="et-swatch" key={swatch.variable}>
                <span
                  className="et-swatch__chip"
                  style={{ background: `var(${swatch.variable})` }}
                />
                <span className="et-swatch__name">{swatch.name}</span>
                <span className="et-swatch__value">{swatch.variable}</span>
              </div>
            ))}
          </div>
          <p className="et-page__description">
            切換右上角的主題按鈕，可即時看到語意 Token 在深色／淺色主題的映射。
          </p>
        </Section>

        <Section title="字體階層">
          <div className="et-type-sample">
            <span style={{ fontSize: 'var(--text-display)', lineHeight: 1.15, fontWeight: 700 }}>
              顯示字級 Display
            </span>
            <span className="et-type-sample__label">--text-display 36px</span>
            <span style={{ fontSize: 'var(--text-3xl)', fontWeight: 600 }}>
              標題 3XL · 每一段連結都值得被記住
            </span>
            <span className="et-type-sample__label">--text-3xl 24px</span>
            <span style={{ fontSize: 'var(--text-xl)' }}>標題 XL · 對話、語音與回憶</span>
            <span className="et-type-sample__label">--text-xl 18px</span>
            <span style={{ fontSize: 'var(--text-md)' }}>
              內文 MD · ETHER 使用 Inter（拉丁）與 Noto Sans TC（繁體中文），皆為 SIL OFL 授權。
            </span>
            <span className="et-type-sample__label">--text-md 14px · 行高 1.5</span>
            <span style={{ fontSize: 'var(--text-sm)', color: 'var(--text-secondary)' }}>
              次要文字 SM
            </span>
            <span className="et-type-sample__label">--text-sm 13px</span>
          </div>
        </Section>

        <Section title="按鈕">
          <div className="et-showcase__row">
            <Button>主要按鈕</Button>
            <Button variant="secondary">次要按鈕</Button>
            <Button variant="ghost">文字按鈕</Button>
            <Button variant="danger">危險操作</Button>
          </div>
          <div className="et-showcase__row">
            <Button size="sm">小型</Button>
            <Button size="md">中型</Button>
            <Button size="lg">大型</Button>
            <Button loading>載入中</Button>
            <Button disabled>停用</Button>
          </div>
          <div className="et-showcase__row">
            <IconButton label="搜尋（展示）">
              <SearchIcon size={20} />
            </IconButton>
            <IconButton label="通知（展示）" variant="filled">
              <BellIcon size={20} />
            </IconButton>
            <IconButton label="設定（展示）" variant="outline">
              <SettingsIcon size={20} />
            </IconButton>
            <Tooltip tip="這是工具提示">
              <span>
                <IconButton label="有提示的按鈕">
                  <SearchIcon size={20} />
                </IconButton>
              </span>
            </Tooltip>
          </div>
        </Section>

        <Section title="表單">
          <Input
            label="電子郵件"
            type="email"
            placeholder="you@example.com"
            hint="我們不會與第三方分享你的 Email。"
          />
          <Input label="有錯誤的欄位" defaultValue="不是正確格式" error="Email 格式不正確。" />
          <TextArea label="狀態訊息" rows={3} defaultValue="在城市與山之間，記得回訊息。" />
          <Select
            label="上線狀態"
            defaultValue="online"
            options={[
              { value: 'online', label: '線上' },
              { value: 'away', label: '離開' },
              { value: 'busy', label: '忙碌中' },
              { value: 'offline', label: '離線' },
            ]}
          />
          <Checkbox
            label="記住我"
            description="在這台裝置上保持登入。"
            checked={checked}
            onChange={(event) => setChecked(event.target.checked)}
          />
          <RadioGroup
            legend="主題偏好（展示）"
            name="showcase-theme"
            value={radioValue}
            options={[
              { value: 'system', label: '跟隨系統' },
              { value: 'dark', label: '深色' },
              { value: 'light', label: '淺色' },
            ]}
            onChange={setRadioValue}
          />
          <Switch
            label="訊息通知"
            description="收到新訊息時顯示通知。"
            checked={switchOn}
            onChange={setSwitchOn}
          />
          <SearchField value={search} onChange={setSearch} placeholder="搜尋展示資料" />
          <Dropdown
            label="更多操作"
            items={[
              { id: 'pin', label: '置頂對話' },
              { id: 'mute', label: '靜音對話' },
              { id: 'delete', label: '刪除對話', danger: true },
            ]}
            onSelect={(id) => {
              notify({ message: `已選擇「${id}」（展示，未執行任何操作）。`, tone: 'info' });
            }}
          />
        </Section>

        <Section title="徽章、頭像與狀態">
          <div className="et-showcase__row">
            <Badge tone="neutral">中性</Badge>
            <Badge tone="accent">強調</Badge>
            <Badge tone="success">成功</Badge>
            <Badge tone="warning">警告</Badge>
            <Badge tone="error">錯誤</Badge>
            <CountBadge count={5} />
            <CountBadge count={128} />
          </div>
          <div className="et-showcase__row">
            <Avatar name="林曉彤" size="xs" presence="online" />
            <Avatar name="林曉彤" size="sm" presence="away" />
            <Avatar name="林曉彤" size="md" presence="busy" />
            <Avatar name="林曉彤" size="lg" presence="offline" />
            <Avatar name="林曉彤" size="xl" presence="online" />
          </div>
          <div className="et-showcase__row">
            <StatusIndicator presence="online" showLabel />
            <StatusIndicator presence="away" showLabel />
            <StatusIndicator presence="busy" showLabel />
            <StatusIndicator presence="offline" showLabel />
          </div>
          <div className="et-showcase__row">
            <Spinner label="載入中示範" />
            <Skeleton width={220} height={14} />
            <Skeleton width={140} height={14} />
          </div>
        </Section>

        <Section title="對話元件">
          <div style={{ maxWidth: 420 }}>
            <ConversationItem
              conversation={SAMPLE_CONVERSATION}
              contact={SAMPLE_CONTACT}
              active
              now={SHOWCASE_NOW}
              onSelect={() => {
                notify({ message: '展示用的對話項目，點擊不會跳轉。', tone: 'info' });
              }}
            />
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
            {SAMPLE_MESSAGES.map((message) => (
              <MessageBubble
                key={message.id}
                message={message}
                isOwn={message.senderId === 'me'}
                senderName={message.senderId === 'me' ? '你' : SAMPLE_CONTACT.name}
                showSenderName={message.senderId !== 'me'}
                now={SHOWCASE_NOW}
              />
            ))}
            <span>
              正在輸入
              <TypingIndicator name={SAMPLE_CONTACT.name} />
            </span>
          </div>
          <MessageComposer
            onSend={(body) => {
              notify({
                message: `展示輸入區收到：「${body}」（未傳送到任何伺服器）。`,
                tone: 'success',
              });
            }}
            onAttach={() => {
              notify({ message: '附加檔案尚未接線（展示）。', tone: 'warning' });
            }}
            onEmoji={() => {
              notify({ message: '表情符號面板尚未接線（展示）。', tone: 'warning' });
            }}
          />
        </Section>

        <Section title="回饋與覆蓋層">
          <div className="et-showcase__row">
            <Button
              variant="secondary"
              onClick={() => notify({ message: '這是一則一般通知。', tone: 'info' })}
            >
              一般通知
            </Button>
            <Button
              variant="secondary"
              onClick={() => notify({ title: '完成', message: '操作成功。', tone: 'success' })}
            >
              成功通知
            </Button>
            <Button
              variant="secondary"
              onClick={() =>
                notify({ title: '注意', message: '這是一則警告通知。', tone: 'warning' })
              }
            >
              警告通知
            </Button>
            <Button
              variant="secondary"
              onClick={() =>
                notify({ title: '發生錯誤', message: '這是一則錯誤通知。', tone: 'error' })
              }
            >
              錯誤通知
            </Button>
            <Button onClick={() => setDialogOpen(true)}>開啟對話框</Button>
          </div>
          <Tabs
            label="展示用頁籤"
            value={tab}
            onChange={setTab}
            tabs={[
              { id: 'buttons', label: '按鈕' },
              { id: 'forms', label: '表單' },
              { id: 'feedback', label: '回饋' },
            ]}
          />
          <p className="et-page__description">
            目前選取的頁籤：{tab}。頁籤支援方向鍵切換，符合 WAI-ARIA Tabs 模式。
          </p>
          <Divider label="狀態範例" />
          <EmptyState
            title="這裡空空如也"
            description="空狀態用來告訴使用者：目前沒有內容，以及下一步可以做什麼。"
          />
          <ErrorState
            title="出了點問題"
            description="錯誤狀態會說明發生了什麼事，以及可以如何重試。"
            action={
              <Button
                variant="secondary"
                onClick={() =>
                  notify({ message: '重試為展示動作，未執行任何操作。', tone: 'info' })
                }
              >
                再試一次
              </Button>
            }
          />
        </Section>

        <Section title="響應式與動態">
          <p className="et-page__description">
            斷點：360／390／430（手機）、768（平板）、1024（小型筆電）、1440（桌面）、1920（大型桌面）。手機版採單欄布局：對話列表與內容分頁顯示，導覽移至底部；平板以上顯示左側邊欄。所有動畫皆尊重
            prefers-reduced-motion，動畫時間以 Token（120／180／280ms）統一管理。
          </p>
        </Section>
      </div>

      <Dialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        title="展示用對話框"
        description="對話框支援 Esc 關閉、點擊遮罩關閉，並會將焦點移入對話框內。"
        footer={
          <>
            <Button variant="secondary" onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button
              onClick={() => {
                setDialogOpen(false);
                notify({ message: '已確認（展示，未執行任何操作）。', tone: 'success' });
              }}
            >
              確認
            </Button>
          </>
        }
      >
        <p>這是對話框的內容區。實際產品中，這裡會放表單或確認說明。</p>
      </Dialog>
    </div>
  );
}
