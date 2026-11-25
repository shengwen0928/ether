import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  compareSearchResults,
  dayLabel,
  decodeSearchCursor,
  encodeSearchCursor,
  filterContacts,
  filterConversations,
  formatMessageTime,
  groupMessagesByDay,
  initialsOf,
  messageMatchesSearchTerms,
  normalizeSearchTerms,
  presenceLabel,
  searchDateToSince,
  searchDateToUntil,
  sortConversations,
  totalUnread,
  validateEmail,
  validateLogin,
  validatePassword,
  validateRegister,
} from './.bundle/entry-logic.cjs';
import { DEMO_CONTACTS, DEMO_CONVERSATIONS, DEMO_MESSAGES } from './.bundle/entry-logic.cjs';

test('initialsOf handles names, single names, empty and CJK input', () => {
  assert.equal(initialsOf('林曉彤'), '林彤');
  assert.equal(initialsOf('Mei Chen'), 'MC');
  assert.equal(initialsOf('  '), '?');
  assert.equal(initialsOf('Ada'), 'A');
});

test('presenceLabel covers all four states', () => {
  assert.equal(presenceLabel('online'), '線上');
  assert.equal(presenceLabel('away'), '離開');
  assert.equal(presenceLabel('busy'), '忙碌中');
  assert.equal(presenceLabel('offline'), '離線');
});

test('formatMessageTime: today shows HH:MM, yesterday shows 昨天, older shows date', () => {
  const now = new Date('2026-11-25T13:05:00+08:00');
  assert.equal(formatMessageTime('2026-11-25T12:41:00+08:00', now), '12:41');
  assert.equal(formatMessageTime('2026-11-24T21:18:00+08:00', now), '昨天');
  assert.equal(formatMessageTime('2026-11-20T09:00:00+08:00', now), '11/20');
  assert.equal(formatMessageTime('2025-12-31T23:59:00+08:00', now), '2025/12/31');
  assert.equal(formatMessageTime('not-a-date', now), '');
});

test('dayLabel groups today and yesterday in words', () => {
  const now = new Date('2026-11-25T13:05:00+08:00');
  assert.equal(dayLabel('2026-11-25T08:00:00+08:00', now), '今天');
  assert.equal(dayLabel('2026-11-24T08:00:00+08:00', now), '昨天');
  assert.equal(dayLabel('2026-11-23T08:00:00+08:00', now), '2026 年 11 月 23 日');
});

test('groupMessagesByDay preserves order and splits days', () => {
  const now = new Date('2026-11-25T13:05:00+08:00');
  const groups = groupMessagesByDay(DEMO_MESSAGES, now);
  assert.equal(groups.length, 2);
  assert.equal(groups[0]?.label, '今天');
  assert.equal(groups[1]?.label, '昨天');
  assert.equal(groups[0]?.messages.length, 7);
  assert.equal(groups[1]?.messages.length, 2);
});

test('filterConversations matches name, handle and preview; filters unread and favorites', () => {
  const byId = new Map(DEMO_CONTACTS.map((contact) => [contact.id, contact]));
  const all = filterConversations(DEMO_CONVERSATIONS, byId, '', 'all');
  assert.equal(all.length, 5);
  const byName = filterConversations(DEMO_CONVERSATIONS, byId, '梓豪', 'all');
  assert.equal(byName.length, 1);
  const byPreview = filterConversations(DEMO_CONVERSATIONS, byId, '提案第三版', 'all');
  assert.equal(byPreview.length, 1);
  const unread = filterConversations(DEMO_CONVERSATIONS, byId, '', 'unread');
  assert.equal(unread.length, 2);
  const favorites = filterConversations(DEMO_CONVERSATIONS, byId, '', 'favorites');
  assert.equal(favorites.length, 3);
});

test('filterContacts filters by group and query', () => {
  assert.equal(filterContacts(DEMO_CONTACTS, '', 'all').length, 6);
  assert.equal(filterContacts(DEMO_CONTACTS, '', '工作').length, 2);
  assert.equal(filterContacts(DEMO_CONTACTS, '亭安', 'all').length, 1);
  assert.equal(filterContacts(DEMO_CONTACTS, '不存在', 'all').length, 0);
});

test('totalUnread sums unread counts', () => {
  assert.equal(totalUnread(DEMO_CONVERSATIONS), 3);
});

test('sortConversations puts pinned first, then most recent', () => {
  const sorted = sortConversations(DEMO_CONVERSATIONS);
  assert.equal(sorted[0]?.id, 'v-hao');
  assert.equal(sorted[1]?.id, 'v-mei');
  assert.equal(sorted[sorted.length - 1]?.id, 'v-wei');
});

test('validators accept valid input and report each problem', () => {
  assert.equal(validateEmail('you@example.com'), null);
  assert.equal(validateEmail(''), '請輸入 Email。');
  assert.equal(validateEmail('not-an-email'), 'Email 格式不正確。');
  assert.equal(validatePassword('12345678'), null);
  assert.equal(validatePassword('short'), '密碼至少需要 8 個字元。');
  assert.deepEqual(validateLogin('you@example.com', '12345678'), []);
  assert.equal(validateLogin('', '').length, 2);
  const registerErrors = validateRegister('', 'bad', '12345678', 'different');
  assert.equal(registerErrors.length, 3);
  assert.deepEqual(validateRegister('林曉彤', 'you@example.com', '12345678', '12345678'), []);
});

test('normalizeSearchTerms trims, lowercases and splits on whitespace', () => {
  assert.deepEqual(normalizeSearchTerms('  提案  客戶 '), ['提案', '客戶']);
  assert.deepEqual(normalizeSearchTerms('HELLO World'), ['hello', 'world']);
  assert.deepEqual(normalizeSearchTerms('   '), []);
  assert.deepEqual(normalizeSearchTerms('提案\t第二版'), ['提案', '第二版']);
});

test('messageMatchesSearchTerms requires every term as a case-insensitive substring', () => {
  const body = '早安，提案第二版客戶看過了。';
  assert.equal(messageMatchesSearchTerms(body, ['提案']), true);
  assert.equal(messageMatchesSearchTerms(body, ['提案', '客戶']), true);
  assert.equal(messageMatchesSearchTerms(body, ['提案', '沒有']), false);
  assert.equal(messageMatchesSearchTerms(body, ['提案第二版']), true);
  assert.equal(messageMatchesSearchTerms(body, []), false);
  assert.equal(messageMatchesSearchTerms('Hello World', ['hello']), true);
});

test('search cursor encodes the sort key and round-trips, rejecting bad input', () => {
  const cursor = encodeSearchCursor('2026-11-25T12:41:00.000Z', 'm-hao-3');
  assert.equal(typeof cursor, 'string');
  assert.ok(!cursor.includes('+') && !cursor.includes('/') && !cursor.includes('='));
  assert.deepEqual(decodeSearchCursor(cursor), {
    createdAt: '2026-11-25T12:41:00.000Z',
    messageId: 'm-hao-3',
  });
  assert.equal(decodeSearchCursor(''), null);
  assert.equal(decodeSearchCursor('not-a-cursor'), null);
  assert.equal(decodeSearchCursor(encodeSearchCursor('', '')), null);
});

test('compareSearchResults orders newest first with the message id as tiebreak', () => {
  const older = { createdAt: '2026-11-25T09:12:00+08:00', id: 'm-1' };
  const newer = { createdAt: '2026-11-25T12:41:00+08:00', id: 'm-2' };
  assert.ok(compareSearchResults(newer, older) < 0);
  assert.ok(compareSearchResults(older, newer) > 0);
  assert.ok(
    compareSearchResults(
      { createdAt: '2026-11-25T09:12:00+08:00', id: 'm-b' },
      { createdAt: '2026-11-25T09:12:00+08:00', id: 'm-a' },
    ) < 0,
  );
});

test('search date bounds map a local day to an inclusive ISO range', () => {
  const since = searchDateToSince('2026-11-25');
  const until = searchDateToUntil('2026-11-25');
  assert.ok(since !== null && until !== null);
  assert.equal(new Date(since).getHours(), 0);
  assert.equal(new Date(since).getMinutes(), 0);
  assert.equal(new Date(until).getHours(), 23);
  assert.equal(new Date(until).getMinutes(), 59);
  assert.equal(new Date(until).getSeconds(), 59);
  assert.ok(new Date(since) <= new Date(until));
  assert.equal(searchDateToSince(''), null);
  assert.equal(searchDateToUntil(''), null);
  assert.equal(searchDateToSince('not-a-date'), null);
});
