import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

/**
 * WCAG contrast check for the Phase 02 token pairs (dark + light themes).
 * Reads the hex values straight from tokens.css so the check follows the
 * file, not a copied list. Run: `node tests/contrast-check.mjs`
 * Exit code 0 only when every pair meets WCAG 2.2 AA (4.5:1).
 */

function luminance(hex) {
  const channels = [1, 3, 5].map((index) => {
    const value = Number.parseInt(hex.slice(index, index + 2), 16) / 255;
    return value <= 0.03928 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
}

function ratio(foreground, background) {
  const l1 = luminance(foreground);
  const l2 = luminance(background);
  const [high, low] = l1 >= l2 ? [l1, l2] : [l2, l1];
  return (high + 0.05) / (low + 0.05);
}

const css = readFileSync(
  fileURLToPath(new URL('../src/styles/tokens.css', import.meta.url)),
  'utf8',
);

function section(theme) {
  const marker = theme === 'dark' ? ':root {' : ":root[data-theme='light'] {";
  const start = css.indexOf(marker);
  const end = css.indexOf('}', start);
  const block = css.slice(start, end);
  const vars = new Map();
  for (const match of block.matchAll(/--([\w-]+):\s*(#[0-9a-fA-F]{6})/g)) {
    vars.set(match[1], match[2].toLowerCase());
  }
  return vars;
}

const dark = section('dark');
const light = section('light');

const pairs = [
  ['dark', 'text-primary', 'bg-base'],
  ['dark', 'text-secondary', 'bg-base'],
  ['dark', 'text-tertiary', 'bg-base'],
  ['dark', 'text-primary', 'bg-surface'],
  ['dark', 'text-secondary', 'bg-surface'],
  // 搜尋面板（§2w）：結果時間（tertiary）與跳轉醒目外框（accent）
  // 皆落在對話清單欄的 bg-surface 上，沿既有文字／焦點配對。
  // 時間線貼文（§2ac）：作者／內容（primary）、時間與媒體詮釋
  // （tertiary）、回覆標籤（accent）皆落在 bg-surface 的貼文卡片
  // 上，沿既有配對，不新增配對。
  // 通話卡片（§2ae）：對象／狀態（primary）、時間與摘要
  // （secondary／tertiary）皆落在 bg-surface 的通話卡片上，沿既
  // 有配對，不新增配對。
  ['dark', 'text-tertiary', 'bg-surface'],
  ['dark', 'accent', 'bg-surface'],
  ['dark', 'accent-ink', 'accent'],
  ['dark', 'success', 'bg-base'],
  ['dark', 'warning', 'bg-base'],
  ['dark', 'error', 'bg-base'],
  ['light', 'text-primary', 'bg-base'],
  ['light', 'text-secondary', 'bg-base'],
  ['light', 'text-tertiary', 'bg-base'],
  ['light', 'text-primary', 'bg-surface'],
  ['light', 'text-secondary', 'bg-surface'],
  ['light', 'text-tertiary', 'bg-surface'],
  ['light', 'accent', 'bg-surface'],
  ['light', 'accent-ink', 'accent'],
  ['light', 'success', 'bg-base'],
  ['light', 'warning', 'bg-base'],
  ['light', 'error', 'bg-base'],
];

let failures = 0;
for (const [theme, fg, bg] of pairs) {
  const vars = theme === 'dark' ? dark : light;
  const fgHex = vars.get(fg);
  const bgHex = vars.get(bg);
  if (fgHex === undefined || bgHex === undefined) {
    console.log(`MISSING ${theme} ${fg} on ${bg}`);
    failures += 1;
    continue;
  }
  const value = ratio(fgHex, bgHex);
  const ok = value >= 4.5;
  if (!ok) {
    failures += 1;
  }
  console.log(
    `${ok ? 'PASS' : 'FAIL'} ${theme} ${fg} (${fgHex}) on ${bg} (${bgHex}): ${value.toFixed(2)}:1`,
  );
}
process.exit(failures === 0 ? 0 : 1);
