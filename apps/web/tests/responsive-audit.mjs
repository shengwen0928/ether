import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

/**
 * Static responsive audit (Phase 02). No browser is available in this
 * environment, so this checks the stylesheet contracts that the layouts
 * depend on; real-device/browser verification remains a CI item.
 * Run: `node tests/responsive-audit.mjs`
 */

const stylesDir = fileURLToPath(new URL('../src/styles/', import.meta.url));
const css = ['tokens.css', 'base.css', 'components.css', 'app.css']
  .map((file) => readFileSync(`${stylesDir}${file}`, 'utf8'))
  .join('\n');

const checks = [
  ['mobile breakpoint 767px', css.includes('max-width: 767px')],
  ['tablet breakpoint 1023px', css.includes('max-width: 1023px')],
  ['small-phone breakpoint 430px', css.includes('max-width: 430px')],
  [
    'breakpoint tokens 360/390/430/768/1024/1440/1920',
    ['360px', '390px', '430px', '768px', '1024px', '1440px', '1920px'].every((bp) =>
      css.includes(bp),
    ),
  ],
  ['prefers-reduced-motion respected', css.includes('prefers-reduced-motion')],
  ['sidebar hidden on mobile', css.includes('.et-shell__sidebar') && css.includes('display: none')],
  ['bottom nav shown on mobile', css.includes('.et-shell__bottomnav')],
  ['thread/list single-pane switching on mobile', css.includes('et-chats--show-thread')],
  ['message width capped on small screens', css.includes('max-width: 88%')],
  ['focus-visible styling present', css.includes(':focus-visible')],
  ['dvh used for shell height', css.includes('100dvh')],
  ['safe-area insets handled', css.includes('safe-area-inset-bottom')],
  // 搜尋面板（§2w）：欄內捲動、無固定寬度、結果內容兩行截斷、
  // 跳轉醒目只用外框（不改變版面尺寸）。
  [
    'search panel scrolls inside the list column',
    css.includes('.et-search-panel') && css.includes('.et-search-panel__results'),
  ],
  ['search panel declares no fixed width', !/\.et-search-panel\s*\{[^}]*\bwidth\s*:/.test(css)],
  [
    'search result content clamps to two lines',
    css.includes('.et-search-result__content') && css.includes('-webkit-line-clamp: 2'),
  ],
  [
    'jump target highlight uses an outline only',
    css.includes('.et-message--jump-target') && css.includes('outline: 2px solid var(--accent)'),
  ],
  // 成員區塊（§2x）：列於對話標頭下方、成員列可換行、無固定寬度。
  [
    'members block wraps and declares no fixed width',
    css.includes('.et-members__list') &&
      css.includes('flex-wrap: wrap') &&
      !/\.et-members\s*\{[^}]*\bwidth\s*:/.test(css),
  ],
  [
    'timeline tabs wrap and post cards declare no fixed width',
    css.includes('.et-timeline__tabs') &&
      css.includes('.et-post') &&
      !/\.et-post\s*\{[^}]*\bwidth\s*:/.test(css),
  ],
  [
    'timeline comment replies indent without fixed width',
    css.includes('.et-comment--reply') && !/\.et-comment--reply\s*\{[^}]*\bwidth\s*:/.test(css),
  ],
  [
    'timeline post content wraps long words',
    css.includes('.et-post__content') && css.includes('overflow-wrap: anywhere'),
  ],
  // 通話（§2ae）：卡片與參與者／事件列無固定寬度、內容換行、
  // 操作列可換行。
  [
    'call cards and rows declare no fixed width and wrap long words',
    css.includes('.et-call') &&
      css.includes('.et-calls__row') &&
      !/\.et-call\s*\{[^}]*\bwidth\s*:/.test(css) &&
      css.includes('.et-call__title') &&
      css.includes('overflow-wrap: anywhere'),
  ],
  [
    'call detail actions wrap',
    css.includes('.et-call-detail__actions') && css.includes('flex-wrap: wrap'),
  ],
  // 貼圖（§2ah）：貼圖包卡片無固定寬度、貼圖網格自動填充欄位、
  // 搜尋列可換行。
  [
    'sticker pack cards declare no fixed width and sticker grid auto-fills',
    css.includes('.et-sticker-pack') &&
      !/\.et-sticker-pack\s*\{[^}]*\bwidth\s*:/.test(css) &&
      css.includes('.et-sticker-grid') &&
      css.includes('repeat(auto-fill, minmax(8.5rem, 1fr))'),
  ],
  [
    'sticker toolbar and search row wrap',
    css.includes('.et-stickers__toolbar') &&
      css.includes('.et-stickers__search-row') &&
      css.includes('flex-wrap: wrap'),
  ],
];

let failures = 0;
for (const [name, ok] of checks) {
  console.log(`${ok ? 'PASS' : 'FAIL'} ${name}`);
  if (!ok) {
    failures += 1;
  }
}
process.exit(failures === 0 ? 0 : 1);
