import type {
  ChatMessage,
  Contact,
  Conversation,
  ConversationFilter,
  PresenceStatus,
} from './types';

/** Pure helpers for the chat UI. No I/O, no framework code: unit-tested directly. */

export function initialsOf(name: string): string {
  const trimmed = name.trim();
  if (trimmed === '') {
    return '?';
  }
  const parts = trimmed.split(/\s+/).filter((part) => part.length > 0);
  const first = parts[0] ?? '';
  if (parts.length > 1) {
    const last = parts[parts.length - 1] ?? '';
    return (first.charAt(0) + last.charAt(0)).toUpperCase();
  }
  // Single token: CJK names have no spaces, so use first + last character
  // (e.g. 林曉彤 → 林彤); single Latin words use the first letter only.
  const hasNonAscii = [...first].some((char) => (char.codePointAt(0) ?? 0) > 127);
  if (first.length >= 2 && hasNonAscii) {
    return (first.charAt(0) + first.charAt(first.length - 1)).toUpperCase();
  }
  return first.charAt(0).toUpperCase();
}

export function presenceLabel(presence: PresenceStatus): string {
  switch (presence) {
    case 'online':
      return '線上';
    case 'away':
      return '離開';
    case 'busy':
      return '忙碌中';
    case 'offline':
      return '離線';
  }
}

const MINUTE_MS = 60_000;
const DAY_MS = 86_400_000;

export function formatMessageTime(iso: string, now: Date): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const diff = now.getTime() - date.getTime();
  if (date.getTime() >= startOfToday && diff >= 0 && diff < DAY_MS) {
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    return `${hours}:${minutes}`;
  }
  const startOfYesterday = startOfToday - DAY_MS;
  if (date.getTime() >= startOfYesterday) {
    return '昨天';
  }
  const month = date.getMonth() + 1;
  const day = date.getDate();
  if (date.getFullYear() === now.getFullYear()) {
    return `${month}/${day}`;
  }
  return `${date.getFullYear()}/${month}/${day}`;
}

export function dayLabel(iso: string, now: Date): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  if (date.getTime() >= startOfToday) {
    return '今天';
  }
  if (date.getTime() >= startOfToday - DAY_MS) {
    return '昨天';
  }
  return `${date.getFullYear()} 年 ${date.getMonth() + 1} 月 ${date.getDate()} 日`;
}

export function groupMessagesByDay(
  messages: ChatMessage[],
  now: Date,
): Array<{
  label: string;
  messages: ChatMessage[];
}> {
  const groups: Array<{ label: string; messages: ChatMessage[] }> = [];
  for (const message of messages) {
    const label = dayLabel(message.sentAt, now);
    const last = groups[groups.length - 1];
    if (last !== undefined && last.label === label) {
      last.messages.push(message);
    } else {
      groups.push({ label, messages: [message] });
    }
  }
  return groups;
}

export function filterConversations(
  conversations: Conversation[],
  contactsById: ReadonlyMap<string, Contact>,
  query: string,
  filter: ConversationFilter,
): Conversation[] {
  const needle = query.trim().toLocaleLowerCase();
  return conversations.filter((conversation) => {
    if (filter === 'unread' && conversation.unreadCount === 0) {
      return false;
    }
    const contact = contactsById.get(conversation.contactId);
    if (filter === 'favorites' && contact?.isFavorite !== true) {
      return false;
    }
    if (needle === '') {
      return true;
    }
    const haystack =
      `${contact?.name ?? ''} ${contact?.handle ?? ''} ${conversation.lastMessagePreview}`.toLocaleLowerCase();
    return haystack.includes(needle);
  });
}

export function filterContacts(contacts: Contact[], query: string, group: string): Contact[] {
  const needle = query.trim().toLocaleLowerCase();
  return contacts.filter((contact) => {
    if (group !== 'all' && contact.group !== group) {
      return false;
    }
    if (needle === '') {
      return true;
    }
    return `${contact.name} ${contact.handle} ${contact.statusMessage}`
      .toLocaleLowerCase()
      .includes(needle);
  });
}

export function totalUnread(conversations: Conversation[]): number {
  return conversations.reduce((sum, conversation) => sum + conversation.unreadCount, 0);
}

export interface FieldError {
  field: string;
  message: string;
}

export function validateEmail(value: string): string | null {
  if (value.trim() === '') {
    return '請輸入 Email。';
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim())) {
    return 'Email 格式不正確。';
  }
  return null;
}

export function validatePassword(value: string): string | null {
  if (value === '') {
    return '請輸入密碼。';
  }
  if (value.length < 8) {
    return '密碼至少需要 8 個字元。';
  }
  return null;
}

export function validateLogin(email: string, password: string): FieldError[] {
  const errors: FieldError[] = [];
  const emailError = validateEmail(email);
  if (emailError !== null) {
    errors.push({ field: 'email', message: emailError });
  }
  const passwordError = validatePassword(password);
  if (passwordError !== null) {
    errors.push({ field: 'password', message: passwordError });
  }
  return errors;
}

export function validateRegister(
  name: string,
  email: string,
  password: string,
  confirm: string,
): FieldError[] {
  const errors = validateLogin(email, password);
  if (name.trim() === '') {
    errors.push({ field: 'name', message: '請輸入顯示名稱。' });
  }
  if (confirm !== password) {
    errors.push({ field: 'confirm', message: '兩次輸入的密碼不一致。' });
  }
  return errors;
}

export function sortConversations(conversations: Conversation[]): Conversation[] {
  return [...conversations].sort((a, b) => {
    if (a.isPinned !== b.isPinned) {
      return a.isPinned ? -1 : 1;
    }
    return b.lastMessageAt.localeCompare(a.lastMessageAt);
  });
}

/**
 * 訊息搜尋的純函式（§2v，後端 internal/search 的前端對映）。
 * 比對語義與後端一致：查詢去空白、小寫、依空白切分，AND 語義、
 * 子字串比對（含 CJK）、停用詞不預設；排序為建立時間降序、訊息
 * 識別碼降序 tiebreak；游標為排序鍵的不透明編碼。
 */
export function normalizeSearchTerms(query: string): string[] {
  return query
    .trim()
    .toLocaleLowerCase()
    .split(/\s+/)
    .filter((term) => term !== '');
}

export function messageMatchesSearchTerms(content: string, terms: string[]): boolean {
  if (terms.length === 0) {
    return false;
  }
  const haystack = content.toLocaleLowerCase();
  return terms.every((term) => haystack.includes(term));
}

function base64UrlEncodeText(text: string): string {
  const bytes = new TextEncoder().encode(text);
  let binary = '';
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  const base64 = btoa(binary);
  return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function base64UrlDecodeText(encoded: string): string | null {
  const normalized = encoded.replace(/-/g, '+').replace(/_/g, '/');
  const padded = normalized + '='.repeat((4 - (normalized.length % 4)) % 4);
  try {
    const binary = atob(padded);
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
    return new TextDecoder().decode(bytes);
  } catch {
    return null;
  }
}

/** 將排序鍵編碼為不透明游標（格式與後端 cursorOf 一致）。 */
export function encodeSearchCursor(createdAt: string, messageId: string): string {
  const instant = new Date(createdAt);
  const normalized = Number.isNaN(instant.getTime()) ? createdAt : instant.toISOString();
  return base64UrlEncodeText(`${normalized}\n${messageId}`);
}

/** 解碼搜尋游標；格式不符時回傳 null（呼叫端視為第一頁）。 */
export function decodeSearchCursor(
  cursor: string,
): { createdAt: string; messageId: string } | null {
  const decoded = base64UrlDecodeText(cursor);
  if (decoded === null) {
    return null;
  }
  const newline = decoded.indexOf('\n');
  if (newline < 0) {
    return null;
  }
  const createdAt = decoded.slice(0, newline);
  const messageId = decoded.slice(newline + 1);
  if (createdAt === '' || messageId === '' || Number.isNaN(Date.parse(createdAt))) {
    return null;
  }
  return { createdAt, messageId };
}

/** 搜尋結果排序鍵比較：建立時間降序、訊息識別碼降序。 */
export function compareSearchResults(
  a: { createdAt: string; id: string },
  b: { createdAt: string; id: string },
): number {
  const timeDelta = Date.parse(b.createdAt) - Date.parse(a.createdAt);
  if (timeDelta !== 0) {
    return timeDelta;
  }
  if (a.id === b.id) {
    return 0;
  }
  return a.id < b.id ? 1 : -1;
}

/**
 * 日期輸入（yyyy-mm-dd，本地時區）轉為搜尋 since／until 的
 * RFC3339 時間：since 取當日 00:00:00（含邊界），until 取當日
 * 23:59:59.999（含邊界）。空字串回傳 null。
 */
export function searchDateToSince(date: string): string | null {
  if (date === '') {
    return null;
  }
  const start = new Date(`${date}T00:00:00`);
  return Number.isNaN(start.getTime()) ? null : start.toISOString();
}

export function searchDateToUntil(date: string): string | null {
  if (date === '') {
    return null;
  }
  const end = new Date(`${date}T23:59:59.999`);
  return Number.isNaN(end.getTime()) ? null : end.toISOString();
}

export { MINUTE_MS };
